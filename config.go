package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Port       int                       `yaml:"port"`
	CacheDir   string                    `yaml:"cache_dir"`
	Registries map[string]RegistryConfig `yaml:"registries"`
}

// RegistryConfig holds credentials for a specific registry
type RegistryConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	config.ApplyDefaults()

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &config, nil
}

// ApplyDefaults sets default values for unspecified configuration options
func (c *Config) ApplyDefaults() {
	if c.Port == 0 {
		c.Port = 8080
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("invalid port: %d (must be between 1 and 65535)", c.Port)
	}
	return nil
}

// ApplyCredentials registers all configured registry credentials
func (c *Config) ApplyCredentials() {
	for registry, creds := range c.Registries {
		SetCredentials(registry, creds.Username, creds.Password)
	}
}
