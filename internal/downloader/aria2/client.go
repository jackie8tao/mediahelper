package aria2

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackie8tao/mediahelper/internal/config"
	"github.com/jackie8tao/mediahelper/internal/downloader"
)

func init() {
	downloader.RegisterFactory("aria2", func(cfg config.DownloaderInstanceConfig) (downloader.Downloader, error) {
		return NewClient(cfg)
	})
}

// Client implements downloader.Downloader for Aria2 JSON-RPC.
type Client struct {
	cfg        config.DownloaderInstanceConfig
	endpoint   string
	token      string
	httpClient *http.Client
	pathMapper *downloader.PathMapper

	// In-memory tag store to support MediaHelper tag lifecycle on Aria2 tasks
	tagsMu sync.RWMutex
	tags   map[string][]string // Keyed by hash and gid
}

// NewClient creates a new Aria2 JSON-RPC client instance.
func NewClient(cfg config.DownloaderInstanceConfig) (*Client, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("aria2 URL is required")
	}

	endpoint := strings.TrimRight(cfg.URL, "/")
	if !strings.HasSuffix(endpoint, "/jsonrpc") {
		endpoint = endpoint + "/jsonrpc"
	}

	secret := cfg.Password
	if secret == "" && cfg.Username != "" {
		secret = cfg.Username
	}

	token := ""
	if secret != "" {
		if strings.HasPrefix(secret, "token:") {
			token = secret
		} else {
			token = "token:" + secret
		}
	}

	timeout := 15 * time.Second
	if cfg.PollInterval > 0 && cfg.PollInterval < 15 {
		timeout = time.Duration(cfg.PollInterval) * time.Second
	}

	return &Client{
		cfg:        cfg,
		endpoint:   endpoint,
		token:      token,
		httpClient: &http.Client{Timeout: timeout},
		pathMapper: downloader.NewPathMapper(cfg.PathMappings),
		tags:       make(map[string][]string),
	}, nil
}

// Name returns the downloader client name.
func (c *Client) Name() string {
	return "Aria2"
}

// jsonRPCRequest represents the JSON-RPC 2.0 request payload.
type jsonRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      string `json:"id"`
	Method  string `json:"method"`
	Params  []any  `json:"params"`
}

// jsonRPCResponse represents the JSON-RPC 2.0 response wrapper.
type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      string          `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

// jsonRPCError represents the error object in JSON-RPC 2.0.
type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// doRPC executes an Aria2 JSON-RPC method with optional token prepending.
func (c *Client) doRPC(ctx context.Context, method string, params []any) (json.RawMessage, error) {
	fullParams := make([]any, 0, len(params)+1)
	if c.token != "" {
		fullParams = append(fullParams, c.token)
	}
	fullParams = append(fullParams, params...)

	reqBody, err := json.Marshal(jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      "mediahelper",
		Method:  method,
		Params:  fullParams,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal aria2 request failed: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.cfg.Username != "" && c.cfg.Password != "" {
		req.SetBasicAuth(c.cfg.Username, c.cfg.Password)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("aria2 request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("aria2 returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read aria2 response failed: %w", err)
	}

	var rpcResp jsonRPCResponse
	if err := json.Unmarshal(respBytes, &rpcResp); err != nil {
		return nil, fmt.Errorf("unmarshal aria2 response failed: %w", err)
	}

	if rpcResp.Error != nil {
		return nil, fmt.Errorf("aria2 error (%d): %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	return rpcResp.Result, nil
}

// TestConnection verifies connectivity and credentials with Aria2.
func (c *Client) TestConnection(ctx context.Context) error {
	result, err := c.doRPC(ctx, "aria2.getVersion", nil)
	if err != nil {
		return fmt.Errorf("aria2 connection test failed: %w", err)
	}

	var ver struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(result, &ver); err != nil {
		return fmt.Errorf("parse aria2 version failed: %w", err)
	}

	slog.Info("Aria2 connected successfully", "version", ver.Version, "url", c.cfg.URL)
	return nil
}

// aria2Task represents download task metadata from Aria2.
type aria2Task struct {
	Gid             string      `json:"gid"`
	Status          string      `json:"status"` // "active", "waiting", "paused", "error", "complete", "removed"
	TotalLength     string      `json:"totalLength"`
	CompletedLength string      `json:"completedLength"`
	DownloadSpeed   string      `json:"downloadSpeed"`
	UploadSpeed     string      `json:"uploadSpeed"`
	InfoHash        string      `json:"infoHash"`
	Dir             string      `json:"dir"`
	Files           []aria2File `json:"files"`
	Bittorrent      *aria2BT    `json:"bittorrent,omitempty"`
}

type aria2File struct {
	Index           string `json:"index"`
	Path            string `json:"path"`
	Length          string `json:"length"`
	CompletedLength string `json:"completedLength"`
	Selected        string `json:"selected"`
}

type aria2BT struct {
	Info struct {
		Name string `json:"name"`
	} `json:"info"`
	Mode string `json:"mode"`
}

func parseNumber(s string) int64 {
	n, _ := strconv.ParseInt(s, 10, 64)
	return n
}

// getTaskTags returns tags associated with a task hash or gid.
func (c *Client) getTaskTags(hash string, gid string) []string {
	c.tagsMu.RLock()
	defer c.tagsMu.RUnlock()

	tagSet := make(map[string]bool)
	if tags, ok := c.tags[strings.ToLower(hash)]; ok {
		for _, t := range tags {
			tagSet[t] = true
		}
	}
	if gid != "" {
		if tags, ok := c.tags[strings.ToLower(gid)]; ok {
			for _, t := range tags {
				tagSet[t] = true
			}
		}
	}

	result := make([]string, 0, len(tagSet))
	for t := range tagSet {
		result = append(result, t)
	}
	return result
}

func (c *Client) convertTask(t aria2Task) downloader.Torrent {
	hash := strings.ToLower(t.InfoHash)
	if hash == "" {
		hash = strings.ToLower(t.Gid)
	}

	var name string
	if t.Bittorrent != nil && t.Bittorrent.Info.Name != "" {
		name = t.Bittorrent.Info.Name
	} else if len(t.Files) > 0 && t.Files[0].Path != "" {
		name = filepath.Base(t.Files[0].Path)
	} else {
		name = t.Gid
	}

	total := parseNumber(t.TotalLength)
	completed := parseNumber(t.CompletedLength)
	var progress float64
	if total > 0 {
		progress = float64(completed) / float64(total)
	}

	isCompleted := t.Status == "complete" || (total > 0 && completed >= total)

	var state string
	switch t.Status {
	case "active":
		if isCompleted {
			state = "uploading"
		} else {
			state = "downloading"
		}
	case "waiting":
		state = "queued"
	case "paused":
		state = "pausedDL"
	case "complete":
		state = "pausedUP"
	case "error":
		state = "error"
	case "removed":
		state = "deleted"
	default:
		state = t.Status
	}

	savePath := c.pathMapper.ToLocal(t.Dir)
	contentPath := savePath
	if t.Bittorrent != nil && t.Bittorrent.Info.Name != "" {
		contentPath = filepath.Join(savePath, t.Bittorrent.Info.Name)
	} else if len(t.Files) > 0 && t.Files[0].Path != "" {
		contentPath = c.pathMapper.ToLocal(t.Files[0].Path)
	}

	tags := c.getTaskTags(hash, t.Gid)

	category := ""
	for _, tag := range tags {
		for _, cat := range c.cfg.Categories {
			if strings.EqualFold(tag, cat) {
				category = tag
				break
			}
		}
		if category != "" {
			break
		}
	}
	if category == "" {
		normDir := strings.ToLower(filepath.ToSlash(t.Dir))
		for _, cat := range c.cfg.Categories {
			if strings.Contains(normDir, "/"+strings.ToLower(cat)) {
				category = cat
				break
			}
		}
	}
	if category == "" && len(tags) > 0 {
		category = tags[0]
	}

	return downloader.Torrent{
		Hash:        hash,
		Name:        name,
		Category:    category,
		Tags:        tags,
		SavePath:    savePath,
		ContentPath: contentPath,
		Progress:    progress,
		State:       state,
		Size:        total,
		IsCompleted: isCompleted,
		DlSpeed:     parseNumber(t.DownloadSpeed),
		UpSpeed:     parseNumber(t.UploadSpeed),
		ETA:         0,
		AddedOn:     0,
	}
}

var taskKeys = []any{"gid", "status", "totalLength", "completedLength", "downloadSpeed", "uploadSpeed", "infoHash", "dir", "files", "bittorrent"}

func (c *Client) fetchAllTasks(ctx context.Context) ([]aria2Task, error) {
	var allTasks []aria2Task

	// 1. Active downloads
	activeRes, err := c.doRPC(ctx, "aria2.tellActive", []any{taskKeys})
	if err != nil {
		return nil, fmt.Errorf("aria2.tellActive failed: %w", err)
	}
	var activeTasks []aria2Task
	if err := json.Unmarshal(activeRes, &activeTasks); err == nil {
		allTasks = append(allTasks, activeTasks...)
	}

	// 2. Waiting downloads
	waitingRes, err := c.doRPC(ctx, "aria2.tellWaiting", []any{0, 1000, taskKeys})
	if err == nil {
		var waitingTasks []aria2Task
		if err := json.Unmarshal(waitingRes, &waitingTasks); err == nil {
			allTasks = append(allTasks, waitingTasks...)
		}
	}

	// 3. Stopped downloads
	stoppedRes, err := c.doRPC(ctx, "aria2.tellStopped", []any{0, 1000, taskKeys})
	if err == nil {
		var stoppedTasks []aria2Task
		if err := json.Unmarshal(stoppedRes, &stoppedTasks); err == nil {
			allTasks = append(allTasks, stoppedTasks...)
		}
	}

	return allTasks, nil
}

// GetAllTorrents returns all torrents matching state filter and category.
func (c *Client) GetAllTorrents(ctx context.Context, filter string, category string) ([]downloader.Torrent, error) {
	tasks, err := c.fetchAllTasks(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]downloader.Torrent, 0, len(tasks))
	for _, t := range tasks {
		dt := c.convertTask(t)

		if category != "" && !strings.EqualFold(dt.Category, category) {
			hasTag := false
			for _, tag := range dt.Tags {
				if strings.EqualFold(tag, category) {
					hasTag = true
					break
				}
			}
			if !hasTag {
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
		for _, tag := range t.Tags {
			if catMap[strings.ToLower(tag)] {
				filtered = append(filtered, t)
				break
			}
		}
	}

	return filtered, nil
}

// GetTorrentFiles returns all files within a specified torrent hash or gid.
func (c *Client) GetTorrentFiles(ctx context.Context, hash string) ([]downloader.TorrentFile, error) {
	targetHash := strings.ToLower(hash)
	tasks, err := c.fetchAllTasks(ctx)
	if err != nil {
		return nil, err
	}

	var matchedTask *aria2Task
	for i := range tasks {
		if strings.EqualFold(tasks[i].InfoHash, targetHash) || strings.EqualFold(tasks[i].Gid, targetHash) {
			matchedTask = &tasks[i]
			break
		}
	}

	if matchedTask == nil {
		return nil, fmt.Errorf("task with hash/gid %s not found in aria2", hash)
	}

	files := make([]downloader.TorrentFile, 0, len(matchedTask.Files))
	for _, f := range matchedTask.Files {
		idx, _ := strconv.Atoi(f.Index)
		length := parseNumber(f.Length)
		completed := parseNumber(f.CompletedLength)
		var progress float64
		if length > 0 {
			progress = float64(completed) / float64(length)
		}
		name := f.Path
		if matchedTask.Dir != "" && strings.HasPrefix(filepath.ToSlash(name), filepath.ToSlash(matchedTask.Dir)) {
			name = strings.TrimPrefix(filepath.ToSlash(name), filepath.ToSlash(matchedTask.Dir))
			name = strings.TrimPrefix(name, "/")
		}

		files = append(files, downloader.TorrentFile{
			Index:    idx,
			Name:     name,
			Size:     length,
			Progress: progress,
		})
	}

	return files, nil
}

// AddTags applies tags to specified hashes/gids in the local in-memory tag store.
func (c *Client) AddTags(ctx context.Context, hashes []string, tags []string) error {
	if len(hashes) == 0 || len(tags) == 0 {
		return nil
	}

	c.tagsMu.Lock()
	defer c.tagsMu.Unlock()

	for _, h := range hashes {
		key := strings.ToLower(strings.TrimSpace(h))
		tagMap := make(map[string]bool)
		for _, existing := range c.tags[key] {
			tagMap[existing] = true
		}
		for _, t := range tags {
			trimmed := strings.TrimSpace(t)
			if trimmed != "" {
				tagMap[trimmed] = true
			}
		}

		merged := make([]string, 0, len(tagMap))
		for t := range tagMap {
			merged = append(merged, t)
		}
		c.tags[key] = merged
	}

	return nil
}

// RemoveTags removes tags from specified hashes/gids in the local in-memory tag store.
func (c *Client) RemoveTags(ctx context.Context, hashes []string, tags []string) error {
	if len(hashes) == 0 || len(tags) == 0 {
		return nil
	}

	c.tagsMu.Lock()
	defer c.tagsMu.Unlock()

	removeMap := make(map[string]bool, len(tags))
	for _, t := range tags {
		removeMap[strings.ToLower(strings.TrimSpace(t))] = true
	}

	for _, h := range hashes {
		key := strings.ToLower(strings.TrimSpace(h))
		var remaining []string
		for _, existing := range c.tags[key] {
			if !removeMap[strings.ToLower(existing)] {
				remaining = append(remaining, existing)
			}
		}
		c.tags[key] = remaining
	}

	return nil
}
