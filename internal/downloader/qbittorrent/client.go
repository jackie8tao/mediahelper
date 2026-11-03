package qbittorrent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	qb "github.com/autobrr/go-qbittorrent"
	"github.com/jackie8tao/mediahelper/internal/downloader"
)

// Config holds connection parameters for a qBittorrent client instance.
type Config struct {
	Addr     string `json:"addr"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// Client wraps autobrr/go-qbittorrent and implements downloader.Downloader.
type Client struct {
	cfg      Config
	client   *qb.Client
	mu       sync.Mutex
	loggedIn bool
}

func init() {
	downloader.RegisterDownloaderFactory("qbittorrent", QbittorrentFactory)
}

// QbittorrentFactory initializes a new qBittorrent client instance.
func QbittorrentFactory(cfgVal json.RawMessage) (downloader.Downloader, error) {
	var cfgSt Config
	if err := json.Unmarshal(cfgVal, &cfgSt); err != nil {
		return nil, err
	}

	targetAddr := strings.TrimSpace(cfgSt.Addr)
	if targetAddr == "" {
		return nil, fmt.Errorf("qbittorrent address is required")
	}
	if !strings.HasPrefix(targetAddr, "http://") && !strings.HasPrefix(targetAddr, "https://") {
		targetAddr = "http://" + targetAddr
	}
	cfgSt.Addr = targetAddr

	qbCfg := qb.Config{
		Host:          cfgSt.Addr,
		Username:      cfgSt.Username,
		Password:      cfgSt.Password,
		RetryAttempts: 3,
		RetryDelay:    1,
	}

	client := qb.NewClient(qbCfg)

	return &Client{
		cfg:    cfgSt,
		client: client,
	}, nil
}

// Name returns the downloader name.
func (c *Client) Name() string {
	return "qBittorrent"
}

// ensureLogin performs lazy authentication against the qBittorrent WebUI.
func (c *Client) ensureLogin(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.loggedIn {
		return nil
	}
	if err := c.client.LoginCtx(ctx); err != nil {
		return fmt.Errorf("qbittorrent login failed: %w", err)
	}
	c.loggedIn = true
	return nil
}

// AddTorrent adds a torrent by local file path or URL/magnet link.
func (c *Client) AddTorrent(ctx context.Context, opts downloader.AddTorrentOptions) ([]string, error) {
	if opts.FilePath == "" && opts.URL == "" {
		return nil, fmt.Errorf("either file_path or url is required to add a torrent")
	}

	if err := c.ensureLogin(ctx); err != nil {
		return nil, err
	}

	qbOpts := make(map[string]string)
	if opts.SavePath != "" {
		qbOpts["savepath"] = opts.SavePath
	}
	if opts.Category != "" {
		qbOpts["category"] = opts.Category
	}
	if len(opts.Tags) > 0 {
		qbOpts["tags"] = strings.Join(opts.Tags, ",")
	}
	if opts.Paused {
		qbOpts["paused"] = "true"
	}

	var resp *qb.TorrentAddResponse
	var err error
	if opts.FilePath != "" {
		resp, err = c.client.AddTorrentFromFileCtx(ctx, opts.FilePath, qbOpts)
	} else {
		resp, err = c.client.AddTorrentFromUrlCtx(ctx, opts.URL, qbOpts)
	}
	if err != nil {
		return nil, fmt.Errorf("qbittorrent add torrent failed: %w", err)
	}

	if resp != nil && len(resp.AddedTorrentIds) > 0 {
		return resp.AddedTorrentIds, nil
	}
	return nil, nil
}

// DeleteTorrent removes one or more torrents and optionally deletes their downloaded data files.
func (c *Client) DeleteTorrent(ctx context.Context, hashes []string, deleteFiles bool) error {
	if len(hashes) == 0 {
		return fmt.Errorf("at least one torrent hash is required")
	}
	if err := c.ensureLogin(ctx); err != nil {
		return err
	}
	if err := c.client.DeleteTorrentsCtx(ctx, hashes, deleteFiles); err != nil {
		return fmt.Errorf("qbittorrent delete torrents failed: %w", err)
	}
	return nil
}

// GetTorrents retrieves a list of torrents matching the given filter options.
func (c *Client) GetTorrents(ctx context.Context, opts downloader.GetTorrentsOptions) ([]downloader.Torrent, error) {
	if err := c.ensureLogin(ctx); err != nil {
		return nil, err
	}

	filter := opts.Filter
	if filter == "" {
		filter = string(qb.TorrentFilterAll)
	}

	qbFilter := qb.TorrentFilterOptions{
		Filter:   qb.TorrentFilter(filter),
		Category: opts.Category,
		Tag:      opts.Tag,
		Limit:    opts.Limit,
		Hashes:   opts.Hashes,
	}

	list, err := c.client.GetTorrentsCtx(ctx, qbFilter)
	if err != nil {
		return nil, fmt.Errorf("qbittorrent get torrents failed: %w", err)
	}

	result := make([]downloader.Torrent, 0, len(list))
	for _, item := range list {
		result = append(result, convertTorrent(item))
	}
	return result, nil
}

// AddTags attaches one or more tags to the specified torrents.
func (c *Client) AddTags(ctx context.Context, hashes []string, tags []string) error {
	if len(hashes) == 0 {
		return fmt.Errorf("at least one torrent hash is required")
	}
	if len(tags) == 0 {
		return fmt.Errorf("at least one tag is required")
	}
	if err := c.ensureLogin(ctx); err != nil {
		return err
	}
	if err := c.client.AddTagsCtx(ctx, hashes, strings.Join(tags, ",")); err != nil {
		return fmt.Errorf("qbittorrent add tags failed: %w", err)
	}
	return nil
}

// RemoveTags removes one or more tags from the specified torrents.
func (c *Client) RemoveTags(ctx context.Context, hashes []string, tags []string) error {
	if len(hashes) == 0 {
		return fmt.Errorf("at least one torrent hash is required")
	}
	if len(tags) == 0 {
		return fmt.Errorf("at least one tag is required")
	}
	if err := c.ensureLogin(ctx); err != nil {
		return err
	}
	if err := c.client.RemoveTagsCtx(ctx, hashes, strings.Join(tags, ",")); err != nil {
		return fmt.Errorf("qbittorrent remove tags failed: %w", err)
	}
	return nil
}

// Pause pauses the specified torrents.
func (c *Client) Pause(ctx context.Context, hashes []string) error {
	if len(hashes) == 0 {
		return fmt.Errorf("at least one torrent hash is required")
	}
	if err := c.ensureLogin(ctx); err != nil {
		return err
	}
	if err := c.client.PauseCtx(ctx, hashes); err != nil {
		return fmt.Errorf("qbittorrent pause torrents failed: %w", err)
	}
	return nil
}

// Resume resumes the specified torrents.
func (c *Client) Resume(ctx context.Context, hashes []string) error {
	if len(hashes) == 0 {
		return fmt.Errorf("at least one torrent hash is required")
	}
	if err := c.ensureLogin(ctx); err != nil {
		return err
	}
	if err := c.client.ResumeCtx(ctx, hashes); err != nil {
		return fmt.Errorf("qbittorrent resume torrents failed: %w", err)
	}
	return nil
}

// convertTorrent maps a qBittorrent Torrent struct to the general downloader.Torrent model.
func convertTorrent(t qb.Torrent) downloader.Torrent {
	var tags []string
	if strings.TrimSpace(t.Tags) != "" {
		for _, part := range strings.Split(t.Tags, ",") {
			if trimmed := strings.TrimSpace(part); trimmed != "" {
				tags = append(tags, trimmed)
			}
		}
	}

	return downloader.Torrent{
		Hash:        t.Hash,
		Name:        t.Name,
		Category:    t.Category,
		Tags:        tags,
		SavePath:    t.SavePath,
		ContentPath: t.ContentPath,
		Progress:    t.Progress,
		State:       string(t.State),
		Size:        t.Size,
		IsCompleted: t.Progress >= 1.0 || t.CompletionOn > 0,
		DlSpeed:     t.DlSpeed,
		UpSpeed:     t.UpSpeed,
		ETA:         t.ETA,
		AddedOn:     t.AddedOn,
	}
}
