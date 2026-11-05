package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackie8tao/mediahelper/internal/config"
	"github.com/jackie8tao/mediahelper/internal/organizer"
	"github.com/jackie8tao/mediahelper/internal/organizer/local"
	"github.com/spf13/cobra"
)

var (
	orgType        string
	orgMovieDir    string
	orgTVDir       string
	orgAnimeDir    string
	orgLinkMode    string
	orgMinFileSize int64
)

var organizerCmd = &cobra.Command{
	Use:   "organizer",
	Short: "Media library file organization operations",
	Long:  `Test and debug the organizer module with custom directory paths and linking modes.`,
}

var (
	orgMediaType  string
	orgTitle      string
	orgYear       int
	orgResolution string
	orgFilesJSON  string
)

var organizeCmd = &cobra.Command{
	Use:   "organize [flags]",
	Short: "Organize media file(s) into standardized library directory structures",
	Example: `  mdhp organizer organize --movie-dir ./tests/library/movies -m movie --title "盗梦空间" -y 2010 -r 1080p --files '[{"file":"./tests/src/Inception.2010.1080p.mkv"}]'
  mdhp organizer organize --tv-dir ./tests/library/tv -m tv --title "葬送的芙莉莲" -y 2023 --files '[{"file":"./tests/src/01.mkv","season":1,"episode":1},{"file":"./tests/src/02.mkv","season":1,"episode":2}]'`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if strings.TrimSpace(orgFilesJSON) == "" {
			return fmt.Errorf("--files JSON array is required")
		}

		var parsedFiles []organizer.EpisodeFileItem
		if err := json.Unmarshal([]byte(orgFilesJSON), &parsedFiles); err != nil {
			return fmt.Errorf("unmarshal --files JSON failed: %w", err)
		}
		if len(parsedFiles) == 0 {
			return fmt.Errorf("--files array cannot be empty")
		}

		detailBytes, err := json.Marshal(local.Config{
			MovieDir:    orgMovieDir,
			TVDir:       orgTVDir,
			AnimeDir:    orgAnimeDir,
			LinkMode:    organizer.LinkMode(orgLinkMode),
			MinFileSize: orgMinFileSize,
		})
		if err != nil {
			return fmt.Errorf("marshal organizer config failed: %w", err)
		}

		org, err := organizer.CreateOrganizer(config.OrganizerComponentConfig{
			Type:   orgType,
			Detail: detailBytes,
		})
		if err != nil {
			return fmt.Errorf("create organizer failed: %w", err)
		}

		task := organizer.OrganizeTask{
			MediaType:  orgMediaType,
			Title:      orgTitle,
			Year:       orgYear,
			Resolution: orgResolution,
			Files:      parsedFiles,
		}

		res, err := org.Organize(context.Background(), task)
		if err != nil {
			return fmt.Errorf("organize failed: %w", err)
		}

		out, err := json.MarshalIndent(res, "", "  ")
		if err != nil {
			return fmt.Errorf("format result failed: %w", err)
		}
		fmt.Println(string(out))
		return nil
	},
}

func init() {
	organizerCmd.PersistentFlags().StringVarP(&orgType, "type", "t", "local", "Organizer provider type (e.g. local)")
	organizerCmd.PersistentFlags().StringVar(&orgMovieDir, "movie-dir", "", "Target directory for movies")
	organizerCmd.PersistentFlags().StringVar(&orgTVDir, "tv-dir", "", "Target directory for TV shows")
	organizerCmd.PersistentFlags().StringVar(&orgAnimeDir, "anime-dir", "", "Target directory for anime")
	organizerCmd.PersistentFlags().StringVarP(&orgLinkMode, "link-mode", "l", "hardlink", "File link mode (hardlink, copy, move, symlink)")
	organizerCmd.PersistentFlags().Int64Var(&orgMinFileSize, "min-size", 0, "Minimum eligible video file size in bytes")

	organizeCmd.Flags().StringVarP(&orgMediaType, "media-type", "m", "movie", "Media category (movie, tv, anime)")
	organizeCmd.Flags().StringVar(&orgTitle, "title", "", "Primary media title")
	organizeCmd.Flags().IntVarP(&orgYear, "year", "y", 0, "Release year")
	organizeCmd.Flags().StringVarP(&orgResolution, "resolution", "r", "", "Video resolution tag (e.g. 1080p, 4K)")
	organizeCmd.Flags().StringVarP(&orgFilesJSON, "files", "f", "", "JSON array of EpisodeFileItem objects to organize")

	organizerCmd.AddCommand(organizeCmd)
	rootCmd.AddCommand(organizerCmd)
}
