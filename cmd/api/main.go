package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"diagnostic-client/internal/api"
	"diagnostic-client/internal/config"
	"diagnostic-client/internal/db"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Received shutdown signal")
		cancel()
	}()

	database, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	server := api.NewServer(cfg, database)

	log.Println("Starting diagnostic client API...")
	if err := server.Run(ctx); err != nil {
		log.Printf("Server shutdown with error: %v", err)
		os.Exit(1)
	}
}
