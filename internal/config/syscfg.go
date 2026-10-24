package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
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
	Server   ServerConfig   `yaml:"server" json:"server"`
	Database DatabaseConfig `yaml:"database" json:"database"`
}

// ServerConfig holds HTTP server network parameters.
type ServerConfig struct {
	Host string `yaml:"host" json:"host"`
	Port int    `yaml:"port" json:"port"`
}

// DatabaseConfig holds database connection parameters.
type DatabaseConfig struct {
	DSN string `yaml:"dsn" json:"dsn"` // Database connection string or SQLite file path (e.g. data/mediahelper.db)
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

// LoadSystemConfigFromFile loads system startup configuration from a specified YAML file path.
func LoadSystemConfigFromFile(path string) (*SystemConfig, error) {
	cfg := DefaultSystemConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read system config file error: %w", err)
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse system config yaml error: %w", err)
	}

	return cfg, nil
}

// SaveSystemConfigToFile writes system startup configuration to a YAML file.
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

	data, err := yaml.Marshal(cfg)
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

// QBittorrentConfig holds legacy connection and management settings for qBittorrent.
type QBittorrentConfig struct {
	ID           string        `yaml:"id" json:"id"` // Optional component ID (defaults to "downloader")
	Enabled      bool          `yaml:"enabled" json:"enabled"`
	URL          string        `yaml:"url" json:"url"`                     // WebUI URL, e.g. http://127.0.0.1:8080
	Username     string        `yaml:"username" json:"username"`           // Username
	Password     string        `yaml:"password" json:"password"`           // Password
	Categories   []string      `yaml:"categories" json:"categories"`       // Watched categories list
	AutoTag      bool          `yaml:"auto_tag" json:"auto_tag"`           // Whether to tag processed torrents
	SuccessTag   string        `yaml:"success_tag" json:"success_tag"`     // Tag applied upon successful organization
	FailTag      string        `yaml:"fail_tag" json:"fail_tag"`           // Tag applied upon failed organization
	PollInterval int           `yaml:"poll_interval" json:"poll_interval"` // Background poll interval in seconds (default: 30)
	PathMappings []PathMapping `yaml:"path_mappings" json:"path_mappings"` // Remote to local path mapping
}

// ToInstanceConfig converts QBittorrentConfig to DownloaderInstanceConfig.
func (q QBittorrentConfig) ToInstanceConfig(defaultID string) DownloaderInstanceConfig {
	id := q.ID
	if id == "" {
		id = defaultID
	}
	return DownloaderInstanceConfig{
		ID:           id,
		Type:         "qbittorrent",
		Name:         "qBittorrent",
		Enabled:      q.Enabled,
		URL:          q.URL,
		Username:     q.Username,
		Password:     q.Password,
		Categories:   q.Categories,
		AutoTag:      q.AutoTag,
		SuccessTag:   q.SuccessTag,
		FailTag:      q.FailTag,
		PollInterval: q.PollInterval,
	}
}

// ToQBittorrentConfig converts DownloaderInstanceConfig to QBittorrentConfig.
func (d DownloaderInstanceConfig) ToQBittorrentConfig() QBittorrentConfig {
	return QBittorrentConfig{
		ID:           d.ID,
		Enabled:      d.Enabled,
		URL:          d.URL,
		Username:     d.Username,
		Password:     d.Password,
		Categories:   d.Categories,
		AutoTag:      d.AutoTag,
		SuccessTag:   d.SuccessTag,
		FailTag:      d.FailTag,
		PollInterval: d.PollInterval,
	}
}

// TMDBConfig holds legacy TheMovieDatabase scraping configuration.
type TMDBConfig struct {
	APIKey         string `yaml:"api_key" json:"api_key"`                 // TMDB API key (v3 auth)
	BaseURL        string `yaml:"base_url" json:"base_url"`               // API base URL, customizable for reverse proxies
	ImageBaseURL   string `yaml:"image_base_url" json:"image_base_url"`   // Image CDN base URL
	Language       string `yaml:"language" json:"language"`               // Target language code, default zh-CN
	Proxy          string `yaml:"proxy" json:"proxy"`                     // Optional HTTP/SOCKS5 proxy address
	GenerateNFO    bool   `yaml:"generate_nfo" json:"generate_nfo"`       // Whether to generate NFO metadata files
	DownloadImages bool   `yaml:"download_images" json:"download_images"` // Whether to download poster and backdrop artwork
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
	MovieDir    string `yaml:"movie_dir" json:"movie_dir"`         // Root directory for movie library
	TVDir       string `yaml:"tv_dir" json:"tv_dir"`               // Root directory for TV show library
	AnimeDir    string `yaml:"anime_dir" json:"anime_dir"`         // Root directory for anime library
	LinkMode    string `yaml:"link_mode" json:"link_mode"`         // Organize mode: hardlink (default), copy, move, symlink
	MinFileSize int64  `yaml:"min_file_size" json:"min_file_size"` // Minimum file size in bytes for valid video files
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
	SystemConfig    `yaml:",inline" json:"system"`
	ComponentConfig `yaml:",inline" json:"component"`

	// Legacy compatibility fields
	QBittorrent QBittorrentConfig `yaml:"qbittorrent,omitempty" json:"qbittorrent,omitempty"`
	TMDB        TMDBConfig        `yaml:"tmdb,omitempty" json:"tmdb,omitempty"`
	Library     LibraryConfig     `yaml:"library,omitempty" json:"library,omitempty"`
}

// DefaultConfig returns default configuration values.
func DefaultConfig() *Config {
	return &Config{
		SystemConfig:    *DefaultSystemConfig(),
		ComponentConfig: *DefaultComponentConfig(),
	}
}

// LoadConfigFromFile loads configuration from a specified YAML file path.
func LoadConfigFromFile(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file error: %w", err)
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config yaml error: %w", err)
	}

	return cfg, nil
}

// SaveConfigToFile writes configuration to a specified YAML file path.
func SaveConfigToFile(path string, cfg *Config) error {
	if cfg == nil {
		return errors.New("config is nil")
	}

	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create directory for config error: %w", err)
		}
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config error: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write config file error: %w", err)
	}

	return nil
}

// GetDownloaderConfigs returns all configured downloader instances, falling back to legacy qbittorrent config.
func (c *Config) GetDownloaderConfigs() []DownloaderInstanceConfig {
	if len(c.ComponentConfig.Downloaders) > 0 {
		return c.ComponentConfig.Downloaders
	}
	if c.QBittorrent.Enabled || c.QBittorrent.URL != "" {
		return []DownloaderInstanceConfig{c.QBittorrent.ToInstanceConfig("downloader")}
	}
	return make([]DownloaderInstanceConfig, 0)
}

// GetPathMappings returns global directory path mappings, falling back to legacy qbittorrent path mappings.
func (c *Config) GetPathMappings() []PathMapperInstanceConfig {
	if len(c.ComponentConfig.PathMappings) > 0 {
		return c.ComponentConfig.PathMappings
	}
	if len(c.QBittorrent.PathMappings) > 0 {
		return c.QBittorrent.PathMappings
	}
	return make([]PathMapperInstanceConfig, 0)
}

// GetParserConfigs returns all configured parser instances, falling back to a default instance.
func (c *Config) GetParserConfigs() []ParserInstanceConfig {
	if len(c.ComponentConfig.Parsers) > 0 {
		return c.ComponentConfig.Parsers
	}
	return []ParserInstanceConfig{
		{
			ID:      "parser",
			Name:    "Default Parser",
			Enabled: true,
		},
	}
}

// GetScraperConfigs returns all configured scraper instances, falling back to legacy TMDBConfig.
func (c *Config) GetScraperConfigs() []ScraperInstanceConfig {
	if len(c.ComponentConfig.Scrapers) > 0 {
		return c.ComponentConfig.Scrapers
	}
	if c.TMDB.APIKey != "" || c.TMDB.BaseURL != "" {
		return []ScraperInstanceConfig{c.TMDB.ToInstanceConfig("scraper")}
	}
	return make([]ScraperInstanceConfig, 0)
}

// GetOrganizerConfigs returns all configured organizer instances, falling back to legacy LibraryConfig.
func (c *Config) GetOrganizerConfigs() []OrganizerInstanceConfig {
	if len(c.ComponentConfig.Organizers) > 0 {
		return c.ComponentConfig.Organizers
	}
	if c.Library.MovieDir != "" || c.Library.TVDir != "" {
		inst := c.Library.ToInstanceConfig("organizer")
		inst.GenerateNFO = c.TMDB.GenerateNFO
		inst.DownloadImages = c.TMDB.DownloadImages
		return []OrganizerInstanceConfig{inst}
	}
	return make([]OrganizerInstanceConfig, 0)
}
