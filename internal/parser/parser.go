package parser

import (
	"context"
	"log/slog"
)

// Recognizer extracts unverified candidate metadata from raw text and files using LLM.
type Recognizer interface {
	// Name returns the recognizer engine name, e.g. "DeepSeek".
	Name() string
	// Recognize extracts structured candidate MediaInfo from a raw filename or torrent title.
	Recognize(ctx context.Context, raw string) (*MediaInfo, error)
	// RecognizeFiles extracts structured candidate MediaInfo and per-file season/episode mappings.
	RecognizeFiles(ctx context.Context, rawTitle string, files []string) (*MediaInfo, error)
}

// Matcher verifies and calibrates recognized media info against third-party platforms (e.g. TMDB, Douban).
type Matcher interface {
	// Name returns the matcher provider name, e.g. "TMDB".
	Name() string
	// Match queries the third-party database to verify and calibrate the candidate info.
	Match(ctx context.Context, candidate *MediaInfo) (*MatchResult, error)
}

// Parser defines the top-level media metadata parser interface.
type Parser interface {
	// Name returns the parser client name.
	Name() string
	// Parse extracts and calibrates structured MediaInfo from a raw filename or torrent title.
	Parse(ctx context.Context, raw string) (*MediaInfo, error)
	// ParseFiles extracts and calibrates structured MediaInfo for a list of video files.
	ParseFiles(ctx context.Context, rawTitle string, files []string) (*MediaInfo, error)
}

// StandardParser combines a Recognizer (LLM) and an optional Matcher (e.g. TMDB)
// to provide a two-stage Recognition -> Matching parsing pipeline.
type StandardParser struct {
	recognizer Recognizer
	matcher    Matcher
}

// NewStandardParser creates a new StandardParser instance.
func NewStandardParser(recognizer Recognizer, matcher Matcher) *StandardParser {
	return &StandardParser{
		recognizer: recognizer,
		matcher:    matcher,
	}
}

// Name returns the combined parser name.
func (p *StandardParser) Name() string {
	if p.recognizer == nil {
		return "StandardParser"
	}
	if p.matcher != nil {
		return p.recognizer.Name() + "+" + p.matcher.Name()
	}
	return p.recognizer.Name()
}

// Parse runs LLM recognition followed by third-party matching verification.
func (p *StandardParser) Parse(ctx context.Context, raw string) (*MediaInfo, error) {
	if p.recognizer == nil {
		return nil, nil
	}

	info, err := p.recognizer.Recognize(ctx, raw)
	if err != nil {
		return nil, err
	}

	p.calibrate(ctx, info)
	return info, nil
}

// ParseFiles runs LLM multi-file recognition followed by third-party matching verification.
func (p *StandardParser) ParseFiles(ctx context.Context, rawTitle string, files []string) (*MediaInfo, error) {
	if p.recognizer == nil {
		return nil, nil
	}

	info, err := p.recognizer.RecognizeFiles(ctx, rawTitle, files)
	if err != nil {
		return nil, err
	}

	p.calibrate(ctx, info)
	return info, nil
}

// calibrate verifies candidate info against the configured matcher and applies authoritative values.
func (p *StandardParser) calibrate(ctx context.Context, info *MediaInfo) {
	if p.matcher == nil || info == nil {
		return
	}

	res, err := p.matcher.Match(ctx, info)
	if err != nil {
		slog.Warn("Matcher verification failed, keeping candidate info", "matcher", p.matcher.Name(), "title", info.Title, "error", err)
		info.Verified = false
		return
	}

	if res == nil || !res.Matched {
		slog.Debug("No authoritative match found on platform", "matcher", p.matcher.Name(), "title", info.Title)
		info.Verified = false
		return
	}

	// Apply authoritative metadata from matcher
	if res.Title != "" {
		info.Title = res.Title
	}
	if res.OriginalTitle != "" && info.EnTitle == "" {
		info.EnTitle = res.OriginalTitle
	}
	if res.Year > 0 {
		info.Year = res.Year
	}
	if res.MediaType != "" {
		info.MediaType = res.MediaType
	}
	info.TMDBID = res.TMDBID
	info.IMDbID = res.IMDbID
	info.Verified = true
	info.MatchedBy = res.Provider
	info.Confidence = res.Confidence

	slog.Info("Calibrated media info via matcher",
		"matcher", p.matcher.Name(),
		"title", info.Title,
		"year", info.Year,
		"media_type", info.MediaType,
		"tmdb_id", info.TMDBID,
		"confidence", info.Confidence)
}
