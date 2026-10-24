package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the system-wide global configuration.
type Config struct {
	Server      ServerConfig      `yaml:"server" json:"server"`
	Database    DatabaseConfig    `yaml:"database" json:"database"`
	QBittorrent QBittorrentConfig `yaml:"qbittorrent" json:"qbittorrent"`
	TMDB        TMDBConfig        `yaml:"tmdb" json:"tmdb"`
	Library     LibraryConfig     `yaml:"library" json:"library"`
	Plugins     PluginConfig      `yaml:"plugins" json:"plugins"`
}

// DatabaseConfig holds database connection parameters.
type DatabaseConfig struct {
	DSN string `yaml:"dsn" json:"dsn"` // Database connection string or SQLite file path (e.g. data/mediahelper.db)
}

// PluginConfig holds global configuration for the plugin system.
type PluginConfig struct {
	Enabled bool                              `yaml:"enabled" json:"enabled"` // Whether the plugin system is enabled
	Items   map[string]map[string]interface{} `yaml:"items" json:"items"`     // Custom configurations per plugin ID
}

// ServerConfig holds HTTP server network parameters.
type ServerConfig struct {
	Port int    `yaml:"port" json:"port"`
	Host string `yaml:"host" json:"host"`
}

// LibraryConfig holds media library directory and linking mode settings.
type LibraryConfig struct {
	MovieDir    string `yaml:"movie_dir" json:"movie_dir"`         // Root directory for movie library
	TVDir       string `yaml:"tv_dir" json:"tv_dir"`               // Root directory for TV show library
	AnimeDir    string `yaml:"anime_dir" json:"anime_dir"`         // Root directory for anime library (falls back to TVDir if empty)
	LinkMode    string `yaml:"link_mode" json:"link_mode"`         // Organize mode: hardlink (default), copy, move, symlink
	MinFileSize int64  `yaml:"min_file_size" json:"min_file_size"` // Minimum file size in bytes for valid video files (default: 50MB)
}

// TMDBConfig holds TheMovieDatabase scraping configuration.
type TMDBConfig struct {
	APIKey         string `yaml:"api_key" json:"api_key"`                   // TMDB API key (v3 auth)
	BaseURL        string `yaml:"base_url" json:"base_url"`                 // API base URL, customizable for reverse proxies
	ImageBaseURL   string `yaml:"image_base_url" json:"image_base_url"`     // Image CDN base URL
	Language       string `yaml:"language" json:"language"`                 // Target language code, default zh-CN
	Proxy          string `yaml:"proxy" json:"proxy"`                       // Optional HTTP/SOCKS5 proxy address
	GenerateNFO    bool   `yaml:"generate_nfo" json:"generate_nfo"`         // Whether to generate NFO metadata files
	DownloadImages bool   `yaml:"download_images" json:"download_images"`   // Whether to download poster and backdrop artwork
}

// PathMapping maps paths between remote/docker and local environments.
type PathMapping struct {
	Remote string `yaml:"remote" json:"remote"` // Remote or container internal path prefix
	Local  string `yaml:"local" json:"local"`   // Local or host accessible path prefix
}

// QBittorrentConfig holds connection and management settings for qBittorrent.
type QBittorrentConfig struct {
	Enabled      bool          `yaml:"enabled" json:"enabled"`
	URL          string        `yaml:"url" json:"url"`                     // WebUI URL, e.g. http://127.0.0.1:8080
	Username     string        `yaml:"username" json:"username"`           // Username
	Password     string        `yaml:"password" json:"password"`           // Password
	Categories   []string      `yaml:"categories" json:"categories"`       // Watched category list; empty means all categories
	AutoTag      bool          `yaml:"auto_tag" json:"auto_tag"`           // Whether to tag processed torrents
	SuccessTag   string        `yaml:"success_tag" json:"success_tag"`     // Tag applied upon successful organization
	FailTag      string        `yaml:"fail_tag" json:"fail_tag"`           // Tag applied upon failed organization
	PollInterval int           `yaml:"poll_interval" json:"poll_interval"` // Background poll interval in seconds (default: 30)
	PathMappings []PathMapping `yaml:"path_mappings" json:"path_mappings"` // Path mapping rules list
}

// DefaultConfig returns default configuration values.
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: 9090,
		},
		Database: DatabaseConfig{
			DSN: "data/mediahelper.db",
		},
		QBittorrent: QBittorrentConfig{
			Enabled:      true,
			URL:          "http://127.0.0.1:8080",
			Username:     "admin",
			Password:     "adminadmin",
			Categories:   []string{"movie", "tv", "anime"},
			AutoTag:      true,
			SuccessTag:   "MediaHelper:Done",
			FailTag:      "MediaHelper:Error",
			PollInterval: 30,
			PathMappings: []PathMapping{},
		},
		TMDB: TMDBConfig{
			APIKey:         "",
			BaseURL:        "https://api.themoviedb.org/3",
			ImageBaseURL:   "https://image.tmdb.org/t/p/original",
			Language:       "zh-CN",
			Proxy:          "",
			GenerateNFO:    true,
			DownloadImages: true,
		},
		Library: LibraryConfig{
			MovieDir:    "D:/Media/Movies",
			TVDir:       "D:/Media/TV",
			AnimeDir:    "D:/Media/Anime",
			LinkMode:    "hardlink",
			MinFileSize: 50 * 1024 * 1024, // 50MB
		},
		Plugins: PluginConfig{
			Enabled: true,
			Items:   make(map[string]map[string]interface{}),
		},
	}
}

// LoadConfig loads configuration from a specified YAML file path.
func LoadConfig(path string) (*Config, error) {
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
