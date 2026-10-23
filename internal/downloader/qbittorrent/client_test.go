package qbittorrent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackie8tao/mediahelper/internal/config"
)

func setupMockQBSrv(t *testing.T) *httptest.Server {
	mux := http.NewServeMux()

	// 1. 登录认证接口
	mux.HandleFunc("/api/v2/auth/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		_ = r.ParseForm()
		user := r.FormValue("username")
		pass := r.FormValue("password")

		if user == "admin" && pass == "secret123" {
			http.SetCookie(w, &http.Cookie{
				Name:  "SID",
				Value: "mock_session_id",
				Path:  "/",
			})
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("Ok."))
		} else {
			http.Error(w, "Fails.", http.StatusForbidden)
		}
	})

	// 2. 版本获取接口
	mux.HandleFunc("/api/v2/app/version", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("v5.0.1"))
	})

	// 3. 种子列表接口
	mux.HandleFunc("/api/v2/torrents/info", func(w http.ResponseWriter, r *http.Request) {
		torrents := []map[string]interface{}{
			{
				"hash":          "hash_movie_1",
				"name":          "Oppenheimer.2023.2160p.UHD.BluRay.x265",
				"category":      "movie",
				"tags":          "4K, MediaHelper:Done",
				"save_path":     "/downloads/movies",
				"content_path":  "/downloads/movies/Oppenheimer.2023.2160p.UHD.BluRay.x265",
				"progress":      1.0,
				"state":         "uploading",
				"size":          32212254720,
				"completion_on": 1698000000,
			},
			{
				"hash":          "hash_tv_1",
				"name":          "Breaking.Bad.S01.1080p",
				"category":      "tv",
				"tags":          "",
				"save_path":     "/downloads/tv",
				"content_path":  "/downloads/tv/Breaking.Bad.S01.1080p",
				"progress":      1.0,
				"state":         "pausedUP",
				"size":          10737418240,
				"completion_on": 1698000100,
			},
			{
				"hash":          "hash_other_1",
				"name":          "Some.Software.ISO",
				"category":      "software",
				"tags":          "",
				"save_path":     "/downloads/other",
				"content_path":  "/downloads/other/software.iso",
				"progress":      1.0,
				"state":         "uploading",
				"size":          2147483648,
				"completion_on": 1698000200,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(torrents)
	})

	// 4. 种子文件详情接口
	mux.HandleFunc("/api/v2/torrents/files", func(w http.ResponseWriter, r *http.Request) {
		hash := r.URL.Query().Get("hash")
		if hash == "hash_tv_1" {
			files := []map[string]interface{}{
				{
					"index":    0,
					"name":     "Breaking.Bad.S01E01.mkv",
					"size":     1073741824,
					"progress": 1.0,
				},
				{
					"index":    1,
					"name":     "Breaking.Bad.S01E02.mkv",
					"size":     1073741824,
					"progress": 1.0,
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(files)
			return
		}

		http.NotFound(w, r)
	})

	// 5. 添加标签接口
	mux.HandleFunc("/api/v2/torrents/addTags", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		hashes := r.FormValue("hashes")
		tags := r.FormValue("tags")
		if hashes == "" || tags == "" {
			http.Error(w, "missing hashes or tags", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	// 6. 移除标签接口
	mux.HandleFunc("/api/v2/torrents/removeTags", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		hashes := r.FormValue("hashes")
		tags := r.FormValue("tags")
		if hashes == "" || tags == "" {
			http.Error(w, "missing hashes or tags", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	return httptest.NewServer(mux)
}

func TestClient_AllOperations(t *testing.T) {
	server := setupMockQBSrv(t)
	defer server.Close()

	cfg := config.QBittorrentConfig{
		Enabled:    true,
		URL:        server.URL,
		Username:   "admin",
		Password:   "secret123",
		Categories: []string{"movie", "tv"},
		PathMappings: []config.PathMapping{
			{
				Remote: "/downloads",
				Local:  "D:/LocalDownloads",
			},
		},
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 1. 测试连通性
	t.Run("TestConnection", func(t *testing.T) {
		err := client.TestConnection(ctx)
		if err != nil {
			t.Errorf("TestConnection failed: %v", err)
		}
	})

	// 1.5 测试获取所有种子任务
	t.Run("GetAllTorrents", func(t *testing.T) {
		torrents, err := client.GetAllTorrents(ctx, "all", "")
		if err != nil {
			t.Fatalf("GetAllTorrents failed: %v", err)
		}
		if len(torrents) != 3 {
			t.Fatalf("expected 3 torrents, got %d", len(torrents))
		}
	})

	// 2. 测试获取已完成种子并校验分类过滤与路径映射
	t.Run("GetCompletedTorrents with Category Filtering and Path Mapping", func(t *testing.T) {
		torrents, err := client.GetCompletedTorrents(ctx, []string{"movie", "tv"})
		if err != nil {
			t.Fatalf("GetCompletedTorrents failed: %v", err)
		}

		// 应该只保留 movie 和 tv，software 被过滤
		if len(torrents) != 2 {
			t.Fatalf("expected 2 torrents, got %d", len(torrents))
		}

		first := torrents[0]
		if first.Hash != "hash_movie_1" {
			t.Errorf("expected hash_movie_1, got %s", first.Hash)
		}

		expectedPath := filepath.FromSlash("D:/LocalDownloads/movies/Oppenheimer.2023.2160p.UHD.BluRay.x265")
		if first.ContentPath != expectedPath {
			t.Errorf("path mapping mismatch: expected %s, got %s", expectedPath, first.ContentPath)
		}

		if len(first.Tags) != 2 || first.Tags[0] != "4K" || first.Tags[1] != "MediaHelper:Done" {
			t.Errorf("tags parsed incorrectly: %+v", first.Tags)
		}
	})

	// 3. 测试获取单个种子包含的文件
	t.Run("GetTorrentFiles", func(t *testing.T) {
		files, err := client.GetTorrentFiles(ctx, "hash_tv_1")
		if err != nil {
			t.Fatalf("GetTorrentFiles failed: %v", err)
		}

		if len(files) != 2 {
			t.Fatalf("expected 2 files, got %d", len(files))
		}

		if files[0].Name != "Breaking.Bad.S01E01.mkv" {
			t.Errorf("unexpected file name: %s", files[0].Name)
		}
	})

	// 4. 测试打标签与去标签
	t.Run("AddTags and RemoveTags", func(t *testing.T) {
		err := client.AddTags(ctx, []string{"hash_tv_1"}, []string{"MediaHelper:Done"})
		if err != nil {
			t.Errorf("AddTags failed: %v", err)
		}

		err = client.RemoveTags(ctx, []string{"hash_tv_1"}, []string{"MediaHelper:Error"})
		if err != nil {
			t.Errorf("RemoveTags failed: %v", err)
		}
	})
}
