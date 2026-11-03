package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Default constants for system configuration
const (
	serverDefaultHost  = "0.0.0.0"
	serverDefaultPort  = 9090
	defaultDatabaseDSN = "data/mediahelper.db"
)

// ============================================================================
// SystemConfig (Startup / Infrastructure Configuration)
// ============================================================================

// SystemConfig holds essential startup configurations required before database connection.
type SystemConfig struct {
	Server   ServerConfig   `json:"server"`
	Database DatabaseConfig `json:"database"`
}

// ServerConfig holds HTTP server network parameters.
type ServerConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

// DatabaseConfig holds database connection parameters.
type DatabaseConfig struct {
	DSN string `json:"dsn"` // Database connection string or SQLite file path (e.g. data/mediahelper.db)
}

// DefaultSystemConfig returns default system startup configuration.
func DefaultSystemConfig() *SystemConfig {
	return &SystemConfig{
		Server: ServerConfig{
			Host: serverDefaultHost,
			Port: serverDefaultPort,
		},
		Database: DatabaseConfig{
			DSN: defaultDatabaseDSN,
		},
	}
}

// LoadSystemConfigFromFile loads system startup configuration from a specified JSON file path.
func LoadSystemConfigFromFile(path string) (*SystemConfig, error) {
	cfg := DefaultSystemConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read system config file error: %w", err)
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse system config json error: %w", err)
	}

	return cfg, nil
}

// SaveSystemConfigToFile writes system startup configuration to a JSON file.
func SaveSystemConfigToFile(path string, cfg *SystemConfig) error {
	if cfg == nil {
		return errors.New("system config is nil")
	}

	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create directory for system config error: %w", err)
		}
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal system config error: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write system config file error: %w", err)
	}

	return nil
}
