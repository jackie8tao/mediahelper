package tmdb

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Default constants for TMDB API configuration.
const (
	DefaultBaseURL      = "https://api.themoviedb.org/3"
	DefaultImageBaseURL = "https://image.tmdb.org/t/p/original"
	DefaultLanguage     = "zh-CN"
	DefaultTimeout      = 20 * time.Second
)

// Config holds client parameters for connecting to TheMovieDatabase API.
type Config struct {
	APIKey       string        `json:"api_key"`
	BaseURL      string        `json:"base_url"`
	ImageBaseURL string        `json:"image_base_url"`
	Language     string        `json:"language"`
	Proxy        string        `json:"proxy"`
	Timeout      time.Duration `json:"timeout"`
	HTTPClient   *http.Client  `json:"-"`
}

// Client provides high-level access to TheMovieDatabase (TMDB) API.
type Client struct {
	cfg        Config
	httpClient *http.Client
}

// NewClient creates a new TMDB client with the provided configuration.
func NewClient(cfg Config) (*Client, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, fmt.Errorf("tmdb api_key is required")
	}

	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL
	}
	if cfg.ImageBaseURL == "" {
		cfg.ImageBaseURL = DefaultImageBaseURL
	}
	if cfg.Language == "" {
		cfg.Language = DefaultLanguage
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = DefaultTimeout
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		transport := &http.Transport{
			Proxy: http.ProxyFromEnvironment,
		}

		if strings.TrimSpace(cfg.Proxy) != "" {
			proxyURL, err := url.Parse(cfg.Proxy)
			if err != nil {
				return nil, fmt.Errorf("invalid proxy URL: %w", err)
			}
			transport.Proxy = http.ProxyURL(proxyURL)
		}

		httpClient = &http.Client{
			Timeout:   cfg.Timeout,
			Transport: transport,
		}
	}

	return &Client{
		cfg:        cfg,
		httpClient: httpClient,
	}, nil
}

// Config returns a copy of the client configuration.
func (c *Client) Config() Config {
	return c.cfg
}

// Language returns the configured default language.
func (c *Client) Language() string {
	return c.cfg.Language
}

// BuildImageURL constructs a complete image URL from a poster/backdrop/profile relative path.
func (c *Client) BuildImageURL(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	return strings.TrimRight(c.cfg.ImageBaseURL, "/") + "/" + strings.TrimLeft(path, "/")
}

// SearchMovie searches for movies by title query and optional release year.
func (c *Client) SearchMovie(ctx context.Context, query string, year int, lang string) (*SearchMovieResponse, error) {
	if lang == "" {
		lang = c.cfg.Language
	}

	endpoint := fmt.Sprintf("%s/search/movie?api_key=%s&query=%s&language=%s",
		strings.TrimRight(c.cfg.BaseURL, "/"), url.QueryEscape(c.cfg.APIKey), url.QueryEscape(query), url.QueryEscape(lang))
	if year > 0 {
		endpoint += fmt.Sprintf("&year=%d", year)
	}

	var res SearchMovieResponse
	if err := c.DoGet(ctx, endpoint, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// SearchTV searches for TV shows by title query and optional first air date year.
func (c *Client) SearchTV(ctx context.Context, query string, year int, lang string) (*SearchTVResponse, error) {
	if lang == "" {
		lang = c.cfg.Language
	}

	endpoint := fmt.Sprintf("%s/search/tv?api_key=%s&query=%s&language=%s",
		strings.TrimRight(c.cfg.BaseURL, "/"), url.QueryEscape(c.cfg.APIKey), url.QueryEscape(query), url.QueryEscape(lang))
	if year > 0 {
		endpoint += fmt.Sprintf("&first_air_date_year=%d", year)
	}

	var res SearchTVResponse
	if err := c.DoGet(ctx, endpoint, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// GetMovieDetails retrieves detailed information for a movie including cast and crew credits.
func (c *Client) GetMovieDetails(ctx context.Context, tmdbID int, lang string) (*MovieDetails, error) {
	if lang == "" {
		lang = c.cfg.Language
	}

	endpoint := fmt.Sprintf("%s/movie/%d?api_key=%s&language=%s&append_to_response=credits",
		strings.TrimRight(c.cfg.BaseURL, "/"), tmdbID, url.QueryEscape(c.cfg.APIKey), url.QueryEscape(lang))

	var details MovieDetails
	if err := c.DoGet(ctx, endpoint, &details); err != nil {
		return nil, err
	}
	return &details, nil
}

// GetTVDetails retrieves detailed information for a TV series including credits and season list.
func (c *Client) GetTVDetails(ctx context.Context, tmdbID int, lang string) (*TVDetails, error) {
	if lang == "" {
		lang = c.cfg.Language
	}

	endpoint := fmt.Sprintf("%s/tv/%d?api_key=%s&language=%s&append_to_response=credits",
		strings.TrimRight(c.cfg.BaseURL, "/"), tmdbID, url.QueryEscape(c.cfg.APIKey), url.QueryEscape(lang))

	var details TVDetails
	if err := c.DoGet(ctx, endpoint, &details); err != nil {
		return nil, err
	}
	return &details, nil
}

// GetSeasonDetails retrieves episode details for a specific season of a TV series.
func (c *Client) GetSeasonDetails(ctx context.Context, tmdbID, seasonNumber int, lang string) (*SeasonDetails, error) {
	if lang == "" {
		lang = c.cfg.Language
	}

	endpoint := fmt.Sprintf("%s/tv/%d/season/%d?api_key=%s&language=%s",
		strings.TrimRight(c.cfg.BaseURL, "/"), tmdbID, seasonNumber, url.QueryEscape(c.cfg.APIKey), url.QueryEscape(lang))

	var details SeasonDetails
	if err := c.DoGet(ctx, endpoint, &details); err != nil {
		return nil, err
	}
	return &details, nil
}

// DownloadImage downloads an image from imgURL and saves it to dstPath atomically.
func (c *Client) DownloadImage(ctx context.Context, imgURL, dstPath string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imgURL, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download image returned status %d", resp.StatusCode)
	}

	tmpFile := dstPath + ".tmp"
	out, err := os.Create(tmpFile)
	if err != nil {
		return err
	}

	if _, err := io.Copy(out, resp.Body); err != nil {
		out.Close()
		_ = os.Remove(tmpFile)
		return err
	}
	out.Close()

	return os.Rename(tmpFile, dstPath)
}

// DoGet performs an HTTP GET request to the given endpoint and JSON-decodes response body into target.
func (c *Client) DoGet(ctx context.Context, endpoint string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("create http request error: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http get error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("tmdb api returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return json.NewDecoder(resp.Body).Decode(target)
}
