package tmdb

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

func TestTMDBScrapeMovie(t *testing.T) {
	// Mock TMDB API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		if strings.HasPrefix(path, "/search/movie") {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"page": 1,
				"results": [
					{
						"id": 550,
						"title": "Fight Club",
						"original_title": "Fight Club",
						"release_date": "1999-10-15",
						"overview": "A ticking-time-bomb insomniac...",
						"vote_average": 8.4,
						"vote_count": 26000,
						"poster_path": "/poster.jpg",
						"backdrop_path": "/backdrop.jpg"
					}
				]
			}`))
			return
		}

		if strings.HasPrefix(path, "/movie/550") {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"id": 550,
				"title": "搏击俱乐部",
				"original_title": "Fight Club",
				"release_date": "1999-10-15",
				"overview": "一个患有严重失眠症的白领...",
				"vote_average": 8.4,
				"vote_count": 26000,
				"poster_path": "/poster.jpg",
				"backdrop_path": "/backdrop.jpg",
				"imdb_id": "tt0137523",
				"genres": [{"name": "剧情"}],
				"production_companies": [{"name": "Fox 2000 Pictures"}],
				"credits": {
					"cast": [
						{"name": "Edward Norton", "character": "Narrator", "profile_path": "/norton.jpg"},
						{"name": "Brad Pitt", "character": "Tyler Durden", "profile_path": "/pitt.jpg"}
					],
					"crew": [
						{"name": "David Fincher", "job": "Director"}
					]
				}
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
	movieDir := filepath.Join(tempDir, "Fight Club (1999)")
	if err := os.MkdirAll(movieDir, 0755); err != nil {
		t.Fatal(err)
	}

	client := &Client{
		cfg: Config{
			APIKey:       "fake_key",
			BaseURL:      server.URL,
			ImageBaseURL: server.URL,
			Language:     "zh-CN",
		},
		httpClient: server.Client(),
	}

	task := scraper.ScrapeTask{
		TargetDir: movieDir,
		MediaType: "movie",
	}

	res, err := client.Scrape(context.Background(), task)
	if err != nil {
		t.Fatalf("Scrape error: %v", err)
	}

	if !res.Success {
		t.Fatalf("expected scrape success, got false")
	}

	if res.Metadata.Title != "搏击俱乐部" {
		t.Errorf("expected title 搏击俱乐部, got %s", res.Metadata.Title)
	}
	if res.Metadata.TMDBID != 550 {
		t.Errorf("expected TMDBID 550, got %d", res.Metadata.TMDBID)
	}
	if res.Metadata.Year != 1999 {
		t.Errorf("expected year 1999, got %d", res.Metadata.Year)
	}

	// Verify NFO file
	nfoPath := filepath.Join(movieDir, "movie.nfo")
	nfoBytes, err := os.ReadFile(nfoPath)
	if err != nil {
		t.Fatalf("expected movie.nfo to exist: %v", err)
	}
	if !strings.Contains(string(nfoBytes), "<title>搏击俱乐部</title>") {
		t.Errorf("movie.nfo does not contain expected title tag: %s", string(nfoBytes))
	}
	if !strings.Contains(string(nfoBytes), "David Fincher") {
		t.Errorf("movie.nfo does not contain director: %s", string(nfoBytes))
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

func TestTMDBScrapeTV(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		if strings.HasPrefix(path, "/search/tv") {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"page": 1,
				"results": [
					{
						"id": 1396,
						"name": "Breaking Bad",
						"original_name": "Breaking Bad",
						"first_air_date": "2008-01-20",
						"overview": "A high school chemistry teacher...",
						"vote_average": 8.9,
						"vote_count": 13000
					}
				]
			}`))
			return
		}

		if strings.HasPrefix(path, "/tv/1396/season/1") {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"season_number": 1,
				"episodes": [
					{
						"episode_number": 1,
						"name": "Pilot",
						"overview": "Walter White is diagnosed with cancer...",
						"vote_average": 8.2,
						"air_date": "2008-01-20"
					}
				]
			}`))
			return
		}

		if strings.HasPrefix(path, "/tv/1396") {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"id": 1396,
				"name": "绝命毒师",
				"original_name": "Breaking Bad",
				"first_air_date": "2008-01-20",
				"overview": "高中化学老师怀特得知自己患上绝症...",
				"vote_average": 8.9,
				"vote_count": 13000,
				"genres": [{"name": "剧情"}],
				"production_companies": [{"name": "Sony Pictures Television"}]
			}`))
			return
		}

		http.NotFound(w, r)
	}))
	defer server.Close()

	tempDir := t.TempDir()
	tvDir := filepath.Join(tempDir, "Breaking Bad (2008)")
	seasonDir := filepath.Join(tvDir, "Season 01")
	if err := os.MkdirAll(seasonDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a dummy episode file
	epFile := filepath.Join(seasonDir, "Breaking Bad - S01E01 - Pilot.mkv")
	if err := os.WriteFile(epFile, []byte("fake video"), 0644); err != nil {
		t.Fatal(err)
	}

	noImages := true
	client := &Client{
		cfg: Config{
			APIKey:   "fake_key",
			BaseURL:  server.URL,
			Language: "zh-CN",
		},
		httpClient: server.Client(),
	}

	task := scraper.ScrapeTask{
		TargetDir:  tvDir,
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

	if res.Metadata.Title != "绝命毒师" {
		t.Errorf("expected title 绝命毒师, got %s", res.Metadata.Title)
	}

	// Verify tvshow.nfo
	tvNFO := filepath.Join(tvDir, "tvshow.nfo")
	if _, err := os.Stat(tvNFO); err != nil {
		t.Errorf("expected tvshow.nfo to exist: %v", err)
	}

	// Verify episode nfo
	epNFO := filepath.Join(seasonDir, "Breaking Bad - S01E01 - Pilot.nfo")
	epBytes, err := os.ReadFile(epNFO)
	if err != nil {
		t.Fatalf("expected episode NFO to exist: %v", err)
	}
	if !strings.Contains(string(epBytes), "<title>Pilot</title>") {
		t.Errorf("episode NFO does not contain expected title: %s", string(epBytes))
	}
}

func TestTMDBScrapeValidationErrors(t *testing.T) {
	client := &Client{
		cfg: Config{APIKey: "fake_key"},
	}

	// 1. Missing target_dir
	_, err := client.Scrape(context.Background(), scraper.ScrapeTask{})
	if err == nil {
		t.Errorf("expected error on empty target_dir, got nil")
	}

	// 2. Non-existent target_dir
	_, err = client.Scrape(context.Background(), scraper.ScrapeTask{
		TargetDir: "/path/that/does/not/exist/at/all",
	})
	if err == nil {
		t.Errorf("expected error on non-existent directory, got nil")
	}
}
