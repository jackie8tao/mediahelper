package douban

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jackie8tao/mediahelper/internal/scraper"
	"github.com/jackie8tao/mediahelper/pkg/utils/douban"
)

// Config aliases douban.Config for scraper configuration.
type Config = douban.Config

// Client implements scraper.Scraper for Douban Movie / TV.
type Client struct {
	cfg        Config
	httpClient *http.Client
	apiClient  *douban.Client
}

func init() {
	scraper.RegisterFactory("douban", DoubanFactory)
}

// DoubanFactory initializes a new Douban scraper client instance.
func DoubanFactory(cfgVal json.RawMessage) (scraper.Scraper, error) {
	var cfgSt Config
	if len(cfgVal) > 0 {
		if err := json.Unmarshal(cfgVal, &cfgSt); err != nil {
			return nil, err
		}
	}
	return NewClient(cfgSt)
}

// NewClient creates a new Douban scraper Client.
func NewClient(cfg Config) (*Client, error) {
	apiClient, err := douban.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	return &Client{
		cfg:       cfg,
		apiClient: apiClient,
	}, nil
}

// Name returns the scraper provider name.
func (c *Client) Name() string {
	return "Douban"
}

func (c *Client) ensureAPIClient() error {
	if c.apiClient != nil {
		return nil
	}

	cfg := c.cfg
	if c.httpClient != nil {
		cfg.HTTPClient = c.httpClient
	}

	client, err := douban.NewClient(cfg)
	if err != nil {
		return err
	}
	c.apiClient = client
	return nil
}

// Scrape retrieves metadata from Douban and writes NFO files and images into task.TargetDir.
func (c *Client) Scrape(ctx context.Context, task scraper.ScrapeTask) (*scraper.ScrapeResult, error) {
	result := &scraper.ScrapeResult{
		TargetDir: task.TargetDir,
	}

	if err := c.ensureAPIClient(); err != nil {
		result.Error = err.Error()
		return result, err
	}

	targetDir := strings.TrimSpace(task.TargetDir)
	if targetDir == "" {
		err := fmt.Errorf("target_dir is required")
		result.Error = err.Error()
		return result, err
	}

	dirInfo, err := os.Stat(targetDir)
	if err != nil || !dirInfo.IsDir() {
		err := fmt.Errorf("target_dir '%s' does not exist or is not a directory", targetDir)
		result.Error = err.Error()
		return result, err
	}

	doubanID := strings.TrimSpace(task.DoubanID)
	if doubanID == "" && task.TMDBID > 0 {
		doubanID = strconv.Itoa(task.TMDBID)
	}
	if doubanID == "" {
		err := fmt.Errorf("douban_id is required")
		result.Error = err.Error()
		return result, err
	}

	mediaType := strings.ToLower(strings.TrimSpace(task.MediaType))
	if mediaType == "" {
		mediaType = "movie"
	}

	saveNFO := task.SaveNFO == nil || *task.SaveNFO
	saveImages := task.SaveImages == nil || *task.SaveImages

	var (
		metadata  *scraper.MediaMetadata
		scrapeErr error
	)
	if mediaType == "tv" || mediaType == "anime" {
		metadata, scrapeErr = c.scrapeTV(ctx, doubanID)
	} else {
		metadata, scrapeErr = c.scrapeMovie(ctx, doubanID)
	}

	if scrapeErr != nil {
		result.Error = scrapeErr.Error()
		return result, scrapeErr
	}

	result.Metadata = metadata

	// 1. Write NFO XML
	if saveNFO {
		if mediaType == "tv" || mediaType == "anime" {
			nfoPath := filepath.Join(targetDir, "tvshow.nfo")
			if nfoBytes, nfoErr := buildTVShowNFO(metadata); nfoErr == nil {
				if writeErr := os.WriteFile(nfoPath, nfoBytes, 0644); writeErr == nil {
					result.CreatedFiles = append(result.CreatedFiles, nfoPath)
				}
			}
		} else {
			nfoPath := filepath.Join(targetDir, "movie.nfo")
			if nfoBytes, nfoErr := buildMovieNFO(metadata); nfoErr == nil {
				if writeErr := os.WriteFile(nfoPath, nfoBytes, 0644); writeErr == nil {
					result.CreatedFiles = append(result.CreatedFiles, nfoPath)
				}
			}
		}
	}

	// 2. Download Images
	if saveImages {
		if metadata.PosterURL != "" {
			posterPath := filepath.Join(targetDir, "poster.jpg")
			if dlErr := c.apiClient.DownloadImage(ctx, metadata.PosterURL, posterPath); dlErr == nil {
				result.CreatedFiles = append(result.CreatedFiles, posterPath)
			} else {
				slog.Warn("Download Douban poster failed", "url", metadata.PosterURL, "error", dlErr)
			}
		}
		if metadata.BackdropURL != "" {
			fanartPath := filepath.Join(targetDir, "fanart.jpg")
			if dlErr := c.apiClient.DownloadImage(ctx, metadata.BackdropURL, fanartPath); dlErr == nil {
				result.CreatedFiles = append(result.CreatedFiles, fanartPath)
			} else {
				slog.Warn("Download Douban backdrop failed", "url", metadata.BackdropURL, "error", dlErr)
			}
		}
	}

	result.Success = true
	return result, nil
}

func (c *Client) scrapeMovie(ctx context.Context, doubanID string) (*scraper.MediaMetadata, error) {
	details, err := c.apiClient.GetMovie(ctx, doubanID)
	if err != nil {
		return nil, fmt.Errorf("fetch douban movie details (id=%s) error: %w", doubanID, err)
	}

	year, _ := strconv.Atoi(details.Year)

	meta := &scraper.MediaMetadata{
		DoubanID:      details.ID,
		MediaType:     "movie",
		Title:         details.Title,
		OriginalTitle: details.OriginalTitle,
		SortTitle:     details.Title,
		Year:          year,
		Plot:          details.Intro,
		Rating:        details.Rating.Value,
		Votes:         details.Rating.Count,
		Genres:        details.Genres,
		PosterURL:     details.PosterURL(),
	}

	for _, d := range details.Directors {
		meta.Directors = append(meta.Directors, d.Name)
	}
	for _, a := range details.Actors {
		thumb := a.Avatar.Large
		if thumb == "" {
			thumb = a.Avatar.Normal
		}
		meta.Actors = append(meta.Actors, scraper.Actor{
			Name:  a.Name,
			Role:  a.Character,
			Thumb: thumb,
		})
	}

	// Fetch backdrop from subject photos
	if photos, pErr := c.apiClient.GetPhotos(ctx, doubanID, "movie"); pErr == nil && len(photos.Photos) > 0 {
		meta.BackdropURL = photos.Photos[0].Image.Large.URL
	}

	return meta, nil
}

func (c *Client) scrapeTV(ctx context.Context, doubanID string) (*scraper.MediaMetadata, error) {
	details, err := c.apiClient.GetTV(ctx, doubanID)
	if err != nil {
		return nil, fmt.Errorf("fetch douban tv details (id=%s) error: %w", doubanID, err)
	}

	year, _ := strconv.Atoi(details.Year)

	meta := &scraper.MediaMetadata{
		DoubanID:      details.ID,
		MediaType:     "tv",
		Title:         details.Title,
		OriginalTitle: details.OriginalTitle,
		SortTitle:     details.Title,
		Year:          year,
		Plot:          details.Intro,
		Rating:        details.Rating.Value,
		Votes:         details.Rating.Count,
		Genres:        details.Genres,
		PosterURL:     details.PosterURL(),
	}

	for _, d := range details.Directors {
		meta.Directors = append(meta.Directors, d.Name)
	}
	for _, a := range details.Actors {
		thumb := a.Avatar.Large
		if thumb == "" {
			thumb = a.Avatar.Normal
		}
		meta.Actors = append(meta.Actors, scraper.Actor{
			Name:  a.Name,
			Role:  a.Character,
			Thumb: thumb,
		})
	}

	// Fetch backdrop from subject photos
	if photos, pErr := c.apiClient.GetPhotos(ctx, doubanID, "tv"); pErr == nil && len(photos.Photos) > 0 {
		meta.BackdropURL = photos.Photos[0].Image.Large.URL
	}

	return meta, nil
}
