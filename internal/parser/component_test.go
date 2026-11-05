package parser

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackie8tao/mediahelper/internal/config"
	"github.com/jackie8tao/mediahelper/internal/pipeline"
)

type mockParser struct {
	parseFunc      func(ctx context.Context, raw string) (*MediaInfo, error)
	parseFilesFunc func(ctx context.Context, rawTitle string, files []string) (*MediaInfo, error)
}

func (m *mockParser) Name() string { return "MockParser" }

func (m *mockParser) Parse(ctx context.Context, raw string) (*MediaInfo, error) {
	if m.parseFunc != nil {
		return m.parseFunc(ctx, raw)
	}
	return &MediaInfo{
		Title:     raw,
		MediaType: MediaTypeMovie,
	}, nil
}

func (m *mockParser) ParseFiles(ctx context.Context, rawTitle string, files []string) (*MediaInfo, error) {
	if m.parseFilesFunc != nil {
		return m.parseFilesFunc(ctx, rawTitle, files)
	}
	var fileItems []FileItem
	for i, f := range files {
		fileItems = append(fileItems, FileItem{
			FilePath: f,
			Season:   1,
			Episode:  i + 1,
		})
	}
	return &MediaInfo{
		Title:     rawTitle,
		MediaType: MediaTypeTV,
		Files:     fileItems,
	}, nil
}

func TestParserComponentExecute_BindParams(t *testing.T) {
	comp := &ParserComponent{
		parser: &mockParser{},
		cfg: config.ParserComponentConfig{
			ID:   "test_parser",
			Name: "Test Parser",
		},
	}

	// 1. Single file parsing: Priority 1 raw_title takes precedence over path
	sCtx := pipeline.NewStepContext("exec-1", 1, "step-1", "Step 1", comp.ID(), map[string]any{
		"raw_title": "Inception.2010.1080p.mkv",
		"path":      "/downloads/other_name.mkv",
	})

	outputs, err := comp.Execute(context.Background(), sCtx)
	if err != nil {
		t.Fatalf("unexpected execute error: %v", err)
	}

	if outputs["title"] != "Inception.2010.1080p.mkv" {
		t.Errorf("expected title to prioritize raw_title 'Inception.2010.1080p.mkv', got %v", outputs["title"])
	}

	files, ok := outputs["files"].([]FileItem)
	if !ok || len(files) != 1 || files[0].FilePath != "/downloads/other_name.mkv" {
		t.Errorf("expected files to contain bound path, got %v", outputs["files"])
	}

	// 2. Multi-file parsing via explicit files array with path
	multiCtx := pipeline.NewStepContext("exec-1", 1, "step-2", "Step 2", comp.ID(), map[string]any{
		"raw_title": "Breaking Bad",
		"path":      "/downloads/Breaking Bad",
		"files":     []string{"S01E01.mkv", "S01E02.mkv"},
	})

	multiOutputs, err := comp.Execute(context.Background(), multiCtx)
	if err != nil {
		t.Fatalf("unexpected execute error: %v", err)
	}

	multiFiles, ok := multiOutputs["files"].([]FileItem)
	if !ok || len(multiFiles) != 2 {
		t.Fatalf("expected 2 files, got %v", multiOutputs["files"])
	}
	if multiFiles[0].Episode != 1 || multiFiles[1].Episode != 2 {
		t.Errorf("unexpected episode mapping: %+v", multiFiles)
	}

	// 3. Directory scanning: Priority 2 extracts title from path when raw_title is empty
	tempDir := t.TempDir()
	subVideo := filepath.Join(tempDir, "01.mkv")
	if err := os.WriteFile(subVideo, []byte("fake video"), 0644); err != nil {
		t.Fatal(err)
	}

	dirCtx := pipeline.NewStepContext("exec-1", 1, "step-dir", "Dir Step", comp.ID(), map[string]any{
		"path": tempDir,
	})
	dirOutputs, err := comp.Execute(context.Background(), dirCtx)
	if err != nil {
		t.Fatalf("unexpected execute error with dir path: %v", err)
	}
	if dirOutputs["title"] != filepath.Base(tempDir) {
		t.Errorf("expected title to fall back to filepath.Base(path) '%s', got %v", filepath.Base(tempDir), dirOutputs["title"])
	}
	dirFiles, ok := dirOutputs["files"].([]FileItem)
	if !ok || len(dirFiles) != 1 {
		t.Fatalf("expected 1 scanned file, got %v", dirOutputs["files"])
	}

	// 4. Missing required path parameter must return error even if raw_title is present
	missingPathCtx := pipeline.NewStepContext("exec-1", 1, "step-no-path", "No Path Step", comp.ID(), map[string]any{
		"raw_title": "Inception.2010.1080p.mkv",
	})
	_, err = comp.Execute(context.Background(), missingPathCtx)
	if err == nil {
		t.Errorf("expected error on missing required path, got nil")
	}

	// 5. Missing both raw_title and path
	emptyCtx := pipeline.NewStepContext("exec-1", 1, "step-3", "Step 3", comp.ID(), map[string]any{})
	_, err = comp.Execute(context.Background(), emptyCtx)
	if err == nil {
		t.Errorf("expected error on empty params, got nil")
	}
}

