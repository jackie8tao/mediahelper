package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the system-wide global configuration.
type Config struct {
	Server       ServerConfig               `yaml:"server" json:"server"`
	Database     DatabaseConfig             `yaml:"database" json:"database"`
	QBittorrent  QBittorrentConfig          `yaml:"qbittorrent" json:"qbittorrent"`
	Downloaders  []DownloaderInstanceConfig `yaml:"downloaders" json:"downloaders"`
	PathMappings []PathMapping              `yaml:"path_mappings" json:"path_mappings"`
	Parsers      []ParserInstanceConfig     `yaml:"parsers" json:"parsers"`
	Scrapers     []ScraperInstanceConfig    `yaml:"scrapers" json:"scrapers"`
	Organizers   []OrganizerInstanceConfig  `yaml:"organizers" json:"organizers"`
	TMDB         TMDBConfig                 `yaml:"tmdb" json:"tmdb"`
	Library      LibraryConfig              `yaml:"library" json:"library"`
	Plugins      PluginConfig               `yaml:"plugins" json:"plugins"`
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
	ID           string        `yaml:"id" json:"id"`                       // Optional component ID (defaults to "downloader")
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

// DownloaderInstanceConfig represents configuration for an individual downloader instance component.
type DownloaderInstanceConfig struct {
	ID           string        `yaml:"id" json:"id"`                       // Unique component ID, e.g. "downloader", "downloader:movie", "downloader:tv"
	Type         string        `yaml:"type" json:"type"`                   // Downloader type, e.g. "qbittorrent"
	Name         string        `yaml:"name" json:"name"`                   // Friendly display name
	Enabled      bool          `yaml:"enabled" json:"enabled"`             // Whether this downloader is enabled
	URL          string        `yaml:"url" json:"url"`                     // WebUI URL, e.g. http://127.0.0.1:8080
	Username     string        `yaml:"username" json:"username"`           // Username
	Password     string        `yaml:"password" json:"password"`           // Password
	Categories   []string      `yaml:"categories" json:"categories"`       // Watched categories list
	AutoTag      bool          `yaml:"auto_tag" json:"auto_tag"`           // Whether to tag processed torrents
	SuccessTag   string        `yaml:"success_tag" json:"success_tag"`     // Tag applied upon successful organization
	FailTag      string        `yaml:"fail_tag" json:"fail_tag"`           // Tag applied upon failed organization
	PollInterval int           `yaml:"poll_interval" json:"poll_interval"` // Background poll interval in seconds (default: 30)
	PathMappings []PathMapping `yaml:"path_mappings" json:"path_mappings"` // Path mapping rules list
}

// ToInstanceConfig converts QBittorrentConfig to DownloaderInstanceConfig.
func (q QBittorrentConfig) ToInstanceConfig(fallbackID string) DownloaderInstanceConfig {
	id := q.ID
	if id == "" {
		id = fallbackID
	}
	if id == "" {
		id = "downloader"
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
		PathMappings: q.PathMappings,
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
		PathMappings: d.PathMappings,
	}
}

// ParserInstanceConfig represents configuration for an individual parser component.
type ParserInstanceConfig struct {
	ID                  string   `yaml:"id" json:"id"`                                     // Unique component ID, e.g. "parser", "parser:anime"
	Name                string   `yaml:"name" json:"name"`                                 // Friendly display name
	Enabled             bool     `yaml:"enabled" json:"enabled"`                           // Whether this parser instance is enabled
	DefaultMediaType    string   `yaml:"default_media_type" json:"default_media_type"`     // Optional default fallback media type (movie/tv/anime)
	CustomReleaseGroups []string `yaml:"custom_release_groups" json:"custom_release_groups"` // Optional additional release group identifiers
}

// ScraperInstanceConfig represents configuration for an individual scraper component.
type ScraperInstanceConfig struct {
	ID             string `yaml:"id" json:"id"`                         // Unique component ID, e.g. "scraper", "scraper:tmdb-en"
	Type           string `yaml:"type" json:"type"`                     // Scraper provider type, e.g. "tmdb"
	Name           string `yaml:"name" json:"name"`                     // Friendly display name
	Enabled        bool   `yaml:"enabled" json:"enabled"`               // Whether this scraper instance is enabled
	APIKey         string `yaml:"api_key" json:"api_key"`               // API authentication key
	BaseURL        string `yaml:"base_url" json:"base_url"`             // API base URL
	ImageBaseURL   string `yaml:"image_base_url" json:"image_base_url"` // Artwork CDN base URL
	Language       string `yaml:"language" json:"language"`             // Preferred metadata language
	Proxy          string `yaml:"proxy" json:"proxy"`                   // Optional proxy URL
	GenerateNFO    bool   `yaml:"generate_nfo" json:"generate_nfo"`     // Whether to generate NFO metadata files
	DownloadImages bool   `yaml:"download_images" json:"download_images"` // Whether to download poster and backdrop artwork
}

// ToInstanceConfig converts TMDBConfig to ScraperInstanceConfig.
func (t TMDBConfig) ToInstanceConfig(fallbackID string) ScraperInstanceConfig {
	id := fallbackID
	if id == "" {
		id = "scraper"
	}
	return ScraperInstanceConfig{
		ID:             id,
		Type:           "tmdb",
		Name:           "TMDB Scraper",
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

// OrganizerInstanceConfig represents configuration for an individual organizer component.
type OrganizerInstanceConfig struct {
	ID             string `yaml:"id" json:"id"`                         // Unique component ID, e.g. "organizer", "organizer:movie"
	Name           string `yaml:"name" json:"name"`                     // Friendly display name
	Enabled        bool   `yaml:"enabled" json:"enabled"`               // Whether this organizer instance is enabled
	MovieDir       string `yaml:"movie_dir" json:"movie_dir"`           // Root directory for movie library
	TVDir          string `yaml:"tv_dir" json:"tv_dir"`                 // Root directory for TV show library
	AnimeDir       string `yaml:"anime_dir" json:"anime_dir"`           // Root directory for anime library
	LinkMode       string `yaml:"link_mode" json:"link_mode"`           // hardlink, copy, move, symlink
	MinFileSize    int64  `yaml:"min_file_size" json:"min_file_size"`   // Minimum file size in bytes for valid video files
	GenerateNFO    bool   `yaml:"generate_nfo" json:"generate_nfo"`     // Whether to generate NFO files
	DownloadImages bool   `yaml:"download_images" json:"download_images"` // Whether to download images
	ScraperID      string `yaml:"scraper_id" json:"scraper_id"`         // Optional target scraper component ID
}

// ToInstanceConfig converts LibraryConfig to OrganizerInstanceConfig.
func (l LibraryConfig) ToInstanceConfig(fallbackID string) OrganizerInstanceConfig {
	id := fallbackID
	if id == "" {
		id = "organizer"
	}
	return OrganizerInstanceConfig{
		ID:             id,
		Name:           "Media Organizer",
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

// GetDownloaderConfigs returns all configured downloader instances.
// If the Downloaders slice is empty but the top-level QBittorrent is configured,
// it wraps QBittorrent into a default DownloaderInstanceConfig for backwards compatibility.
func (c *Config) GetDownloaderConfigs() []DownloaderInstanceConfig {
	if len(c.Downloaders) > 0 {
		return c.Downloaders
	}

	if c.QBittorrent.Enabled || c.QBittorrent.URL != "" {
		return []DownloaderInstanceConfig{c.QBittorrent.ToInstanceConfig("downloader")}
	}

	return nil
}

// GetPathMappings returns global directory path mappings, falling back to legacy qbittorrent path mappings.
func (c *Config) GetPathMappings() []PathMapping {
	if len(c.PathMappings) > 0 {
		return c.PathMappings
	}
	if len(c.QBittorrent.PathMappings) > 0 {
		return c.QBittorrent.PathMappings
	}
	return nil
}

// GetParserConfigs returns all configured parser instances, falling back to a default instance.
func (c *Config) GetParserConfigs() []ParserInstanceConfig {
	if len(c.Parsers) > 0 {
		return c.Parsers
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
	if len(c.Scrapers) > 0 {
		return c.Scrapers
	}
	if c.TMDB.APIKey != "" || c.TMDB.BaseURL != "" {
		return []ScraperInstanceConfig{c.TMDB.ToInstanceConfig("scraper")}
	}
	return nil
}

// GetOrganizerConfigs returns all configured organizer instances, falling back to legacy LibraryConfig.
func (c *Config) GetOrganizerConfigs() []OrganizerInstanceConfig {
	if len(c.Organizers) > 0 {
		return c.Organizers
	}
	if c.Library.MovieDir != "" || c.Library.TVDir != "" {
		inst := c.Library.ToInstanceConfig("organizer")
		inst.GenerateNFO = c.TMDB.GenerateNFO
		inst.DownloadImages = c.TMDB.DownloadImages
		return []OrganizerInstanceConfig{inst}
	}
	return nil
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
