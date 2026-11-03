package parser

import "context"

// Parser defines the universal AIGC media title parser interface.
type Parser interface {
	// Name returns the parser client name, e.g. "DeepSeek"
	Name() string
	// Parse extracts structured MediaInfo from a raw filename or torrent title.
	Parse(ctx context.Context, raw string) (*MediaInfo, error)
}
