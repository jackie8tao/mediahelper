package pipeline

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackie8tao/mediahelper/internal/database"
)

// SeedDefaultPipelines checks whether any pipelines exist; if empty, it seeds the default organize workflow.
func SeedDefaultPipelines(ctx context.Context, repo *Repository) error {
	existing, err := repo.ListPipelines(false)
	if err != nil {
		return fmt.Errorf("check existing pipelines error: %w", err)
	}

	if len(existing) > 0 {
		return nil
	}

	slog.Info("No existing pipelines found, seeding default media organize pipeline...")

	defaultPipeline := &database.Pipeline{
		Name:        "Default Media Organizer",
		Description: "Automated media workflow: Title Parsing -> TMDB Scraping -> Media Library Organizing -> Downloader Tagging",
		Enabled:     true,
		Triggers: []database.TriggerConfig{
			{
				Type:   TriggerTypeEvent,
				Source: "webhook:qbittorrent",
			},
			{
				Type:   TriggerTypeEvent,
				Source: "sync:manual",
			},
		},
		Steps: []database.StepConfig{
			{
				ID:        "parser",
				Name:      "Parse Media Title",
				Component: "parser",
				Params: map[string]any{
					"raw_title": "{{ event.name }}",
					"path":      "{{ event.content_path }}",
				},
			},
			{
				ID:        "scraper",
				Name:      "Scrape TMDB Metadata",
				Component: "scraper",
				Params: map[string]any{
					"title":      "{{ steps.parser.title }}",
					"year":       "{{ steps.parser.year }}",
					"media_type": "{{ steps.parser.media_type }}",
					"files":      "{{ steps.parser.files }}",
				},
			},
			{
				ID:        "organizer",
				Name:      "Organize Media Files",
				Component: "organizer",
				Params: map[string]any{
					"action":     "organize",
					"media_type": "{{ steps.scraper.media_type }}",
					"title":      "{{ steps.scraper.title }}",
					"year":       "{{ steps.scraper.year }}",
					"resolution": "{{ steps.parser.resolution }}",
					"files":      "{{ steps.parser.files }}",
				},
			},
			{
				ID:        "tag_done",
				Name:      "Tag Torrent Completed",
				Component: "downloader",
				Params: map[string]any{
					"action": "add_tags",
					"hash":   "{{ event.hash }}",
					"tags":   []string{"MediaHelper:Done"},
				},
				ContinueOnError: true,
			},
		},
	}

	if err := repo.CreatePipeline(defaultPipeline); err != nil {
		return fmt.Errorf("seed default pipeline failed: %w", err)
	}

	slog.Info("Default media organize pipeline seeded successfully", "id", defaultPipeline.ID, "name", defaultPipeline.Name)
	return nil
}
