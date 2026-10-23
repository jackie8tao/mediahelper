package tmdb

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackie8tao/mediahelper/internal/config"
	"github.com/jackie8tao/mediahelper/internal/parser"
)

func setupMockTMDBServer(t *testing.T) *httptest.Server {
	mux := http.NewServeMux()

	// 1. 搜索电影接口
	mux.HandleFunc("/search/movie", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")
		w.Header().Set("Content-Type", "application/json")

		if query == "奥本海默" || query == "Oppenheimer" {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"results": []map[string]interface{}{
					{
						"id":           872585,
						"title":        "奥本海默",
						"release_date": "2023-07-19",
					},
				},
			})
			return
		}

		_ = json.NewEncoder(w).Encode(map[string]interface{}{"results": []interface{}{}})
	})

	// 2. 电影详情接口
	mux.HandleFunc("/movie/872585", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":             872585,
			"imdb_id":        "tt15398776",
			"title":          "奥本海默",
			"original_title": "Oppenheimer",
			"overview":       "电影讲述了罗伯特·奥本海默主导曼哈顿计划的过程...",
			"release_date":   "2023-07-19",
			"vote_average":   8.1,
			"vote_count":     8900,
			"poster_path":    "/poster.jpg",
			"backdrop_path":  "/backdrop.jpg",
			"genres": []map[string]interface{}{
				{"id": 18, "name": "剧情"},
				{"id": 36, "name": "历史"},
			},
			"production_companies": []map[string]interface{}{
				{"name": "Syncopy"},
				{"name": "Universal Pictures"},
			},
			"credits": map[string]interface{}{
				"cast": []map[string]interface{}{
					{"name": "基里安·墨菲", "character": "J. Robert Oppenheimer", "profile_path": "/cillian.jpg"},
				},
				"crew": []map[string]interface{}{
					{"name": "克里斯托弗·诺兰", "job": "Director"},
				},
			},
		})
	})

	// 3. 搜索电视剧接口
	mux.HandleFunc("/search/tv", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")
		w.Header().Set("Content-Type", "application/json")

		if query == "绝命毒师" || query == "Breaking Bad" {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"results": []map[string]interface{}{
					{
						"id":             1396,
						"name":           "绝命毒师",
						"first_air_date": "2008-01-20",
					},
				},
			})
			return
		}

		_ = json.NewEncoder(w).Encode(map[string]interface{}{"results": []interface{}{}})
	})

	// 4. 电视剧详情接口
	mux.HandleFunc("/tv/1396", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":             1396,
			"name":           "绝命毒师",
			"original_name":  "Breaking Bad",
			"overview":       "老白制毒传奇...",
			"first_air_date": "2008-01-20",
			"vote_average":   8.9,
			"vote_count":     14000,
			"poster_path":    "/tv_poster.jpg",
			"backdrop_path":  "/tv_backdrop.jpg",
			"genres": []map[string]interface{}{
				{"id": 18, "name": "剧情"},
			},
			"external_ids": map[string]interface{}{
				"imdb_id": "tt0903747",
			},
			"credits": map[string]interface{}{
				"cast": []map[string]interface{}{
					{"name": "布莱恩·科兰斯顿", "character": "Walter White", "profile_path": "/walter.jpg"},
				},
			},
		})
	})

	// 5. 电视剧单集接口
	mux.HandleFunc("/tv/1396/season/1/episode/1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"name":         "首播集",
			"overview":     "身患绝症的高中化学老师...",
			"still_path":   "/still1.jpg",
			"vote_average": 8.5,
			"air_date":     "2008-01-20",
		})
	})

	return httptest.NewServer(mux)
}

func TestTMDBClient_MovieAndTV(t *testing.T) {
	srv := setupMockTMDBServer(t)
	defer srv.Close()

	cfg := config.TMDBConfig{
		APIKey:       "mock_key",
		BaseURL:      srv.URL,
		ImageBaseURL: "https://image.tmdb.org/t/p/original",
		Language:     "zh-CN",
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 1. 测试电影搜索与详情抓取
	t.Run("SearchAndGetMovie", func(t *testing.T) {
		meta, err := client.SearchAndGetMovie(ctx, "奥本海默", 2023)
		if err != nil {
			t.Fatalf("SearchAndGetMovie error: %v", err)
		}

		if meta.TMDBID != 872585 {
			t.Errorf("expected TMDBID 872585, got %d", meta.TMDBID)
		}
		if meta.Title != "奥本海默" || meta.OriginalTitle != "Oppenheimer" {
			t.Errorf("title mismatch: %+v", meta)
		}
		if meta.Year != 2023 {
			t.Errorf("expected year 2023, got %d", meta.Year)
		}
		if meta.PosterURL != "https://image.tmdb.org/t/p/original/poster.jpg" {
			t.Errorf("poster url mismatch: %s", meta.PosterURL)
		}
		if len(meta.Directors) == 0 || meta.Directors[0] != "克里斯托弗·诺兰" {
			t.Errorf("director mismatch: %+v", meta.Directors)
		}
	})

	// 2. 测试电视剧及单集搜索抓取
	t.Run("SearchAndGetTV with Episode", func(t *testing.T) {
		meta, err := client.SearchAndGetTV(ctx, "绝命毒师", 2008, 1, 1)
		if err != nil {
			t.Fatalf("SearchAndGetTV error: %v", err)
		}

		if meta.TMDBID != 1396 {
			t.Errorf("expected TMDBID 1396, got %d", meta.TMDBID)
		}
		if meta.MediaType != parser.MediaTypeTV {
			t.Errorf("expected MediaTypeTV, got %v", meta.MediaType)
		}
		if meta.Season != 1 || meta.Episode != 1 {
			t.Errorf("expected S01E01, got S%02dE%02d", meta.Season, meta.Episode)
		}
		if meta.EpisodeTitle != "首播集" {
			t.Errorf("expected episode title '首播集', got %s", meta.EpisodeTitle)
		}
		if meta.EpisodeThumb != "https://image.tmdb.org/t/p/original/still1.jpg" {
			t.Errorf("episode thumb mismatch: %s", meta.EpisodeThumb)
		}
	})
}
