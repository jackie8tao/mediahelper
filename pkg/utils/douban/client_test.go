package douban

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestDoubanClient_Methods(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/movie/1292052":
			_ = json.NewEncoder(w).Encode(SubjectDetails{
				ID:            "1292052",
				Title:         "肖申克的救赎",
				OriginalTitle: "The Shawshank Redemption",
				Year:          "1994",
				Intro:         "一场谋杀案...",
				Rating:        Rating{Value: 9.7, Count: 2900000},
				CoverURL:      "https://example.com/poster.jpg",
				Genres:        []string{"犯罪", "剧情"},
				Directors:     []Person{{Name: "弗兰克·德拉邦特"}},
				Actors:        []Person{{Name: "蒂姆·罗宾斯"}, {Name: "摩根·弗里曼"}},
			})
		case "/tv/35465232":
			_ = json.NewEncoder(w).Encode(SubjectDetails{
				ID:            "35465232",
				Title:         "狂飙",
				OriginalTitle: "Punch Out",
				Year:          "2023",
				Rating:        Rating{Value: 8.5, Count: 320000},
				EpisodesCount: 39,
				CoverURL:      "https://example.com/tv_poster.jpg",
			})
		case "/movie/1292052/photos":
			_ = json.NewEncoder(w).Encode(PhotosResponse{
				Total: 1,
				Photos: []PhotoItem{
					{
						ID: "101",
						Image: struct {
							Large ImageDetails `json:"large"`
						}{
							Large: ImageDetails{URL: "https://example.com/backdrop.jpg"},
						},
					},
				},
			})
		case "/search/weixin":
			_ = json.NewEncoder(w).Encode(SearchResponse{
				Total: 1,
				Items: []SearchItem{
					{
						TargetType: "movie",
						Target: SearchTarget{
							ID:    "1292052",
							Title: "肖申克的救赎",
							Year:  "1994",
						},
					},
				},
			})
		case "/poster.jpg":
			w.Header().Set("Content-Type", "image/jpeg")
			_, _ = w.Write([]byte("fake-douban-image"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewClient(Config{
		BaseURL: server.URL,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	ctx := context.Background()

	// 1. GetMovie
	movie, err := client.GetMovie(ctx, "1292052")
	if err != nil {
		t.Fatalf("GetMovie error: %v", err)
	}
	if movie.Title != "肖申克的救赎" || movie.Rating.Value != 9.7 {
		t.Errorf("unexpected movie result: %+v", movie)
	}

	// 2. GetTV
	tv, err := client.GetTV(ctx, "35465232")
	if err != nil {
		t.Fatalf("GetTV error: %v", err)
	}
	if tv.Title != "狂飙" || tv.EpisodesCount != 39 {
		t.Errorf("unexpected tv result: %+v", tv)
	}

	// 3. GetPhotos
	photos, err := client.GetPhotos(ctx, "1292052", "movie")
	if err != nil {
		t.Fatalf("GetPhotos error: %v", err)
	}
	if len(photos.Photos) != 1 || photos.Photos[0].Image.Large.URL != "https://example.com/backdrop.jpg" {
		t.Errorf("unexpected photos: %+v", photos)
	}

	// 4. Search
	searchRes, err := client.Search(ctx, "肖申克")
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(searchRes.Items) != 1 || searchRes.Items[0].Target.ID != "1292052" {
		t.Errorf("unexpected search result: %+v", searchRes)
	}

	// 5. DownloadImage
	tmpDir := t.TempDir()
	imgDst := filepath.Join(tmpDir, "poster.jpg")
	if err := client.DownloadImage(ctx, server.URL+"/poster.jpg", imgDst); err != nil {
		t.Fatalf("DownloadImage error: %v", err)
	}
	data, err := os.ReadFile(imgDst)
	if err != nil || string(data) != "fake-douban-image" {
		t.Errorf("unexpected downloaded image data: %s, err: %v", string(data), err)
	}
}

