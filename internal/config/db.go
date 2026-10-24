package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackie8tao/mediahelper/internal/database"
	"gorm.io/gorm"
)

// Section key constants for structured configuration modules
const (
	SectionServer       = "server"
	SectionDatabase     = "database"
	SectionQBittorrent  = "qbittorrent"
	SectionDownloaders  = "downloaders"
	SectionPathMappings = "path_mappings"
	SectionTMDB         = "tmdb"
	SectionLibrary      = "library"
	SectionPlugins      = "plugins"
)

// ErrConfigNotFound indicates no configuration was found in the database.
var ErrConfigNotFound = errors.New("no system configuration found in database")

// LoadFromDB loads global configuration from database records.
func LoadFromDB(db *gorm.DB) (*Config, error) {
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

	// Verify that at least one core configuration section exists
	hasSection := false
	for _, sec := range []string{SectionServer, SectionDatabase, SectionQBittorrent, SectionDownloaders, SectionPathMappings, SectionTMDB, SectionLibrary, SectionPlugins} {
		if _, exists := settingsMap[sec]; exists {
			hasSection = true
			break
		}
	}
	if !hasSection {
		return nil, ErrConfigNotFound
	}

	cfg := DefaultConfig()

	// 1. Populate module sections from JSON payloads
	if val, ok := settingsMap[SectionServer]; ok && strings.TrimSpace(val) != "" {
		_ = json.Unmarshal([]byte(val), &cfg.Server)
	}
	if val, ok := settingsMap[SectionDatabase]; ok && strings.TrimSpace(val) != "" {
		_ = json.Unmarshal([]byte(val), &cfg.Database)
	}
	if val, ok := settingsMap[SectionQBittorrent]; ok && strings.TrimSpace(val) != "" {
		_ = json.Unmarshal([]byte(val), &cfg.QBittorrent)
	}
	if val, ok := settingsMap[SectionDownloaders]; ok && strings.TrimSpace(val) != "" {
		_ = json.Unmarshal([]byte(val), &cfg.Downloaders)
	}
	if val, ok := settingsMap[SectionPathMappings]; ok && strings.TrimSpace(val) != "" {
		_ = json.Unmarshal([]byte(val), &cfg.PathMappings)
	}
	if val, ok := settingsMap[SectionTMDB]; ok && strings.TrimSpace(val) != "" {
		_ = json.Unmarshal([]byte(val), &cfg.TMDB)
	}
	if val, ok := settingsMap[SectionLibrary]; ok && strings.TrimSpace(val) != "" {
		_ = json.Unmarshal([]byte(val), &cfg.Library)
	}
	if val, ok := settingsMap[SectionPlugins]; ok && strings.TrimSpace(val) != "" {
		_ = json.Unmarshal([]byte(val), &cfg.Plugins)
	}

	// 2. Apply granular key overrides (e.g. server.port, qbittorrent.url)
	applyGranularOverrides(cfg, settingsMap)

	return cfg, nil
}

// SaveToDB persists global configuration sections into the database.
func SaveToDB(db *gorm.DB, cfg *Config) error {
	if db == nil {
		return errors.New("database connection is nil")
	}
	if cfg == nil {
		return errors.New("config is nil")
	}

	sections := []struct {
		key      string
		category string
		desc     string
		data     any
	}{
		{SectionServer, "server", "HTTP server configuration", cfg.Server},
		{SectionDatabase, "database", "Database connection configuration", cfg.Database},
		{SectionQBittorrent, "downloader", "qBittorrent client configuration", cfg.QBittorrent},
		{SectionDownloaders, "downloader", "Configured downloaders list", cfg.Downloaders},
		{SectionPathMappings, "pathmapper", "Directory path mapping rules", cfg.PathMappings},
		{SectionTMDB, "scraper", "TMDB metadata scraping configuration", cfg.TMDB},
		{SectionLibrary, "library", "Media library directory and link mode configuration", cfg.Library},
		{SectionPlugins, "plugins", "Plugin manager configuration", cfg.Plugins},
	}

	now := time.Now()
	for _, sec := range sections {
		bytes, err := json.Marshal(sec.data)
		if err != nil {
			return fmt.Errorf("marshal section '%s' error: %w", sec.key, err)
		}

		setting := database.SystemSetting{
			Key:         sec.key,
			Value:       string(bytes),
			Category:    sec.category,
			Description: sec.desc,
			UpdatedAt:   now,
		}

		if err := db.Save(&setting).Error; err != nil {
			return fmt.Errorf("save section '%s' error: %w", sec.key, err)
		}
	}

	return nil
}

// LoadOrInit loads configuration from the database, or initializes it with fallback config if empty.
func LoadOrInit(db *gorm.DB, fallback *Config) (*Config, error) {
	if db == nil {
		return fallback, errors.New("database connection is nil")
	}
	if fallback == nil {
		fallback = DefaultConfig()
	}

	loaded, err := LoadFromDB(db)
	if err == nil {
		return loaded, nil
	}

	if errors.Is(err, ErrConfigNotFound) {
		// Initialize database with fallback config on first start
		if saveErr := SaveToDB(db, fallback); saveErr != nil {
			return fallback, fmt.Errorf("initialize database config error: %w", saveErr)
		}
		return fallback, nil
	}

	return fallback, err
}

// GetSetting retrieves a single configuration value by key.
func GetSetting(db *gorm.DB, key string) (string, error) {
	if db == nil {
		return "", errors.New("database connection is nil")
	}

	var setting database.SystemSetting
	if err := db.First(&setting, "key = ?", key).Error; err != nil {
		return "", err
	}
	return setting.Value, nil
}

// SetSetting saves or updates a single configuration key-value entry.
func SetSetting(db *gorm.DB, key string, value string, category string, desc string) error {
	if db == nil {
		return errors.New("database connection is nil")
	}

	setting := database.SystemSetting{
		Key:         key,
		Value:       value,
		Category:    category,
		Description: desc,
		UpdatedAt:   time.Now(),
	}

	return db.Save(&setting).Error
}

// GetAllSettings retrieves all raw configuration settings as a key-value map.
func GetAllSettings(db *gorm.DB) (map[string]string, error) {
	if db == nil {
		return nil, errors.New("database connection is nil")
	}

	var settings []database.SystemSetting
	if err := db.Find(&settings).Error; err != nil {
		return nil, err
	}

	res := make(map[string]string, len(settings))
	for _, s := range settings {
		res[s.Key] = s.Value
	}
	return res, nil
}

// LoadSection loads a specific configuration module object from the database.
func LoadSection(db *gorm.DB, section string, target any) error {
	val, err := GetSetting(db, section)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(val), target)
}

// SaveSection persists a specific configuration module object to the database.
func SaveSection(db *gorm.DB, section string, category string, desc string, val any) error {
	bytes, err := json.Marshal(val)
	if err != nil {
		return fmt.Errorf("marshal section '%s' error: %w", section, err)
	}
	return SetSetting(db, section, string(bytes), category, desc)
}

// applyGranularOverrides handles dot-separated granular overrides.
func applyGranularOverrides(cfg *Config, settings map[string]string) {
	for k, v := range settings {
		switch strings.ToLower(k) {
		case "server.host":
			cfg.Server.Host = v
		case "server.port":
			if port, err := strconv.Atoi(v); err == nil {
				cfg.Server.Port = port
			}
		case "database.dsn":
			cfg.Database.DSN = v
		case "qbittorrent.url":
			cfg.QBittorrent.URL = v
		case "qbittorrent.username":
			cfg.QBittorrent.Username = v
		case "qbittorrent.password":
			cfg.QBittorrent.Password = v
		case "qbittorrent.enabled":
			cfg.QBittorrent.Enabled = (strings.ToLower(v) == "true" || v == "1")
		case "tmdb.api_key":
			cfg.TMDB.APIKey = v
		case "tmdb.base_url":
			cfg.TMDB.BaseURL = v
		case "tmdb.language":
			cfg.TMDB.Language = v
		case "library.movie_dir":
			cfg.Library.MovieDir = v
		case "library.tv_dir":
			cfg.Library.TVDir = v
		case "library.anime_dir":
			cfg.Library.AnimeDir = v
		case "library.link_mode":
			cfg.Library.LinkMode = v
		}
	}
}
