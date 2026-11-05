package scraper

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackie8tao/mediahelper/internal/config"
	"github.com/jackie8tao/mediahelper/internal/pipeline"
)

type mockScraper struct {
	scrapeFunc func(ctx context.Context, task ScrapeTask) (*ScrapeResult, error)
}

func (m *mockScraper) Name() string { return "MockScraper" }

func (m *mockScraper) Scrape(ctx context.Context, task ScrapeTask) (*ScrapeResult, error) {
	if m.scrapeFunc != nil {
		return m.scrapeFunc(ctx, task)
	}
	return &ScrapeResult{
		Success:   true,
		TargetDir: task.TargetDir,
		Metadata: &MediaMetadata{
			TMDBID: 12345,
			Title:  task.Title,
			Year:   task.Year,
		},
		CreatedFiles: []string{filepath.Join(task.TargetDir, "movie.nfo")},
	}, nil
}

func TestScraperComponent_Execute(t *testing.T) {
	tempDir := t.TempDir()
	comp := &ScraperComponent{
		scraper: &mockScraper{},
		cfg: config.ScraperComponentConfig{
			ID:   "test_scraper",
			Name: "Test Scraper",
		},
	}

	sCtx := pipeline.NewStepContext("exec-1", 1, "step-scrape", "Scrape Step", comp.ID(), map[string]any{
		"target_dir": tempDir,
		"title":      "Inception",
		"year":       2010,
		"media_type": "movie",
	})

	outputs, err := comp.Execute(context.Background(), sCtx)
	if err != nil {
		t.Fatalf("unexpected execute error: %v", err)
	}

	if outputs["title"] != "Inception" {
		t.Errorf("expected title Inception, got %v", outputs["title"])
	}
	if outputs["tmdb_id"] != 12345 {
		t.Errorf("expected tmdb_id 12345, got %v", outputs["tmdb_id"])
	}
	if outputs["target_dir"] != tempDir {
		t.Errorf("expected target_dir %s, got %v", tempDir, outputs["target_dir"])
	}
}

func TestScraperComponent_NilTarget(t *testing.T) {
	comp := &ScraperComponent{
		scraper: &mockScraper{
			scrapeFunc: func(ctx context.Context, task ScrapeTask) (*ScrapeResult, error) {
				if task.TargetDir == "" {
					return nil, os.ErrNotExist
				}
				return &ScrapeResult{Success: true}, nil
			},
		},
		cfg: config.ScraperComponentConfig{ID: "test"},
	}

	sCtx := pipeline.NewStepContext("exec-1", 1, "step-err", "Err", comp.ID(), map[string]any{})
	_, err := comp.Execute(context.Background(), sCtx)
	if err == nil {
		t.Errorf("expected error on empty target_dir, got nil")
	}
}

