package downloader

import (
	"context"
	"testing"

	"github.com/jackie8tao/mediahelper/internal/config"
	"github.com/jackie8tao/mediahelper/internal/pipeline"
)

type mockDownloader struct {
	addedOpts     AddTorrentOptions
	deletedHashes []string
	deleteFiles   bool
	torrents      map[string]Torrent
	addedTags     []string
	removedTags   []string
	pausedHashes  []string
	resumedHashes []string
}

func (m *mockDownloader) Name() string { return "MockDownloader" }

func (m *mockDownloader) AddTorrent(ctx context.Context, opts AddTorrentOptions) ([]string, error) {
	m.addedOpts = opts
	return []string{"hash_added_123"}, nil
}

func (m *mockDownloader) DeleteTorrent(ctx context.Context, hashes []string, deleteFiles bool) error {
	m.deletedHashes = hashes
	m.deleteFiles = deleteFiles
	return nil
}

func (m *mockDownloader) GetTorrents(ctx context.Context, opts GetTorrentsOptions) ([]Torrent, error) {
	var list []Torrent
	if len(opts.Hashes) > 0 {
		for _, h := range opts.Hashes {
			if t, ok := m.torrents[h]; ok {
				list = append(list, t)
			}
		}
		return list, nil
	}
	for _, t := range m.torrents {
		list = append(list, t)
	}
	return list, nil
}

func (m *mockDownloader) AddTags(ctx context.Context, hashes []string, tags []string) error {
	m.addedTags = tags
	return nil
}

func (m *mockDownloader) RemoveTags(ctx context.Context, hashes []string, tags []string) error {
	m.removedTags = tags
	return nil
}

func (m *mockDownloader) Pause(ctx context.Context, hashes []string) error {
	m.pausedHashes = hashes
	return nil
}

func (m *mockDownloader) Resume(ctx context.Context, hashes []string) error {
	m.resumedHashes = hashes
	return nil
}

func TestDownloaderComponent_ExecuteActions(t *testing.T) {
	mock := &mockDownloader{
		torrents: map[string]Torrent{
			"abc12345": {
				Hash:        "abc12345",
				Name:        "Inception.2010.1080p",
				Category:    "movie",
				Tags:        []string{"pt"},
				SavePath:    "/downloads",
				ContentPath: "/downloads/Inception.2010.1080p",
				Progress:    1.0,
				State:       "uploading",
				IsCompleted: true,
			},
		},
	}

	comp := &DownloaderComponent{
		dl: mock,
		cfg: config.DownloaderComponentConfig{
			ID:   "downloader:test",
			Name: "Test Downloader",
			Type: "mock",
		},
	}

	ctx := context.Background()

	// 1. Test add_torrent
	addCtx := pipeline.NewStepContext("exec-1", 1, "step-add", "Add", comp.ID(), map[string]any{
		"action":    ActionAddTorrent,
		"url":       "magnet:?xt=urn:btih:12345",
		"save_path": "/downloads/movies",
		"category":  "movie",
		"tags":      "MediaHelper,Auto",
		"paused":    true,
	})

	out, err := comp.Execute(ctx, addCtx)
	if err != nil {
		t.Fatalf("add_torrent failed: %v", err)
	}
	if out["hash"] != "hash_added_123" {
		t.Fatalf("expected hash 'hash_added_123', got %v", out["hash"])
	}
	if mock.addedOpts.Category != "movie" || !mock.addedOpts.Paused || len(mock.addedOpts.Tags) != 2 {
		t.Fatalf("unexpected AddTorrentOptions: %+v", mock.addedOpts)
	}

	// 2. Test get_torrents
	getCtx := pipeline.NewStepContext("exec-1", 1, "step-get", "Get", comp.ID(), map[string]any{
		"action": ActionGetTorrents,
		"hash":   "abc12345",
	})

	out, err = comp.Execute(ctx, getCtx)
	if err != nil {
		t.Fatalf("get_torrents failed: %v", err)
	}
	if out["name"] != "Inception.2010.1080p" || out["content_path"] != "/downloads/Inception.2010.1080p" {
		t.Fatalf("unexpected get_torrents output: %+v", out)
	}

	// 3. Test delete_torrent
	delCtx := pipeline.NewStepContext("exec-1", 1, "step-del", "Delete", comp.ID(), map[string]any{
		"action":       ActionDeleteTorrent,
		"hash":         "abc12345",
		"delete_files": true,
	})

	out, err = comp.Execute(ctx, delCtx)
	if err != nil {
		t.Fatalf("delete_torrent failed: %v", err)
	}
	if len(mock.deletedHashes) != 1 || mock.deletedHashes[0] != "abc12345" || !mock.deleteFiles {
		t.Fatalf("unexpected delete call: hashes=%v, deleteFiles=%v", mock.deletedHashes, mock.deleteFiles)
	}

	// 4. Test add_tags
	tagCtx := pipeline.NewStepContext("exec-1", 1, "step-tag", "Tag", comp.ID(), map[string]any{
		"action": ActionAddTags,
		"hash":   "abc12345",
		"tags":   []string{"MediaHelper:Done"},
	})

	if _, err := comp.Execute(ctx, tagCtx); err != nil {
		t.Fatalf("add_tags failed: %v", err)
	}
	if len(mock.addedTags) != 1 || mock.addedTags[0] != "MediaHelper:Done" {
		t.Fatalf("unexpected addedTags: %v", mock.addedTags)
	}
}
