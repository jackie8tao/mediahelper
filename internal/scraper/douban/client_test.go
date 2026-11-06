package douban

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackie8tao/mediahelper/internal/scraper"
)

func TestDoubanScraper_ScrapeMovie(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		if strings.HasPrefix(path, "/movie/1292052/photos") {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"total": 1,
				"photos": [
					{
						"id": "1",
						"image": {
							"large": {"url": "` + server.URL + `/backdrop.jpg"}
						}
					}
				]
			}`))
			return
		}

		if strings.HasPrefix(path, "/movie/1292052") {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"id": "1292052",
				"title": "肖申克的救赎",
				"original_title": "The Shawshank Redemption",
				"year": "1994",
				"intro": "一场谋杀案使银行家安迪入狱...",
				"rating": {"value": 9.7, "count": 2900000},
				"cover_url": "` + server.URL + `/poster.jpg",
				"genres": ["犯罪", "剧情"],
				"directors": [{"name": "弗兰克·德拉邦特"}],
				"actors": [
					{"name": "蒂姆·罗宾斯", "character": "安迪"},
					{"name": "摩根·弗里曼", "character": "瑞德"}
				]
			}`))
			return
		}

		if strings.HasSuffix(path, ".jpg") {
			w.Header().Set("Content-Type", "image/jpeg")
			_, _ = w.Write([]byte("fake image data"))
			return
		}

		http.NotFound(w, r)
	}))
	defer server.Close()

	tempDir := t.TempDir()
	movieDir := filepath.Join(tempDir, "The Shawshank Redemption (1994)")
	if err := os.MkdirAll(movieDir, 0755); err != nil {
		t.Fatal(err)
	}

	client := &Client{
		cfg: Config{
			BaseURL: server.URL,
		},
		httpClient: server.Client(),
	}

	task := scraper.ScrapeTask{
		TargetDir: movieDir,
		DoubanID:  "1292052",
		MediaType: "movie",
	}

	res, err := client.Scrape(context.Background(), task)
	if err != nil {
		t.Fatalf("Scrape error: %v", err)
	}

	if !res.Success {
		t.Fatalf("expected scrape success, got false")
	}

	if res.Metadata.Title != "肖申克的救赎" {
		t.Errorf("expected title 肖申克的救赎, got %s", res.Metadata.Title)
	}
	if res.Metadata.DoubanID != "1292052" {
		t.Errorf("expected DoubanID 1292052, got %s", res.Metadata.DoubanID)
	}
	if res.Metadata.Year != 1994 {
		t.Errorf("expected year 1994, got %d", res.Metadata.Year)
	}

	// Verify NFO file
	nfoPath := filepath.Join(movieDir, "movie.nfo")
	nfoBytes, err := os.ReadFile(nfoPath)
	if err != nil {
		t.Fatalf("expected movie.nfo to exist: %v", err)
	}
	nfoContent := string(nfoBytes)
	if !strings.Contains(nfoContent, "<title>肖申克的救赎</title>") {
		t.Errorf("movie.nfo missing title: %s", nfoContent)
	}
	if !strings.Contains(nfoContent, `<uniqueid type="douban" default="true">1292052</uniqueid>`) {
		t.Errorf("movie.nfo missing douban uniqueid: %s", nfoContent)
	}
	if !strings.Contains(nfoContent, "弗兰克·德拉邦特") {
		t.Errorf("movie.nfo missing director: %s", nfoContent)
	}

	// Verify poster and fanart
	posterPath := filepath.Join(movieDir, "poster.jpg")
	if _, err := os.Stat(posterPath); err != nil {
		t.Errorf("expected poster.jpg to exist: %v", err)
	}
	fanartPath := filepath.Join(movieDir, "fanart.jpg")
	if _, err := os.Stat(fanartPath); err != nil {
		t.Errorf("expected fanart.jpg to exist: %v", err)
	}
}

func TestDoubanScraper_ScrapeTV(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		if strings.HasPrefix(path, "/tv/35465232/photos") {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"total": 0, "photos": []}`))
			return
		}

		if strings.HasPrefix(path, "/tv/35465232") {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"id": "35465232",
				"title": "狂飙",
				"original_title": "Punch Out",
				"year": "2023",
				"intro": "京海市一线刑警安欣...",
				"rating": {"value": 8.5, "count": 320000},
				"episodes_count": 39,
				"genres": ["剧情", "犯罪"]
			}`))
			return
		}

		http.NotFound(w, r)
	}))
	defer server.Close()

	tempDir := t.TempDir()
	tvDir := filepath.Join(tempDir, "The Knockout (2023)")
	if err := os.MkdirAll(tvDir, 0755); err != nil {
		t.Fatal(err)
	}

	client := &Client{
		cfg: Config{
			BaseURL: server.URL,
		},
		httpClient: server.Client(),
	}

	noImages := true
	task := scraper.ScrapeTask{
		TargetDir:  tvDir,
		DoubanID:   "35465232",
		MediaType:  "tv",
		SaveImages: &noImages,
	}

	res, err := client.Scrape(context.Background(), task)
	if err != nil {
		t.Fatalf("Scrape TV error: %v", err)
	}

	if !res.Success {
		t.Fatalf("expected scrape TV success, got false")
	}

	if res.Metadata.Title != "狂飙" {
		t.Errorf("expected title 狂飙, got %s", res.Metadata.Title)
	}

	tvNFO := filepath.Join(tvDir, "tvshow.nfo")
	if _, err := os.Stat(tvNFO); err != nil {
		t.Errorf("expected tvshow.nfo to exist: %v", err)
	}
}

func TestDoubanScraper_ValidationErrors(t *testing.T) {
	tempDir := t.TempDir()
	client := &Client{
		cfg: Config{},
	}

	// 1. Missing target_dir
	_, err := client.Scrape(context.Background(), scraper.ScrapeTask{DoubanID: "123"})
	if err == nil {
		t.Errorf("expected error on empty target_dir, got nil")
	}

	// 2. Missing douban_id
	_, err = client.Scrape(context.Background(), scraper.ScrapeTask{TargetDir: tempDir})
	if err == nil {
		t.Errorf("expected error on missing douban_id, got nil")
	}
}
