package config

import (
	"errors"
	"testing"

	"github.com/jackie8tao/mediahelper/internal/database"
)

func TestDBConfig(t *testing.T) {
	db, err := database.InitDB(":memory:")
	if err != nil {
		t.Fatalf("InitDB in memory error: %v", err)
	}

	// 1. Loading from empty database should return ErrConfigNotFound
	t.Run("LoadFromDB_Empty", func(t *testing.T) {
		_, err := LoadFromDB(db)
		if !errors.Is(err, ErrConfigNotFound) {
			t.Fatalf("expected ErrConfigNotFound, got: %v", err)
		}
	})

	// 2. SaveToDB and LoadFromDB round-trip test
	t.Run("SaveToDB_and_LoadFromDB", func(t *testing.T) {
		orig := DefaultConfig()
		orig.Server.Port = 9999
		orig.QBittorrent.URL = "http://192.168.1.100:8080"
		orig.TMDB.APIKey = "test_api_key_12345"
		orig.Library.LinkMode = "symlink"
		orig.Library.MovieDir = "/data/media/movies"

		if err := SaveToDB(db, orig); err != nil {
			t.Fatalf("SaveToDB error: %v", err)
		}

		loaded, err := LoadFromDB(db)
		if err != nil {
			t.Fatalf("LoadFromDB error: %v", err)
		}

		if loaded.Server.Port != 9999 {
			t.Errorf("expected server port 9999, got %d", loaded.Server.Port)
		}
		if loaded.QBittorrent.URL != "http://192.168.1.100:8080" {
			t.Errorf("expected qb url mismatch: %s", loaded.QBittorrent.URL)
		}
		if loaded.TMDB.APIKey != "test_api_key_12345" {
			t.Errorf("expected tmdb api key mismatch: %s", loaded.TMDB.APIKey)
		}
		if loaded.Library.LinkMode != "symlink" {
			t.Errorf("expected link mode symlink, got %s", loaded.Library.LinkMode)
		}
		if loaded.Library.MovieDir != "/data/media/movies" {
			t.Errorf("expected movie dir mismatch: %s", loaded.Library.MovieDir)
		}
	})

	// 3. LoadOrInit initial seeding and subsequent reload test
	t.Run("LoadOrInit", func(t *testing.T) {
		memDB, err := database.InitDB(":memory:")
		if err != nil {
			t.Fatalf("InitDB memory error: %v", err)
		}

		initial := DefaultConfig()
		initial.Server.Port = 8888
		initial.TMDB.Language = "en-US"

		// First call should initialize and write the fallback to DB
		cfg1, err := LoadOrInit(memDB, initial)
		if err != nil {
			t.Fatalf("LoadOrInit error: %v", err)
		}
		if cfg1.Server.Port != 8888 {
			t.Errorf("expected port 8888, got %d", cfg1.Server.Port)
		}

		// Second call should load the persisted config from DB, ignoring new fallback
		fallbackWithDiffPort := DefaultConfig()
		fallbackWithDiffPort.Server.Port = 7777

		cfg2, err := LoadOrInit(memDB, fallbackWithDiffPort)
		if err != nil {
			t.Fatalf("second LoadOrInit error: %v", err)
		}
		if cfg2.Server.Port != 8888 {
			t.Errorf("expected to load preserved port 8888 from DB, got %d", cfg2.Server.Port)
		}
	})

	// 4. Granular dot-notation key override test
	t.Run("GranularOverrides", func(t *testing.T) {
		if err := SetSetting(db, "server.port", "8081", "server", "override port"); err != nil {
			t.Fatalf("SetSetting error: %v", err)
		}
		if err := SetSetting(db, "tmdb.language", "zh-TW", "scraper", "override lang"); err != nil {
			t.Fatalf("SetSetting error: %v", err)
		}

		loaded, err := LoadFromDB(db)
		if err != nil {
			t.Fatalf("LoadFromDB error: %v", err)
		}

		if loaded.Server.Port != 8081 {
			t.Errorf("expected overridden port 8081, got %d", loaded.Server.Port)
		}
		if loaded.TMDB.Language != "zh-TW" {
			t.Errorf("expected overridden language zh-TW, got %s", loaded.TMDB.Language)
		}
	})

	// 5. Section-level SaveSection / LoadSection test
	t.Run("SectionSaveLoad", func(t *testing.T) {
		qbCfg := QBittorrentConfig{
			Enabled:      true,
			URL:          "http://nas-local:9091",
			Username:     "qbadmin",
			Password:     "qbpass",
			Categories:   []string{"movie"},
			AutoTag:      false,
			PollInterval: 15,
		}

		if err := SaveSection(db, SectionQBittorrent, "downloader", "custom qb", qbCfg); err != nil {
			t.Fatalf("SaveSection error: %v", err)
		}

		var loadedQB QBittorrentConfig
		if err := LoadSection(db, SectionQBittorrent, &loadedQB); err != nil {
			t.Fatalf("LoadSection error: %v", err)
		}

		if loadedQB.URL != "http://nas-local:9091" || loadedQB.Username != "qbadmin" || loadedQB.PollInterval != 15 {
			t.Errorf("loaded section mismatch: %+v", loadedQB)
		}
	})

	// 6. GetAllSettings test
	t.Run("GetAllSettings", func(t *testing.T) {
		all, err := GetAllSettings(db)
		if err != nil {
			t.Fatalf("GetAllSettings error: %v", err)
		}
		if len(all) == 0 {
			t.Errorf("expected non-empty settings map")
		}
		if _, ok := all[SectionServer]; !ok {
			t.Errorf("expected key '%s' in all settings", SectionServer)
		}
	})
}
