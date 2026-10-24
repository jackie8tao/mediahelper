package pathmapper

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/jackie8tao/mediahelper/internal/config"
	"github.com/jackie8tao/mediahelper/internal/downloader"
	"github.com/jackie8tao/mediahelper/internal/pipeline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPathMapper_ToLocal(t *testing.T) {
	mappings := []config.PathMapping{
		{Remote: "/downloads", Local: "D:/LocalDownloads"},
		{Remote: "/data/media", Local: "/mnt/storage/media"},
	}

	mapper := NewPathMapper(mappings)

	// Unix to Windows local path
	res := mapper.ToLocal("/downloads/movies/Avatar (2009)/Avatar.mkv")
	assert.Equal(t, filepath.FromSlash("D:/LocalDownloads/movies/Avatar (2009)/Avatar.mkv"), res)

	// Prefix boundary check (should not match /downloads2)
	res2 := mapper.ToLocal("/downloads2/test.mkv")
	assert.Equal(t, filepath.Clean("/downloads2/test.mkv"), res2)

	// Empty mappings
	emptyMapper := NewPathMapper(nil)
	assert.Equal(t, filepath.Clean("/downloads/test.mkv"), emptyMapper.ToLocal("/downloads/test.mkv"))
}

func TestPathMapper_ToRemote(t *testing.T) {
	mappings := []config.PathMapping{
		{Remote: "/downloads", Local: "D:/LocalDownloads"},
	}

	mapper := NewPathMapper(mappings)

	res := mapper.ToRemote("D:/LocalDownloads/movies/file.mkv")
	assert.Equal(t, "/downloads/movies/file.mkv", res)
}

func TestPathMapper_MapTorrent(t *testing.T) {
	mappings := []config.PathMapping{
		{Remote: "/downloads", Local: "/local/downloads"},
	}
	mapper := NewPathMapper(mappings)

	tor := downloader.Torrent{
		Hash:        "hash123",
		Name:        "Test Movie",
		SavePath:    "/downloads/movies",
		ContentPath: "/downloads/movies/Test Movie.mkv",
	}

	mapped := mapper.MapTorrent(tor)
	assert.Equal(t, filepath.FromSlash("/local/downloads/movies"), mapped.SavePath)
	assert.Equal(t, filepath.FromSlash("/local/downloads/movies/Test Movie.mkv"), mapped.ContentPath)
}

func TestComponent_Execute(t *testing.T) {
	mappings := []config.PathMapping{
		{Remote: "/downloads", Local: "/local/downloads"},
	}
	mapper := NewPathMapper(mappings)
	comp := NewComponent("path_mapper", "Path Mapper", mapper)

	assert.Equal(t, "path_mapper", comp.ID())
	assert.Equal(t, "Path Mapper", comp.Name())
	assert.NotEmpty(t, comp.Description())

	ctx := context.Background()

	// 1. With path parameter
	sCtx1 := pipeline.NewStepContext("exec-1", 1, "step-1", "Map Path", "path_mapper",
		map[string]any{"path": "/downloads/tv/Show.S01E01.mkv"}, nil, nil)
	out1, err := comp.Execute(ctx, sCtx1)
	require.NoError(t, err)
	assert.Equal(t, filepath.FromSlash("/local/downloads/tv/Show.S01E01.mkv"), out1["local_path"])

	// 2. With torrent parameter
	tor := downloader.Torrent{
		Hash:        "abc",
		Name:        "Show",
		SavePath:    "/downloads/tv",
		ContentPath: "/downloads/tv/Show",
	}
	sCtx2 := pipeline.NewStepContext("exec-2", 1, "step-2", "Map Torrent", "path_mapper",
		map[string]any{"torrent": tor}, nil, nil)
	out2, err := comp.Execute(ctx, sCtx2)
	require.NoError(t, err)
	mappedTor, ok := out2["torrent"].(downloader.Torrent)
	require.True(t, ok)
	assert.Equal(t, filepath.FromSlash("/local/downloads/tv"), mappedTor.SavePath)
	assert.Equal(t, filepath.FromSlash("/local/downloads/tv/Show"), mappedTor.ContentPath)

	// 3. Fallback from Event payload
	eventPayload := map[string]any{
		"save_path":    "/downloads/movies",
		"content_path": "/downloads/movies/Movie.mkv",
	}
	sCtx3 := pipeline.NewStepContext("exec-3", 1, "step-3", "Map Event", "path_mapper",
		nil, eventPayload, nil)
	out3, err := comp.Execute(ctx, sCtx3)
	require.NoError(t, err)
	assert.Equal(t, filepath.FromSlash("/local/downloads/movies"), out3["local_save_path"])
	assert.Equal(t, filepath.FromSlash("/local/downloads/movies/Movie.mkv"), out3["local_content_path"])
}

func TestRegisterComponent(t *testing.T) {
	cfg := &config.Config{
		PathMappings: []config.PathMapping{
			{Remote: "/downloads", Local: "/local"},
		},
	}
	registry := pipeline.NewComponentRegistry()

	comp, err := RegisterComponent(cfg, registry)
	require.NoError(t, err)
	assert.NotNil(t, comp)

	registeredComp, ok := registry.Get("path_mapper")
	require.True(t, ok)
	assert.Equal(t, comp, registeredComp)
}
