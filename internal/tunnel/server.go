package tunnel

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"diagnostic-client/internal/config"
)

type Server struct {
	cfg      *config.Config
	handler  *Handler
	listener net.Listener

	// Connection management
	activeConns sync.WaitGroup
	mu          sync.Mutex
	connections map[net.Conn]struct{}

	// Shutdown coordination
	shutdownCh   chan struct{}
	shutdownOnce sync.Once
}

func NewServer(cfg *config.Config, handler *Handler) (*Server, error) {
	// Create TCP listener
	listener, err := net.Listen("tcp", cfg.AgentAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to create listener: %w", err)
	}

	server := &Server{
		cfg:         cfg,
		handler:     handler,
		listener:    listener,
		connections: make(map[net.Conn]struct{}),
		shutdownCh:  make(chan struct{}),
	}

	return server, nil
}

func (s *Server) Run(ctx context.Context) error {
	log.Printf("[TUNNEL] Server listening on %s", s.cfg.AgentAddr)

	// Create error channel for accept loop
	acceptErrors := make(chan error, 1)

	// Start accept loop in goroutine
	go s.acceptLoop(ctx, acceptErrors)

	// Wait for shutdown signal or accept error
	select {
	case <-ctx.Done():
		return s.shutdown(ctx.Err())
	case err := <-acceptErrors:
		return s.shutdown(err)
	}
}

func (s *Server) acceptLoop(ctx context.Context, acceptErrors chan<- error) {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				// Normal shutdown, don't report error
				return
			default:
				// Unexpected error
				acceptErrors <- fmt.Errorf("accept error: %w", err)
				return
			}
		}

		// Register new connection
		s.trackConnection(conn)

		// Handle connection in goroutine
		go func() {
			defer s.untrackConnection(conn)

			// Create connection-specific context
			connCtx, cancel := context.WithCancel(ctx)
			defer cancel()

			// Handle connection
			if err := s.handleConnection(connCtx, conn); err != nil {
				if ctx.Err() == nil { // Only log if not shutting down
					log.Printf("[TUNNEL] Connection error: %v", err)
				}
			}
		}()
	}
}

func (s *Server) handleConnection(ctx context.Context, conn net.Conn) error {
	// Set TCP keepalive
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		if err := tcpConn.SetKeepAlive(true); err != nil {
			return fmt.Errorf("failed to set keepalive: %w", err)
		}
		if err := tcpConn.SetKeepAlivePeriod(30 * time.Second); err != nil {
			return fmt.Errorf("failed to set keepalive period: %w", err)
		}
	}

	// Create done channel for this connection
	done := make(chan struct{})
	defer close(done)

	// Monitor connection in separate goroutine
	go func() {
		select {
		case <-ctx.Done():
			conn.Close()
		case <-s.shutdownCh:
			conn.Close()
		case <-done:
			return
		}
	}()

	// Handle connection using tunnel handler
	s.handler.HandleConnection(ctx, conn)
	return nil
}

func (s *Server) trackConnection(conn net.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.connections[conn] = struct{}{}
	s.activeConns.Add(1)
}

func (s *Server) untrackConnection(conn net.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.connections[conn]; exists {
		delete(s.connections, conn)
		s.activeConns.Done()
	}
}

func (s *Server) shutdown(err error) error {
	s.shutdownOnce.Do(func() {
		// Signal shutdown to all goroutines
		close(s.shutdownCh)

		// Close listener
		if s.listener != nil {
			s.listener.Close()
		}

		// Close all active connections
		s.mu.Lock()
		for conn := range s.connections {
			conn.Close()
		}
		s.mu.Unlock()

		// Wait for all connections to finish
		shutdownTimeout := time.NewTimer(10 * time.Second)
		shutdownComplete := make(chan struct{})

		go func() {
			s.activeConns.Wait()
			close(shutdownComplete)
		}()

		select {
		case <-shutdownComplete:
			log.Printf("[TUNNEL] Server shutdown complete")
		case <-shutdownTimeout.C:
			log.Printf("[TUNNEL] Server shutdown timed out")
		}
	})

	// Return the original error that triggered shutdown
	return err
}

// GetConnCount returns the current number of active connections
func (s *Server) GetConnCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.connections)
}

// Close initiates a graceful shutdown of the server
func (s *Server) Close() error {
	return s.shutdown(nil)
}
