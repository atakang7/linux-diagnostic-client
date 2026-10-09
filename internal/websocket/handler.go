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

const (
	clientQueueSize = 128
	pingInterval    = 30 * time.Second
)

type wsMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// Handler consumes each upstream stream once and fans out events to connected viewers.
type Handler struct {
	cfg     *config.Config
	tunnel  *tunnel.Handler
	mu      sync.RWMutex
	viewers map[*websocket.Conn]string
	clients map[*websocket.Conn]chan wsMessage
	done    chan struct{}
	once    sync.Once
}

func NewHandler(cfg *config.Config, stream *tunnel.Handler) *Handler {
	h := &Handler{
		cfg:     cfg,
		tunnel:  stream,
		viewers: make(map[*websocket.Conn]string),
		clients: make(map[*websocket.Conn]chan wsMessage),
		done:    make(chan struct{}),
	}
	go h.forwardEvents()
	return h
}

func (h *Handler) forwardEvents() {
	for {
		select {
		case <-h.done:
			return
		case packets, ok := <-h.tunnel.NetworkStream():
			if !ok {
				return
			}
			h.broadcast("network", packets, "")
		case entry, ok := <-h.tunnel.LogStream():
			if !ok {
				return
			}
			h.broadcast("log", entry, entry.Filename)
		case file, ok := <-h.tunnel.FileUpdates():
			if !ok {
				return
			}
			h.broadcast("file_update", file, "")
		}
	}
}

func (h *Handler) broadcast(kind string, payload any, filterPath string) {
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("WebSocket marshal %s: %v", kind, err)
		return
	}
	message := wsMessage{Type: kind, Payload: data}

	h.mu.RLock()
	defer h.mu.RUnlock()
	for conn, queue := range h.clients {
		if filterPath != "" && h.viewers[conn] != filterPath {
			continue
		}
		select {
		case queue <- message:
		default:
			// A slow subscriber must not block delivery to other viewers.
			log.Printf("WebSocket subscriber queue full; dropping %s event", kind)
		}
	}
}

func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	ctx, cancel := context.WithCancel(r.Context())
	queue := make(chan wsMessage, clientQueueSize)

	h.mu.Lock()
	h.clients[conn] = queue
	h.mu.Unlock()

	defer func() {
		cancel()
		h.mu.Lock()
		delete(h.clients, conn)
		delete(h.viewers, conn)
		h.mu.Unlock()
		_ = conn.Close()
	}()

	go h.readPump(conn, cancel)
	h.writePump(ctx, conn, queue)
}

func (h *Handler) readPump(conn *websocket.Conn, cancel context.CancelFunc) {
	defer cancel()
	for {
		var msg wsMessage
		if err := conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket read error: %v", err)
			}
			return
		}
		if msg.Type == "view_file" {
			var filePath string
			if json.Unmarshal(msg.Payload, &filePath) != nil {
				continue
			}
			h.mu.Lock()
			h.viewers[conn] = filePath
			h.mu.Unlock()
		}
	}
}

func (h *Handler) writePump(ctx context.Context, conn *websocket.Conn, queue <-chan wsMessage) {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-h.done:
			return
		case event := <-queue:
			if err := conn.WriteJSON(event); err != nil {
				return
			}
		case <-ticker.C:
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (h *Handler) Close() {
	h.once.Do(func() { close(h.done) })
}
