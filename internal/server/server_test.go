package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackie8tao/mediahelper/internal/config"
	"github.com/jackie8tao/mediahelper/internal/downloader"
	"github.com/jackie8tao/mediahelper/internal/service"
)

type mockDownloader struct{}

func (m *mockDownloader) Name() string                                 { return "mock" }
func (m *mockDownloader) TestConnection(ctx context.Context) error      { return nil }
func (m *mockDownloader) GetAllTorrents(ctx context.Context, filter, category string) ([]downloader.Torrent, error) {
	return []downloader.Torrent{
		{Hash: "h1", Name: "Movie.1.mkv", State: "downloading", Progress: 0.5},
	}, nil
}
func (m *mockDownloader) GetCompletedTorrents(ctx context.Context, categories []string) ([]downloader.Torrent, error) {
	return nil, nil
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
	cfg := config.DefaultConfig()
	pipeline := service.NewPipeline(cfg, &mockDownloader{}, nil, nil, nil)
	srv := New(cfg, pipeline)

	// 1. 测试 /health
	t.Run("Health Check", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})

	// 1.5 测试 Swagger UI 接口
	t.Run("Swagger UI", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/swagger/index.html", nil)
		req.RequestURI = "/swagger/index.html"
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200 for swagger ui, got %d", w.Code)
		}
	})

	// 2. 测试 /api/v1/status
	t.Run("Status", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/status", nil)
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})

	// 3. 测试 /api/v1/history
	t.Run("History", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/history", nil)
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})

	// 4. 测试 /api/v1/plugins
	t.Run("Plugins", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/plugins", nil)
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})

	// 5. 测试 /api/v1/downloader/torrents
	t.Run("Downloader Torrents", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/downloader/torrents?filter=all", nil)
		w := httptest.NewRecorder()
		srv.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}

		var res struct {
			Code  int                  `json:"code"`
			Total int                  `json:"total"`
			Data  []downloader.Torrent `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}
		if res.Total != 1 || res.Data[0].Hash != "h1" {
			t.Errorf("unexpected torrent data: %+v", res)
		}
	})

	// 6. 测试 /api/v1/webhook/qbittorrent
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

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})
}
