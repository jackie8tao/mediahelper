package tmdb

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/jackie8tao/mediahelper/internal/scraper"
	"github.com/jackie8tao/mediahelper/pkg/utils/tmdb"
)

var (
	titleYearRegex = regexp.MustCompile(`^(.*?)\s*[\(\[]?((?:19|20)\d{2})[\)\]]?$`)
	seasonDirRegex = regexp.MustCompile(`(?i)^Season\s*(\d+)$`)
	episodeRegex   = regexp.MustCompile(`(?i)S(\d+)E(\d+)`)
)

// Config aliases tmdb.Config for scraper configuration.
type Config = tmdb.Config

// Client implements scraper.Scraper for TheMovieDatabase (TMDB).
type Client struct {
	cfg        Config
	httpClient *http.Client
	apiClient  *tmdb.Client
}

func init() {
	scraper.RegisterFactory("tmdb", TMDBFactory)
}

// TMDBFactory initializes a new TMDB scraper client instance.
func TMDBFactory(cfgVal json.RawMessage) (scraper.Scraper, error) {
	var cfgSt Config
	if err := json.Unmarshal(cfgVal, &cfgSt); err != nil {
		return nil, err
	}
	return NewClient(cfgSt)
}

// NewClient creates a new TMDB scraper Client with the given configuration.
func NewClient(cfg Config) (*Client, error) {
	apiClient, err := tmdb.NewClient(cfg)
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
	return "TMDB"
}

func (c *Client) ensureAPIClient() error {
	if c.apiClient != nil {
		return nil
	}

	cfg := c.cfg
	if c.httpClient != nil {
		cfg.HTTPClient = c.httpClient
	}

	client, err := tmdb.NewClient(cfg)
	if err != nil {
		return err
	}
	c.apiClient = client
	return nil
}

// Scrape retrieves metadata from TMDB and writes NFO files and images into task.TargetDir.
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

	// Infer title and year from target directory name if not provided
	title := strings.TrimSpace(task.Title)
	year := task.Year
	if title == "" {
		baseName := filepath.Base(targetDir)
		if m := titleYearRegex.FindStringSubmatch(baseName); len(m) == 3 {
			title = strings.TrimSpace(m[1])
			if year <= 0 {
				year, _ = strconv.Atoi(m[2])
			}
		} else {
			title = baseName
		}
	}

	// Detect media type if omitted
	mediaType := strings.ToLower(strings.TrimSpace(task.MediaType))
	if mediaType == "" {
		mediaType = detectMediaType(targetDir)
	}

	lang := strings.TrimSpace(task.Language)
	if lang == "" {
		lang = c.apiClient.Language()
	}

	saveNFO := task.SaveNFO == nil || *task.SaveNFO
	saveImages := task.SaveImages == nil || *task.SaveImages

	var metadata *scraper.MediaMetadata
	if mediaType == "tv" || mediaType == "anime" {
		metadata, err = c.scrapeTV(ctx, title, year, task.TMDBID, lang, targetDir)
	} else {
		metadata, err = c.scrapeMovie(ctx, title, year, task.TMDBID, lang)
	}

	if err != nil {
		result.Error = err.Error()
		return result, err
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
			// Write episode NFOs if matching video files exist
			c.writeEpisodeNFOs(targetDir, metadata, result)
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
				slog.Warn("Download poster failed", "url", metadata.PosterURL, "error", dlErr)
			}
		}
		if metadata.BackdropURL != "" {
			fanartPath := filepath.Join(targetDir, "fanart.jpg")
			if dlErr := c.apiClient.DownloadImage(ctx, metadata.BackdropURL, fanartPath); dlErr == nil {
				result.CreatedFiles = append(result.CreatedFiles, fanartPath)
			} else {
				slog.Warn("Download backdrop failed", "url", metadata.BackdropURL, "error", dlErr)
			}
		}
	}

	result.Success = true
	return result, nil
}

func (c *Client) scrapeMovie(ctx context.Context, title string, year int, tmdbID int, lang string) (*scraper.MediaMetadata, error) {
	if tmdbID <= 0 {
		res, err := c.apiClient.SearchMovie(ctx, title, year, lang)
		if err != nil {
			return nil, err
		}
		if len(res.Results) == 0 {
			// Fallback: search without year if year was set
			if year > 0 {
				res, _ = c.apiClient.SearchMovie(ctx, title, 0, lang)
			}
		}
		if len(res.Results) == 0 {
			return nil, fmt.Errorf("no movie found on TMDB for query '%s' (%d)", title, year)
		}
		tmdbID = res.Results[0].ID
	}

	details, err := c.apiClient.GetMovieDetails(ctx, tmdbID, lang)
	if err != nil {
		return nil, fmt.Errorf("fetch movie details (id=%d) error: %w", tmdbID, err)
	}

	meta := &scraper.MediaMetadata{
		TMDBID:        details.ID,
		IMDbID:        details.IMDbID,
		MediaType:     "movie",
		Title:         details.Title,
		OriginalTitle: details.OriginalTitle,
		SortTitle:     details.Title,
		Premiered:     details.ReleaseDate,
		Plot:          details.Overview,
		Rating:        details.VoteAverage,
		Votes:         details.VoteCount,
	}

	if len(details.ReleaseDate) >= 4 {
		meta.Year, _ = strconv.Atoi(details.ReleaseDate[:4])
	}
	for _, g := range details.Genres {
		meta.Genres = append(meta.Genres, g.Name)
	}
	for _, comp := range details.ProductionCompanies {
		meta.Studios = append(meta.Studios, comp.Name)
	}
	for _, crew := range details.Credits.Crew {
		if strings.EqualFold(crew.Job, "Director") {
			meta.Directors = append(meta.Directors, crew.Name)
		}
	}
	for _, cast := range details.Credits.Cast {
		meta.Actors = append(meta.Actors, scraper.Actor{
			Name:  cast.Name,
			Role:  cast.Character,
			Thumb: c.apiClient.BuildImageURL(cast.ProfilePath),
		})
	}
	if details.PosterPath != "" {
		meta.PosterURL = c.apiClient.BuildImageURL(details.PosterPath)
	}
	if details.BackdropPath != "" {
		meta.BackdropURL = c.apiClient.BuildImageURL(details.BackdropPath)
	}

	return meta, nil
}

func (c *Client) scrapeTV(ctx context.Context, title string, year int, tmdbID int, lang, targetDir string) (*scraper.MediaMetadata, error) {
	if tmdbID <= 0 {
		res, err := c.apiClient.SearchTV(ctx, title, year, lang)
		if err != nil {
			return nil, err
		}
		if len(res.Results) == 0 && year > 0 {
			res, _ = c.apiClient.SearchTV(ctx, title, 0, lang)
		}
		if len(res.Results) == 0 {
			return nil, fmt.Errorf("no TV show found on TMDB for query '%s' (%d)", title, year)
		}
		tmdbID = res.Results[0].ID
	}

	details, err := c.apiClient.GetTVDetails(ctx, tmdbID, lang)
	if err != nil {
		return nil, fmt.Errorf("fetch TV details (id=%d) error: %w", tmdbID, err)
	}

	meta := &scraper.MediaMetadata{
		TMDBID:        details.ID,
		MediaType:     "tv",
		Title:         details.Name,
		OriginalTitle: details.OriginalName,
		SortTitle:     details.Name,
		Premiered:     details.FirstAirDate,
		Plot:          details.Overview,
		Rating:        details.VoteAverage,
		Votes:         details.VoteCount,
	}

	if len(details.FirstAirDate) >= 4 {
		meta.Year, _ = strconv.Atoi(details.FirstAirDate[:4])
	}
	for _, g := range details.Genres {
		meta.Genres = append(meta.Genres, g.Name)
	}
	for _, comp := range details.ProductionCompanies {
		meta.Studios = append(meta.Studios, comp.Name)
	}
	for _, cast := range details.Credits.Cast {
		meta.Actors = append(meta.Actors, scraper.Actor{
			Name:  cast.Name,
			Role:  cast.Character,
			Thumb: c.apiClient.BuildImageURL(cast.ProfilePath),
		})
	}
	if details.PosterPath != "" {
		meta.PosterURL = c.apiClient.BuildImageURL(details.PosterPath)
	}
	if details.BackdropPath != "" {
		meta.BackdropURL = c.apiClient.BuildImageURL(details.BackdropPath)
	}

	// Find required seasons in targetDir and query episodes
	seasons := detectSeasonsInDir(targetDir)
	if len(seasons) == 0 {
		seasons = []int{1}
	}

	for _, sNum := range seasons {
		sDetails, sErr := c.apiClient.GetSeasonDetails(ctx, tmdbID, sNum, lang)
		if sErr != nil {
			slog.Warn("Fetch season details error", "tmdb_id", tmdbID, "season", sNum, "error", sErr)
			continue
		}
		for _, ep := range sDetails.Episodes {
			meta.Episodes = append(meta.Episodes, scraper.EpisodeMetadata{
				Season:       sNum,
				Episode:      ep.EpisodeNumber,
				Title:        ep.Name,
				Plot:         ep.Overview,
				Rating:       ep.VoteAverage,
				Aired:        ep.AirDate,
				ThumbnailURL: c.apiClient.BuildImageURL(ep.StillPath),
			})
		}
	}

	return meta, nil
}

func (c *Client) writeEpisodeNFOs(targetDir string, meta *scraper.MediaMetadata, result *scraper.ScrapeResult) {
	if len(meta.Episodes) == 0 {
		return
	}

	epMap := make(map[string]scraper.EpisodeMetadata)
	for _, ep := range meta.Episodes {
		key := fmt.Sprintf("S%02dE%02d", ep.Season, ep.Episode)
		epMap[key] = ep
	}

	_ = filepath.Walk(targetDir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(info.Name()))
		if ext != ".mkv" && ext != ".mp4" && ext != ".ts" {
			return nil
		}

		if m := episodeRegex.FindStringSubmatch(info.Name()); len(m) == 3 {
			s, _ := strconv.Atoi(m[1])
			e, _ := strconv.Atoi(m[2])
			key := fmt.Sprintf("S%02dE%02d", s, e)
			if ep, found := epMap[key]; found {
				nfoPath := strings.TrimSuffix(p, ext) + ".nfo"
				if nfoBytes, nfoErr := buildEpisodeNFO(&ep); nfoErr == nil {
					if writeErr := os.WriteFile(nfoPath, nfoBytes, 0644); writeErr == nil {
						result.CreatedFiles = append(result.CreatedFiles, nfoPath)
					}
				}
			}
		}
		return nil
	})
}

func detectMediaType(dirPath string) string {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return "movie"
	}

	for _, entry := range entries {
		if entry.IsDir() && seasonDirRegex.MatchString(entry.Name()) {
			return "tv"
		}
		if !entry.IsDir() && episodeRegex.MatchString(entry.Name()) {
			return "tv"
		}
	}
	return "movie"
}

func detectSeasonsInDir(dirPath string) []int {
	var seasons []int
	seen := make(map[int]bool)

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return seasons
	}

	for _, entry := range entries {
		if entry.IsDir() {
			if m := seasonDirRegex.FindStringSubmatch(entry.Name()); len(m) == 2 {
				s, _ := strconv.Atoi(m[1])
				if s > 0 && !seen[s] {
					seen[s] = true
					seasons = append(seasons, s)
				}
			}
		}
	}
	return seasons
}
