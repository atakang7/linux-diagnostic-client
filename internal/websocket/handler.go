package websocket

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"diagnostic-client/internal/config"
	"diagnostic-client/internal/tunnel"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type Handler struct {
	cfg    *config.Config
	tunnel *tunnel.Handler
	// Map to track which file each client is viewing
	viewers map[*websocket.Conn]string
	mu      sync.RWMutex
}

func NewHandler(cfg *config.Config, tunnel *tunnel.Handler) *Handler {
	return &Handler{
		cfg:     cfg,
		tunnel:  tunnel,
		viewers: make(map[*websocket.Conn]string),
	}
}

type wsMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	// Start handler goroutines
	ctx, cancel := context.WithCancel(r.Context())
	defer func() {
		cancel()
		h.mu.Lock()
		delete(h.viewers, conn)
		h.mu.Unlock()
		conn.Close()
	}()

	// Handle client messages
	go h.readPump(ctx, conn)

	// Handle data streams
	h.writePump(ctx, conn)
}

func (h *Handler) readPump(ctx context.Context, conn *websocket.Conn) {
	for {
		var msg wsMessage
		err := conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket read error: %v", err)
			}
			return
		}

		switch msg.Type {
		case "view_file":
			var filePath string
			if err := json.Unmarshal(msg.Payload, &filePath); err != nil {
				continue
			}
			h.mu.Lock()
			h.viewers[conn] = filePath
			h.mu.Unlock()

		case "speed_control":
			var speed float64
			if err := json.Unmarshal(msg.Payload, &speed); err != nil {
				continue
			}
			// Store speed preference for this connection
			// Implementation depends on your rate limiting strategy
		}
	}
}

func (h *Handler) writePump(ctx context.Context, conn *websocket.Conn) {
	// Create ticker for network updates
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case packets := <-h.tunnel.NetworkStream():
			err := conn.WriteJSON(wsMessage{
				Type:    "network",
				Payload: json.RawMessage(mustMarshal(packets)),
			})
			if err != nil {
				return
			}

		case log := <-h.tunnel.LogStream():
			// Check if client is viewing this file
			h.mu.RLock()
			viewingFile := h.viewers[conn]
			h.mu.RUnlock()

			if viewingFile == log.Filename {
				err := conn.WriteJSON(wsMessage{
					Type:    "log",
					Payload: json.RawMessage(mustMarshal(log)),
				})
				if err != nil {
					return
				}
			}

		case file := <-h.tunnel.FileUpdates():
			err := conn.WriteJSON(wsMessage{
				Type:    "file_update",
				Payload: json.RawMessage(mustMarshal(file)),
			})
			if err != nil {
				return
			}

		case <-ticker.C:
			// Send ping to keep connection alive
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Helper function to handle JSON marshaling
func mustMarshal(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		log.Printf("Error marshaling JSON: %v", err)
		return []byte("{}")
	}
	return data
}
