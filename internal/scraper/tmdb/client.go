package tmdb

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/jackie8tao/mediahelper/internal/scraper"
)

// Config holds connection and scraping parameters for a TMDB client instance.
type Config struct {
	APIKey       string `json:"api_key"`
	BaseURL      string `json:"base_url"`
	ImageBaseURL string `json:"image_base_url"`
	Language     string `json:"language"`
	Proxy        string `json:"proxy"`
}

// Client implements scraper.Scraper for TheMovieDatabase (TMDB).
type Client struct {
	cfg        Config
	httpClient *http.Client
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

	if cfgSt.APIKey == "" {
		return nil, fmt.Errorf("tmdb api_key is required")
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}

	if cfgSt.Proxy != "" {
		proxyURL, err := url.Parse(cfgSt.Proxy)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy URL: %w", err)
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}

	if cfgSt.BaseURL == "" {
		cfgSt.BaseURL = "https://api.themoviedb.org/3"
	}
	if cfgSt.ImageBaseURL == "" {
		cfgSt.ImageBaseURL = "https://image.tmdb.org/t/p/original"
	}
	if cfgSt.Language == "" {
		cfgSt.Language = "zh-CN"
	}

	return &Client{
		cfg: cfgSt,
		httpClient: &http.Client{
			Timeout:   15 * time.Second,
			Transport: transport,
		},
	}, nil
}

// Name returns the scraper provider name.
func (c *Client) Name() string {
	return "TMDB"
}
