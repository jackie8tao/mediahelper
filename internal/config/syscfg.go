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

// ============================================================================
// Combined Config & Backward Compatibility
// ============================================================================

// PathMapping is an alias for PathMapperInstanceConfig for backward compatibility.
type PathMapping = PathMapperInstanceConfig

// TMDBConfig holds legacy TheMovieDatabase scraping configuration.
type TMDBConfig struct {
	APIKey         string `json:"api_key"`         // TMDB API key (v3 auth)
	BaseURL        string `json:"base_url"`        // API base URL, customizable for reverse proxies
	ImageBaseURL   string `json:"image_base_url"`  // Image CDN base URL
	Language       string `json:"language"`        // Target language code, default zh-CN
	Proxy          string `json:"proxy"`           // Optional HTTP/SOCKS5 proxy address
	GenerateNFO    bool   `json:"generate_nfo"`    // Whether to generate NFO metadata files
	DownloadImages bool   `json:"download_images"` // Whether to download poster and backdrop artwork
}

// ToInstanceConfig converts TMDBConfig to ScraperInstanceConfig.
func (t TMDBConfig) ToInstanceConfig(defaultID string) ScraperInstanceConfig {
	id := defaultID
	return ScraperInstanceConfig{
		ID:             id,
		Type:           "tmdb",
		Name:           "TMDB",
		Enabled:        t.APIKey != "",
		APIKey:         t.APIKey,
		BaseURL:        t.BaseURL,
		ImageBaseURL:   t.ImageBaseURL,
		Language:       t.Language,
		Proxy:          t.Proxy,
		GenerateNFO:    t.GenerateNFO,
		DownloadImages: t.DownloadImages,
	}
}

// ToTMDBConfig converts ScraperInstanceConfig to TMDBConfig.
func (s ScraperInstanceConfig) ToTMDBConfig() TMDBConfig {
	return TMDBConfig{
		APIKey:         s.APIKey,
		BaseURL:        s.BaseURL,
		ImageBaseURL:   s.ImageBaseURL,
		Language:       s.Language,
		Proxy:          s.Proxy,
		GenerateNFO:    s.GenerateNFO,
		DownloadImages: s.DownloadImages,
	}
}

// LibraryConfig holds legacy media library directory and linking mode settings.
type LibraryConfig struct {
	MovieDir    string `json:"movie_dir"`     // Root directory for movie library
	TVDir       string `json:"tv_dir"`        // Root directory for TV show library
	AnimeDir    string `json:"anime_dir"`     // Root directory for anime library
	LinkMode    string `json:"link_mode"`     // Organize mode: hardlink (default), copy, move, symlink
	MinFileSize int64  `json:"min_file_size"` // Minimum file size in bytes for valid video files
}

// ToInstanceConfig converts LibraryConfig to OrganizerInstanceConfig.
func (l LibraryConfig) ToInstanceConfig(defaultID string) OrganizerInstanceConfig {
	return OrganizerInstanceConfig{
		ID:             defaultID,
		Name:           "Default Library",
		Enabled:        true,
		MovieDir:       l.MovieDir,
		TVDir:          l.TVDir,
		AnimeDir:       l.AnimeDir,
		LinkMode:       l.LinkMode,
		MinFileSize:    l.MinFileSize,
		GenerateNFO:    true,
		DownloadImages: true,
	}
}

// ToLibraryConfig converts OrganizerInstanceConfig to LibraryConfig.
func (o OrganizerInstanceConfig) ToLibraryConfig() LibraryConfig {
	return LibraryConfig{
		MovieDir:    o.MovieDir,
		TVDir:       o.TVDir,
		AnimeDir:    o.AnimeDir,
		LinkMode:    o.LinkMode,
		MinFileSize: o.MinFileSize,
	}
}

// Config represents the system-wide global configuration composing SystemConfig and ComponentConfig.
type Config struct {
	SystemConfig    `json:"system"`
	ComponentConfig `json:"component"`

	TMDB    TMDBConfig    `json:"tmdb,omitempty"`
	Library LibraryConfig `json:"library,omitempty"`
}
