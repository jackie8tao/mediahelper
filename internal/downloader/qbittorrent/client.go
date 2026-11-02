package qbittorrent

import (
	"encoding/json"
	"fmt"

	qb "github.com/autobrr/go-qbittorrent"
	"github.com/jackie8tao/mediahelper/internal/downloader"
)

// Config holds connection parameters for a qBittorrent client instance.
type Config struct {
	Addr     string `json:"addr"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// Client wraps autobrr/go-qbittorrent and implements downloader.Downloader.
type Client struct {
	cfg    Config
	client *qb.Client
}

func init() {
	downloader.RegisterFactory("qbittorrent", QbittorrentFactory)
}

// QbittorrentFactory initializes a new qBittorrent client instance.
func QbittorrentFactory(cfgVal json.RawMessage) (downloader.Downloader, error) {
	var cfgSt Config
	if err := json.Unmarshal(cfgVal, &cfgSt); err != nil {
		return nil, err
	}

	if cfgSt.Addr == "" {
		return nil, fmt.Errorf("qbittorrent address is required")
	}

	qbCfg := qb.Config{
		Host:          cfgSt.Addr,
		Username:      cfgSt.Username,
		Password:      cfgSt.Password,
		RetryAttempts: 3,
		RetryDelay:    1,
	}

	client := qb.NewClient(qbCfg)

	return &Client{
		cfg:    cfgSt,
		client: client,
	}, nil
}

// Name returns the downloader name.
func (c *Client) Name() string {
	return "qBittorrent"
}
