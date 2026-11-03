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
	SectionDownloaders = "downloaders"
	SectionParsers     = "parsers"
	SectionScrapers    = "scrapers"
	SectionOrganizers  = "organizers"
	SectionPlugins     = "plugins"
)

// ErrConfigNotFound indicates no component configuration was found in the database.
var ErrConfigNotFound = errors.New("no component configuration found in database")

// ============================================================================
// ComponentConfig (Dynamic Component Configurations Managed via API & DB)
// ============================================================================

// ComponentConfig holds dynamic pipeline and component configurations managed via API and stored in the database.
type ComponentConfig struct {
	Downloaders []DownloaderComponentConfig `json:"downloaders"`
	Parsers     []ParserComponentConfig     `json:"parsers"`
	Scrapers    []ScraperComponentConfig    `json:"scrapers"`
	Organizers  []OrganizerComponentConfig  `json:"organizers"`
	Plugins     PluginConfig                `json:"plugins"`
}

// DownloaderComponentConfig represents configuration for an individual downloader instance component.
type DownloaderComponentConfig struct {
	Enabled     bool            `json:"enabled"`     // Whether this downloader is enabled
	ID          string          `json:"id"`          // Unique component ID, e.g. "downloader", "downloader:movie", "downloader:tv"
	Type        string          `json:"type"`        // Downloader type, e.g. "qbittorrent", "transmission"
	Name        string          `json:"name"`        // Friendly display name
	Description string          `json:"description"` // Friendly display description
	Detail      json.RawMessage `json:"detail"`      // Downloader-specific configuration
}

// ParserComponentConfig represents configuration for an individual AIGC parser instance component.
type ParserComponentConfig struct {
	Enabled     bool            `json:"enabled"`     // Whether this parser instance is enabled
	ID          string          `json:"id"`          // Unique component ID, e.g. "parser", "parser:openai", "parser:gemini"
	Type        string          `json:"type"`        // Parser provider type, e.g. "openai", "gemini"
	Name        string          `json:"name"`        // Friendly display name
	Description string          `json:"description"` // Friendly display description
	Detail      json.RawMessage `json:"detail"`      // Parser-specific configuration
}

// ScraperComponentConfig represents configuration for an individual scraper instance component.
type ScraperComponentConfig struct {
	Enabled     bool            `json:"enabled"`     // Whether this scraper instance is enabled
	ID          string          `json:"id"`          // Unique component ID, e.g. "scraper", "scraper:tmdb-zh", "scraper:tmdb-en"
	Type        string          `json:"type"`        // Scraper provider type, e.g. "tmdb"
	Name        string          `json:"name"`        // Friendly display name
	Description string          `json:"description"` // Friendly display description
	Detail      json.RawMessage `json:"detail"`      // Scraper-specific configuration
}

// OrganizerComponentConfig represents configuration for an individual organizer instance component.
type OrganizerComponentConfig struct {
	Enabled     bool            `json:"enabled"`     // Whether this organizer instance is enabled
	ID          string          `json:"id"`          // Unique component ID, e.g. "organizer", "organizer:movie", "organizer:tv"
	Type        string          `json:"type"`        // Organizer type, e.g. "local"
	Name        string          `json:"name"`        // Friendly display name
	Description string          `json:"description"` // Friendly display description
	Detail      json.RawMessage `json:"detail"`      // Organizer-specific configuration
}

// PluginConfig holds global configuration for the plugin system.
type PluginConfig struct {
	Enabled bool                              `json:"enabled"` // Whether the plugin system is enabled
	Items   map[string]map[string]interface{} `json:"items"`   // Custom configurations per plugin ID
}

// DefaultComponentConfig returns default component configuration.
func DefaultComponentConfig() *ComponentConfig {
	return &ComponentConfig{
		Downloaders: make([]DownloaderComponentConfig, 0),
		Parsers:     make([]ParserComponentConfig, 0),
		Scrapers:    make([]ScraperComponentConfig, 0),
		Organizers:  make([]OrganizerComponentConfig, 0),
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

// GetParserConfigs returns all configured parser instances.
func (c *ComponentConfig) GetParserConfigs() []ParserComponentConfig {
	if len(c.Parsers) > 0 {
		return c.Parsers
	}
	return make([]ParserComponentConfig, 0)
}

// GetScraperConfigs returns all configured scraper instances.
func (c *ComponentConfig) GetScraperConfigs() []ScraperComponentConfig {
	if len(c.Scrapers) > 0 {
		return c.Scrapers
	}
	return make([]ScraperComponentConfig, 0)
}

// GetOrganizerConfigs returns all configured organizer instances.
func (c *ComponentConfig) GetOrganizerConfigs() []OrganizerComponentConfig {
	if len(c.Organizers) > 0 {
		return c.Organizers
	}
	return make([]OrganizerComponentConfig, 0)
}

// LoadComponentConfigFromFile loads component configuration from a specified JSON file path.
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

// SaveComponentConfigToFile writes component configuration to a JSON file.
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
