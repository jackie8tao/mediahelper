package aria2

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackie8tao/mediahelper/internal/config"
	"github.com/jackie8tao/mediahelper/internal/downloader"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupMockAria2Server(t *testing.T, expectedToken string) *httptest.Server {
	activeTasks := []aria2Task{
		{
			Gid:             "aria-active-1",
			Status:          "active",
			TotalLength:     "1000000",
			CompletedLength: "500000",
			DownloadSpeed:   "204800",
			UploadSpeed:     "0",
			InfoHash:        "ARIA_ACTIVE_HASH",
			Dir:             "/downloads/tv",
			Files: []aria2File{
				{
					Index:           "1",
					Path:            "/downloads/tv/Show.S01E01.mkv",
					Length:          "1000000",
					CompletedLength: "500000",
					Selected:        "true",
				},
			},
			Bittorrent: &aria2BT{
				Info: struct {
					Name string `json:"name"`
				}{
					Name: "Show.S01E01.mkv",
				},
			},
		},
	}

	stoppedTasks := []aria2Task{
		{
			Gid:             "aria-stopped-2",
			Status:          "complete",
			TotalLength:     "2000000",
			CompletedLength: "2000000",
			DownloadSpeed:   "0",
			UploadSpeed:     "0",
			InfoHash:        "ARIA_COMPLETED_HASH",
			Dir:             "/downloads/movie",
			Files: []aria2File{
				{
					Index:           "1",
					Path:            "/downloads/movie/Movie.2023.mkv",
					Length:          "2000000",
					CompletedLength: "2000000",
					Selected:        "true",
				},
			},
			Bittorrent: &aria2BT{
				Info: struct {
					Name string `json:"name"`
				}{
					Name: "Movie.2023.mkv",
				},
			},
		},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/jsonrpc", r.URL.Path)

		var req jsonRPCRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		// Verify token if expected
		if expectedToken != "" {
			require.NotEmpty(t, req.Params)
			firstParam, ok := req.Params[0].(string)
			require.True(t, ok)
			assert.Equal(t, "token:"+expectedToken, firstParam)
		}

		resp := jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
		}

		switch req.Method {
		case "aria2.getVersion":
			resBytes, _ := json.Marshal(map[string]any{
				"version":         "1.36.0",
				"enabledFeatures": []string{"BitTorrent"},
			})
			resp.Result = resBytes

		case "aria2.tellActive":
			resBytes, _ := json.Marshal(activeTasks)
			resp.Result = resBytes

		case "aria2.tellWaiting":
			resBytes, _ := json.Marshal([]aria2Task{})
			resp.Result = resBytes

		case "aria2.tellStopped":
			resBytes, _ := json.Marshal(stoppedTasks)
			resp.Result = resBytes

		default:
			http.Error(w, "unknown method", http.StatusBadRequest)
			return
		}

		_ = json.NewEncoder(w).Encode(resp)
	})

	return httptest.NewServer(handler)
}

func TestAria2Client(t *testing.T) {
	secret := "my-secret-token"
	srv := setupMockAria2Server(t, secret)
	defer srv.Close()

	cfg := config.DownloaderInstanceConfig{
		ID:         "aria2-test",
		Type:       "aria2",
		Name:       "Aria2 Client",
		Enabled:    true,
		URL:        srv.URL,
		Password:   secret,
		Categories: []string{"movie", "tv"},
		PathMappings: []config.PathMapping{
			{Remote: "/downloads", Local: "/local/downloads"},
		},
	}

	// 1. Factory check
	dl, err := downloader.CreateDownloader(cfg)
	require.NoError(t, err)
	assert.Equal(t, "Aria2", dl.Name())

	ctx := context.Background()

	// 2. TestConnection
	err = dl.TestConnection(ctx)
	assert.NoError(t, err)

	// 3. GetAllTorrents
	all, err := dl.GetAllTorrents(ctx, "all", "")
	require.NoError(t, err)
	assert.Len(t, all, 2)

	// Verify path mapping and properties
	var completedTask, activeTask downloader.Torrent
	for _, tor := range all {
		if tor.Hash == "aria_completed_hash" {
			completedTask = tor
		} else if tor.Hash == "aria_active_hash" {
			activeTask = tor
		}
	}

	assert.Equal(t, "aria_completed_hash", completedTask.Hash)
	assert.True(t, completedTask.IsCompleted)
	assert.Equal(t, "Movie.2023.mkv", completedTask.Name)
	assert.Equal(t, "/downloads/movie", completedTask.SavePath)
	assert.Equal(t, "movie", completedTask.Category)

	assert.Equal(t, "aria_active_hash", activeTask.Hash)
	assert.False(t, activeTask.IsCompleted)
	assert.Equal(t, "downloading", activeTask.State)
	assert.Equal(t, 0.5, activeTask.Progress)

	// 4. GetCompletedTorrents
	completed, err := dl.GetCompletedTorrents(ctx, []string{"movie"})
	require.NoError(t, err)
	assert.Len(t, completed, 1)
	assert.Equal(t, "aria_completed_hash", completed[0].Hash)

	// 5. GetTorrentFiles
	files, err := dl.GetTorrentFiles(ctx, "aria_completed_hash")
	require.NoError(t, err)
	assert.Len(t, files, 1)
	assert.Equal(t, "Movie.2023.mkv", files[0].Name)
	assert.Equal(t, 1.0, files[0].Progress)

	// 6. AddTags & RemoveTags
	err = dl.AddTags(ctx, []string{"aria_completed_hash"}, []string{"MediaHelper:Done", "TestTag"})
	assert.NoError(t, err)

	updatedAll, err := dl.GetAllTorrents(ctx, "all", "")
	require.NoError(t, err)
	for _, tor := range updatedAll {
		if tor.Hash == "aria_completed_hash" {
			assert.Contains(t, tor.Tags, "MediaHelper:Done")
			assert.Contains(t, tor.Tags, "TestTag")
		}
	}

	err = dl.RemoveTags(ctx, []string{"aria_completed_hash"}, []string{"TestTag"})
	assert.NoError(t, err)

	updatedAll2, err := dl.GetAllTorrents(ctx, "all", "")
	require.NoError(t, err)
	for _, tor := range updatedAll2 {
		if tor.Hash == "aria_completed_hash" {
			assert.Contains(t, tor.Tags, "MediaHelper:Done")
			assert.NotContains(t, tor.Tags, "TestTag")
		}
	}
}
