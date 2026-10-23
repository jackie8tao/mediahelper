package scraper

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackie8tao/mediahelper/internal/config"
	"github.com/jackie8tao/mediahelper/internal/parser"
)

type mockTMDBClient struct{}

func (m *mockTMDBClient) SearchAndGetMovie(ctx context.Context, title string, year int) (*MediaMetadata, error) {
	if title == "奥本海默" || title == "Oppenheimer" {
		return &MediaMetadata{
			TMDBID:    872585,
			MediaType: parser.MediaTypeMovie,
			Title:     "奥本海默",
			Year:      2023,
		}, nil
	}
	return nil, fmt.Errorf("movie not found")
}

func (m *mockTMDBClient) SearchAndGetTV(ctx context.Context, title string, year int, season int, episode int) (*MediaMetadata, error) {
	if title == "繁花" || title == "Blossoms Shanghai" {
		return &MediaMetadata{
			TMDBID:       12345,
			MediaType:    parser.MediaTypeTV,
			Title:        "繁花",
			Year:         2023,
			Season:       season,
			Episode:      episode,
			EpisodeTitle: "第一集",
		}, nil
	}
	return nil, fmt.Errorf("tv show not found")
}

func TestService_Scrape(t *testing.T) {
	cfg := config.TMDBConfig{
		GenerateNFO:    true,
		DownloadImages: true,
	}

	service := NewService(cfg, &mockTMDBClient{})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// 1. 测试电影刮削 (支持中文检索)
	t.Run("Scrape Movie", func(t *testing.T) {
		info := &parser.MediaInfo{
			MediaType: parser.MediaTypeMovie,
			Title:     "奥本海默",
			CnTitle:   "奥本海默",
			EnTitle:   "Oppenheimer",
			Year:      2023,
		}

		meta, err := service.Scrape(ctx, info)
		if err != nil {
			t.Fatalf("Scrape movie error: %v", err)
		}
		if meta.TMDBID != 872585 {
			t.Errorf("expected TMDBID 872585, got %d", meta.TMDBID)
		}
	})

	// 2. 测试剧集刮削
	t.Run("Scrape TV Show", func(t *testing.T) {
		info := &parser.MediaInfo{
			MediaType: parser.MediaTypeTV,
			Title:     "繁花",
			CnTitle:   "繁花",
			Season:    1,
			Episode:   1,
		}

		meta, err := service.Scrape(ctx, info)
		if err != nil {
			t.Fatalf("Scrape TV error: %v", err)
		}
		if meta.TMDBID != 12345 {
			t.Errorf("expected TMDBID 12345, got %d", meta.TMDBID)
		}
	})
}

func TestService_DownloadImage(t *testing.T) {
	// 启动一个简易静态图片文件服务
	imgServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write([]byte("fake_image_bytes"))
	}))
	defer imgServer.Close()

	service := NewService(config.TMDBConfig{}, &mockTMDBClient{})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	tempDir, err := os.MkdirTemp("", "mediahelper_test_*")
	if err != nil {
		t.Fatalf("MkdirTemp failed: %v", err)
	}
	defer os.RemoveAll(tempDir)

	targetFile := filepath.Join(tempDir, "poster.jpg")
	err = service.DownloadImage(ctx, imgServer.URL+"/poster.jpg", targetFile)
	if err != nil {
		t.Fatalf("DownloadImage failed: %v", err)
	}

	content, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	if string(content) != "fake_image_bytes" {
		t.Errorf("unexpected image content: %s", string(content))
	}
}
