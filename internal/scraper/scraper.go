package scraper

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/jackie8tao/mediahelper/internal/parser"
)

// Scraper 刮削服务接口
type Scraper interface {
	// Scrape 根据文件名解析结果检索元数据
	Scrape(ctx context.Context, info *parser.MediaInfo) (*MediaMetadata, error)
	// DownloadImage 下载图片保存至本地路径
	DownloadImage(ctx context.Context, imageURL string, destPath string) error
}

// TMDBScraperClient TMDB 客户端接口，便于 Mock 测试
type TMDBScraperClient interface {
	SearchAndGetMovie(ctx context.Context, title string, year int) (*MediaMetadata, error)
	SearchAndGetTV(ctx context.Context, title string, year int, season int, episode int) (*MediaMetadata, error)
}

// Service 刮削服务协调器
type Service struct {
	tmdbClient TMDBScraperClient
	httpClient *http.Client
}

// NewService 创建刮削服务实例
func NewService(tmdbClient TMDBScraperClient) *Service {
	return &Service{
		tmdbClient: tmdbClient,
		httpClient: &http.Client{},
	}
}

// Scrape 核心刮削入口
func (s *Service) Scrape(ctx context.Context, info *parser.MediaInfo) (*MediaMetadata, error) {
	if info == nil {
		return nil, fmt.Errorf("media info cannot be nil")
	}

	searchTitle := info.Title
	if info.CnTitle != "" {
		searchTitle = info.CnTitle
	}

	if searchTitle == "" {
		return nil, fmt.Errorf("no title available for scraping")
	}

	switch info.MediaType {
	case parser.MediaTypeMovie:
		meta, err := s.tmdbClient.SearchAndGetMovie(ctx, searchTitle, info.Year)
		if err != nil && info.EnTitle != "" && info.EnTitle != searchTitle {
			// 回退尝试使用英文名检索
			meta, err = s.tmdbClient.SearchAndGetMovie(ctx, info.EnTitle, info.Year)
		}
		if err != nil {
			return nil, fmt.Errorf("scrape movie failed (%s): %w", searchTitle, err)
		}
		return meta, nil

	case parser.MediaTypeTV, parser.MediaTypeAnime:
		meta, err := s.tmdbClient.SearchAndGetTV(ctx, searchTitle, info.Year, info.Season, info.Episode)
		if err != nil && info.EnTitle != "" && info.EnTitle != searchTitle {
			// 回退尝试使用英文名检索
			meta, err = s.tmdbClient.SearchAndGetTV(ctx, info.EnTitle, info.Year, info.Season, info.Episode)
		}
		if err != nil {
			return nil, fmt.Errorf("scrape tv show failed (%s): %w", searchTitle, err)
		}
		if info.MediaType == parser.MediaTypeAnime {
			meta.MediaType = parser.MediaTypeAnime
		}
		return meta, nil

	default:
		return nil, fmt.Errorf("unsupported media type: %s", info.MediaType)
	}
}

// DownloadImage 下载图片并安全写入指定路径
func (s *Service) DownloadImage(ctx context.Context, imageURL string, destPath string) error {
	if imageURL == "" {
		return fmt.Errorf("image URL cannot be empty")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("download image error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download image status code: %d", resp.StatusCode)
	}

	// 确保父目录存在
	dir := filepath.Dir(destPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create dir error: %w", err)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("create dest file error: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("write image error: %w", err)
	}

	return nil
}
