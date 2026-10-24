package organizer

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackie8tao/mediahelper/internal/config"
	"github.com/jackie8tao/mediahelper/internal/downloader"
	"github.com/jackie8tao/mediahelper/internal/parser"
	"github.com/jackie8tao/mediahelper/internal/pipeline"
	"github.com/jackie8tao/mediahelper/internal/scraper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrganizerComponent_Execute(t *testing.T) {
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "downloads")
	movieLib := filepath.Join(tempDir, "movies")
	tvLib := filepath.Join(tempDir, "tv")
	_ = os.MkdirAll(sourceDir, 0755)

	// Create a dummy video file
	srcFile := filepath.Join(sourceDir, "Oppenheimer.2023.1080p.mkv")
	_ = os.WriteFile(srcFile, make([]byte, 1024), 0644)

	orgCfg := config.OrganizerInstanceConfig{
		ID:          "organizer:test",
		Name:        "Test Organizer",
		Enabled:     true,
		MovieDir:    movieLib,
		TVDir:       tvLib,
		LinkMode:    "copy",
		MinFileSize: 100, // Small threshold for test
		GenerateNFO: false,
	}

	org := NewOrganizerFromInstance(orgCfg, nil)
	comp := NewComponent(orgCfg, org)

	assert.Equal(t, "organizer:test", comp.ID())
	assert.Equal(t, "Test Organizer", comp.Name())
	assert.NotEmpty(t, comp.Description())
	assert.Equal(t, org, comp.Organizer())

	ctx := context.Background()

	// 1. Direct parameters
	t.Run("DirectParams", func(t *testing.T) {
		mediaInfo := &parser.MediaInfo{Title: "Oppenheimer", Year: 2023, MediaType: parser.MediaTypeMovie}
		metadata := &scraper.MediaMetadata{Title: "Oppenheimer", Year: 2023, MediaType: parser.MediaTypeMovie}

		sCtx := pipeline.NewStepContext("exec-1", 1, "step-1", "Organize", "organizer:test",
			map[string]any{
				"source_path": srcFile,
				"media_info":  mediaInfo,
				"metadata":    metadata,
			}, nil, nil)

		out, err := comp.Execute(ctx, sCtx)
		require.NoError(t, err)
		assert.True(t, out["success"].(bool))
		assert.Equal(t, filepath.Join(movieLib, "Oppenheimer (2023)"), out["target_dir"])
	})

	// 2. Chained upstream step extraction
	t.Run("ChainedUpstreamSteps", func(t *testing.T) {
		srcFile2 := filepath.Join(sourceDir, "Interstellar.2014.mkv")
		_ = os.WriteFile(srcFile2, make([]byte, 1024), 0644)

		steps := map[string]map[string]any{
			"path_mapper": {
				"local_content_path": srcFile2,
			},
			"parser": {
				"media_info": &parser.MediaInfo{Title: "Interstellar", Year: 2014, MediaType: parser.MediaTypeMovie},
			},
			"scraper": {
				"metadata": &scraper.MediaMetadata{Title: "Interstellar", Year: 2014, MediaType: parser.MediaTypeMovie},
			},
		}

		sCtx := pipeline.NewStepContext("exec-2", 1, "step-2", "Organize", "organizer:test",
			nil, nil, steps)

		out, err := comp.Execute(ctx, sCtx)
		require.NoError(t, err)
		assert.True(t, out["success"].(bool))
		assert.Equal(t, filepath.Join(movieLib, "Interstellar (2014)"), out["target_dir"])
	})

	// 3. Torrent in Event payload
	t.Run("TorrentInEvent", func(t *testing.T) {
		srcFile3 := filepath.Join(sourceDir, "Inception.2010.mkv")
		_ = os.WriteFile(srcFile3, make([]byte, 1024), 0644)

		tor := downloader.Torrent{
			Name:        "Inception.2010.mkv",
			ContentPath: srcFile3,
		}

		event := map[string]any{
			"torrent": tor,
		}

		sCtx := pipeline.NewStepContext("exec-3", 1, "step-3", "Organize", "organizer:test",
			nil, event, nil)

		out, err := comp.Execute(ctx, sCtx)
		require.NoError(t, err)
		assert.True(t, out["success"].(bool))
	})

	// 4. Missing source path error
	t.Run("MissingSourcePath", func(t *testing.T) {
		sCtx := pipeline.NewStepContext("exec-4", 1, "step-4", "Organize", "organizer:test",
			nil, nil, nil)
		_, err := comp.Execute(ctx, sCtx)
		assert.Error(t, err)
	})
}

func TestOrganizerComponent_RegisterComponents(t *testing.T) {
	cfg := &config.ComponentConfig{
		Organizers: []config.OrganizerInstanceConfig{
			{ID: "organizer:movie", Name: "Movie Organizer", MovieDir: "/data/movies", Enabled: true},
			{ID: "organizer:tv", Name: "TV Organizer", TVDir: "/data/tv", Enabled: true},
			{ID: "organizer:disabled", Enabled: false},
		},
	}

	registry := pipeline.NewComponentRegistry()
	comps, err := RegisterComponents(cfg, registry)
	require.NoError(t, err)
	assert.Len(t, comps, 2)

	org1, ok1 := registry.Get("organizer:movie")
	assert.True(t, ok1)
	assert.Equal(t, "Movie Organizer", org1.Name())

	alias, okAlias := registry.Get("organizer")
	assert.True(t, okAlias)
	assert.Equal(t, "organizer", alias.ID())
}
