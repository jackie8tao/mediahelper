package transmission

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/jackie8tao/mediahelper/internal/config"
	"github.com/jackie8tao/mediahelper/internal/downloader"
)

const (
	headerSessionID = "X-Transmission-Session-Id"

	// Transmission torrent status codes
	statusStopped      = 0 // TR_STATUS_STOPPED
	statusCheckWait    = 1 // TR_STATUS_CHECK_WAIT
	statusCheck        = 2 // TR_STATUS_CHECK
	statusDownloadWait = 3 // TR_STATUS_DOWNLOAD_WAIT
	statusDownload     = 4 // TR_STATUS_DOWNLOAD
	statusSeedWait     = 5 // TR_STATUS_SEED_WAIT
	statusSeed         = 6 // TR_STATUS_SEED
)

func init() {
	downloader.RegisterFactory("transmission", func(cfg config.DownloaderInstanceConfig) (downloader.Downloader, error) {
		return NewClient(cfg)
	})
}

// Client implements downloader.Downloader for Transmission RPC.
type Client struct {
	cfg        config.DownloaderInstanceConfig
	endpoint   string
	httpClient *http.Client
	pathMapper *downloader.PathMapper

	sessionMu sync.RWMutex
	sessionID string
}

// NewClient creates a new Transmission RPC client instance.
func NewClient(cfg config.DownloaderInstanceConfig) (*Client, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("transmission URL is required")
	}

	endpoint := strings.TrimRight(cfg.URL, "/")
	if !strings.HasSuffix(endpoint, "/transmission/rpc") {
		endpoint = endpoint + "/transmission/rpc"
	}

	timeout := 15 * time.Second
	if cfg.PollInterval > 0 && cfg.PollInterval < 15 {
		timeout = time.Duration(cfg.PollInterval) * time.Second
	}

	return &Client{
		cfg:        cfg,
		endpoint:   endpoint,
		httpClient: &http.Client{Timeout: timeout},
		pathMapper: downloader.NewPathMapper(cfg.PathMappings),
	}, nil
}

// Name returns the downloader client name.
func (c *Client) Name() string {
	return "Transmission"
}

// rpcRequest defines the standard JSON-RPC payload for Transmission.
type rpcRequest struct {
	Method    string         `json:"method"`
	Arguments map[string]any `json:"arguments,omitempty"`
	Tag       int            `json:"tag,omitempty"`
}

// rpcResponse defines the standard response wrapper from Transmission.
type rpcResponse struct {
	Result    string          `json:"result"`
	Arguments json.RawMessage `json:"arguments"`
	Tag       int             `json:"tag"`
}

// doRequest sends an HTTP POST request to the Transmission RPC endpoint,
// handling X-Transmission-Session-Id negotiation and Basic Auth.
func (c *Client) doRequest(ctx context.Context, method string, args map[string]any) ([]byte, error) {
	reqBody, err := json.Marshal(rpcRequest{
		Method:    method,
		Arguments: args,
		Tag:       1,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal transmission request failed: %w", err)
	}

	send := func(sessID string) (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(reqBody))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		if sessID != "" {
			req.Header.Set(headerSessionID, sessID)
		}
		if c.cfg.Username != "" || c.cfg.Password != "" {
			req.SetBasicAuth(c.cfg.Username, c.cfg.Password)
		}
		return c.httpClient.Do(req)
	}

	c.sessionMu.RLock()
	sessID := c.sessionID
	c.sessionMu.RUnlock()

	resp, err := send(sessID)
	if err != nil {
		return nil, fmt.Errorf("transmission request failed: %w", err)
	}
	defer resp.Body.Close()

	// Transmission returns 409 Conflict with the required session token in headers
	if resp.StatusCode == http.StatusConflict {
		newSessID := resp.Header.Get(headerSessionID)
		if newSessID == "" {
			return nil, fmt.Errorf("received 409 Conflict from transmission without %s header", headerSessionID)
		}

		c.sessionMu.Lock()
		c.sessionID = newSessID
		c.sessionMu.Unlock()

		resp2, err := send(newSessID)
		if err != nil {
			return nil, fmt.Errorf("transmission request retry failed: %w", err)
		}
		defer resp2.Body.Close()
		resp = resp2
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("transmission returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read transmission response failed: %w", err)
	}

	var rpcResp rpcResponse
	if err := json.Unmarshal(respBytes, &rpcResp); err != nil {
		return nil, fmt.Errorf("unmarshal transmission response failed: %w", err)
	}

	if rpcResp.Result != "success" {
		return nil, fmt.Errorf("transmission rpc error: %s", rpcResp.Result)
	}

	return rpcResp.Arguments, nil
}

// TestConnection verifies connectivity and credentials with Transmission.
func (c *Client) TestConnection(ctx context.Context) error {
	rawArgs, err := c.doRequest(ctx, "session-get", nil)
	if err != nil {
		return fmt.Errorf("transmission connection test failed: %w", err)
	}

	var sessInfo struct {
		Version    string `json:"version"`
		RPCVersion int    `json:"rpc-version"`
	}
	if err := json.Unmarshal(rawArgs, &sessInfo); err != nil {
		return fmt.Errorf("parse session-get response failed: %w", err)
	}

	slog.Info("Transmission connected successfully", "version", sessInfo.Version, "rpc_version", sessInfo.RPCVersion, "url", c.cfg.URL)
	return nil
}

// trTorrent represents raw torrent structure returned by Transmission RPC.
type trTorrent struct {
	ID           int64    `json:"id"`
	HashString   string   `json:"hashString"`
	Name         string   `json:"name"`
	Status       int      `json:"status"`
	PercentDone  float64  `json:"percentDone"`
	RateDownload int64    `json:"rateDownload"`
	RateUpload   int64    `json:"rateUpload"`
	ETA          int64    `json:"eta"`
	TotalSize    int64    `json:"totalSize"`
	Labels       []string `json:"labels"`
	DownloadDir  string   `json:"downloadDir"`
	IsFinished   bool     `json:"isFinished"`
	AddedDate    int64    `json:"addedDate"`
	Files        []trFile `json:"files,omitempty"`
}

type trFile struct {
	BytesCompleted int64  `json:"bytesCompleted"`
	Length         int64  `json:"length"`
	Name           string `json:"name"`
}

func (c *Client) convertTorrent(t trTorrent) downloader.Torrent {
	hash := strings.ToLower(t.HashString)
	savePath := c.pathMapper.ToLocal(t.DownloadDir)
	contentPath := filepath.Join(savePath, t.Name)

	isCompleted := t.PercentDone >= 1.0 || t.IsFinished || t.Status == statusSeed || t.Status == statusSeedWait

	var state string
	switch t.Status {
	case statusStopped:
		if isCompleted {
			state = "pausedUP"
		} else {
			state = "pausedDL"
		}
	case statusCheckWait, statusCheck:
		state = "checking"
	case statusDownloadWait, statusDownload:
		state = "downloading"
	case statusSeedWait, statusSeed:
		state = "uploading"
	default:
		state = "unknown"
	}

	category := ""
	for _, l := range t.Labels {
		for _, cat := range c.cfg.Categories {
			if strings.EqualFold(l, cat) {
				category = l
				break
			}
		}
		if category != "" {
			break
		}
	}
	if category == "" && len(t.Labels) > 0 {
		category = t.Labels[0]
	}

	return downloader.Torrent{
		Hash:        hash,
		Name:        t.Name,
		Category:    category,
		Tags:        t.Labels,
		SavePath:    savePath,
		ContentPath: contentPath,
		Progress:    t.PercentDone,
		State:       state,
		Size:        t.TotalSize,
		IsCompleted: isCompleted,
		DlSpeed:     t.RateDownload,
		UpSpeed:     t.RateUpload,
		ETA:         t.ETA,
		AddedOn:     t.AddedDate,
	}
}

// GetAllTorrents returns all torrents matching state filter and category.
func (c *Client) GetAllTorrents(ctx context.Context, filter string, category string) ([]downloader.Torrent, error) {
	fields := []string{
		"id", "hashString", "name", "status", "percentDone",
		"rateDownload", "rateUpload", "eta", "totalSize",
		"labels", "downloadDir", "isFinished", "addedDate",
	}

	args := map[string]any{
		"fields": fields,
	}

	rawArgs, err := c.doRequest(ctx, "torrent-get", args)
	if err != nil {
		return nil, fmt.Errorf("get all torrents error: %w", err)
	}

	var resp struct {
		Torrents []trTorrent `json:"torrents"`
	}
	if err := json.Unmarshal(rawArgs, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal torrent-get response failed: %w", err)
	}

	result := make([]downloader.Torrent, 0, len(resp.Torrents))
	for _, t := range resp.Torrents {
		dt := c.convertTorrent(t)

		// Filter matching
		if category != "" && !strings.EqualFold(dt.Category, category) {
			hasLabel := false
			for _, l := range dt.Tags {
				if strings.EqualFold(l, category) {
					hasLabel = true
					break
				}
			}
			if !hasLabel {
				continue
			}
		}

		if filter != "" && filter != "all" {
			match := false
			switch strings.ToLower(filter) {
			case "completed":
				match = dt.IsCompleted
			case "downloading":
				match = (dt.State == "downloading")
			case "uploading", "seeding":
				match = (dt.State == "uploading")
			case "paused", "stopped":
				match = strings.HasPrefix(dt.State, "paused")
			default:
				match = strings.EqualFold(dt.State, filter)
			}
			if !match {
				continue
			}
		}

		result = append(result, dt)
	}

	return result, nil
}

// GetCompletedTorrents returns completed torrents matching the specified categories.
func (c *Client) GetCompletedTorrents(ctx context.Context, categories []string) ([]downloader.Torrent, error) {
	allTorrents, err := c.GetAllTorrents(ctx, "completed", "")
	if err != nil {
		return nil, err
	}

	if len(categories) == 0 {
		return allTorrents, nil
	}

	catMap := make(map[string]bool, len(categories))
	for _, cat := range categories {
		catMap[strings.ToLower(strings.TrimSpace(cat))] = true
	}

	filtered := make([]downloader.Torrent, 0, len(allTorrents))
	for _, t := range allTorrents {
		if catMap[strings.ToLower(t.Category)] {
			filtered = append(filtered, t)
			continue
		}
		// Also inspect tags for category matching
		for _, tag := range t.Tags {
			if catMap[strings.ToLower(tag)] {
				filtered = append(filtered, t)
				break
			}
		}
	}

	return filtered, nil
}

// GetTorrentFiles returns all files within a specified torrent hash.
func (c *Client) GetTorrentFiles(ctx context.Context, hash string) ([]downloader.TorrentFile, error) {
	args := map[string]any{
		"ids":    []string{hash},
		"fields": []string{"id", "hashString", "files"},
	}

	rawArgs, err := c.doRequest(ctx, "torrent-get", args)
	if err != nil {
		return nil, fmt.Errorf("get torrent files error: %w", err)
	}

	var resp struct {
		Torrents []trTorrent `json:"torrents"`
	}
	if err := json.Unmarshal(rawArgs, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal torrent files response failed: %w", err)
	}

	if len(resp.Torrents) == 0 {
		return nil, fmt.Errorf("torrent %s not found in transmission", hash)
	}

	t := resp.Torrents[0]
	files := make([]downloader.TorrentFile, 0, len(t.Files))
	for i, f := range t.Files {
		var progress float64
		if f.Length > 0 {
			progress = float64(f.BytesCompleted) / float64(f.Length)
		}
		files = append(files, downloader.TorrentFile{
			Index:    i,
			Name:     f.Name,
			Size:     f.Length,
			Progress: progress,
		})
	}

	return files, nil
}

// AddTags applies tags (labels) to the specified torrent hashes in Transmission.
func (c *Client) AddTags(ctx context.Context, hashes []string, tags []string) error {
	if len(hashes) == 0 || len(tags) == 0 {
		return nil
	}

	// Fetch current labels for target torrents
	args := map[string]any{
		"ids":    hashes,
		"fields": []string{"id", "hashString", "labels"},
	}
	rawArgs, err := c.doRequest(ctx, "torrent-get", args)
	if err != nil {
		return fmt.Errorf("fetch torrents for add tags error: %w", err)
	}

	var resp struct {
		Torrents []trTorrent `json:"torrents"`
	}
	if err := json.Unmarshal(rawArgs, &resp); err != nil {
		return fmt.Errorf("unmarshal torrents for add tags failed: %w", err)
	}

	for _, t := range resp.Torrents {
		labelSet := make(map[string]bool)
		for _, l := range t.Labels {
			labelSet[l] = true
		}
		for _, tag := range tags {
			trimmed := strings.TrimSpace(tag)
			if trimmed != "" {
				labelSet[trimmed] = true
			}
		}

		mergedLabels := make([]string, 0, len(labelSet))
		for l := range labelSet {
			mergedLabels = append(mergedLabels, l)
		}

		setArgs := map[string]any{
			"ids":    []string{t.HashString},
			"labels": mergedLabels,
		}
		if _, err := c.doRequest(ctx, "torrent-set", setArgs); err != nil {
			return fmt.Errorf("set labels on torrent %s failed: %w", t.HashString, err)
		}
	}

	return nil
}

// RemoveTags removes tags (labels) from the specified torrent hashes in Transmission.
func (c *Client) RemoveTags(ctx context.Context, hashes []string, tags []string) error {
	if len(hashes) == 0 || len(tags) == 0 {
		return nil
	}

	args := map[string]any{
		"ids":    hashes,
		"fields": []string{"id", "hashString", "labels"},
	}
	rawArgs, err := c.doRequest(ctx, "torrent-get", args)
	if err != nil {
		return fmt.Errorf("fetch torrents for remove tags error: %w", err)
	}

	var resp struct {
		Torrents []trTorrent `json:"torrents"`
	}
	if err := json.Unmarshal(rawArgs, &resp); err != nil {
		return fmt.Errorf("unmarshal torrents for remove tags failed: %w", err)
	}

	removeSet := make(map[string]bool, len(tags))
	for _, tag := range tags {
		removeSet[strings.ToLower(strings.TrimSpace(tag))] = true
	}

	for _, t := range resp.Torrents {
		var remainingLabels []string
		for _, l := range t.Labels {
			if !removeSet[strings.ToLower(l)] {
				remainingLabels = append(remainingLabels, l)
			}
		}

		setArgs := map[string]any{
			"ids":    []string{t.HashString},
			"labels": remainingLabels,
		}
		if _, err := c.doRequest(ctx, "torrent-set", setArgs); err != nil {
			return fmt.Errorf("update labels on torrent %s failed: %w", t.HashString, err)
		}
	}

	return nil
}
