package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/jackie8tao/mediahelper/internal/plugin"
)

// Plugin 通用 Webhook 消息通知插件 (兼容企业微信/钉钉/飞书/自定义 Webhook)
type Plugin struct {
	webhookURL string
	httpClient *http.Client
}

// New 创建通知插件实例
func New() *Plugin {
	return &Plugin{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (p *Plugin) ID() string {
	return "webhook_notify"
}

func (p *Plugin) Name() string {
	return "Webhook 消息通知"
}

func (p *Plugin) Version() string {
	return "1.0.0"
}

func (p *Plugin) Description() string {
	return "影视整理完成后向企业微信/钉钉/飞书/自定义 Webhook 发送卡片通知"
}

func (p *Plugin) Init(ctx context.Context, cfg map[string]interface{}) error {
	if url, ok := cfg["webhook_url"].(string); ok {
		p.webhookURL = strings.TrimSpace(url)
	}

	if p.webhookURL == "" {
		return fmt.Errorf("webhook_url is required")
	}
	return nil
}

func (p *Plugin) Start(ctx context.Context) error {
	return nil
}

func (p *Plugin) Stop(ctx context.Context) error {
	return nil
}

// OnPostOrganize 当文件整理归档完成后发送通知
func (p *Plugin) OnPostOrganize(ctx context.Context, event plugin.OrganizeEvent) error {
	if p.webhookURL == "" || event.Metadata == nil {
		return nil
	}

	title := event.Metadata.Title
	if event.Metadata.Year > 0 {
		title = fmt.Sprintf("%s (%d)", title, event.Metadata.Year)
	}

	mType := "电影"
	if string(event.Metadata.MediaType) == "tv" {
		mType = "电视剧"
	} else if string(event.Metadata.MediaType) == "anime" {
		mType = "动漫"
	}

	targetDir := ""
	fileCount := 0
	if event.Result != nil {
		targetDir = event.Result.TargetDir
		fileCount = len(event.Result.LinkedFiles)
	}

	nowStr := time.Now().Format("2006-01-02 15:04:05")

	// 构造多平台兼容 Payload (同时支持标准 JSON、企业微信与钉钉 markdown)
	markdownText := fmt.Sprintf("### 🎬 影视整理成功通知\n- **作品**: %s\n- **类型**: %s\n- **归档路径**: %s\n- **文件数量**: %d 个\n- **完成时间**: %s",
		title, mType, targetDir, fileCount, nowStr)

	payload := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"content": markdownText, // 企业微信
			"title":   "影视整理通知",     // 钉钉
			"text":    markdownText, // 钉钉
		},
		"title":       title,
		"media_type":  mType,
		"target_dir":  targetDir,
		"file_count":  fileCount,
		"poster_url":  event.Metadata.PosterURL,
		"finished_at": nowStr,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal notify payload error: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.webhookURL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create notify request error: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send webhook notify error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		slog.Info("Successfully sent webhook notification", "title", title)
		return nil
	}

	return fmt.Errorf("webhook endpoint returned status: %d", resp.StatusCode)
}
