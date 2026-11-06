package parser

import (
	"context"
	"testing"
)

type mockRecognizer struct {
	recognizeFunc func(ctx context.Context, raw string) (*MediaInfo, error)
}

func (m *mockRecognizer) Name() string { return "MockLLM" }

func (m *mockRecognizer) Recognize(ctx context.Context, raw string) (*MediaInfo, error) {
	if m.recognizeFunc != nil {
		return m.recognizeFunc(ctx, raw)
	}
	return &MediaInfo{
		Title:        "Unverified Movie",
		Year:         2024,
		MediaType:    MediaTypeMovie,
		OriginalText: raw,
	}, nil
}

func (m *mockRecognizer) RecognizeFiles(ctx context.Context, rawTitle string, files []string) (*MediaInfo, error) {
	return m.Recognize(ctx, rawTitle)
}

type mockMatcher struct {
	matchFunc func(ctx context.Context, candidate *MediaInfo) (*MatchResult, error)
}

func (m *mockMatcher) Name() string { return "MockTMDB" }

func (m *mockMatcher) Match(ctx context.Context, candidate *MediaInfo) (*MatchResult, error) {
	if m.matchFunc != nil {
		return m.matchFunc(ctx, candidate)
	}
	return &MatchResult{
		Matched:    true,
		Provider:   "tmdb",
		TMDBID:     999,
		Title:      "Authoritative Movie",
		Year:       2023,
		MediaType:  MediaTypeMovie,
		Confidence: 0.95,
	}, nil
}

func TestStandardParser_ParseWithMatcherCalibration(t *testing.T) {
	parser := NewStandardParser(&mockRecognizer{}, &mockMatcher{})

	if parser.Name() != "MockLLM+MockTMDB" {
		t.Errorf("expected name MockLLM+MockTMDB, got %s", parser.Name())
	}

	info, err := parser.Parse(context.Background(), "dirty.movie.filename.1080p.mkv")
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	// Verify calibration
	if !info.Verified {
		t.Errorf("expected Verified true, got false")
	}
	if info.TMDBID != 999 {
		t.Errorf("expected TMDBID 999, got %d", info.TMDBID)
	}
	if info.Title != "Authoritative Movie" {
		t.Errorf("expected calibrated title Authoritative Movie, got %s", info.Title)
	}
	if info.Year != 2023 {
		t.Errorf("expected calibrated year 2023, got %d", info.Year)
	}
	if info.MatchedBy != "tmdb" {
		t.Errorf("expected MatchedBy tmdb, got %s", info.MatchedBy)
	}
}

func TestStandardParser_ParseWithoutMatcher(t *testing.T) {
	parser := NewStandardParser(&mockRecognizer{}, nil)

	info, err := parser.Parse(context.Background(), "test.mkv")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if info.Verified {
		t.Errorf("expected Verified false without matcher, got true")
	}
	if info.Title != "Unverified Movie" {
		t.Errorf("expected unverified title, got %s", info.Title)
	}
}

func TestStandardParser_MatcherUnmatchedFallback(t *testing.T) {
	unmatchedMatcher := &mockMatcher{
		matchFunc: func(ctx context.Context, candidate *MediaInfo) (*MatchResult, error) {
			return parserMatchResultUnmatched(), nil
		},
	}
	parser := NewStandardParser(&mockRecognizer{}, unmatchedMatcher)

	info, err := parser.Parse(context.Background(), "test.mkv")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if info.Verified {
		t.Errorf("expected Verified false on unmatched, got true")
	}
	if info.Title != "Unverified Movie" {
		t.Errorf("expected original candidate title, got %s", info.Title)
	}
}

func parserMatchResultUnmatched() *MatchResult {
	return &MatchResult{Matched: false}
}
