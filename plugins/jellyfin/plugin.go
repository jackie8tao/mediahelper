package jellyfin

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/jackie8tao/mediahelper/internal/plugin"
)

// Plugin Jellyfin 媒体库刷新插件
type Plugin struct {
	serverURL  string
	apiKey     string
	httpClient *http.Client
}

// New 创建 Jellyfin 插件实例
func New() *Plugin {
	return &Plugin{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (p *Plugin) ID() string {
	return "jellyfin_refresh"
}

func (p *Plugin) Name() string {
	return "Jellyfin 媒体库刷新"
}

func (p *Plugin) Version() string {
	return "1.0.0"
}

func (p *Plugin) Description() string {
	return "影视归档完成后自动通知 Jellyfin 刷新媒体库"
}

func (p *Plugin) Init(ctx context.Context, cfg map[string]interface{}) error {
	if url, ok := cfg["server_url"].(string); ok {
		p.serverURL = strings.TrimRight(url, "/")
	}
	if key, ok := cfg["api_key"].(string); ok {
		p.apiKey = strings.TrimSpace(key)
	}

	if p.serverURL == "" {
		return fmt.Errorf("jellyfin server_url is required")
	}
	return nil
}

func (p *Plugin) Start(ctx context.Context) error {
	return nil
}

func (p *Plugin) Stop(ctx context.Context) error {
	return nil
}

// OnPostOrganize 当文件整理成功后触发 Jellyfin 媒体库全局或目录扫描
func (p *Plugin) OnPostOrganize(ctx context.Context, event plugin.OrganizeEvent) error {
	if p.serverURL == "" {
		return nil
	}

	refreshURL := fmt.Sprintf("%s/Library/Refresh", p.serverURL)
	if p.apiKey != "" {
		refreshURL = fmt.Sprintf("%s?api_key=%s", refreshURL, p.apiKey)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, refreshURL, nil)
	if err != nil {
		return fmt.Errorf("create jellyfin refresh request error: %w", err)
	}

	if p.apiKey != "" {
		req.Header.Set("X-Emby-Token", p.apiKey)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("jellyfin refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		slog.Info("Successfully triggered Jellyfin library refresh", "media", event.Metadata.Title)
		return nil
	}

	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("jellyfin returned status %d: %s", resp.StatusCode, string(body))
}
