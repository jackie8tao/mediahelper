package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackie8tao/mediahelper/internal/database"
	"gorm.io/gorm"
)

// Section key constants for structured configuration modules
const (
	SectionDownloaders  = "downloaders"
	SectionPathMappings = "path_mappings"
	SectionPathMappers  = "pathmappers" // alias for backwards compatibility
	SectionParsers      = "parsers"
	SectionScrapers     = "scrapers"
	SectionOrganizers   = "organizers"
	SectionPlugins      = "plugins"
)

// ErrConfigNotFound indicates no component configuration was found in the database.
var ErrConfigNotFound = errors.New("no component configuration found in database")

// ============================================================================
// ComponentConfig (Dynamic Component Configurations Managed via API & DB)
// ============================================================================

// ComponentConfig holds dynamic pipeline and component configurations managed via API and stored in the database.
type ComponentConfig struct {
	Downloaders  []DownloaderComponentConfig `yaml:"downloaders" json:"downloaders"`
	PathMappings []PathMapperInstanceConfig  `yaml:"path_mappings" json:"path_mappings"`
	Parsers      []ParserInstanceConfig      `yaml:"parsers" json:"parsers"`
	Scrapers     []ScraperInstanceConfig     `yaml:"scrapers" json:"scrapers"`
	Organizers   []OrganizerInstanceConfig   `yaml:"organizers" json:"organizers"`
	Plugins      PluginConfig                `yaml:"plugins" json:"plugins"`
}

// DownloaderComponentConfig represents configuration for an individual downloader instance component.
type DownloaderComponentConfig struct {
	Enabled     bool            `yaml:"enabled" json:"enabled"`         // Whether this downloader is enabled
	ID          string          `yaml:"id" json:"id"`                   // Unique component ID, e.g. "downloader", "downloader:movie", "downloader:tv"
	Type        string          `yaml:"type" json:"type"`               // Downloader type, e.g. "qbittorrent", "transmission"
	Name        string          `yaml:"name" json:"name"`               // Friendly display name
	Description string          `yaml:"description" json:"description"` // Friendly display description
	Detail      json.RawMessage `yaml:"detail" json:"detail"`           // download-specific configuration
}

// PathMapperInstanceConfig maps paths between remote/docker and local environments.
type PathMapperInstanceConfig struct {
	Remote string `yaml:"remote" json:"remote"` // Remote or container internal path prefix
	Local  string `yaml:"local" json:"local"`   // Local or host accessible path prefix
}

// ParserInstanceConfig represents configuration for an individual parser component.
type ParserInstanceConfig struct {
	ID                  string   `yaml:"id" json:"id"`                                       // Unique component ID, e.g. "parser", "parser:anime"
	Name                string   `yaml:"name" json:"name"`                                   // Friendly display name
	Enabled             bool     `yaml:"enabled" json:"enabled"`                             // Whether this parser instance is enabled
	DefaultMediaType    string   `yaml:"default_media_type" json:"default_media_type"`       // Optional default fallback media type (movie/tv/anime)
	CustomReleaseGroups []string `yaml:"custom_release_groups" json:"custom_release_groups"` // Optional additional release group identifiers
}

// ScraperInstanceConfig represents configuration for an individual scraper component.
type ScraperInstanceConfig struct {
	ID             string `yaml:"id" json:"id"`                           // Unique component ID, e.g. "scraper", "scraper:tmdb-en"
	Type           string `yaml:"type" json:"type"`                       // Scraper provider type, e.g. "tmdb"
	Name           string `yaml:"name" json:"name"`                       // Friendly display name
	Enabled        bool   `yaml:"enabled" json:"enabled"`                 // Whether this scraper instance is enabled
	APIKey         string `yaml:"api_key" json:"api_key"`                 // API authentication key
	BaseURL        string `yaml:"base_url" json:"base_url"`               // API base URL
	ImageBaseURL   string `yaml:"image_base_url" json:"image_base_url"`   // Artwork CDN base URL
	Language       string `yaml:"language" json:"language"`               // Preferred metadata language
	Proxy          string `yaml:"proxy" json:"proxy"`                     // Optional proxy URL
	GenerateNFO    bool   `yaml:"generate_nfo" json:"generate_nfo"`       // Whether to generate NFO metadata files
	DownloadImages bool   `yaml:"download_images" json:"download_images"` // Whether to download poster and backdrop artwork
}

// OrganizerInstanceConfig represents configuration for an individual organizer component.
type OrganizerInstanceConfig struct {
	ID             string `yaml:"id" json:"id"`                           // Unique component ID, e.g. "organizer", "organizer:movie"
	Name           string `yaml:"name" json:"name"`                       // Friendly display name
	Enabled        bool   `yaml:"enabled" json:"enabled"`                 // Whether this organizer instance is enabled
	MovieDir       string `yaml:"movie_dir" json:"movie_dir"`             // Root directory for movie library
	TVDir          string `yaml:"tv_dir" json:"tv_dir"`                   // Root directory for TV show library
	AnimeDir       string `yaml:"anime_dir" json:"anime_dir"`             // Root directory for anime library
	LinkMode       string `yaml:"link_mode" json:"link_mode"`             // hardlink, copy, move, symlink
	MinFileSize    int64  `yaml:"min_file_size" json:"min_file_size"`     // Minimum file size in bytes for valid video files
	GenerateNFO    bool   `yaml:"generate_nfo" json:"generate_nfo"`       // Whether to generate NFO files
	DownloadImages bool   `yaml:"download_images" json:"download_images"` // Whether to download images
	ScraperID      string `yaml:"scraper_id" json:"scraper_id"`           // Optional target scraper component ID
}

// PluginConfig holds global configuration for the plugin system.
type PluginConfig struct {
	Enabled bool                              `yaml:"enabled" json:"enabled"` // Whether the plugin system is enabled
	Items   map[string]map[string]interface{} `yaml:"items" json:"items"`     // Custom configurations per plugin ID
}

// DefaultComponentConfig returns default component configuration.
func DefaultComponentConfig() *ComponentConfig {
	return &ComponentConfig{
		Downloaders:  make([]DownloaderComponentConfig, 0),
		PathMappings: make([]PathMapperInstanceConfig, 0),
		Parsers:      make([]ParserInstanceConfig, 0),
		Scrapers:     make([]ScraperInstanceConfig, 0),
		Organizers:   make([]OrganizerInstanceConfig, 0),
		Plugins: PluginConfig{
			Enabled: true,
			Items:   make(map[string]map[string]interface{}),
		},
	}
}

// GetDownloaderConfigs returns all configured downloader instances.
func (c *ComponentConfig) GetDownloaderConfigs() []DownloaderComponentConfig {
	if len(c.Downloaders) > 0 {
		return c.Downloaders
	}
	return make([]DownloaderComponentConfig, 0)
}

// GetPathMappings returns global directory path mappings.
func (c *ComponentConfig) GetPathMappings() []PathMapperInstanceConfig {
	if len(c.PathMappings) > 0 {
		return c.PathMappings
	}
	return make([]PathMapperInstanceConfig, 0)
}

// GetParserConfigs returns all configured parser instances.
func (c *ComponentConfig) GetParserConfigs() []ParserInstanceConfig {
	if len(c.Parsers) > 0 {
		return c.Parsers
	}
	return make([]ParserInstanceConfig, 0)
}

// GetScraperConfigs returns all configured scraper instances.
func (c *ComponentConfig) GetScraperConfigs() []ScraperInstanceConfig {
	if len(c.Scrapers) > 0 {
		return c.Scrapers
	}
	return make([]ScraperInstanceConfig, 0)
}

// GetOrganizerConfigs returns all configured organizer instances.
func (c *ComponentConfig) GetOrganizerConfigs() []OrganizerInstanceConfig {
	if len(c.Organizers) > 0 {
		return c.Organizers
	}
	return make([]OrganizerInstanceConfig, 0)
}

// LoadComponentConfigFromFile loads component configuration from a specified YAML file path.
func LoadComponentConfigFromFile(path string) (*ComponentConfig, error) {
	cfg := DefaultComponentConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read component config file error: %w", err)
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse component config json error: %w", err)
	}

	return cfg, nil
}

// SaveComponentConfigToFile writes component configuration to a YAML file.
func SaveComponentConfigToFile(path string, cfg *ComponentConfig) error {
	if cfg == nil {
		return errors.New("component config is nil")
	}

	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create directory for component config error: %w", err)
		}
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal component config error: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write component config file error: %w", err)
	}

	return nil
}

// LoadComponentConfigFromDB loads dynamic component configurations from database records.
func LoadComponentConfigFromDB(db *gorm.DB) (*ComponentConfig, error) {
	if db == nil {
		return nil, errors.New("database connection is nil")
	}

	var settings []database.SystemSetting
	if err := db.Find(&settings).Error; err != nil {
		return nil, fmt.Errorf("query system settings error: %w", err)
	}

	if len(settings) == 0 {
		return nil, ErrConfigNotFound
	}

	// Build key-value lookup map
	settingsMap := make(map[string]string, len(settings))
	for _, s := range settings {
		settingsMap[s.Key] = s.Value
	}

	// Verify that at least one component configuration section exists
	componentSections := []string{
		SectionDownloaders,
		SectionPathMappings,
		SectionPathMappers,
		SectionParsers,
		SectionScrapers,
		SectionOrganizers,
		SectionPlugins,
	}
	hasSection := false
	for _, sec := range componentSections {
		if _, exists := settingsMap[sec]; exists {
			hasSection = true
			break
		}
	}
	if !hasSection {
		return nil, ErrConfigNotFound
	}

	cfg := DefaultComponentConfig()

	// Populate component sections from JSON payloads
	if val, ok := settingsMap[SectionDownloaders]; ok && strings.TrimSpace(val) != "" {
		if err := json.Unmarshal([]byte(val), &cfg.Downloaders); err != nil {
			return nil, fmt.Errorf("unmarshal downloaders error: %w", err)
		}
	}
	if val, ok := settingsMap[SectionPathMappings]; ok && strings.TrimSpace(val) != "" {
		if err := json.Unmarshal([]byte(val), &cfg.PathMappings); err != nil {
			return nil, fmt.Errorf("unmarshal path_mappings error: %w", err)
		}
	} else if val, ok := settingsMap[SectionPathMappers]; ok && strings.TrimSpace(val) != "" {
		if err := json.Unmarshal([]byte(val), &cfg.PathMappings); err != nil {
			return nil, fmt.Errorf("unmarshal pathmappers error: %w", err)
		}
	}
	if val, ok := settingsMap[SectionParsers]; ok && strings.TrimSpace(val) != "" {
		if err := json.Unmarshal([]byte(val), &cfg.Parsers); err != nil {
			return nil, fmt.Errorf("unmarshal parsers error: %w", err)
		}
	}
	if val, ok := settingsMap[SectionScrapers]; ok && strings.TrimSpace(val) != "" {
		if err := json.Unmarshal([]byte(val), &cfg.Scrapers); err != nil {
			return nil, fmt.Errorf("unmarshal scrapers error: %w", err)
		}
	}
	if val, ok := settingsMap[SectionOrganizers]; ok && strings.TrimSpace(val) != "" {
		if err := json.Unmarshal([]byte(val), &cfg.Organizers); err != nil {
			return nil, fmt.Errorf("unmarshal organizers error: %w", err)
		}
	}
	if val, ok := settingsMap[SectionPlugins]; ok && strings.TrimSpace(val) != "" {
		if err := json.Unmarshal([]byte(val), &cfg.Plugins); err != nil {
			return nil, fmt.Errorf("unmarshal plugins error: %w", err)
		}
	}

	return cfg, nil
}

// SaveComponentConfigToDB persists dynamic component configuration sections into the database.
func SaveComponentConfigToDB(db *gorm.DB, cfg *ComponentConfig) error {
	if db == nil {
		return errors.New("database connection is nil")
	}
	if cfg == nil {
		return errors.New("component config is nil")
	}

	sections := []struct {
		key      string
		category string
		desc     string
		data     any
	}{
		{SectionDownloaders, "component", "Downloader component instances configuration", cfg.Downloaders},
		{SectionPathMappings, "component", "Path mapper instances configuration", cfg.PathMappings},
		{SectionParsers, "component", "Parser component instances configuration", cfg.Parsers},
		{SectionScrapers, "component", "Scraper component instances configuration", cfg.Scrapers},
		{SectionOrganizers, "component", "Organizer component instances configuration", cfg.Organizers},
		{SectionPlugins, "component", "Plugin system configuration", cfg.Plugins},
	}

	now := time.Now()
	for _, sec := range sections {
		bytes, err := json.Marshal(sec.data)
		if err != nil {
			return fmt.Errorf("marshal section '%s' error: %w", sec.key, err)
		}

		setting := database.SystemSetting{
			Key:       sec.key,
			Value:     string(bytes),
			UpdatedAt: now,
		}
		if err := db.Save(&setting).Error; err != nil {
			return fmt.Errorf("save section '%s' error: %w", sec.key, err)
		}
	}

	return nil
}

// SaveComponentSection saves a single component configuration section to the database.
func SaveComponentSection(db *gorm.DB, section string, data any) error {
	if db == nil {
		return errors.New("database connection is nil")
	}

	bytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal section '%s' error: %w", section, err)
	}

	setting := database.SystemSetting{
		Key:       section,
		Value:     string(bytes),
		UpdatedAt: time.Now(),
	}

	if err := db.Save(&setting).Error; err != nil {
		return fmt.Errorf("save section '%s' error: %w", section, err)
	}

	return nil
}

// LoadComponentSection loads a single component configuration section from the database into target.
func LoadComponentSection(db *gorm.DB, section string, target any) error {
	if db == nil {
		return errors.New("database connection is nil")
	}

	var setting database.SystemSetting
	if err := db.Where("key = ?", section).First(&setting).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrConfigNotFound
		}
		return fmt.Errorf("query section '%s' error: %w", section, err)
	}

	if err := json.Unmarshal([]byte(setting.Value), target); err != nil {
		return fmt.Errorf("unmarshal section '%s' error: %w", section, err)
	}

	return nil
}
