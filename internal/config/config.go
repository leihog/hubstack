package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Service represents a single service in the dashboard
type Service struct {
	Name        string `yaml:"name"`
	URL         string `yaml:"url"`
	Icon        string `yaml:"icon"`
	Description string `yaml:"description"`
}

// Config represents the application configuration
type Config struct {
	Title      string    `yaml:"title"`
	Subtitle   string    `yaml:"subtitle"`
	Services   []Service `yaml:"services"`
	ConfigPath string    `yaml:"-"` // used for dynamic service addition
}

// DefaultSubtitles contains the list of subtitles, a random one is shown when none is configured
var DefaultSubtitles = []string{
	"Because remembering ports is for humans.",
	"The map of your digital kingdom.",
	"Find it. Click it. Run it.",
	"Ports, hosts, and glory.",
	"Your digital command hub.",
	"The front door to your homelab.",
	"A single hub for your entire stack.",
	"The portal for your private cloud.",
}

// Load reads and parses the configuration file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set default title if not specified
	if config.Title == "" {
		config.Title = "HubStack"
	}

	// store the config path for later use in dynamic service addition
	config.ConfigPath = path

	// subtitle is left empty if not specified in config
	// One of the default subtitles will be picked randomly on each page load in the handler

	return &config, nil
}

// IsWriteable checks if the config file is writeable
func (c *Config) IsWriteable() bool {
	if c.ConfigPath == "" {
		return false
	}

	// Try to open the file with write permissions
	file, err := os.OpenFile(c.ConfigPath, os.O_WRONLY, 0)
	if err != nil {
		return false
	}
	defer file.Close()

	return true
}

// Save writes the config back to the file
func (c *Config) Save() error {
	if c.ConfigPath == "" {
		return fmt.Errorf("config path not set")
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	err = os.WriteFile(c.ConfigPath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// AddService adds a new service to the config
func (c *Config) AddService(service Service) error {
	// Check for duplicate names
	for _, s := range c.Services {
		if s.Name == service.Name {
			return fmt.Errorf("service with name '%s' already exists", service.Name)
		}
	}

	c.Services = append(c.Services, service)
	return nil
}
