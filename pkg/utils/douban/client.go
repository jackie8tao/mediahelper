package douban

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

// Default constants for Douban Frodo API.
const (
	DefaultBaseURL   = "https://frodo.douban.com/api/v2"
	DefaultAPIKey    = "0ac44ae016490db2204ce0a042db2916"
	DefaultUserAgent = "MicroMessenger/"
	DefaultReferer   = "https://servicewechat.com/wx2f9b06c1de1ccfca/91/page-frame.html"
	DefaultTimeout   = 20 * time.Second
)

// Config holds client parameters for connecting to Douban API.
type Config struct {
	APIKey     string        `json:"api_key"`
	BaseURL    string        `json:"base_url"`
	UserAgent  string        `json:"user_agent"`
	Referer    string        `json:"referer"`
	Proxy      string        `json:"proxy"`
	Timeout    time.Duration `json:"timeout"`
	HTTPClient *http.Client  `json:"-"`
}

// Client provides access to Douban Movie and TV metadata.
type Client struct {
	cfg        Config
	httpClient *http.Client
}

// NewClient creates a new Douban client with the provided configuration.
func NewClient(cfg Config) (*Client, error) {
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL
	}
	if cfg.APIKey == "" {
		cfg.APIKey = DefaultAPIKey
	}
	if cfg.UserAgent == "" {
		cfg.UserAgent = DefaultUserAgent
	}
	if cfg.Referer == "" {
		cfg.Referer = DefaultReferer
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

// GetMovie retrieves detailed metadata for a Douban movie by subject ID.
func (c *Client) GetMovie(ctx context.Context, id string) (*SubjectDetails, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("douban movie id cannot be empty")
	}

	endpoint := fmt.Sprintf("%s/movie/%s?apiKey=%s",
		strings.TrimRight(c.cfg.BaseURL, "/"), url.PathEscape(id), url.QueryEscape(c.cfg.APIKey))

	var details SubjectDetails
	if err := c.DoGet(ctx, endpoint, &details); err != nil {
		return nil, err
	}
	return &details, nil
}

// GetTV retrieves detailed metadata for a Douban TV series by subject ID.
func (c *Client) GetTV(ctx context.Context, id string) (*SubjectDetails, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("douban tv id cannot be empty")
	}

	endpoint := fmt.Sprintf("%s/tv/%s?apiKey=%s",
		strings.TrimRight(c.cfg.BaseURL, "/"), url.PathEscape(id), url.QueryEscape(c.cfg.APIKey))

	var details SubjectDetails
	if err := c.DoGet(ctx, endpoint, &details); err != nil {
		return nil, err
	}
	return &details, nil
}

// GetPhotos retrieves still and poster photos for a Douban subject.
func (c *Client) GetPhotos(ctx context.Context, id string, mediaType string) (*PhotosResponse, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("douban id cannot be empty")
	}

	category := "movie"
	if strings.EqualFold(mediaType, "tv") || strings.EqualFold(mediaType, "anime") {
		category = "tv"
	}

	endpoint := fmt.Sprintf("%s/%s/%s/photos?apiKey=%s",
		strings.TrimRight(c.cfg.BaseURL, "/"), category, url.PathEscape(id), url.QueryEscape(c.cfg.APIKey))

	var resp PhotosResponse
	if err := c.DoGet(ctx, endpoint, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Search searches for Douban subjects matching query keyword.
func (c *Client) Search(ctx context.Context, query string) (*SearchResponse, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}

	endpoint := fmt.Sprintf("%s/search/weixin?q=%s&apiKey=%s",
		strings.TrimRight(c.cfg.BaseURL, "/"), url.QueryEscape(query), url.QueryEscape(c.cfg.APIKey))

	var resp SearchResponse
	if err := c.DoGet(ctx, endpoint, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// DownloadImage downloads an image from Douban CDN and saves it to dstPath atomically.
func (c *Client) DownloadImage(ctx context.Context, imgURL, dstPath string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imgURL, nil)
	if err != nil {
		return err
	}
	// Douban CDN requires Referer header to bypass anti-hotlinking 403
	req.Header.Set("Referer", "https://movie.douban.com/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download douban image returned status %d", resp.StatusCode)
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

// DoGet executes a GET request against Douban API with required authentication and client headers.
func (c *Client) DoGet(ctx context.Context, endpoint string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("create http request error: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	if c.cfg.UserAgent != "" {
		req.Header.Set("User-Agent", c.cfg.UserAgent)
	}
	if c.cfg.Referer != "" {
		req.Header.Set("Referer", c.cfg.Referer)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http get error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("douban api returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return json.NewDecoder(resp.Body).Decode(target)
}
