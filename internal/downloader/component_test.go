package downloader

import (
	"context"
	"errors"
	"testing"

	"github.com/jackie8tao/mediahelper/internal/config"
	"github.com/jackie8tao/mediahelper/internal/pipeline"
)

func init() {
	RegisterFactory("mock", func(cfg config.DownloaderInstanceConfig) (Downloader, error) {
		return &mockDownloader{name: cfg.Name}, nil
	})
	RegisterFactory("qbittorrent", func(cfg config.DownloaderInstanceConfig) (Downloader, error) {
		return &mockDownloader{name: cfg.Name}, nil
	})
}

type mockDownloader struct {
	name          string
	addTagsCalled bool
	lastHashes    []string
	lastTags      []string
	torrents      []Torrent
	files         []TorrentFile
	shouldFail    bool
}

func (m *mockDownloader) Name() string {
	if m.name != "" {
		return m.name
	}
	return "MockDownloader"
}

func (m *mockDownloader) TestConnection(ctx context.Context) error {
	if m.shouldFail {
		return errors.New("connection failed")
	}
	return nil
}

func (m *mockDownloader) GetAllTorrents(ctx context.Context, filter string, category string) ([]Torrent, error) {
	if m.shouldFail {
		return nil, errors.New("get all torrents failed")
	}
	return m.torrents, nil
}

func (m *mockDownloader) GetCompletedTorrents(ctx context.Context, categories []string) ([]Torrent, error) {
	if m.shouldFail {
		return nil, errors.New("get completed torrents failed")
	}
	return m.torrents, nil
}

func (m *mockDownloader) GetTorrentFiles(ctx context.Context, hash string) ([]TorrentFile, error) {
	if m.shouldFail {
		return nil, errors.New("get files failed")
	}
	return m.files, nil
}

func (m *mockDownloader) AddTags(ctx context.Context, hashes []string, tags []string) error {
	if m.shouldFail {
		return errors.New("add tags failed")
	}
	m.addTagsCalled = true
	m.lastHashes = hashes
	m.lastTags = tags
	return nil
}

func (m *mockDownloader) RemoveTags(ctx context.Context, hashes []string, tags []string) error {
	if m.shouldFail {
		return errors.New("remove tags failed")
	}
	m.lastHashes = hashes
	m.lastTags = tags
	return nil
}

func TestDownloaderComponent_Execute(t *testing.T) {
	mockDL := &mockDownloader{
		torrents: []Torrent{
			{Hash: "hash123", Name: "Movie.2023.mkv", IsCompleted: true},
		},
		files: []TorrentFile{
			{Index: 0, Name: "Movie.2023.mkv", Size: 1048576},
		},
	}

	dlCfg := config.DownloaderInstanceConfig{
		ID:         "downloader:test",
		Name:       "Test Downloader",
		SuccessTag: "Test:Done",
		FailTag:    "Test:Fail",
		Categories: []string{"movie", "tv"},
	}

	comp := NewDownloaderComponent("downloader:test", "Test Downloader", mockDL, dlCfg)

	if comp.ID() != "downloader:test" || comp.Name() != "Test Downloader" {
		t.Fatalf("unexpected component ID/name: %s / %s", comp.ID(), comp.Name())
	}

	ctx := context.Background()

	// 1. Test add_tags action
	t.Run("Action_AddTags", func(t *testing.T) {
		sCtx := pipeline.NewStepContext("exec1", 1, "step1", "Tag Step", "downloader:test", map[string]any{
			"action": "add_tags",
			"hash":   "hash123",
			"tags":   []any{"MediaHelper:Done", "CustomTag"},
		}, nil, nil)

		res, err := comp.Execute(ctx, sCtx)
		if err != nil {
			t.Fatalf("Execute add_tags error: %v", err)
		}
		if res["success"] != true {
			t.Errorf("expected success true, got %v", res["success"])
		}
		if len(mockDL.lastTags) != 2 || mockDL.lastTags[0] != "MediaHelper:Done" {
			t.Errorf("expected tags registered, got %v", mockDL.lastTags)
		}
	})

	// 2. Test auto-deduce action from tag param and event hash
	t.Run("AutoDeduce_AddTags_From_Event", func(t *testing.T) {
		event := map[string]any{"hash": "event_hash_456"}
		sCtx := pipeline.NewStepContext("exec1", 1, "step2", "Auto Tag Step", "downloader:test", map[string]any{
			"tag": "AutoTagged",
		}, event, nil)

		res, err := comp.Execute(ctx, sCtx)
		if err != nil {
			t.Fatalf("Execute error: %v", err)
		}
		if res["action"] != "add_tags" {
			t.Errorf("expected auto action add_tags, got %v", res["action"])
		}
		if len(mockDL.lastHashes) != 1 || mockDL.lastHashes[0] != "event_hash_456" {
			t.Errorf("expected event hash used, got %v", mockDL.lastHashes)
		}
	})

	// 3. Test remove_tags action
	t.Run("Action_RemoveTags", func(t *testing.T) {
		sCtx := pipeline.NewStepContext("exec1", 1, "step3", "Untag Step", "downloader:test", map[string]any{
			"action": "remove_tags",
			"hashes": "hash1, hash2",
			"tags":   "TagToRemove",
		}, nil, nil)

		res, err := comp.Execute(ctx, sCtx)
		if err != nil {
			t.Fatalf("Execute remove_tags error: %v", err)
		}
		if res["action"] != "remove_tags" {
			t.Errorf("expected remove_tags action, got %v", res["action"])
		}
		if len(mockDL.lastHashes) != 2 {
			t.Errorf("expected 2 parsed hashes, got %v", mockDL.lastHashes)
		}
	})

	// 4. Test get_completed action
	t.Run("Action_GetCompleted", func(t *testing.T) {
		sCtx := pipeline.NewStepContext("exec1", 1, "step4", "Query Step", "downloader:test", map[string]any{
			"action": "get_completed",
		}, nil, nil)

		res, err := comp.Execute(ctx, sCtx)
		if err != nil {
			t.Fatalf("Execute get_completed error: %v", err)
		}
		if res["count"] != 1 {
			t.Errorf("expected count 1, got %v", res["count"])
		}
	})

	// 5. Test get_files action
	t.Run("Action_GetFiles", func(t *testing.T) {
		sCtx := pipeline.NewStepContext("exec1", 1, "step5", "Files Step", "downloader:test", map[string]any{
			"action": "get_files",
			"hash":   "hash123",
		}, nil, nil)

		res, err := comp.Execute(ctx, sCtx)
		if err != nil {
			t.Fatalf("Execute get_files error: %v", err)
		}
		if res["count"] != 1 {
			t.Errorf("expected file count 1, got %v", res["count"])
		}
	})

	// 6. Test connection test action
	t.Run("Action_TestConnection", func(t *testing.T) {
		sCtx := pipeline.NewStepContext("exec1", 1, "step6", "Ping Step", "downloader:test", map[string]any{
			"action": "test_connection",
		}, nil, nil)

		res, err := comp.Execute(ctx, sCtx)
		if err != nil {
			t.Fatalf("Execute test_connection error: %v", err)
		}
		if res["success"] != true {
			t.Errorf("expected test_connection success")
		}
	})
}

func TestRegisterComponents(t *testing.T) {
	// 1. Multiple configured downloaders
	t.Run("MultipleDownloadersConfig", func(t *testing.T) {
		cfg := &config.ComponentConfig{
			Downloaders: []config.DownloaderInstanceConfig{
				{
					ID:       "downloader:movie",
					Type:     "qbittorrent",
					Name:     "Movie QB",
					Enabled:  true,
					URL:      "http://127.0.0.1:8080",
					Username: "admin",
				},
				{
					ID:       "downloader:tv",
					Type:     "qbittorrent",
					Name:     "TV QB",
					Enabled:  true,
					URL:      "http://127.0.0.1:8081",
					Username: "admin",
				},
				{
					ID:      "downloader:disabled",
					Type:    "qbittorrent",
					Enabled: false,
					URL:     "http://127.0.0.1:8082",
				},
			},
		}

		registry := pipeline.NewComponentRegistry()
		comps, err := RegisterComponents(cfg, registry)
		if err != nil {
			t.Fatalf("RegisterComponents error: %v", err)
		}

		if len(comps) != 2 {
			t.Fatalf("expected 2 active downloader components, got %d", len(comps))
		}

		// Verify registration in registry
		c1, ok1 := registry.Get("downloader:movie")
		if !ok1 || c1 == nil {
			t.Errorf("expected downloader:movie in registry")
		}

		c2, ok2 := registry.Get("downloader:tv")
		if !ok2 || c2 == nil {
			t.Errorf("expected downloader:tv in registry")
		}

		_, ok3 := registry.Get("downloader:disabled")
		if ok3 {
			t.Errorf("expected disabled downloader NOT to be in registry")
		}
	})

	// 3. Error on nil registry
	t.Run("NilRegistry", func(t *testing.T) {
		cfg := config.DefaultComponentConfig()
		_, err := RegisterComponents(cfg, nil)
		if err == nil {
			t.Errorf("expected error on nil registry")
		}
	})
}
