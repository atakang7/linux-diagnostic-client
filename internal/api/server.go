package api

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"diagnostic-client/internal/config"
	"diagnostic-client/internal/db"
	"diagnostic-client/internal/tunnel"
	"diagnostic-client/internal/websocket"
)

type Server struct {
	cfg    *config.Config
	db     *db.DB
	tunnel *tunnel.Handler
	ws     *websocket.Handler
	http   *Handler
	server *http.Server
}

func NewServer(cfg *config.Config, db *db.DB) *Server {
	// Initialize components
	tunnelHandler := tunnel.NewHandler(cfg, db)
	wsHandler := websocket.NewHandler(cfg, tunnelHandler)
	httpHandler := NewHandler(db)

	// Create server with routing
	mux := http.NewServeMux()

	// WebSocket endpoint
	mux.HandleFunc("/ws", wsHandler.ServeWS)

	// REST endpoints
	mux.HandleFunc("/api/files", httpHandler.GetFiles)
	mux.HandleFunc("/api/logs", httpHandler.GetLogs)
	mux.HandleFunc("/api/logs/search", httpHandler.SearchLogs)
	mux.HandleFunc("/api/network/metrics", httpHandler.GetNetworkMetrics)

	// Create HTTP server with timeouts
	server := &http.Server{
		Addr:         cfg.ServerAddr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &Server{
		cfg:    cfg,
		db:     db,
		tunnel: tunnelHandler,
		ws:     wsHandler,
		http:   httpHandler,
		server: server,
	}
}

func (s *Server) Run(ctx context.Context) error {
	// Bind both listeners before reporting startup as successful.
	httpListener, err := net.Listen("tcp", s.cfg.ServerAddr)
	if err != nil {
		return fmt.Errorf("bind HTTP listener: %w", err)
	}
	defer httpListener.Close()

	tunnelServer, err := tunnel.NewServer(s.cfg, s.tunnel)
	if err != nil {
		return fmt.Errorf("bind agent listener: %w", err)
	}
	defer tunnelServer.Close()

	errCh := make(chan error, 2)
	go func() {
		if err := tunnelServer.Run(ctx); err != nil && ctx.Err() == nil {
			errCh <- fmt.Errorf("agent listener: %w", err)
		}
	}()
	go func() {
		log.Printf("HTTP server listening on %s", httpListener.Addr())
		if err := s.server.Serve(httpListener); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("HTTP server: %w", err)
		}
	}()

	var runErr error
	select {
	case <-ctx.Done():
	case runErr = <-errCh:
	}

	log.Println("Shutting down servers...")
	s.ws.Close()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.server.Shutdown(shutdownCtx); err != nil && runErr == nil {
		runErr = err
	}
	return runErr
}
