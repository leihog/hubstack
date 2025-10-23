package handlers

import (
	"encoding/json"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"sync"

	"hubstack/internal/config"
	"hubstack/internal/icondetector"
)

type Handler struct {
	config       *config.Config
	template     *template.Template
	iconDetector icondetector.Detector
	mu           sync.RWMutex
}

// New creates a new Handler with the given config, template, and icon detector
func New(cfg *config.Config, tmpl *template.Template, detector icondetector.Detector) *Handler {
	return &Handler{
		config:       cfg,
		template:     tmpl,
		iconDetector: detector,
	}
}

// ServeHome handles requests to the home page
func (h *Handler) ServeHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	h.mu.RLock()
	// Create a copy of config for this request
	// the config only contains template data so we can safely copy it
	pageConfig := *h.config
	h.mu.RUnlock()

	// If no subtitle is configured, pick a random one for this page load
	if pageConfig.Subtitle == "" {
		pageConfig.Subtitle = config.DefaultSubtitles[rand.Intn(len(config.DefaultSubtitles))]
	}

	err := h.template.Execute(w, pageConfig)
	if err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// CheckWriteable handles requests to check if config is writeable
func (h *Handler) CheckWriteable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.mu.RLock()
	writeable := h.config.IsWriteable()
	h.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{
		"writeable": writeable,
	})
}

// AddService handles requests to add a new service
func (h *Handler) AddService(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse the JSON request body
	var service config.Service
	if err := json.NewDecoder(r.Body).Decode(&service); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	// TODO: add better validation
	if service.Name == "" || service.URL == "" {
		http.Error(w, "Name and URL are required", http.StatusBadRequest)
		return
	}

	// Auto-detect icon if not provided
	if service.Icon == "" && h.iconDetector != nil {
		log.Printf("Auto-detecting icon for service: %s (%s)", service.Name, service.URL)
		if detectedIcon, err := h.iconDetector.DetectIcon(service.URL); err == nil {
			service.Icon = detectedIcon
			log.Printf("Auto-detected icon: %s", detectedIcon)
		} else {
			// not a fatal error, just log it and continue without icon
			log.Printf("Failed to auto-detect icon: %v", err)
		}
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.config.IsWriteable() {
		http.Error(w, "Config file is not writeable", http.StatusForbidden)
		return
	}

	if err := h.config.AddService(service); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.config.Save(); err != nil {
		log.Printf("Error saving config: %v", err)
		http.Error(w, "Failed to save config", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Service added successfully",
	})
}
