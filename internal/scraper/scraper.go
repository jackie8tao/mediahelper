package scraper

import "context"

// Scraper defines the universal metadata scraper client interface.
type Scraper interface {
	// Name returns the scraper provider name, e.g. "TMDB".
	Name() string

	// Scrape fetches media metadata and generates local assets (NFO, images) into task.TargetDir.
	Scrape(ctx context.Context, task ScrapeTask) (*ScrapeResult, error)
}
