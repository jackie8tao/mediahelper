package organizer

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackie8tao/mediahelper/internal/config"
	"github.com/jackie8tao/mediahelper/internal/parser"
	"github.com/jackie8tao/mediahelper/internal/scraper"
)

type dummyScraper struct{}

func (d *dummyScraper) Scrape(ctx context.Context, info *parser.MediaInfo) (*scraper.MediaMetadata, error) {
	return nil, nil
}

func (d *dummyScraper) DownloadImage(ctx context.Context, imageURL string, destPath string) error {
	return os.WriteFile(destPath, []byte("fake_image_bytes"), 0644)
}

func TestOrganizer_Movie(t *testing.T) {
	tempBase, err := os.MkdirTemp("", "organizer_movie_test_*")
	if err != nil {
		t.Fatalf("MkdirTemp failed: %v", err)
	}
	defer os.RemoveAll(tempBase)

	downloadDir := filepath.Join(tempBase, "downloads")
	movieLibDir := filepath.Join(tempBase, "media", "movies")
	_ = os.MkdirAll(downloadDir, 0755)

	// 准备源文件 (大于 1024 字节模拟有效视频)
	srcVideo := filepath.Join(downloadDir, "Oppenheimer.2023.2160p.mkv")
	srcSub := filepath.Join(downloadDir, "Oppenheimer.2023.2160p.zh.srt")
	_ = os.WriteFile(srcVideo, make([]byte, 2048), 0644)
	_ = os.WriteFile(srcSub, []byte("1\n00:00:01 --> 00:00:03\nHello"), 0644)

	cfg := config.Config{
		TMDB: config.TMDBConfig{
			GenerateNFO:    true,
			DownloadImages: true,
		},
		Library: config.LibraryConfig{
			MovieDir:    movieLibDir,
			LinkMode:    "hardlink",
			MinFileSize: 1024, // 1KB
		},
	}

	org := NewOrganizer(cfg, &dummyScraper{})

	meta := &scraper.MediaMetadata{
		TMDBID:      872585,
		MediaType:   parser.MediaTypeMovie,
		Title:       "奥本海默",
		Year:        2023,
		PosterURL:   "http://example.com/poster.jpg",
		BackdropURL: "http://example.com/backdrop.jpg",
	}

	mediaInfo := &parser.MediaInfo{
		MediaType:  parser.MediaTypeMovie,
		Title:      "奥本海默",
		Resolution: "4K",
		Year:       2023,
	}

	task := OrganizeTask{
		SourcePath: srcVideo,
		MediaInfo:  mediaInfo,
		Metadata:   meta,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := org.Organize(ctx, task)
	if err != nil {
		t.Fatalf("Organize movie failed: %v", err)
	}

	if !result.Success {
		t.Fatalf("expected success, got false")
	}

	// 校验目标视频文件是否存在
	expectedVideo := filepath.Join(movieLibDir, "奥本海默 (2023)", "奥本海默 (2023) - 4K.mkv")
	if _, err := os.Stat(expectedVideo); os.IsNotExist(err) {
		t.Errorf("expected video file not found: %s", expectedVideo)
	}

	// 校验伴生字幕是否存在
	expectedSub := filepath.Join(movieLibDir, "奥本海默 (2023)", "奥本海默 (2023) - 4K.zh.srt")
	if _, err := os.Stat(expectedSub); os.IsNotExist(err) {
		t.Errorf("expected sub file not found: %s", expectedSub)
	}

	// 校验 movie.nfo 是否生成
	expectedNFO := filepath.Join(movieLibDir, "奥本海默 (2023)", "movie.nfo")
	if _, err := os.Stat(expectedNFO); os.IsNotExist(err) {
		t.Errorf("expected movie.nfo not found: %s", expectedNFO)
	}

	// 校验 poster.jpg 与 fanart.jpg 是否生成
	expectedPoster := filepath.Join(movieLibDir, "奥本海默 (2023)", "poster.jpg")
	if _, err := os.Stat(expectedPoster); os.IsNotExist(err) {
		t.Errorf("expected poster.jpg not found: %s", expectedPoster)
	}
}

func TestOrganizer_TVShow(t *testing.T) {
	tempBase, err := os.MkdirTemp("", "organizer_tv_test_*")
	if err != nil {
		t.Fatalf("MkdirTemp failed: %v", err)
	}
	defer os.RemoveAll(tempBase)

	downloadDir := filepath.Join(tempBase, "downloads", "Breaking.Bad.S01.1080p")
	tvLibDir := filepath.Join(tempBase, "media", "tv")
	_ = os.MkdirAll(downloadDir, 0755)

	// 创建剧集目录内的多个文件
	ep1 := filepath.Join(downloadDir, "Breaking.Bad.S01E01.mkv")
	ep2 := filepath.Join(downloadDir, "Breaking.Bad.S01E02.mkv")
	_ = os.WriteFile(ep1, make([]byte, 2048), 0644)
	_ = os.WriteFile(ep2, make([]byte, 2048), 0644)

	cfg := config.Config{
		TMDB: config.TMDBConfig{
			GenerateNFO:    true,
			DownloadImages: false,
		},
		Library: config.LibraryConfig{
			TVDir:       tvLibDir,
			LinkMode:    "hardlink",
			MinFileSize: 1024,
		},
	}

	org := NewOrganizer(cfg, &dummyScraper{})

	meta := &scraper.MediaMetadata{
		TMDBID:       1396,
		MediaType:    parser.MediaTypeTV,
		Title:        "绝命毒师",
		Year:         2008,
		Season:       1,
		Episode:      1,
		EpisodeTitle: "首播集",
	}

	mediaInfo := &parser.MediaInfo{
		MediaType:    parser.MediaTypeTV,
		Title:        "绝命毒师",
		Year:         2008,
		Season:       1,
		IsSeasonPack: true,
	}

	task := OrganizeTask{
		SourcePath: downloadDir,
		MediaInfo:  mediaInfo,
		Metadata:   meta,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := org.Organize(ctx, task)
	if err != nil {
		t.Fatalf("Organize TV failed: %v", err)
	}

	if !result.Success {
		t.Fatalf("expected success, got false")
	}

	// 校验 S01E01 文件与 NFO
	expectedEp1 := filepath.Join(tvLibDir, "绝命毒师 (2008)", "Season 01", "绝命毒师 - S01E01 - 首播集.mkv")
	if _, err := os.Stat(expectedEp1); os.IsNotExist(err) {
		t.Errorf("expected Ep1 not found: %s", expectedEp1)
	}

	expectedEp1NFO := filepath.Join(tvLibDir, "绝命毒师 (2008)", "Season 01", "绝命毒师 - S01E01 - 首播集.nfo")
	if _, err := os.Stat(expectedEp1NFO); os.IsNotExist(err) {
		t.Errorf("expected Ep1 NFO not found: %s", expectedEp1NFO)
	}

	// 校验 S01E02 文件
	expectedEp2 := filepath.Join(tvLibDir, "绝命毒师 (2008)", "Season 01", "绝命毒师 - S01E02.mkv")
	if _, err := os.Stat(expectedEp2); os.IsNotExist(err) {
		t.Errorf("expected Ep2 not found: %s", expectedEp2)
	}

	// 校验 tvshow.nfo
	expectedTVNFO := filepath.Join(tvLibDir, "绝命毒师 (2008)", "tvshow.nfo")
	if _, err := os.Stat(expectedTVNFO); os.IsNotExist(err) {
		t.Errorf("expected tvshow.nfo not found: %s", expectedTVNFO)
	}
}
