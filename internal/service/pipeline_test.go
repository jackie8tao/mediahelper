package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackie8tao/mediahelper/internal/config"
	"github.com/jackie8tao/mediahelper/internal/downloader"
	"github.com/jackie8tao/mediahelper/internal/organizer"
	"github.com/jackie8tao/mediahelper/internal/parser"
	"github.com/jackie8tao/mediahelper/internal/scraper"
)

type mockDownloader struct {
	addedTags   map[string][]string
	removedTags map[string][]string
}

func newMockDownloader() *mockDownloader {
	return &mockDownloader{
		addedTags:   make(map[string][]string),
		removedTags: make(map[string][]string),
	}
}

func (m *mockDownloader) Name() string { return "MockQB" }
func (m *mockDownloader) TestConnection(ctx context.Context) error {
	return nil
}
func (m *mockDownloader) GetAllTorrents(ctx context.Context, filter string, category string) ([]downloader.Torrent, error) {
	return []downloader.Torrent{
		{
			Hash:        "mock_hash_1",
			Name:        "Oppenheimer.2023.2160p.mkv",
			Category:    "movie",
			Tags:        []string{},
			ContentPath: "D:/fake/Oppenheimer.2023.2160p.mkv",
			IsCompleted: true,
		},
	}, nil
}
func (m *mockDownloader) GetCompletedTorrents(ctx context.Context, categories []string) ([]downloader.Torrent, error) {
	return []downloader.Torrent{
		{
			Hash:        "mock_hash_1",
			Name:        "Oppenheimer.2023.2160p.mkv",
			Category:    "movie",
			Tags:        []string{},
			ContentPath: "D:/fake/Oppenheimer.2023.2160p.mkv",
			IsCompleted: true,
		},
	}, nil
}
func (m *mockDownloader) GetTorrentFiles(ctx context.Context, hash string) ([]downloader.TorrentFile, error) {
	return nil, nil
}
func (m *mockDownloader) AddTags(ctx context.Context, hashes []string, tags []string) error {
	for _, h := range hashes {
		m.addedTags[h] = append(m.addedTags[h], tags...)
	}
	return nil
}
func (m *mockDownloader) RemoveTags(ctx context.Context, hashes []string, tags []string) error {
	for _, h := range hashes {
		m.removedTags[h] = append(m.removedTags[h], tags...)
	}
	return nil
}

type mockScraper struct{}

func (s *mockScraper) Scrape(ctx context.Context, info *parser.MediaInfo) (*scraper.MediaMetadata, error) {
	return &scraper.MediaMetadata{
		TMDBID:    872585,
		Title:     "奥本海默",
		Year:      2023,
		MediaType: parser.MediaTypeMovie,
	}, nil
}
func (s *mockScraper) DownloadImage(ctx context.Context, imageURL string, destPath string) error {
	return nil
}

func TestPipeline_ProcessTorrent(t *testing.T) {
	tempBase, err := os.MkdirTemp("", "pipeline_test_*")
	if err != nil {
		t.Fatalf("MkdirTemp failed: %v", err)
	}
	defer os.RemoveAll(tempBase)

	srcFile := filepath.Join(tempBase, "Oppenheimer.2023.2160p.mkv")
	_ = os.WriteFile(srcFile, make([]byte, 2048), 0644)

	movieLib := filepath.Join(tempBase, "movies")

	cfg := &config.Config{
		QBittorrent: config.QBittorrentConfig{
			Enabled:    true,
			AutoTag:    true,
			SuccessTag: "MediaHelper:Done",
			FailTag:    "MediaHelper:Error",
		},
		TMDB: config.TMDBConfig{
			APIKey:      "mock_key",
			GenerateNFO: false,
		},
		Library: config.LibraryConfig{
			MovieDir:    movieLib,
			LinkMode:    "hardlink",
			MinFileSize: 1024,
		},
	}

	dl := newMockDownloader()
	sc := &mockScraper{}
	org := organizer.NewOrganizer(*cfg, sc)
	p := NewPipeline(cfg, dl, sc, org, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	torrent := downloader.Torrent{
		Hash:        "hash_oppenheimer",
		Name:        "Oppenheimer.2023.2160p.mkv",
		Category:    "movie",
		ContentPath: srcFile,
		Tags:        []string{},
		IsCompleted: true,
	}

	// 1. 处理新种子任务
	res, err := p.ProcessTorrent(ctx, torrent)
	if err != nil {
		t.Fatalf("ProcessTorrent error: %v", err)
	}
	if !res.Success {
		t.Fatalf("expected success, got false")
	}

	// 校验打标
	tags := dl.addedTags["hash_oppenheimer"]
	if len(tags) == 0 || tags[0] != "MediaHelper:Done" {
		t.Errorf("expected tag MediaHelper:Done, got %+v", tags)
	}

	// 校验历史记录
	history := p.History().GetAll()
	if len(history) != 1 || history[0].Status != "success" {
		t.Errorf("history record mismatch: %+v", history)
	}

	// 2. 再次处理已打标种子 (应该自动跳过)
	torrent.Tags = []string{"MediaHelper:Done"}
	res, err = p.ProcessTorrent(ctx, torrent)
	if err != nil {
		t.Fatalf("second call error: %v", err)
	}
	if !res.Success {
		t.Errorf("expected skipped success true")
	}
}
