package tmdb

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestClient_SearchAndDetails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/search/movie":
			_ = json.NewEncoder(w).Encode(SearchMovieResponse{
				Page: 1,
				Results: []MovieResult{
					{
						ID:          100,
						Title:       "Inception",
						ReleaseDate: "2010-07-16",
					},
				},
			})
		case "/search/tv":
			_ = json.NewEncoder(w).Encode(SearchTVResponse{
				Page: 1,
				Results: []TVResult{
					{
						ID:           200,
						Name:         "Breaking Bad",
						FirstAirDate: "2008-01-20",
					},
				},
			})
		case "/movie/100":
			_ = json.NewEncoder(w).Encode(MovieDetails{
				ID:          100,
				Title:       "Inception",
				ReleaseDate: "2010-07-16",
				IMDbID:      "tt1375666",
				Genres:      []Genre{{Name: "Action"}},
			})
		case "/tv/200":
			_ = json.NewEncoder(w).Encode(TVDetails{
				ID:           200,
				Name:         "Breaking Bad",
				FirstAirDate: "2008-01-20",
				Seasons:      []SeasonSummary{{SeasonNumber: 1, EpisodeCount: 7}},
			})
		case "/tv/200/season/1":
			_ = json.NewEncoder(w).Encode(SeasonDetails{
				SeasonNumber: 1,
				Episodes: []EpisodeItem{
					{EpisodeNumber: 1, Name: "Pilot"},
				},
			})
		case "/poster.jpg":
			w.Header().Set("Content-Type", "image/jpeg")
			_, _ = w.Write([]byte("fake-image-data"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewClient(Config{
		APIKey:  "test-api-key",
		BaseURL: server.URL,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	ctx := context.Background()

	// 1. Test SearchMovie
	movieRes, err := client.SearchMovie(ctx, "Inception", 2010, "")
	if err != nil {
		t.Fatalf("SearchMovie failed: %v", err)
	}
	if len(movieRes.Results) != 1 || movieRes.Results[0].ID != 100 {
		t.Errorf("unexpected SearchMovie result: %+v", movieRes)
	}

	// 2. Test SearchTV
	tvRes, err := client.SearchTV(ctx, "Breaking Bad", 2008, "")
	if err != nil {
		t.Fatalf("SearchTV failed: %v", err)
	}
	if len(tvRes.Results) != 1 || tvRes.Results[0].ID != 200 {
		t.Errorf("unexpected SearchTV result: %+v", tvRes)
	}

	// 3. Test GetMovieDetails
	movieDetails, err := client.GetMovieDetails(ctx, 100, "")
	if err != nil {
		t.Fatalf("GetMovieDetails failed: %v", err)
	}
	if movieDetails.IMDbID != "tt1375666" {
		t.Errorf("unexpected IMDbID: %s", movieDetails.IMDbID)
	}

	// 4. Test GetTVDetails
	tvDetails, err := client.GetTVDetails(ctx, 200, "")
	if err != nil {
		t.Fatalf("GetTVDetails failed: %v", err)
	}
	if len(tvDetails.Seasons) != 1 || tvDetails.Seasons[0].SeasonNumber != 1 {
		t.Errorf("unexpected TVDetails: %+v", tvDetails)
	}

	// 5. Test GetSeasonDetails
	seasonDetails, err := client.GetSeasonDetails(ctx, 200, 1, "")
	if err != nil {
		t.Fatalf("GetSeasonDetails failed: %v", err)
	}
	if len(seasonDetails.Episodes) != 1 || seasonDetails.Episodes[0].Name != "Pilot" {
		t.Errorf("unexpected SeasonDetails: %+v", seasonDetails)
	}

	// 6. Test DownloadImage
	tmpDir := t.TempDir()
	imgFile := filepath.Join(tmpDir, "test.jpg")
	if err := client.DownloadImage(ctx, server.URL+"/poster.jpg", imgFile); err != nil {
		t.Fatalf("DownloadImage failed: %v", err)
	}
	content, err := os.ReadFile(imgFile)
	if err != nil || string(content) != "fake-image-data" {
		t.Errorf("unexpected downloaded image content: %s, err: %v", string(content), err)
	}
}
