package tmdb

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackie8tao/mediahelper/internal/parser"
)

func TestTMDBMatcher_MatchMovie(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasPrefix(path, "/search/movie") {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"page": 1,
				"results": [
					{
						"id": 27205,
						"title": "盗梦空间",
						"original_title": "Inception",
						"release_date": "2010-07-15"
					}
				]
			}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	matcher, err := NewMatcher(Config{
		APIKey:  "fake_key",
		BaseURL: server.URL,
	})
	if err != nil {
		t.Fatalf("unexpected new matcher error: %v", err)
	}

	candidate := &parser.MediaInfo{
		Title:     "盗梦空间",
		EnTitle:   "Inception",
		Year:      2010,
		MediaType: parser.MediaTypeMovie,
	}

	res, err := matcher.Match(context.Background(), candidate)
	if err != nil {
		t.Fatalf("match error: %v", err)
	}

	if !res.Matched {
		t.Fatalf("expected matched true, got false")
	}
	if res.TMDBID != 27205 {
		t.Errorf("expected TMDBID 27205, got %d", res.TMDBID)
	}
	if res.Title != "盗梦空间" {
		t.Errorf("expected title 盗梦空间, got %s", res.Title)
	}
	if res.Year != 2010 {
		t.Errorf("expected year 2010, got %d", res.Year)
	}
	if res.Confidence < 0.80 {
		t.Errorf("expected confidence >= 0.80, got %f", res.Confidence)
	}
}

func TestTMDBMatcher_MatchTVFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		query := r.URL.Query().Get("query")
		year := r.URL.Query().Get("first_air_date_year")

		if strings.HasPrefix(path, "/search/tv") {
			w.Header().Set("Content-Type", "application/json")
			// Simulate that searching with year 2024 returns empty, but fallback without year returns result
			if year == "2024" {
				_, _ = w.Write([]byte(`{"results": []}`))
				return
			}
			if strings.Contains(query, "Frieren") || strings.Contains(query, "芙莉莲") {
				_, _ = w.Write([]byte(`{
					"results": [
						{
							"id": 209867,
							"name": "葬送的芙莉莲",
							"original_name": "葬送のフリーレン",
							"first_air_date": "2023-09-29"
						}
					]
				}`))
				return
			}
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	matcher, err := NewMatcher(Config{
		APIKey:  "fake_key",
		BaseURL: server.URL,
	})
	if err != nil {
		t.Fatalf("unexpected new matcher error: %v", err)
	}

	// LLM extracted release group year 2024 instead of official broadcast year 2023
	candidate := &parser.MediaInfo{
		Title:     "Sousou no Frieren",
		EnTitle:   "Frieren",
		Year:      2024,
		MediaType: parser.MediaTypeAnime,
	}

	res, err := matcher.Match(context.Background(), candidate)
	if err != nil {
		t.Fatalf("match error: %v", err)
	}

	if !res.Matched {
		t.Fatalf("expected matched true, got false")
	}
	if res.TMDBID != 209867 {
		t.Errorf("expected TMDBID 209867, got %d", res.TMDBID)
	}
	if res.Title != "葬送的芙莉莲" {
		t.Errorf("expected title 葬送的芙莉莲, got %s", res.Title)
	}
	// Year should be calibrated to 2023
	if res.Year != 2023 {
		t.Errorf("expected calibrated year 2023, got %d", res.Year)
	}
	// MediaType should be calibrated to TV
	if res.MediaType != parser.MediaTypeTV {
		t.Errorf("expected MediaType TV, got %s", res.MediaType)
	}
}

func TestTMDBMatcher_Unmatched(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results": []}`))
	}))
	defer server.Close()

	matcher, _ := NewMatcher(Config{
		APIKey:  "fake_key",
		BaseURL: server.URL,
	})

	candidate := &parser.MediaInfo{
		Title: "NonExistentMovie999999",
	}

	res, err := matcher.Match(context.Background(), candidate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Matched {
		t.Errorf("expected matched false, got true")
	}
}
