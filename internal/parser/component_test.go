package parser

import (
	"context"
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

	// 1. Single file parsing via title parameter
	sCtx := pipeline.NewStepContext("exec-1", 1, "step-1", "Step 1", comp.ID(), map[string]any{
		"title": "Inception.2010.1080p.mkv",
		"path":  "/downloads/Inception.2010.1080p.mkv",
	})

	outputs, err := comp.Execute(context.Background(), sCtx)
	if err != nil {
		t.Fatalf("unexpected execute error: %v", err)
	}

	if outputs["title"] != "Inception.2010.1080p.mkv" {
		t.Errorf("expected title Inception.2010.1080p.mkv, got %v", outputs["title"])
	}

	files, ok := outputs["files"].([]FileItem)
	if !ok || len(files) != 1 || files[0].FilePath != "/downloads/Inception.2010.1080p.mkv" {
		t.Errorf("expected files to contain bound path, got %v", outputs["files"])
	}

	// 2. Multi-file parsing via explicit files array
	multiCtx := pipeline.NewStepContext("exec-1", 1, "step-2", "Step 2", comp.ID(), map[string]any{
		"title": "Breaking Bad",
		"files": []string{"S01E01.mkv", "S01E02.mkv"},
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

	// 3. Missing both title and path
	emptyCtx := pipeline.NewStepContext("exec-1", 1, "step-3", "Step 3", comp.ID(), map[string]any{})
	_, err = comp.Execute(context.Background(), emptyCtx)
	if err == nil {
		t.Errorf("expected error on empty params, got nil")
	}
}
