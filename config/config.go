package config

import (
	"claude-squad/log"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const ConfigFileName = "config.json"

// GetConfigDir returns the path to the application's configuration directory
func GetConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get config home directory: %w", err)
	}
	return filepath.Join(homeDir, ".claude-squad"), nil
}

// Config represents the application configuration
type Config struct {
	// DefaultProgram is the default program to run in new instances
	DefaultProgram string `json:"default_program"`
	// AutoYes is a flag to automatically accept all prompts.
	AutoYes bool `json:"auto_yes"`
	// DaemonPollInterval is the interval (ms) at which the daemon polls sessions for autoyes mode.
	DaemonPollInterval int `json:"daemon_poll_interval"`
	
	// Web Server Configuration
	WebServerEnabled     bool   `json:"web_server_enabled"`
	WebServerPort        int    `json:"web_server_port"`
	WebServerHost        string `json:"web_server_host"`
	WebServerAuthToken   string `json:"web_server_auth_token"`
	WebServerAllowLocalhost bool `json:"web_server_allow_localhost"`
	WebServerUseTLS      bool   `json:"web_server_use_tls"`
	WebServerTLSCert     string `json:"web_server_tls_cert"`
	WebServerTLSKey      string `json:"web_server_tls_key"`
	WebServerCorsOrigin  string `json:"web_server_cors_origin"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	// Generate a simple default auth token
	// In production, a more secure token would be generated
	defaultToken := "claude-squad-monitoring-token"
	
	return &Config{
		DefaultProgram:     "claude",
		AutoYes:            false,
		DaemonPollInterval: 1000,
		
		// Web Server defaults
		WebServerEnabled:      false,
		WebServerPort:         8080,
		WebServerHost:         "127.0.0.1",
		WebServerAuthToken:    defaultToken,
		WebServerAllowLocalhost: true,
		WebServerUseTLS:       false,
		WebServerTLSCert:      "",
		WebServerTLSKey:       "",
		WebServerCorsOrigin:   "http://localhost:3000",
	}
}

// LoadConfig loads the configuration from disk. If it cannot be done, we return the default configuration.
func LoadConfig() *Config {
	configDir, err := GetConfigDir()
	if err != nil {
		log.ErrorLog.Printf("failed to get config directory: %v", err)
		return DefaultConfig()
	}

	configPath := filepath.Join(configDir, ConfigFileName)
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Create and save default config if file doesn't exist
			defaultCfg := DefaultConfig()
			if saveErr := saveConfig(defaultCfg); saveErr != nil {
				log.WarningLog.Printf("failed to save default config: %v", saveErr)
			}
			return defaultCfg
		}

		log.WarningLog.Printf("failed to get config file: %v", err)
		return DefaultConfig()
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		log.ErrorLog.Printf("failed to parse config file: %v", err)
		return DefaultConfig()
	}

	return &config
}

// saveConfig saves the configuration to disk
func saveConfig(config *Config) error {
	configDir, err := GetConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get config directory: %w", err)
	}

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configPath := filepath.Join(configDir, ConfigFileName)
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return os.WriteFile(configPath, data, 0644)
}

// SaveConfig exports the saveConfig function for use by other packages
func SaveConfig(config *Config) error {
	return saveConfig(config)
}
