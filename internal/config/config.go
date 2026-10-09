package config

import (
	"os"
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
	return &Config{
		DatabaseURL:       getEnv("DATABASE_URL", "postgres://postgres:postgres@127.0.0.1:5432/diagnostic?sslmode=disable"),
		ServerAddr:        getEnv("SERVER_ADDR", "127.0.0.1:8080"),
		AgentAddr:         getEnv("AGENT_ADDR", "127.0.0.1:8081"),
		LogBufferSize:     10000, // Larger buffer for logs
		NetworkBufferSize: 50000, // Larger buffer for network packets
		BatchSize:         10000, // Database batch size
		StreamBatchSize:   100,   // WebSocket stream batch size
	}, nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
