package tmdb

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jackie8tao/mediahelper/internal/config"
	"github.com/jackie8tao/mediahelper/internal/parser"
	"github.com/jackie8tao/mediahelper/internal/scraper"
)

// Client TMDB API 客户端
type Client struct {
	cfg        config.TMDBConfig
	httpClient *http.Client
}

func init() {
	scraper.RegisterFactory("tmdb", func(cfg config.ScraperInstanceConfig) (scraper.Scraper, error) {
		tmdbCli, err := NewClient(cfg.ToTMDBConfig())
		if err != nil {
			return nil, err
		}
		return scraper.NewService(cfg.ToTMDBConfig(), tmdbCli), nil
	})
}

// NewClient 创建 TMDB 客户端
func NewClient(cfg config.TMDBConfig) (*Client, error) {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}

	// 支持自定义 HTTP/SOCKS5 代理
	if cfg.Proxy != "" {
		proxyURL, err := url.Parse(cfg.Proxy)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy URL: %w", err)
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}

	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.themoviedb.org/3"
	}
	if cfg.ImageBaseURL == "" {
		cfg.ImageBaseURL = "https://image.tmdb.org/t/p/original"
	}
	if cfg.Language == "" {
		cfg.Language = "zh-CN"
	}

	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout:   15 * time.Second,
			Transport: transport,
		},
	}, nil
}

// SearchAndGetMovie 根据片名和年份搜索电影并获取详细信息
func (c *Client) SearchAndGetMovie(ctx context.Context, title string, year int) (*scraper.MediaMetadata, error) {
	// 1. 搜索电影 ID
	movieID, err := c.searchMovieID(ctx, title, year)
	if err != nil {
		return nil, err
	}

	// 2. 获取电影详情及演职人员
	return c.GetMovieDetails(ctx, movieID)
}

// SearchAndGetTV 根据剧名和年份搜索电视剧，并可选抓取单集详情
func (c *Client) SearchAndGetTV(ctx context.Context, title string, year int, season int, episode int) (*scraper.MediaMetadata, error) {
	// 1. 搜索剧集 ID
	tvID, err := c.searchTVID(ctx, title, year)
	if err != nil {
		return nil, err
	}

	// 2. 获取剧集主详情
	meta, err := c.GetTVDetails(ctx, tvID)
	if err != nil {
		return nil, err
	}

	// 3. 如果指定了集数，继续获取该集信息
	if season > 0 && episode > 0 {
		epInfo, err := c.getEpisodeDetails(ctx, tvID, season, episode)
		if err == nil && epInfo != nil {
			meta.Season = season
			meta.Episode = episode
			meta.EpisodeTitle = epInfo.Name
			meta.EpisodePlot = epInfo.Overview
			meta.EpisodeRating = epInfo.VoteAverage
			meta.EpisodeAired = epInfo.AirDate
			if epInfo.StillPath != "" {
				meta.EpisodeThumb = c.buildImageURL(epInfo.StillPath)
			}
		}
	}

	return meta, nil
}

// searchMovieID 检索电影 ID
func (c *Client) searchMovieID(ctx context.Context, query string, year int) (int, error) {
	endpoint := fmt.Sprintf("%s/search/movie", strings.TrimSuffix(c.cfg.BaseURL, "/"))
	params := url.Values{}
	params.Set("query", query)
	params.Set("language", c.cfg.Language)
	if year > 0 {
		params.Set("year", strconv.Itoa(year))
	}

	body, err := c.get(ctx, endpoint, params)
	if err != nil {
		return 0, err
	}

	var res struct {
		Results []struct {
			ID          int    `json:"id"`
			Title       string `json:"title"`
			ReleaseDate string `json:"release_date"`
		} `json:"results"`
	}

	if err := json.Unmarshal(body, &res); err != nil {
		return 0, fmt.Errorf("parse tmdb search movie response error: %w", err)
	}

	if len(res.Results) == 0 {
		// 如果带年份搜索未找到，尝试不带年份二次检索 (允许首映年份轻微偏差)
		if year > 0 {
			return c.searchMovieID(ctx, query, 0)
		}
		return 0, fmt.Errorf("movie not found on tmdb: %s", query)
	}

	return res.Results[0].ID, nil
}

// searchTVID 检索电视剧 ID
func (c *Client) searchTVID(ctx context.Context, query string, year int) (int, error) {
	endpoint := fmt.Sprintf("%s/search/tv", strings.TrimSuffix(c.cfg.BaseURL, "/"))
	params := url.Values{}
	params.Set("query", query)
	params.Set("language", c.cfg.Language)
	if year > 0 {
		params.Set("first_air_date_year", strconv.Itoa(year))
	}

	body, err := c.get(ctx, endpoint, params)
	if err != nil {
		return 0, err
	}

	var res struct {
		Results []struct {
			ID           int    `json:"id"`
			Name         string `json:"name"`
			FirstAirDate string `json:"first_air_date"`
		} `json:"results"`
	}

	if err := json.Unmarshal(body, &res); err != nil {
		return 0, fmt.Errorf("parse tmdb search tv response error: %w", err)
	}

	if len(res.Results) == 0 {
		if year > 0 {
			return c.searchTVID(ctx, query, 0)
		}
		return 0, fmt.Errorf("tv show not found on tmdb: %s", query)
	}

	return res.Results[0].ID, nil
}

// GetMovieDetails 获取电影完整详情
func (c *Client) GetMovieDetails(ctx context.Context, movieID int) (*scraper.MediaMetadata, error) {
	endpoint := fmt.Sprintf("%s/movie/%d", strings.TrimSuffix(c.cfg.BaseURL, "/"), movieID)
	params := url.Values{}
	params.Set("language", c.cfg.Language)
	params.Set("append_to_response", "credits")

	body, err := c.get(ctx, endpoint, params)
	if err != nil {
		return nil, err
	}

	var raw struct {
		ID            int     `json:"id"`
		IMDbID        string  `json:"imdb_id"`
		Title         string  `json:"title"`
		OriginalTitle string  `json:"original_title"`
		Overview      string  `json:"overview"`
		ReleaseDate   string  `json:"release_date"`
		VoteAverage   float64 `json:"vote_average"`
		VoteCount     int     `json:"vote_count"`
		PosterPath    string  `json:"poster_path"`
		BackdropPath  string  `json:"backdrop_path"`
		Genres        []struct {
			Name string `json:"name"`
		} `json:"genres"`
		ProductionCompanies []struct {
			Name string `json:"name"`
		} `json:"production_companies"`
		Credits struct {
			Cast []struct {
				Name        string `json:"name"`
				Character   string `json:"character"`
				ProfilePath string `json:"profile_path"`
			} `json:"cast"`
			Crew []struct {
				Name string `json:"name"`
				Job  string `json:"job"`
			} `json:"crew"`
		} `json:"credits"`
	}

	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parse movie details error: %w", err)
	}

	meta := &scraper.MediaMetadata{
		TMDBID:        raw.ID,
		IMDbID:        raw.IMDbID,
		MediaType:     parser.MediaTypeMovie,
		Title:         raw.Title,
		OriginalTitle: raw.OriginalTitle,
		SortTitle:     raw.Title,
		Premiered:     raw.ReleaseDate,
		Plot:          raw.Overview,
		Rating:        raw.VoteAverage,
		Votes:         raw.VoteCount,
	}

	if len(raw.ReleaseDate) >= 4 {
		meta.Year, _ = strconv.Atoi(raw.ReleaseDate[:4])
	}
	if raw.PosterPath != "" {
		meta.PosterURL = c.buildImageURL(raw.PosterPath)
	}
	if raw.BackdropPath != "" {
		meta.BackdropURL = c.buildImageURL(raw.BackdropPath)
	}

	for _, g := range raw.Genres {
		meta.Genres = append(meta.Genres, g.Name)
	}
	for _, comp := range raw.ProductionCompanies {
		meta.Studios = append(meta.Studios, comp.Name)
	}

	// 演职员信息 (取前 15 位主要演员)
	maxCast := 15
	if len(raw.Credits.Cast) < maxCast {
		maxCast = len(raw.Credits.Cast)
	}
	for i := 0; i < maxCast; i++ {
		actor := raw.Credits.Cast[i]
		thumb := ""
		if actor.ProfilePath != "" {
			thumb = c.buildImageURL(actor.ProfilePath)
		}
		meta.Actors = append(meta.Actors, scraper.Actor{
			Name:  actor.Name,
			Role:  actor.Character,
			Thumb: thumb,
		})
	}

	// 导演信息
	for _, crew := range raw.Credits.Crew {
		if crew.Job == "Director" {
			meta.Directors = append(meta.Directors, crew.Name)
		}
	}

	return meta, nil
}

// GetTVDetails 获取电视剧完整详情
func (c *Client) GetTVDetails(ctx context.Context, tvID int) (*scraper.MediaMetadata, error) {
	endpoint := fmt.Sprintf("%s/tv/%d", strings.TrimSuffix(c.cfg.BaseURL, "/"), tvID)
	params := url.Values{}
	params.Set("language", c.cfg.Language)
	params.Set("append_to_response", "credits,external_ids")

	body, err := c.get(ctx, endpoint, params)
	if err != nil {
		return nil, err
	}

	var raw struct {
		ID            int     `json:"id"`
		Name          string  `json:"name"`
		OriginalName  string  `json:"original_name"`
		Overview      string  `json:"overview"`
		FirstAirDate  string  `json:"first_air_date"`
		VoteAverage   float64 `json:"vote_average"`
		VoteCount     int     `json:"vote_count"`
		PosterPath    string  `json:"poster_path"`
		BackdropPath  string  `json:"backdrop_path"`
		Genres        []struct {
			Name string `json:"name"`
		} `json:"genres"`
		ProductionCompanies []struct {
			Name string `json:"name"`
		} `json:"production_companies"`
		ExternalIDs struct {
			IMDbID string `json:"imdb_id"`
		} `json:"external_ids"`
		Credits struct {
			Cast []struct {
				Name        string `json:"name"`
				Character   string `json:"character"`
				ProfilePath string `json:"profile_path"`
			} `json:"cast"`
		} `json:"credits"`
	}

	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parse tv details error: %w", err)
	}

	meta := &scraper.MediaMetadata{
		TMDBID:        raw.ID,
		IMDbID:        raw.ExternalIDs.IMDbID,
		MediaType:     parser.MediaTypeTV,
		Title:         raw.Name,
		OriginalTitle: raw.OriginalName,
		SortTitle:     raw.Name,
		Premiered:     raw.FirstAirDate,
		Plot:          raw.Overview,
		Rating:        raw.VoteAverage,
		Votes:         raw.VoteCount,
	}

	if len(raw.FirstAirDate) >= 4 {
		meta.Year, _ = strconv.Atoi(raw.FirstAirDate[:4])
	}
	if raw.PosterPath != "" {
		meta.PosterURL = c.buildImageURL(raw.PosterPath)
	}
	if raw.BackdropPath != "" {
		meta.BackdropURL = c.buildImageURL(raw.BackdropPath)
	}

	for _, g := range raw.Genres {
		meta.Genres = append(meta.Genres, g.Name)
	}
	for _, comp := range raw.ProductionCompanies {
		meta.Studios = append(meta.Studios, comp.Name)
	}

	maxCast := 15
	if len(raw.Credits.Cast) < maxCast {
		maxCast = len(raw.Credits.Cast)
	}
	for i := 0; i < maxCast; i++ {
		actor := raw.Credits.Cast[i]
		thumb := ""
		if actor.ProfilePath != "" {
			thumb = c.buildImageURL(actor.ProfilePath)
		}
		meta.Actors = append(meta.Actors, scraper.Actor{
			Name:  actor.Name,
			Role:  actor.Character,
			Thumb: thumb,
		})
	}

	return meta, nil
}

type tmdbEpisodeResponse struct {
	Name        string  `json:"name"`
	Overview    string  `json:"overview"`
	StillPath   string  `json:"still_path"`
	VoteAverage float64 `json:"vote_average"`
	AirDate     string  `json:"air_date"`
}

func (c *Client) getEpisodeDetails(ctx context.Context, tvID int, season int, episode int) (*tmdbEpisodeResponse, error) {
	endpoint := fmt.Sprintf("%s/tv/%d/season/%d/episode/%d", strings.TrimSuffix(c.cfg.BaseURL, "/"), tvID, season, episode)
	params := url.Values{}
	params.Set("language", c.cfg.Language)

	body, err := c.get(ctx, endpoint, params)
	if err != nil {
		return nil, err
	}

	var res tmdbEpisodeResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// buildImageURL 生成完整的图片绝对 CDN URL
func (c *Client) buildImageURL(path string) string {
	if path == "" {
		return ""
	}
	return strings.TrimSuffix(c.cfg.ImageBaseURL, "/") + "/" + strings.TrimPrefix(path, "/")
}

// get 执行带有 APIKey 的 GET 请求
func (c *Client) get(ctx context.Context, endpoint string, params url.Values) ([]byte, error) {
	if params == nil {
		params = url.Values{}
	}
	if c.cfg.APIKey != "" {
		params.Set("api_key", c.cfg.APIKey)
	}

	reqURL := endpoint
	if len(params) > 0 {
		reqURL = fmt.Sprintf("%s?%s", endpoint, params.Encode())
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request error: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request error to %s: %w", endpoint, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body error: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("tmdb api returned status %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}
