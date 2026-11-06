package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackie8tao/mediahelper/internal/config"
	"github.com/jackie8tao/mediahelper/internal/scraper"
	_ "github.com/jackie8tao/mediahelper/internal/scraper/douban"
	_ "github.com/jackie8tao/mediahelper/internal/scraper/tmdb"
	"github.com/spf13/cobra"
)

var (
	scraperType       string
	scraperAPIKey     string
	scraperBaseURL    string
	scraperProxy      string
	scraperLanguage   string
	scrapeDir         string
	scrapeTitle       string
	scrapeYear        int
	scrapeMediaType   string
	scrapeTMDBID      int
	scrapeDoubanID    string
	scrapeNoNFO       bool
	scrapeNoImages    bool
)

var scraperCmd = &cobra.Command{
	Use:   "scraper",
	Short: "Media metadata scraping operations",
	Long:  `Scrape rich metadata, posters, and NFO files directly into local media directories.`,
}

var scrapeCmd = &cobra.Command{
	Use:   "scrape [flags]",
	Short: "Scrape and enrich a local media directory with NFO files and artwork",
	Example: `  mdhp scraper scrape -k <api_key> -d "/library/movies/Inception (2010)" --tmdb-id 27205
  mdhp scraper scrape -t douban -d "/library/movies/The Shawshank Redemption" --douban-id 1292052
  mdhp scraper scrape -k <api_key> -d "/library/tv/Breaking Bad (2008)" -m tv --tmdb-id 1396`,
	RunE: func(cmd *cobra.Command, args []string) error {
		targetDir := strings.TrimSpace(scrapeDir)
		if targetDir == "" {
			return fmt.Errorf("--dir is required")
		}

		sProvider := strings.ToLower(strings.TrimSpace(scraperType))
		if sProvider == "douban" {
			if strings.TrimSpace(scrapeDoubanID) == "" && scrapeTMDBID > 0 {
				scrapeDoubanID = fmt.Sprintf("%d", scrapeTMDBID)
			}
			if strings.TrimSpace(scrapeDoubanID) == "" {
				return fmt.Errorf("--douban-id is required for douban scraper")
			}
		} else {
			if scrapeTMDBID <= 0 {
				return fmt.Errorf("--tmdb-id is required for tmdb scraper")
			}
		}

		cfgMap := map[string]any{
			"api_key":   scraperAPIKey,
			"base_url":  scraperBaseURL,
			"proxy":     scraperProxy,
			"language":  scraperLanguage,
		}

		detailBytes, err := json.Marshal(cfgMap)
		if err != nil {
			return fmt.Errorf("marshal scraper config error: %w", err)
		}

		sc, err := scraper.CreateScraper(config.ScraperComponentConfig{
			Type:   scraperType,
			Detail: detailBytes,
		})
		if err != nil {
			return fmt.Errorf("create scraper failed: %w", err)
		}

		saveNFO := !scrapeNoNFO
		saveImages := !scrapeNoImages

		task := scraper.ScrapeTask{
			TargetDir:  targetDir,
			Title:      scrapeTitle,
			Year:       scrapeYear,
			MediaType:  scrapeMediaType,
			TMDBID:     scrapeTMDBID,
			DoubanID:   scrapeDoubanID,
			Language:   scraperLanguage,
			SaveNFO:    &saveNFO,
			SaveImages: &saveImages,
		}

		res, err := sc.Scrape(context.Background(), task)
		if err != nil {
			return fmt.Errorf("scraping failed: %w", err)
		}

		out, err := json.MarshalIndent(res, "", "  ")
		if err != nil {
			return fmt.Errorf("format result error: %w", err)
		}
		fmt.Println(string(out))
		return nil
	},
}

func init() {
	scraperCmd.PersistentFlags().StringVarP(&scraperType, "type", "t", "tmdb", "Scraper provider type (e.g. tmdb)")
	scraperCmd.PersistentFlags().StringVarP(&scraperAPIKey, "apikey", "k", "", "TMDB API key")
	scraperCmd.PersistentFlags().StringVarP(&scraperBaseURL, "baseurl", "u", "", "API base URL")
	scraperCmd.PersistentFlags().StringVar(&scraperProxy, "proxy", "", "HTTP / SOCKS proxy URL")
	scraperCmd.PersistentFlags().StringVarP(&scraperLanguage, "lang", "l", "zh-CN", "Metadata language (e.g. zh-CN)")

	scrapeCmd.Flags().StringVarP(&scrapeDir, "dir", "d", "", "Target media directory containing media files (required)")
	scrapeCmd.Flags().StringVar(&scrapeTitle, "title", "", "Explicit media title (optional, inferred from directory name if empty)")
	scrapeCmd.Flags().IntVarP(&scrapeYear, "year", "y", 0, "Explicit release year (optional)")
	scrapeCmd.Flags().StringVarP(&scrapeMediaType, "media-type", "m", "", "Media type: movie / tv / anime (optional)")
	scrapeCmd.Flags().IntVarP(&scrapeTMDBID, "tmdb-id", "i", 0, "TMDB unique ID (required when type=tmdb)")
	scrapeCmd.Flags().StringVar(&scrapeDoubanID, "douban-id", "", "Douban Subject ID (required when type=douban)")
	scrapeCmd.Flags().BoolVar(&scrapeNoNFO, "no-nfo", false, "Disable writing .nfo XML files")
	scrapeCmd.Flags().BoolVar(&scrapeNoImages, "no-images", false, "Disable downloading poster and backdrop images")

	scraperCmd.AddCommand(scrapeCmd)
	rootCmd.AddCommand(scraperCmd)
}

