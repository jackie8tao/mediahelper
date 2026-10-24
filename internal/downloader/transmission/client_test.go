package transmission

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/jackie8tao/mediahelper/internal/config"
	"github.com/jackie8tao/mediahelper/internal/downloader"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupMockTransmissionServer(t *testing.T) (*httptest.Server, *[]trTorrent) {
	torrents := []trTorrent{
		{
			ID:           1,
			HashString:   "TR_HASH_1",
			Name:         "Movie.Title.2023.1080p.mkv",
			Status:       statusSeed,
			PercentDone:  1.0,
			RateDownload: 0,
			RateUpload:   1024,
			ETA:          0,
			TotalSize:    2147483648,
			Labels:       []string{"movie"},
			DownloadDir:  "/downloads/movies",
			IsFinished:   true,
			AddedDate:    1700000000,
			Files: []trFile{
				{BytesCompleted: 2147483648, Length: 2147483648, Name: "Movie.Title.2023.1080p.mkv"},
			},
		},
		{
			ID:           2,
			HashString:   "TR_HASH_2",
			Name:         "TV.Show.S01E01.mkv",
			Status:       statusDownload,
			PercentDone:  0.45,
			RateDownload: 2048,
			RateUpload:   0,
			ETA:          600,
			TotalSize:    1073741824,
			Labels:       []string{"tv"},
			DownloadDir:  "/downloads/tv",
			IsFinished:   false,
			AddedDate:    1700000100,
			Files: []trFile{
				{BytesCompleted: 483183820, Length: 1073741824, Name: "TV.Show.S01E01.mkv"},
			},
		},
	}

	var mu sync.Mutex
	validSessionID := "mock-session-token-xyz"

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/transmission/rpc", r.URL.Path)

		// 1. Session ID validation (Transmission CSRF check)
		incomingSess := r.Header.Get(headerSessionID)
		if incomingSess != validSessionID {
			w.Header().Set(headerSessionID, validSessionID)
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte("Conflict"))
			return
		}

		var req rpcRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		mu.Lock()
		defer mu.Unlock()

		switch req.Method {
		case "session-get":
			resp := map[string]any{
				"result": "success",
				"arguments": map[string]any{
					"version":     "4.0.5 (4f2165089c)",
					"rpc-version": 17,
				},
				"tag": req.Tag,
			}
			_ = json.NewEncoder(w).Encode(resp)

		case "torrent-get":
			var matched []trTorrent
			ids, hasIDs := req.Arguments["ids"].([]any)
			if hasIDs && len(ids) > 0 {
				idStrMap := make(map[string]bool)
				for _, id := range ids {
					idStrMap[strings.ToLower(id.(string))] = true
				}
				for _, tor := range torrents {
					if idStrMap[strings.ToLower(tor.HashString)] {
						matched = append(matched, tor)
					}
				}
			} else {
				matched = torrents
			}

			resp := map[string]any{
				"result": "success",
				"arguments": map[string]any{
					"torrents": matched,
				},
				"tag": req.Tag,
			}
			_ = json.NewEncoder(w).Encode(resp)

		case "torrent-set":
			ids, _ := req.Arguments["ids"].([]any)
			labels, _ := req.Arguments["labels"].([]any)
			newLabels := make([]string, 0, len(labels))
			for _, l := range labels {
				newLabels = append(newLabels, l.(string))
			}

			for _, id := range ids {
				for i := range torrents {
					if strings.EqualFold(torrents[i].HashString, id.(string)) {
						torrents[i].Labels = newLabels
					}
				}
			}

			resp := map[string]any{
				"result":    "success",
				"arguments": map[string]any{},
				"tag":       req.Tag,
			}
			_ = json.NewEncoder(w).Encode(resp)

		default:
			http.Error(w, "unknown method", http.StatusBadRequest)
		}
	})

	srv := httptest.NewServer(handler)
	return srv, &torrents
}

func TestTransmissionClient(t *testing.T) {
	srv, _ := setupMockTransmissionServer(t)
	defer srv.Close()

	cfg := config.DownloaderInstanceConfig{
		ID:         "transmission-test",
		Type:       "transmission",
		Name:       "Transmission Client",
		Enabled:    true,
		URL:        srv.URL,
		Categories: []string{"movie", "tv"},
	}

	// 1. Factory check
	dl, err := downloader.CreateDownloader(cfg)
	require.NoError(t, err)
	assert.Equal(t, "Transmission", dl.Name())

	ctx := context.Background()

	// 2. Test Connection (should succeed through 409 session handshake)
	err = dl.TestConnection(ctx)
	assert.NoError(t, err)

	// 3. GetAllTorrents
	all, err := dl.GetAllTorrents(ctx, "all", "")
	require.NoError(t, err)
	assert.Len(t, all, 2)
	assert.Equal(t, "tr_hash_1", all[0].Hash)
	assert.True(t, all[0].IsCompleted)
	assert.Equal(t, "uploading", all[0].State)
	assert.Equal(t, "/downloads/movies", all[0].SavePath)

	// 4. Filter by completed
	completed, err := dl.GetCompletedTorrents(ctx, []string{"movie"})
	require.NoError(t, err)
	assert.Len(t, completed, 1)
	assert.Equal(t, "tr_hash_1", completed[0].Hash)

	// 5. GetTorrentFiles
	files, err := dl.GetTorrentFiles(ctx, "tr_hash_1")
	require.NoError(t, err)
	assert.Len(t, files, 1)
	assert.Equal(t, "Movie.Title.2023.1080p.mkv", files[0].Name)
	assert.Equal(t, 1.0, files[0].Progress)

	// 6. AddTags & RemoveTags
	err = dl.AddTags(ctx, []string{"tr_hash_1"}, []string{"MediaHelper:Done", "Processed"})
	assert.NoError(t, err)

	updated, err := dl.GetAllTorrents(ctx, "all", "")
	require.NoError(t, err)
	assert.Contains(t, updated[0].Tags, "MediaHelper:Done")
	assert.Contains(t, updated[0].Tags, "Processed")
	assert.Contains(t, updated[0].Tags, "movie")

	err = dl.RemoveTags(ctx, []string{"tr_hash_1"}, []string{"Processed"})
	assert.NoError(t, err)

	updated2, err := dl.GetAllTorrents(ctx, "all", "")
	require.NoError(t, err)
	assert.Contains(t, updated2[0].Tags, "MediaHelper:Done")
	assert.NotContains(t, updated2[0].Tags, "Processed")
}
