package scraper

import (
	"context"
	"errors"
	"testing"

	"github.com/jackie8tao/mediahelper/internal/config"
	"github.com/jackie8tao/mediahelper/internal/parser"
	"github.com/jackie8tao/mediahelper/internal/pipeline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockScraper struct {
	meta       *MediaMetadata
	shouldFail bool
}

func (m *mockScraper) Scrape(ctx context.Context, info *parser.MediaInfo) (*MediaMetadata, error) {
	if m.shouldFail {
		return nil, errors.New("mock scrape failed")
	}
	if m.meta != nil {
		return m.meta, nil
	}
	return &MediaMetadata{
		TMDBID:    12345,
		Title:     info.Title,
		Year:      info.Year,
		MediaType: info.MediaType,
		Plot:      "Mock Overview",
	}, nil
}

func (m *mockScraper) DownloadImage(ctx context.Context, imageURL string, destPath string) error {
	return nil
}

func TestScraperComponent_Execute(t *testing.T) {
	mock := &mockScraper{
		meta: &MediaMetadata{
			TMDBID:        872585,
			Title:         "Oppenheimer",
			OriginalTitle: "Oppenheimer",
			Year:          2023,
			MediaType:     parser.MediaTypeMovie,
			PosterURL:     "https://image.tmdb.org/t/p/original/poster.jpg",
			BackdropURL:   "https://image.tmdb.org/t/p/original/backdrop.jpg",
			Plot:          "The story of J. Robert Oppenheimer's role in the development of the atomic bomb.",
		},
	}

	comp := NewComponent(config.ScraperInstanceConfig{
		ID:      "scraper:test",
		Name:    "Test Scraper",
		Enabled: true,
	}, mock)

	assert.Equal(t, "scraper:test", comp.ID())
	assert.Equal(t, "Test Scraper", comp.Name())
	assert.NotEmpty(t, comp.Description())
	assert.Equal(t, mock, comp.Scraper())

	ctx := context.Background()

	// 1. MediaInfo in params
	t.Run("MediaInfoInParams", func(t *testing.T) {
		info := &parser.MediaInfo{Title: "Oppenheimer", Year: 2023, MediaType: parser.MediaTypeMovie}
		sCtx := pipeline.NewStepContext("exec-1", 1, "step-1", "Scrape", "scraper:test",
			map[string]any{"media_info": info}, nil, nil)
		out, err := comp.Execute(ctx, sCtx)
		require.NoError(t, err)
		assert.Equal(t, 872585, out["tmdb_id"])
		assert.Equal(t, "Oppenheimer", out["title"])
		assert.Equal(t, 2023, out["year"])
		assert.Equal(t, "movie", out["media_type"])
	})

	// 2. MediaInfo in upstream steps
	t.Run("MediaInfoInSteps", func(t *testing.T) {
		steps := map[string]map[string]any{
			"parser": {
				"media_info": &parser.MediaInfo{Title: "Oppenheimer", Year: 2023},
			},
		}
		sCtx := pipeline.NewStepContext("exec-2", 1, "step-2", "Scrape", "scraper:test",
			nil, nil, steps)
		out, err := comp.Execute(ctx, sCtx)
		require.NoError(t, err)
		assert.Equal(t, 872585, out["tmdb_id"])
	})

	// 3. Constructed parameters
	t.Run("ConstructedParams", func(t *testing.T) {
		sCtx := pipeline.NewStepContext("exec-3", 1, "step-3", "Scrape", "scraper:test",
			map[string]any{"title": "Oppenheimer", "year": 2023, "media_type": "movie"}, nil, nil)
		out, err := comp.Execute(ctx, sCtx)
		require.NoError(t, err)
		assert.Equal(t, 872585, out["tmdb_id"])
	})

	// 4. Missing title error
	t.Run("MissingTitle", func(t *testing.T) {
		sCtx := pipeline.NewStepContext("exec-4", 1, "step-4", "Scrape", "scraper:test",
			nil, nil, nil)
		_, err := comp.Execute(ctx, sCtx)
		assert.Error(t, err)
	})

	// 5. Scraper error
	t.Run("ScraperFailure", func(t *testing.T) {
		failComp := NewComponent(config.ScraperInstanceConfig{ID: "scraper:fail"}, &mockScraper{shouldFail: true})
		sCtx := pipeline.NewStepContext("exec-5", 1, "step-5", "Scrape", "scraper:fail",
			map[string]any{"title": "Fail Movie"}, nil, nil)
		_, err := failComp.Execute(ctx, sCtx)
		assert.Error(t, err)
	})
}

func TestScraperComponent_RegisterComponents(t *testing.T) {
	RegisterFactory("mock", func(cfg config.ScraperInstanceConfig) (Scraper, error) {
		return &mockScraper{}, nil
	})

	cfg := &config.ComponentConfig{
		Scrapers: []config.ScraperInstanceConfig{
			{ID: "scraper:movie", Type: "mock", Name: "Movie Scraper", Enabled: true},
			{ID: "scraper:tv", Type: "mock", Name: "TV Scraper", Enabled: true},
			{ID: "scraper:disabled", Type: "mock", Enabled: false},
		},
	}

	registry := pipeline.NewComponentRegistry()
	comps, err := RegisterComponents(cfg, registry)
	require.NoError(t, err)
	assert.Len(t, comps, 2)

	sc1, ok1 := registry.Get("scraper:movie")
	assert.True(t, ok1)
	assert.Equal(t, "Movie Scraper", sc1.Name())

	alias, okAlias := registry.Get("scraper")
	assert.True(t, okAlias)
	assert.Equal(t, "scraper", alias.ID())
}
