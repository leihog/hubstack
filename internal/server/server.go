package server

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"hubstack/internal/config"
	"hubstack/internal/handlers"
	"hubstack/internal/icondetector"
)

// Server represents the HTTP server
type Server struct {
	config       *config.Config
	handler      *handlers.Handler
	template     *template.Template
	iconDetector icondetector.Detector
	port         int
	httpServer   *http.Server
}

// New creates a new Server instance
func New(cfg *config.Config, port int, detector icondetector.Detector) (*Server, error) {
	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	// Pass the icon detector to the handler so we can
	// trigger icon auto-detection when adding a new service
	handler := handlers.New(cfg, tmpl, detector)

	srv := &Server{
		config:       cfg,
		handler:      handler,
		template:     tmpl,
		iconDetector: detector,
		port:         port,
	}

	// Run pre-flight check to auto-detect missing icons
	if detector != nil {
		if err := srv.preflightIconCheck(); err != nil {
			log.Printf("Warning: pre-flight icon check failed: %v", err)
		}
	}

	return srv, nil
}

// preflightIconCheck checks all services and auto-detects missing icons
func (s *Server) preflightIconCheck() error {
	log.Println("Running pre-flight icon check...")

	updated := false
	for i := range s.config.Services {
		service := &s.config.Services[i]

		if service.Icon != "" {
			continue
		}

		log.Printf("  [%s] No icon configured, attempting auto-detection...", service.Name)

		detectedIcon, err := s.iconDetector.DetectIcon(service.URL)
		if err != nil {
			log.Printf("  [%s] Failed to detect icon: %v", service.Name, err)
			continue
		}

		service.Icon = detectedIcon
		updated = true
		log.Printf("  [%s] ✓ Icon detected: %s", service.Name, detectedIcon)
	}

	// Update the config file if any icons were detected, and the config is writeable
	if updated && s.config.IsWriteable() {
		log.Println("Saving detected icons to config...")
		if err := s.config.Save(); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
		log.Println("✓ Config saved successfully")
	} else if updated {
		log.Println("Warning: Config is not writeable, detected icons will not be persisted")
	}

	log.Println("Pre-flight check complete")
	return nil
}

// Start starts the HTTP server
func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handler.ServeHome)
	mux.HandleFunc("/api/config/writeable", s.handler.CheckWriteable)
	mux.HandleFunc("/api/services", s.handler.AddService)

	addr := fmt.Sprintf(":%d", s.port)

	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Starting server on http://localhost%s", addr)
	log.Printf("Loaded %d services from config.yml", len(s.config.Services))
	log.Printf("Page title: %s", s.config.Title)

	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server failed to start: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}

	log.Println("Shutting down server gracefully...")

	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	log.Println("Server stopped")
	return nil
}
