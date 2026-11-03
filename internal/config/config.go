package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	DatabaseURL       string
	ServerAddr        string
	AgentAddr         string
	LogBufferSize     int
	NetworkBufferSize int
	BatchSize         int
	StreamBatchSize   int // How many packets to send in one websocket message
	ProcessingWorkers int
	MaxBackoff        time.Duration
	InitialBackoff    time.Duration
}

func Load() (*Config, error) {
	cfg := &Config{
		DatabaseURL:       getEnv("DATABASE_URL", "postgres://postgres:postgres@127.0.0.1:5432/diagnostic?sslmode=disable"),
		ServerAddr:        getEnv("SERVER_ADDR", "127.0.0.1:8080"),
		AgentAddr:         getEnv("AGENT_ADDR", "127.0.0.1:8081"),
		LogBufferSize:     10000, // Larger buffer for logs
		NetworkBufferSize: 50000, // Larger buffer for network packets
		BatchSize:         10000, // Database batch size
		StreamBatchSize:   100,   // WebSocket stream batch size
	}
	if strings.TrimSpace(cfg.DatabaseURL) == "" {
		return nil, fmt.Errorf("DATABASE_URL must not be empty")
	}
	for key, address := range map[string]string{"SERVER_ADDR": cfg.ServerAddr, "AGENT_ADDR": cfg.AgentAddr} {
		_, portStr, err := net.SplitHostPort(address)
		if err != nil {
			return nil, fmt.Errorf("%s: expected host:port: %w", key, err)
		}
		port, err := strconv.Atoi(portStr)
		if err != nil || port < 1 || port > 65535 {
			return nil, fmt.Errorf("%s: invalid port %q", key, portStr)
		}
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
