package downloader

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackie8tao/mediahelper/internal/config"
	"github.com/jackie8tao/mediahelper/internal/pipeline"
)

// Supported downloader component action constants.
const (
	ActionAddTorrent    = "add_torrent"
	ActionDeleteTorrent = "delete_torrent"
	ActionGetTorrents   = "get_torrents"
	ActionAddTags       = "add_tags"
	ActionRemoveTags    = "remove_tags"
	ActionPause         = "pause"
	ActionResume        = "resume"
)

// DownloaderComponent adapts a Downloader implementation to the pipeline.Component interface.
type DownloaderComponent struct {
	dl  Downloader
	cfg config.DownloaderComponentConfig
}

func (c *DownloaderComponent) ID() string                               { return c.cfg.ID }
func (c *DownloaderComponent) Name() string                             { return c.cfg.Name }
func (c *DownloaderComponent) Description() string                      { return c.cfg.Description }
func (c *DownloaderComponent) Downloader() Downloader                   { return c.dl }
func (c *DownloaderComponent) Config() config.DownloaderComponentConfig { return c.cfg }

// Execute dispatches downloader operations within a pipeline step according to the "action" parameter.
func (c *DownloaderComponent) Execute(ctx context.Context, sCtx *pipeline.StepContext) (map[string]any, error) {
	if c.dl == nil {
		return nil, fmt.Errorf("downloader client is not initialized for component '%s'", c.cfg.ID)
	}
	if sCtx == nil {
		return nil, fmt.Errorf("step context cannot be nil for downloader component '%s'", c.cfg.ID)
	}

	action := strings.ToLower(strings.TrimSpace(sCtx.GetStringParam("action", "")))
	if action == "" {
		return nil, fmt.Errorf("missing required parameter 'action' for downloader component '%s'", c.cfg.ID)
	}

	switch action {
	case ActionAddTorrent:
		return c.execAddTorrent(ctx, sCtx)
	case ActionDeleteTorrent:
		return c.execDeleteTorrent(ctx, sCtx)
	case ActionGetTorrents:
		return c.execGetTorrents(ctx, sCtx)
	case ActionAddTags:
		return c.execAddTags(ctx, sCtx)
	case ActionRemoveTags:
		return c.execRemoveTags(ctx, sCtx)
	case ActionPause:
		return c.execPause(ctx, sCtx)
	case ActionResume:
		return c.execResume(ctx, sCtx)
	default:
		return nil, fmt.Errorf("unsupported action '%s' for downloader component '%s'", action, c.cfg.ID)
	}
}

// execAddTorrent handles adding a torrent from a local .torrent file or URL/magnet link.
func (c *DownloaderComponent) execAddTorrent(ctx context.Context, sCtx *pipeline.StepContext) (map[string]any, error) {
	opts := AddTorrentOptions{
		FilePath: strings.TrimSpace(sCtx.GetStringParam("file_path", "")),
		URL:      strings.TrimSpace(sCtx.GetStringParam("url", "")),
		SavePath: strings.TrimSpace(sCtx.GetStringParam("save_path", "")),
		Category: strings.TrimSpace(sCtx.GetStringParam("category", "")),
		Tags:     sCtx.GetStringSliceParam("tags"),
		Paused:   sCtx.GetBoolParam("paused", false),
	}

	if opts.FilePath == "" && opts.URL == "" {
		return nil, fmt.Errorf("action '%s' requires 'file_path' or 'url' parameter", ActionAddTorrent)
	}

	hashes, err := c.dl.AddTorrent(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("downloader '%s' add_torrent failed: %w", c.cfg.ID, err)
	}

	var primaryHash string
	if len(hashes) > 0 {
		primaryHash = hashes[0]
	}

	sourceDesc := opts.FilePath
	if sourceDesc == "" {
		sourceDesc = opts.URL
	}
	sCtx.Logf("Added torrent '%s' via %s (hashes=%v, category=%q, tags=%v)",
		sourceDesc, c.dl.Name(), hashes, opts.Category, opts.Tags)

	return map[string]any{
		"action":    ActionAddTorrent,
		"hash":      primaryHash,
		"hashes":    hashes,
		"file_path": opts.FilePath,
		"url":       opts.URL,
		"save_path": opts.SavePath,
		"category":  opts.Category,
		"tags":      opts.Tags,
		"paused":    opts.Paused,
	}, nil
}

// execDeleteTorrent handles deleting one or more torrents and optionally removing downloaded files.
func (c *DownloaderComponent) execDeleteTorrent(ctx context.Context, sCtx *pipeline.StepContext) (map[string]any, error) {
	hashes := getHashesParam(sCtx)
	if len(hashes) == 0 {
		return nil, fmt.Errorf("action '%s' requires 'hashes' or 'hash' parameter", ActionDeleteTorrent)
	}

	deleteFiles := sCtx.GetBoolParam("delete_files", true)

	if err := c.dl.DeleteTorrent(ctx, hashes, deleteFiles); err != nil {
		return nil, fmt.Errorf("downloader '%s' delete_torrent failed: %w", c.cfg.ID, err)
	}

	sCtx.Logf("Deleted torrent(s) %v via %s (delete_files=%v)", hashes, c.dl.Name(), deleteFiles)

	return map[string]any{
		"action":       ActionDeleteTorrent,
		"hash":         hashes[0],
		"hashes":       hashes,
		"delete_files": deleteFiles,
	}, nil
}

// execGetTorrents retrieves torrents matching hash or filter parameters.
func (c *DownloaderComponent) execGetTorrents(ctx context.Context, sCtx *pipeline.StepContext) (map[string]any, error) {
	opts := GetTorrentsOptions{
		Hashes:   getHashesParam(sCtx),
		Filter:   strings.TrimSpace(sCtx.GetStringParam("filter", "")),
		Category: strings.TrimSpace(sCtx.GetStringParam("category", "")),
		Tag:      strings.TrimSpace(sCtx.GetStringParam("tag", "")),
		Limit:    sCtx.GetIntParam("limit", 0),
	}

	torrents, err := c.dl.GetTorrents(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("downloader '%s' get_torrents failed: %w", c.cfg.ID, err)
	}

	sCtx.Logf("Fetched %d torrent(s) via %s (hashes=%v, filter=%q, category=%q, tag=%q)",
		len(torrents), c.dl.Name(), opts.Hashes, opts.Filter, opts.Category, opts.Tag)

	if len(torrents) > 0 {
		return buildTorrentOutputs(torrents[0], torrents), nil
	}

	return map[string]any{
		"action":   ActionGetTorrents,
		"torrents": []map[string]any{},
		"count":    0,
	}, nil
}

// execAddTags adds one or more tags to the specified torrent hashes.
func (c *DownloaderComponent) execAddTags(ctx context.Context, sCtx *pipeline.StepContext) (map[string]any, error) {
	hashes := getHashesParam(sCtx)
	if len(hashes) == 0 {
		return nil, fmt.Errorf("action '%s' requires 'hashes' or 'hash' parameter", ActionAddTags)
	}

	tags := sCtx.GetStringSliceParam("tags")
	if len(tags) == 0 {
		return nil, fmt.Errorf("action '%s' requires 'tags' parameter", ActionAddTags)
	}

	if err := c.dl.AddTags(ctx, hashes, tags); err != nil {
		return nil, fmt.Errorf("downloader '%s' add_tags failed: %w", c.cfg.ID, err)
	}

	sCtx.Logf("Added tag(s) %v to torrent(s) %v via %s", tags, hashes, c.dl.Name())

	return map[string]any{
		"action": ActionAddTags,
		"hash":   hashes[0],
		"hashes": hashes,
		"tags":   tags,
	}, nil
}

// execRemoveTags removes one or more tags from the specified torrent hashes.
func (c *DownloaderComponent) execRemoveTags(ctx context.Context, sCtx *pipeline.StepContext) (map[string]any, error) {
	hashes := getHashesParam(sCtx)
	if len(hashes) == 0 {
		return nil, fmt.Errorf("action '%s' requires 'hashes' or 'hash' parameter", ActionRemoveTags)
	}

	tags := sCtx.GetStringSliceParam("tags")
	if len(tags) == 0 {
		return nil, fmt.Errorf("action '%s' requires 'tags' parameter", ActionRemoveTags)
	}

	if err := c.dl.RemoveTags(ctx, hashes, tags); err != nil {
		return nil, fmt.Errorf("downloader '%s' remove_tags failed: %w", c.cfg.ID, err)
	}

	sCtx.Logf("Removed tag(s) %v from torrent(s) %v via %s", tags, hashes, c.dl.Name())

	return map[string]any{
		"action": ActionRemoveTags,
		"hash":   hashes[0],
		"hashes": hashes,
		"tags":   tags,
	}, nil
}

// execPause pauses the specified torrent hashes.
func (c *DownloaderComponent) execPause(ctx context.Context, sCtx *pipeline.StepContext) (map[string]any, error) {
	hashes := getHashesParam(sCtx)
	if len(hashes) == 0 {
		return nil, fmt.Errorf("action '%s' requires 'hashes' or 'hash' parameter", ActionPause)
	}

	if err := c.dl.Pause(ctx, hashes); err != nil {
		return nil, fmt.Errorf("downloader '%s' pause failed: %w", c.cfg.ID, err)
	}

	sCtx.Logf("Paused torrent(s) %v via %s", hashes, c.dl.Name())

	return map[string]any{
		"action": ActionPause,
		"hash":   hashes[0],
		"hashes": hashes,
	}, nil
}

// execResume resumes the specified torrent hashes.
func (c *DownloaderComponent) execResume(ctx context.Context, sCtx *pipeline.StepContext) (map[string]any, error) {
	hashes := getHashesParam(sCtx)
	if len(hashes) == 0 {
		return nil, fmt.Errorf("action '%s' requires 'hashes' or 'hash' parameter", ActionResume)
	}

	if err := c.dl.Resume(ctx, hashes); err != nil {
		return nil, fmt.Errorf("downloader '%s' resume failed: %w", c.cfg.ID, err)
	}

	sCtx.Logf("Resumed torrent(s) %v via %s", hashes, c.dl.Name())

	return map[string]any{
		"action": ActionResume,
		"hash":   hashes[0],
		"hashes": hashes,
	}, nil
}

// getHashesParam extracts torrent hashes strictly from step parameters ("hashes" or "hash").
func getHashesParam(sCtx *pipeline.StepContext) []string {
	if hashes := sCtx.GetStringSliceParam("hashes"); len(hashes) > 0 {
		return hashes
	}
	return sCtx.GetStringSliceParam("hash")
}

func torrentToMap(t Torrent) map[string]any {
	return map[string]any{
		"hash":         t.Hash,
		"name":         t.Name,
		"category":     t.Category,
		"tags":         t.Tags,
		"save_path":    t.SavePath,
		"content_path": t.ContentPath,
		"progress":     t.Progress,
		"state":        t.State,
		"size":         t.Size,
		"is_completed": t.IsCompleted,
		"dl_speed":     t.DlSpeed,
		"up_speed":     t.UpSpeed,
		"eta":          t.ETA,
		"added_on":     t.AddedOn,
	}
}

func buildTorrentOutputs(primary Torrent, all []Torrent) map[string]any {
	torrents := make([]map[string]any, 0, len(all))
	for _, t := range all {
		torrents = append(torrents, torrentToMap(t))
	}

	return map[string]any{
		"action":       ActionGetTorrents,
		"torrent":      torrentToMap(primary),
		"torrents":     torrents,
		"count":        len(all),
		"hash":         primary.Hash,
		"name":         primary.Name,
		"category":     primary.Category,
		"tags":         primary.Tags,
		"save_path":    primary.SavePath,
		"content_path": primary.ContentPath,
		"progress":     primary.Progress,
		"state":        primary.State,
		"size":         primary.Size,
		"is_completed": primary.IsCompleted,
	}
}
