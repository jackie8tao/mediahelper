package parser

import "context"

// Parser defines the universal AIGC media title parser interface.
type Parser interface {
	// Name returns the parser client name, e.g. "DeepSeek".
	Name() string
	// Parse extracts structured MediaInfo from a raw filename or torrent title.
	Parse(ctx context.Context, raw string) (*MediaInfo, error)
	// ParseFiles extracts structured MediaInfo and per-file season/episode mappings for a list of video files.
	ParseFiles(ctx context.Context, rawTitle string, files []string) (*MediaInfo, error)
}
