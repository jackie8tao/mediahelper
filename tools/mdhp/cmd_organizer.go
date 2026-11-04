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
	orgSourcePath   string
	orgMediaType    string
	orgTitle        string
	orgYear         int
	orgSeason       int
	orgEpisode      int
	orgEpisodeTitle string
	orgResolution   string
)

var organizeCmd = &cobra.Command{
	Use:   "organize [flags] [source_path]",
	Short: "Organize media file(s) into standardized library directory structures",
	Example: `  mdhp organizer organize --movie-dir ./tmp/movies -s ./tmp/src/Inception.2010.1080p.mkv --title "盗梦空间" -y 2010 -r 1080p
  mdhp organizer organize --tv-dir ./tmp/tv -m tv -s ./tmp/src/Frieren.S01E01.mkv --title "葬送的芙莉莲" -y 2023 --season 1 --episode 1`,
	RunE: func(cmd *cobra.Command, args []string) error {
		srcPath := strings.TrimSpace(orgSourcePath)
		if srcPath == "" && len(args) > 0 {
			srcPath = strings.TrimSpace(args[0])
		}
		if srcPath == "" {
			return fmt.Errorf("source path is required (provide as argument or via -s/--source)")
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
			SourcePath:   srcPath,
			MediaType:    orgMediaType,
			Title:        orgTitle,
			Year:         orgYear,
			Season:       orgSeason,
			Episode:      orgEpisode,
			EpisodeTitle: orgEpisodeTitle,
			Resolution:   orgResolution,
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

	organizeCmd.Flags().StringVarP(&orgSourcePath, "source", "s", "", "Source media file or directory path")
	organizeCmd.Flags().StringVarP(&orgMediaType, "media-type", "m", "movie", "Media category (movie, tv, anime)")
	organizeCmd.Flags().StringVar(&orgTitle, "title", "", "Primary media title")
	organizeCmd.Flags().IntVarP(&orgYear, "year", "y", 0, "Release year")
	organizeCmd.Flags().IntVar(&orgSeason, "season", 1, "Season number (for tv/anime)")
	organizeCmd.Flags().IntVar(&orgEpisode, "episode", 1, "Episode number (for tv/anime)")
	organizeCmd.Flags().StringVar(&orgEpisodeTitle, "episode-title", "", "Episode title (optional)")
	organizeCmd.Flags().StringVarP(&orgResolution, "resolution", "r", "", "Video resolution tag (e.g. 1080p, 4K)")

	organizerCmd.AddCommand(organizeCmd)
	rootCmd.AddCommand(organizerCmd)
}
