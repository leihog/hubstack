package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"hubstack/internal/config"
	"hubstack/internal/icondetector"
	"hubstack/internal/server"
)

func main() {
	port := flag.Int("port", 8080, "Port to listen on")
	configPath := flag.String("config", "config.yml", "Path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	detector := icondetector.NewHTTPDetector()

	srv, err := server.New(cfg, *port, detector)
	if err != nil {
		log.Fatalf("Error creating server: %v", err)
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	serverErr := make(chan error, 1)
	go func() {
		if err := srv.Start(); err != nil {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		log.Fatalf("Server error: %v", err)
	case sig := <-sigChan:
		log.Printf("Received signal: %v", sig)

		// Create a context with timeout for shutdown
		// and then attempt graceful shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Fatalf("Error during shutdown: %v", err)
		}
	}
}
