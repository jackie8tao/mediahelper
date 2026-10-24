package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackie8tao/mediahelper/internal/config"
	"github.com/jackie8tao/mediahelper/internal/database"
	"github.com/jackie8tao/mediahelper/internal/downloader"
	"github.com/jackie8tao/mediahelper/internal/pipeline"
	"github.com/jackie8tao/mediahelper/internal/plugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockDownloader struct{}

func (m *mockDownloader) Name() string                             { return "mock" }
func (m *mockDownloader) TestConnection(ctx context.Context) error { return nil }
func (m *mockDownloader) GetAllTorrents(ctx context.Context, filter, category string) ([]downloader.Torrent, error) {
	return []downloader.Torrent{
		{Hash: "h1", Name: "Movie.1.mkv", State: "downloading", Progress: 0.5},
	}, nil
}
func (m *mockDownloader) GetCompletedTorrents(ctx context.Context, categories []string) ([]downloader.Torrent, error) {
	return []downloader.Torrent{
		{Hash: "h1", Name: "Movie.1.mkv", State: "uploading", Progress: 1.0, IsCompleted: true},
	}, nil
}
func (m *mockDownloader) GetTorrentFiles(ctx context.Context, hash string) ([]downloader.TorrentFile, error) {
	return nil, nil
}
func (m *mockDownloader) AddTags(ctx context.Context, hashes []string, tags []string) error {
	return nil
}
func (m *mockDownloader) RemoveTags(ctx context.Context, hashes []string, tags []string) error {
	return nil
}

func TestServerRoutes(t *testing.T) {
	cfg := config.DefaultSystemConfig()
	db, err := database.InitDB(":memory:")
	require.NoError(t, err)

	registry := pipeline.NewComponentRegistry()

	// Register a mock downloader component
	dlCfg := config.DownloaderInstanceConfig{
		ID:      "downloader",
		Name:    "Default Downloader",
		Enabled: true,
	}
	dlComp := downloader.NewDownloaderComponent("downloader", "Default Downloader", &mockDownloader{}, dlCfg)
	err = registry.Register(dlComp)
	require.NoError(t, err)

	engine := pipeline.NewEngine(db, registry)
	err = pipeline.SeedDefaultPipelines(context.Background(), engine.Repo())
	require.NoError(t, err)

	pluginMgr := plugin.NewManager()
	srv := New(cfg, engine, registry, pluginMgr)

	// 1. Health check
	t.Run("Health Check", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	// 2. Swagger UI
	t.Run("Swagger UI", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/swagger/index.html", nil)
		req.RequestURI = "/swagger/index.html"
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	// 3. Status
	t.Run("Status", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/status", nil)
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	// 4. History
	t.Run("History", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/history", nil)
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	// 5. Plugins
	t.Run("Plugins", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/plugins", nil)
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	// 6. Downloader Torrents
	t.Run("Downloader Torrents", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/downloader/torrents?filter=all", nil)
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var res struct {
			Code  int                  `json:"code"`
			Total int                  `json:"total"`
			Data  []downloader.Torrent `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &res)
		require.NoError(t, err)
		assert.Equal(t, 1, res.Total)
		assert.Equal(t, "h1", res.Data[0].Hash)
	})

	// 7. Manual Sync
	t.Run("Manual Sync", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/sync", nil)
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	// 8. Webhook qBittorrent
	t.Run("QB Webhook", func(t *testing.T) {
		payload := QBWebhookPayload{
			Hash:        "hash_test_123",
			Name:        "Test.Movie.2023.mkv",
			Category:    "movie",
			ContentPath: "/downloads/Test.Movie.2023.mkv",
		}
		data, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/webhook/qbittorrent", bytes.NewReader(data))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	// 9. Pipelines list
	t.Run("Pipelines List", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/pipelines", nil)
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var res struct {
			Code  int                 `json:"code"`
			Total int                 `json:"total"`
			Data  []database.Pipeline `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &res)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, res.Total, 1)
	})
}
