package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	qbt "github.com/autobrr/go-qbittorrent"
	"github.com/spf13/cobra"
)

func init() {
	torrentCmd.AddCommand(torrentAddCmd)
	torrentAddCmd.Flags().StringVarP(&torrentAddFile, "file", "f", "", "Path to the local .torrent file")
	torrentAddCmd.Flags().StringVarP(&torrentAddSavePath, "savepath", "s", "", "Download/Save directory for the torrent data")
	torrentAddCmd.Flags().StringVarP(&torrentAddCategory, "category", "C", "", "Category to assign (e.g. movie, tv)")
	torrentAddCmd.Flags().StringVarP(&torrentAddTags, "tag", "t", "", "Tag(s) to assign (comma-separated)")
	torrentAddCmd.Flags().BoolVar(&torrentAddPaused, "paused", false, "Add torrent in paused/stopped state")

	torrentCmd.AddCommand(torrentListCmd)
	torrentListCmd.Flags().StringVarP(&torrentListFilter, "filter", "f", "all", "Filter status (all, completed, downloading, paused, active, inactive)")
	torrentListCmd.Flags().StringVarP(&torrentListCategory, "category", "C", "", "Filter by category (e.g. movie, tv)")
	torrentListCmd.Flags().StringVarP(&torrentListTag, "tag", "t", "", "Filter by tag (e.g. MediaHelper:Done)")
	torrentListCmd.Flags().IntVarP(&torrentListLimit, "limit", "n", 0, "Limit maximum number of torrents returned (0 for unlimited)")
	torrentListCmd.Flags().BoolVar(&torrentListJSON, "json", false, "Output raw JSON response")

	torrentCmd.AddCommand(torrentResumeCmd)
	torrentResumeCmd.Flags().StringVar(&torrentResumeHash, "hash", "", "Torrent hash (required)")
	_ = torrentResumeCmd.MarkFlagRequired("hash")

	torrentCmd.AddCommand(torrentPauseCmd)
	torrentPauseCmd.Flags().StringVar(&torrentPauseHash, "hash", "", "Torrent hash (required)")
	_ = torrentPauseCmd.MarkFlagRequired("hash")
}

var torrentCmd = &cobra.Command{
	Use:   "torrent",
	Short: "Torrent management operations",
}

//--------------------------------------------------------------------------------------

var (
	torrentAddFile     string
	torrentAddSavePath string
	torrentAddCategory string
	torrentAddTags     string
	torrentAddPaused   bool
)

var torrentAddCmd = &cobra.Command{
	Use:   "add [flags] [torrent_file]",
	Short: "Add a local .torrent file to qBittorrent",
	Example: `  qbt torrent add -f /path/to/movie.torrent -C movie
  qbt torrent add /path/to/movie.torrent --savepath /downloads/movies --tag "MediaHelper"
  qbt torrent add /path/to/movie.torrent --paused`,
	RunE: func(cmd *cobra.Command, args []string) error {
		targetFile := torrentAddFile
		if targetFile == "" && len(args) > 0 {
			targetFile = args[0]
		}
		if targetFile == "" {
			return fmt.Errorf("torrent file path is required (provide as argument or via -f/--file)")
		}

		// Verify local file exists
		fileInfo, err := os.Stat(targetFile)
		if err != nil {
			return fmt.Errorf("failed to access torrent file %q: %w", targetFile, err)
		}
		if fileInfo.IsDir() {
			return fmt.Errorf("%q is a directory, not a .torrent file", targetFile)
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		cli, err := newCtxClient(ctx)
		if err != nil {
			return fmt.Errorf("failed to create qbittorrent client: %w", err)
		}

		// Prepare options
		options := make(map[string]string)
		if torrentAddSavePath != "" {
			options["savepath"] = torrentAddSavePath
		}
		if torrentAddCategory != "" {
			options["category"] = torrentAddCategory
		}
		if torrentAddTags != "" {
			options["tags"] = torrentAddTags
		}
		if torrentAddPaused {
			options["paused"] = "true"
		}

		resp, err := cli.AddTorrentFromFileCtx(ctx, targetFile, options)
		if err != nil {
			return fmt.Errorf("failed to add torrent: %w", err)
		}
		fmt.Printf("Torrent added successfully: %s\n", targetFile)
		if resp != nil {
			if len(resp.AddedTorrentIds) > 0 {
				fmt.Printf("  • Added Torrent Hash: %s\n", strings.Join(resp.AddedTorrentIds, ", "))
			}
			if resp.FailureCount > 0 {
				fmt.Printf("  • Failures: %d\n", resp.FailureCount)
			}
		}
		return nil
	},
}

//--------------------------------------------------------------------------------------

var (
	torrentListFilter   string
	torrentListCategory string
	torrentListTag      string
	torrentListLimit    int
	torrentListJSON     bool
)

var torrentListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List torrents with optional filters",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		cli, err := newCtxClient(ctx)
		if err != nil {
			return fmt.Errorf("failed to create qbittorrent client: %w", err)
		}

		filterOpts := qbt.TorrentFilterOptions{
			Filter:   qbt.TorrentFilter(torrentListFilter),
			Category: torrentListCategory,
			Tag:      torrentListTag,
			Limit:    torrentListLimit,
		}

		torrents, err := cli.GetTorrentsCtx(ctx, filterOpts)
		if err != nil {
			return fmt.Errorf("failed to get torrents: %w", err)
		}

		if torrentListJSON {
			data, err := json.MarshalIndent(torrents, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf(
			"Found %d torrent(s) (Filter: %s, Category: %q, Tag: %q):\n\n",
			len(torrents), torrentListFilter, torrentListCategory, torrentListTag,
		)
		for i, t := range torrents {
			fmt.Printf("[%d] %s\n", i+1, t.Name)
			fmt.Printf("    Hash:        %s\n", t.Hash)
			fmt.Printf("    State:       %s (Progress: %.1f%%)\n", t.State, t.Progress*100)
			fmt.Printf("    Size:        %s\n", formatBytes(t.Size))
			fmt.Printf("    Category:    %s\n", defaultStr(t.Category, "(none)"))
			fmt.Printf("    Tags:        %s\n", defaultStr(t.Tags, "(none)"))
			fmt.Printf("    SavePath:    %s\n", t.SavePath)
			fmt.Printf("    DownloadPath:%s\n", t.DownloadPath)
			fmt.Printf("    ContentPath: %s\n", t.ContentPath)
			fmt.Println()
		}
		return nil
	},
}

//--------------------------------------------------------------------------------------

var torrentPauseHash string

var torrentPauseCmd = &cobra.Command{
	Use:   "pause",
	Short: "Pause a specific torrent",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		cli, err := newCtxClient(ctx)
		if err != nil {
			return fmt.Errorf("failed to create qbittorrent client: %w", err)
		}

		if err := cli.PauseCtx(ctx, []string{torrentPauseHash}); err != nil {
			return fmt.Errorf("failed to pause torrent: %w", err)
		}
		fmt.Printf("Torrent [%s] paused successfully\n", torrentPauseHash)
		return nil
	},
}

//--------------------------------------------------------------------------------------

var torrentResumeHash string

var torrentResumeCmd = &cobra.Command{
	Use:   "resume",
	Short: "Resume a specific torrent",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		cli, err := newCtxClient(ctx)
		if err != nil {
			return fmt.Errorf("failed to create qbittorrent client: %w", err)
		}

		if err := cli.ResumeCtx(ctx, []string{torrentResumeHash}); err != nil {
			return fmt.Errorf("failed to resume torrent: %w", err)
		}
		fmt.Printf("Torrent [%s] resumed successfully\n", torrentResumeHash)
		return nil
	},
}
