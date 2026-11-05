package local

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackie8tao/mediahelper/internal/organizer"
)

func TestLocalFactoryAndName(t *testing.T) {
	cfgJSON := []byte(`{"movie_dir":"/movies","tv_dir":"/tv","link_mode":"copy"}`)
	org, err := LocalFactory(json.RawMessage(cfgJSON))
	if err != nil {
		t.Fatalf("unexpected error creating organizer: %v", err)
	}

	if org.Name() != "LocalOrganizer" {
		t.Errorf("expected name LocalOrganizer, got %s", org.Name())
	}
}

func TestOrganizeMovie(t *testing.T) {
	tempDir := t.TempDir()
	srcDir := filepath.Join(tempDir, "src")
	movieDir := filepath.Join(tempDir, "movies")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}

	srcFile := filepath.Join(srcDir, "Inception.2010.1080p.mkv")
	if err := os.WriteFile(srcFile, []byte("fake video content"), 0644); err != nil {
		t.Fatal(err)
	}
	srcSub := filepath.Join(srcDir, "Inception.2010.1080p.zh.srt")
	if err := os.WriteFile(srcSub, []byte("fake subtitle"), 0644); err != nil {
		t.Fatal(err)
	}

	org := &Organizer{
		cfg: Config{
			MovieDir: movieDir,
			LinkMode: organizer.LinkModeCopy,
		},
	}

	task := organizer.OrganizeTask{
		Title:      "Inception",
		Year:       2010,
		MediaType:  "movie",
		Resolution: "1080p",
		Files: []organizer.FileItem{
			{FilePath: srcFile},
		},
	}

	res, err := org.Organize(context.Background(), task)
	if err != nil {
		t.Fatalf("Organize failed: %v", err)
	}
	if !res.Success {
		t.Fatalf("expected success, got false")
	}

	expectedTargetDir := filepath.Join(movieDir, "Inception (2010)")
	expectedVideoFile := filepath.Join(expectedTargetDir, "Inception (2010) - 1080p.mkv")
	expectedSubFile := filepath.Join(expectedTargetDir, "Inception (2010) - 1080p.zh.srt")

	if _, err := os.Stat(expectedVideoFile); err != nil {
		t.Errorf("expected movie file %s to exist: %v", expectedVideoFile, err)
	}
	if _, err := os.Stat(expectedSubFile); err != nil {
		t.Errorf("expected subtitle file %s to exist: %v", expectedSubFile, err)
	}
}

func TestOrganizeTV(t *testing.T) {
	tempDir := t.TempDir()
	srcDir := filepath.Join(tempDir, "src")
	tvDir := filepath.Join(tempDir, "tv")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}

	srcFile := filepath.Join(srcDir, "Breaking.Bad.S01E01.Pilot.mkv")
	if err := os.WriteFile(srcFile, []byte("fake video content"), 0644); err != nil {
		t.Fatal(err)
	}

	org := &Organizer{
		cfg: Config{
			TVDir:    tvDir,
			LinkMode: organizer.LinkModeCopy,
		},
	}

	task := organizer.OrganizeTask{
		Title:     "Breaking Bad",
		Year:      2008,
		MediaType: "tv",
		Files: []organizer.FileItem{
			{
				FilePath:     srcFile,
				Season:       1,
				Episode:      1,
				EpisodeTitle: "Pilot",
			},
		},
	}

	res, err := org.Organize(context.Background(), task)
	if err != nil {
		t.Fatalf("Organize failed: %v", err)
	}
	if !res.Success {
		t.Fatalf("expected success, got false")
	}

	expectedSeasonDir := filepath.Join(tvDir, "Breaking Bad (2008)", "Season 01")
	expectedEpFile := filepath.Join(expectedSeasonDir, "Breaking Bad - S01E01 - Pilot.mkv")

	if _, err := os.Stat(expectedEpFile); err != nil {
		t.Errorf("expected tv episode file %s to exist: %v", expectedEpFile, err)
	}
}

func TestOrganizeErrors(t *testing.T) {
	org := &Organizer{}

	// Test empty files
	_, err := org.Organize(context.Background(), organizer.OrganizeTask{Title: "Test"})
	if err == nil {
		t.Errorf("expected error on empty files, got nil")
	}

	// Test empty title
	_, err = org.Organize(context.Background(), organizer.OrganizeTask{
		Files: []organizer.FileItem{{FilePath: "/tmp/test.mkv"}},
	})
	if err == nil {
		t.Errorf("expected error on empty title, got nil")
	}

	// Test invalid title with only illegal characters
	_, err = org.Organize(context.Background(), organizer.OrganizeTask{
		Title: "???***",
		Files: []organizer.FileItem{{FilePath: "/tmp/test.mkv"}},
	})
	if err == nil {
		t.Errorf("expected error on title with only illegal characters, got nil")
	}

	tempDir := t.TempDir()
	testVideo := filepath.Join(tempDir, "ep.mkv")
	if err := os.WriteFile(testVideo, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}
	orgWithDir := &Organizer{
		cfg: Config{TVDir: tempDir},
	}

	// Test TV with invalid season
	_, err = orgWithDir.Organize(context.Background(), organizer.OrganizeTask{
		Title:     "Show",
		MediaType: "tv",
		Files: []organizer.FileItem{
			{FilePath: testVideo, Season: 0, Episode: 1},
		},
	})
	if err == nil {
		t.Errorf("expected error on season <= 0, got nil")
	}

	// Test TV with invalid episode
	_, err = orgWithDir.Organize(context.Background(), organizer.OrganizeTask{
		Title:     "Show",
		MediaType: "tv",
		Files: []organizer.FileItem{
			{FilePath: testVideo, Season: 1, Episode: 0},
		},
	})
	if err == nil {
		t.Errorf("expected error on episode <= 0, got nil")
	}
}

