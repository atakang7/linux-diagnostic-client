//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func unusedAddress(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	listener.Close()
	return address
}

func readJSONArray(t *testing.T, address string, target any) bool {
	t.Helper()
	resp, err := http.Get(address)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false
	}
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		t.Fatalf("decode %s: %v", address, err)
	}
	return true
}

func eventually(t *testing.T, description string, fn func() bool) {
	t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(150 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", description)
}

func TestTCPIngestionRESTAndWebSocket(t *testing.T) {
	binary := os.Getenv("CLIENT_BINARY")
	databaseURL := os.Getenv("DATABASE_URL")
	if binary == "" || databaseURL == "" {
		t.Fatal("CLIENT_BINARY and DATABASE_URL are required for integration tests")
	}

	httpAddr := unusedAddress(t)
	tcpAddr := unusedAddress(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmd := exec.CommandContext(ctx, binary)
	cmd.Env = append(os.Environ(),
		"SERVER_ADDR="+httpAddr, "AGENT_ADDR="+tcpAddr, "DATABASE_URL="+databaseURL)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		cancel()
		_ = cmd.Wait()
	}()

	baseURL := "http://" + httpAddr
	eventually(t, "HTTP startup", func() bool {
		var files []json.RawMessage
		return readJSONArray(t, baseURL+"/api/files", &files)
	})

	headers := http.Header{"Origin": []string{baseURL}}
	ws, _, err := websocket.DefaultDialer.Dial("ws://"+httpAddr+"/ws", headers)
	if err != nil {
		t.Fatalf("websocket handshake: %v", err)
	}
	defer ws.Close()

	// Every subscribed client should receive the update, not just whichever
	// goroutine happened to consume a shared upstream channel.
	otherWS, _, err := websocket.DefaultDialer.Dial("ws://"+httpAddr+"/ws", headers)
	if err != nil {
		t.Fatalf("second websocket handshake: %v", err)
	}
	defer otherWS.Close()

	var agent net.Conn
	eventually(t, "agent listener", func() bool {
		var err error
		agent, err = net.DialTimeout("tcp", tcpAddr, 500*time.Millisecond)
		return err == nil
	})
	defer agent.Close()

	encoder := json.NewEncoder(agent)
	modified := time.Now().UTC().Format(time.RFC3339Nano)
	file := map[string]any{
		"path": "/var/log/diagnostic-e2e.log",
		"parent_path": "/var/log",
		"name": "diagnostic-e2e.log",
		"is_directory": false, "size": 123,
		"mod_time": modified, "is_gzipped": false, "is_scraped": false,
	}
	if err := encoder.Encode(map[string]any{
		"type": "log_list", "payload": []any{file},
	}); err != nil {
		t.Fatal(err)
	}

	// The WebSocket must deliver the same file update to both viewers.
	for _, subscriber := range []*websocket.Conn{ws, otherWS} {
		_ = subscriber.SetReadDeadline(time.Now().Add(10 * time.Second))
		for {
			var event struct {
				Type string `json:"type"`
			Payload json.RawMessage `json:"payload"`
		}
		if err := subscriber.ReadJSON(&event); err != nil {
			t.Fatalf("did not observe file_update over WebSocket: %v", err)
		}
			if event.Type == "file_update" {
				if !strings.Contains(string(event.Payload), "diagnostic-e2e.log") {
					t.Fatalf("unexpected file event: %s", event.Payload)
				}
				break
			}
		}
	}

	fileURL := baseURL + "/api/files?path=" + url.QueryEscape("/var/log") + "&depth=2"
	eventually(t, "nested file-tree endpoint", func() bool {
		var files []map[string]any
		if !readJSONArray(t, fileURL, &files) {
			return false
		}
		for _, f := range files {
			if f["path"] == "/var/log/diagnostic-e2e.log" {
				return true
			}
		}
		return false
	})

	occurrence := time.Now().UTC().Format(time.RFC3339Nano)
	entry := map[string]any{
		"filename": "/var/log/diagnostic-e2e.log",
		"line": "diagnostic e2e error detected",
		"line_num": 1,
		"timestamp": occurrence,
		"level": "error",
	}
	if err := encoder.Encode(map[string]any{
		"type": "log_data", "payload": []any{entry},
	}); err != nil {
		t.Fatal(err)
	}
	logURL := baseURL + "/api/logs?file=" + url.QueryEscape("/var/log/diagnostic-e2e.log")
	eventually(t, "persisted agent logs", func() bool {
		var rows []map[string]any
		if !readJSONArray(t, logURL, &rows) {
			return false
		}
		for _, row := range rows {
			if row["line"] == "diagnostic e2e error detected" {
				return true
			}
		}
		return false
	})

	now := time.Now().UTC()
	networkPayload := map[string]any{
		"timestamp": now.Format(time.RFC3339Nano),
		"packets": []any{map[string]any{
			"timestamp": now.Format(time.RFC3339Nano),
			"protocol": "TCP", "src_ip": "10.0.0.1", "dst_ip": "10.0.0.2",
			"src_port": 5123, "dst_port": 443, "length": 64, "payload_size": 10,
			"tcp_flags": "SYN",
		}},
	}
	if err := encoder.Encode(map[string]any{
		"type": "metrics", "payload": networkPayload,
	}); err != nil {
		t.Fatal(err)
	}
	start := url.QueryEscape(now.Add(-time.Minute).Format(time.RFC3339))
	end := url.QueryEscape(now.Add(time.Minute).Format(time.RFC3339))
	metricsURL := fmt.Sprintf("%s/api/network/metrics?start=%s&end=%s&protocol=TCP", baseURL, start, end)
	lastResponse := ""
	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("last network response: %s", lastResponse)
		}
	})
	eventually(t, "persisted network packet", func() bool {
		response, err := http.Get(metricsURL)
		if err != nil {
			lastResponse = err.Error()
			return false
		}
		defer response.Body.Close()
		var stats struct {
			PacketCount int64 `json:"packet_count"`
			Packets []struct {
				SrcIP string `json:"src_ip"`
				Protocol string `json:"protocol"`
			} `json:"packets"`
		}
		if response.StatusCode != http.StatusOK {
			lastResponse = fmt.Sprintf("HTTP %d", response.StatusCode)
			return false
		}
		if err := json.NewDecoder(response.Body).Decode(&stats); err != nil {
			lastResponse = "invalid JSON: " + err.Error()
			return false
		}
		lastResponse = fmt.Sprintf("packet_count=%d packets=%+v", stats.PacketCount, stats.Packets)
		if stats.PacketCount < 1 {
			return false
		}
		for _, row := range stats.Packets {
			if row.SrcIP == "10.0.0.1" && row.Protocol == "TCP" {
				return true
			}
		}
		return false
	})
}
