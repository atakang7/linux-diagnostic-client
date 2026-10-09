package websocket

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
)

func TestWebSocketOriginPolicy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		_ = conn.Close()
	}))
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http")

	for _, tc := range []struct {
		name   string
		origin string
		allow  bool
	}{
		{"same origin", server.URL, true},
		{"foreign origin", "https://untrusted.example", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			headers := http.Header{}
			headers.Set("Origin", tc.origin)
			conn, resp, err := websocket.DefaultDialer.Dial(url, headers)
			if conn != nil {
				_ = conn.Close()
			}
			if tc.allow {
				if err != nil {
					t.Fatalf("same-origin upgrade rejected: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("cross-origin upgrade was allowed")
			}
			if resp == nil || resp.StatusCode != http.StatusForbidden {
				t.Fatalf("cross-origin response = %v, want 403", resp)
			}
		})
	}
}
