package qbittorrent

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	qb "github.com/autobrr/go-qbittorrent"
	"github.com/jackie8tao/mediahelper/internal/config"
	"github.com/jackie8tao/mediahelper/internal/downloader"
)

// Client wraps autobrr/go-qbittorrent and implements downloader.Downloader.
type Client struct {
	cfg        config.QBittorrentConfig
	client     *qb.Client
	pathMapper *downloader.PathMapper
}

func init() {
	downloader.RegisterFactory("qbittorrent", func(cfg config.DownloaderInstanceConfig) (downloader.Downloader, error) {
		return NewClient(cfg.ToQBittorrentConfig())
	})
}

// NewClient initializes a new qBittorrent client instance.
func NewClient(cfg config.QBittorrentConfig) (*Client, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("qbittorrent URL is required")
	}

	timeout := 15
	if cfg.PollInterval > 0 && cfg.PollInterval < timeout {
		timeout = cfg.PollInterval
	}

	qbCfg := qb.Config{
		Host:          cfg.URL,
		Username:      cfg.Username,
		Password:      cfg.Password,
		Timeout:       timeout,
		RetryAttempts: 3,
		RetryDelay:    1,
	}

	client := qb.NewClient(qbCfg)
	pathMapper := downloader.NewPathMapper(cfg.PathMappings)

	return &Client{
		cfg:        cfg,
		client:     client,
		pathMapper: pathMapper,
	}, nil
}

// Name returns the downloader name.
func (c *Client) Name() string {
	return "qBittorrent"
}

// TestConnection verifies connectivity and authentication with qBittorrent WebUI.
func (c *Client) TestConnection(ctx context.Context) error {
	// Attempt login
	if err := c.client.LoginCtx(ctx); err != nil {
		return fmt.Errorf("qbittorrent login failed: %w", err)
	}

	// Fetch version to confirm API responsiveness
	version, err := c.client.GetAppVersionCtx(ctx)
	if err != nil {
		return fmt.Errorf("qbittorrent get version failed: %w", err)
	}

	slog.Info("qBittorrent connected successfully", "version", version, "url", c.cfg.URL)
	return nil
}

// GetAllTorrents returns all torrents matching state filter and category.
func (c *Client) GetAllTorrents(ctx context.Context, filter string, category string) ([]downloader.Torrent, error) {
	opts := qb.TorrentFilterOptions{}
	if filter != "" && filter != "all" {
		opts.Filter = qb.ParseTorrentFilter(filter)
	} else {
		opts.Filter = qb.TorrentFilterAll
	}
	if category != "" {
		opts.Category = category
	}

	torrents, err := c.client.GetTorrentsCtx(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("get all torrents error: %w", err)
	}

	result := make([]downloader.Torrent, 0, len(torrents))
	for _, t := range torrents {
		result = append(result, c.convertTorrent(t))
	}

	return result, nil
}

// GetCompletedTorrents returns completed torrents matching the specified categories.
func (c *Client) GetCompletedTorrents(ctx context.Context, categories []string) ([]downloader.Torrent, error) {
	// Query completed torrents
	opts := qb.TorrentFilterOptions{
		Filter: qb.TorrentFilterCompleted,
	}

	torrents, err := c.client.GetTorrentsCtx(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("get completed torrents error: %w", err)
	}

	// Build allowed categories lookup set
	allowedCategories := make(map[string]struct{})
	for _, cat := range categories {
		cat = strings.TrimSpace(cat)
		if cat != "" {
			allowedCategories[cat] = struct{}{}
		}
	}

	var result []downloader.Torrent
	for _, t := range torrents {
		// Category filtering if categories specified
		if len(allowedCategories) > 0 {
			if _, ok := allowedCategories[t.Category]; !ok {
				continue
			}
		}

		result = append(result, c.convertTorrent(t))
	}

	return result, nil
}

// convertTorrent maps qBittorrent entity to the domain Torrent model.
func (c *Client) convertTorrent(t qb.Torrent) downloader.Torrent {
	tags := parseTags(t.Tags)
	savePath := c.pathMapper.ToLocal(t.SavePath)
	contentPath := c.pathMapper.ToLocal(t.ContentPath)
	isCompleted := t.Progress >= 1.0 || t.CompletionOn > 0

	return downloader.Torrent{
		Hash:        t.Hash,
		Name:        t.Name,
		Category:    t.Category,
		Tags:        tags,
		SavePath:    savePath,
		ContentPath: contentPath,
		Progress:    t.Progress,
		State:       string(t.State),
		Size:        t.Size,
		IsCompleted: isCompleted,
		DlSpeed:     t.DlSpeed,
		UpSpeed:     t.UpSpeed,
		ETA:         t.ETA,
		AddedOn:     t.AddedOn,
	}
}

// GetTorrentFiles retrieves file information for a specific torrent.
func (c *Client) GetTorrentFiles(ctx context.Context, hash string) ([]downloader.TorrentFile, error) {
	files, err := c.client.GetFilesInformationCtx(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("get torrent files error (hash=%s): %w", hash, err)
	}

	if files == nil {
		return nil, nil
	}

	result := make([]downloader.TorrentFile, len(*files))
	for i, f := range *files {
		result[i] = downloader.TorrentFile{
			Index:    f.Index,
			Name:     f.Name,
			Size:     f.Size,
			Progress: float64(f.Progress),
		}
	}

	return result, nil
}

// AddTags applies tags to the specified torrent hashes.
func (c *Client) AddTags(ctx context.Context, hashes []string, tags []string) error {
	if len(hashes) == 0 || len(tags) == 0 {
		return nil
	}

	tagStr := strings.Join(tags, ",")
	if err := c.client.AddTagsCtx(ctx, hashes, tagStr); err != nil {
		return fmt.Errorf("add tags failed: %w", err)
	}

	return nil
}

// RemoveTags removes tags from the specified torrent hashes.
func (c *Client) RemoveTags(ctx context.Context, hashes []string, tags []string) error {
	if len(hashes) == 0 || len(tags) == 0 {
		return nil
	}

	tagStr := strings.Join(tags, ",")
	if err := c.client.RemoveTagsCtx(ctx, hashes, tagStr); err != nil {
		return fmt.Errorf("remove tags failed: %w", err)
	}

	return nil
}

// parseTags parses a comma-separated tag string into a string slice.
func parseTags(tagStr string) []string {
	if strings.TrimSpace(tagStr) == "" {
		return []string{}
	}

	parts := strings.Split(tagStr, ",")
	var tags []string
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t != "" {
			tags = append(tags, t)
		}
	}
	return tags
}
