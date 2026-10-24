package parser

import (
	"context"
	"testing"

	"github.com/jackie8tao/mediahelper/internal/config"
	"github.com/jackie8tao/mediahelper/internal/downloader"
	"github.com/jackie8tao/mediahelper/internal/pipeline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParserComponent_Execute(t *testing.T) {
	comp := NewComponent(config.ParserInstanceConfig{
		ID:      "parser:test",
		Name:    "Test Parser",
		Enabled: true,
	})

	assert.Equal(t, "parser:test", comp.ID())
	assert.Equal(t, "Test Parser", comp.Name())
	assert.NotEmpty(t, comp.Description())

	ctx := context.Background()

	// 1. Direct name parameter
	t.Run("DirectTitleParam", func(t *testing.T) {
		sCtx := pipeline.NewStepContext("exec-1", 1, "step-1", "Parse Title", "parser:test",
			map[string]any{"raw": "Oppenheimer.2023.2160p.UHD.BluRay.x265-FLUX"}, nil, nil)
		out, err := comp.Execute(ctx, sCtx)
		require.NoError(t, err)
		assert.Equal(t, "Oppenheimer", out["title"])
		assert.Equal(t, 2023, out["year"])
		assert.Equal(t, "movie", out["media_type"])
		assert.Equal(t, "4K", out["resolution"])
		assert.Equal(t, "FLUX", out["release_group"])
	})

	// 2. Torrent in parameters
	t.Run("TorrentInParams", func(t *testing.T) {
		tor := downloader.Torrent{
			Name: "Loki.S02E01.1080p.WEB-DL.DDP5.1.Atmos.H.264",
		}
		sCtx := pipeline.NewStepContext("exec-2", 1, "step-2", "Parse Torrent", "parser:test",
			map[string]any{"torrent": tor}, nil, nil)
		out, err := comp.Execute(ctx, sCtx)
		require.NoError(t, err)
		assert.Equal(t, "Loki", out["title"])
		assert.Equal(t, 2, out["season"])
		assert.Equal(t, 1, out["episode"])
		assert.Equal(t, "tv", out["media_type"])
	})

	// 3. File path in Event
	t.Run("FilePathInEvent", func(t *testing.T) {
		event := map[string]any{
			"file_path": "/downloads/movies/Dune.Part.Two.2024.1080p.mkv",
		}
		sCtx := pipeline.NewStepContext("exec-3", 1, "step-3", "Parse Event", "parser:test",
			nil, event, nil)
		out, err := comp.Execute(ctx, sCtx)
		require.NoError(t, err)
		assert.Equal(t, "Dune Part Two", out["title"])
		assert.Equal(t, 2024, out["year"])
	})

	// 4. Upstream step output
	t.Run("UpstreamStepOutput", func(t *testing.T) {
		steps := map[string]map[string]any{
			"path_mapper": {
				"local_content_path": "D:/Downloads/Breaking.Bad.S05E14.720p.mkv",
			},
		}
		sCtx := pipeline.NewStepContext("exec-4", 1, "step-4", "Parse Step", "parser:test",
			nil, nil, steps)
		out, err := comp.Execute(ctx, sCtx)
		require.NoError(t, err)
		assert.Equal(t, "Breaking Bad", out["title"])
		assert.Equal(t, 5, out["season"])
		assert.Equal(t, 14, out["episode"])
	})

	// 5. Category hint
	t.Run("CategoryHint", func(t *testing.T) {
		sCtx := pipeline.NewStepContext("exec-5", 1, "step-5", "Parse Hint", "parser:test",
			map[string]any{"raw": "Sousou no Frieren", "category": "anime"}, nil, nil)
		out, err := comp.Execute(ctx, sCtx)
		require.NoError(t, err)
		assert.Equal(t, "anime", out["media_type"])
	})

	// 6. Empty input error
	t.Run("EmptyInput", func(t *testing.T) {
		sCtx := pipeline.NewStepContext("exec-6", 1, "step-6", "Parse Empty", "parser:test",
			nil, nil, nil)
		_, err := comp.Execute(ctx, sCtx)
		assert.Error(t, err)
	})
}

func TestParserComponent_RegisterComponents(t *testing.T) {
	cfg := &config.ComponentConfig{
		Parsers: []config.ParserInstanceConfig{
			{ID: "parser:movie", Name: "Movie Parser", Enabled: true, DefaultMediaType: "movie"},
			{ID: "parser:anime", Name: "Anime Parser", Enabled: true, DefaultMediaType: "anime"},
			{ID: "parser:disabled", Name: "Disabled", Enabled: false},
		},
	}

	registry := pipeline.NewComponentRegistry()
	comps, err := RegisterComponents(cfg, registry)
	require.NoError(t, err)
	assert.Len(t, comps, 2)

	// Check registry contents
	p1, ok1 := registry.Get("parser:movie")
	assert.True(t, ok1)
	assert.Equal(t, "Movie Parser", p1.Name())

	p2, ok2 := registry.Get("parser:anime")
	assert.True(t, ok2)
	assert.Equal(t, "Anime Parser", p2.Name())

	// Default alias check
	alias, okAlias := registry.Get("parser")
	assert.True(t, okAlias)
	assert.Equal(t, "parser", alias.ID())
}
