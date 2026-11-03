package scraper

// Scraper defines the universal metadata scraper client interface.
type Scraper interface {
	// Name returns the scraper provider name, e.g. "TMDB"
	Name() string
}
