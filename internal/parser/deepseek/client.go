package deepseek

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackie8tao/mediahelper/internal/parser"
	openai "github.com/sashabaranov/go-openai"
)

const (
	defaultBaseURL = "https://api.deepseek.com/v1"
	defaultModel   = "deepseek-chat"

	systemPrompt = `You are a professional media filename and torrent title parser.
Analyze the user-provided raw filename or torrent title and extract structured metadata into a single JSON object.

Output strictly valid JSON matching this schema:
{
  "media_type": "movie | tv | anime",
  "title": "Primary clean title (prefer Chinese title if present, otherwise English title)",
  "cn_title": "Clean Chinese title without season/episode/specs (empty string if none)",
  "en_title": "Clean English title without season/episode/specs (empty string if none)",
  "year": 2023,
  "season": 0,
  "episode": 0,
  "resolution": "4K | 1080p | 720p | 480p | empty string",
  "source": "Remux | BluRay | WEB-DL | HDTV | DVD | empty string",
  "video_codec": "HEVC | H.264 | AV1 | VC-1 | empty string",
  "audio_codec": "Atmos | TrueHD | DTS-HD MA | DTS | DDP5.1 | AC3 | FLAC | AAC | empty string"
}

Rules:
1. For movies, set "media_type" to "movie", "season" to 0, and "episode" to 0.
2. For TV shows or anime with season/episode indicators, set "season" (default 1 if unspecified) and "episode".
3. If a numeric title like "1917" appears alongside a release year like "2019", "1917" is the title and 2019 is the year.
4. Strip all technical tags, brackets, noise words, and file extensions from title fields.`
)

// Config holds connection and model parameters for a DeepSeek parser instance.
type Config struct {
	APIKey  string `json:"api_key"`
	BaseURL string `json:"base_url"`
	Model   string `json:"model"`
}

// Client wraps go-openai client configured for DeepSeek API and implements parser.Parser.
type Client struct {
	cfg    Config
	client *openai.Client
}

func init() {
	parser.RegisterParserFactory("deepseek", DeepSeekFactory)
}

// DeepSeekFactory initializes a new DeepSeek parser client instance.
func DeepSeekFactory(cfgVal json.RawMessage) (parser.Parser, error) {
	var cfgSt Config
	if err := json.Unmarshal(cfgVal, &cfgSt); err != nil {
		return nil, err
	}

	if strings.TrimSpace(cfgSt.APIKey) == "" {
		return nil, fmt.Errorf("deepseek api_key is required")
	}

	if strings.TrimSpace(cfgSt.BaseURL) == "" {
		cfgSt.BaseURL = defaultBaseURL
	}
	if strings.TrimSpace(cfgSt.Model) == "" {
		cfgSt.Model = defaultModel
	}

	openaiCfg := openai.DefaultConfig(cfgSt.APIKey)
	openaiCfg.BaseURL = strings.TrimRight(cfgSt.BaseURL, "/")

	return &Client{
		cfg:    cfgSt,
		client: openai.NewClientWithConfig(openaiCfg),
	}, nil
}

// Name returns the parser provider name.
func (c *Client) Name() string {
	return "DeepSeek"
}

// Parse calls DeepSeek Chat Completions in JSON mode to parse a media filename or torrent title.
func (c *Client) Parse(ctx context.Context, raw string) (*parser.MediaInfo, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, fmt.Errorf("raw input title cannot be empty")
	}

	req := openai.ChatCompletionRequest{
		Model:       c.cfg.Model,
		Temperature: 0.1,
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		},
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: systemPrompt,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: trimmed,
			},
		},
	}

	resp, err := c.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("deepseek chat completion error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("deepseek returned empty choices")
	}

	content := cleanJSONBlock(resp.Choices[0].Message.Content)

	var info parser.MediaInfo
	if err := json.Unmarshal([]byte(content), &info); err != nil {
		return nil, fmt.Errorf("unmarshal deepseek response json error: %w (raw: %s)", err, content)
	}

	info.OriginalText = raw
	if info.MediaType == "" {
		info.MediaType = parser.MediaTypeMovie
	}
	if info.Title == "" {
		if info.CnTitle != "" {
			info.Title = info.CnTitle
		} else if info.EnTitle != "" {
			info.Title = info.EnTitle
		}
	}

	return &info, nil
}

// cleanJSONBlock strips optional markdown code fences around JSON output.
func cleanJSONBlock(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}
