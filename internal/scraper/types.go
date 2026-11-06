package scraper

// Actor represents a cast member in a media item.
type Actor struct {
	Name  string `json:"name"`  // Actor full name
	Role  string `json:"role"`  // Character role name
	Thumb string `json:"thumb"` // Profile image URL
}

// EpisodeMetadata holds metadata for a single television or anime episode.
type EpisodeMetadata struct {
	Season       int     `json:"season"`        // Season number (1+)
	Episode      int     `json:"episode"`       // Episode number (1+)
	Title        string  `json:"title"`         // Episode title
	Plot         string  `json:"plot"`          // Episode overview / plot
	Rating       float64 `json:"rating"`        // Episode rating score (0.0 - 10.0)
	Aired        string  `json:"aired"`         // Air date (YYYY-MM-DD)
	ThumbnailURL string  `json:"thumbnail_url"` // Episode still / thumbnail URL
}

// MediaMetadata holds standardized multimedia metadata scraped from external providers.
type MediaMetadata struct {
	TMDBID        int               `json:"tmdb_id"`             // TMDB ID
	IMDbID        string            `json:"imdb_id"`             // IMDb ID (e.g. tt15398776)
	DoubanID      string            `json:"douban_id,omitempty"` // Douban Subject ID (e.g. 1292052)
	MediaType     string            `json:"media_type"`          // Media type: movie / tv / anime
	Title         string            `json:"title"`               // Official localized title
	OriginalTitle string            `json:"original_title"`      // Original language title
	SortTitle     string            `json:"sort_title"`          // Sort title
	Year          int               `json:"year"`                // Release year
	Premiered     string            `json:"premiered"`           // Premiere date (YYYY-MM-DD)
	Plot          string            `json:"plot"`                // Overview / plot summary
	Rating        float64           `json:"rating"`              // Rating score (0.0 - 10.0)
	Votes         int               `json:"votes"`               // Rating vote count
	Genres        []string          `json:"genres"`              // Genre tags
	Studios       []string          `json:"studios"`             // Production companies / studios
	Directors     []string          `json:"directors"`           // Director names
	Actors        []Actor           `json:"actors"`              // Principal cast list
	PosterURL     string            `json:"poster_url"`          // Full poster image URL
	BackdropURL   string            `json:"backdrop_url"`        // Full backdrop / fanart URL
	Episodes      []EpisodeMetadata `json:"episodes"`            // Episodes metadata (for TV / anime series)
}

// ScrapeTask represents incoming parameters to scrape media metadata and write assets into target_dir.
type ScrapeTask struct {
	TargetDir     string `json:"target_dir"`               // Local directory containing the organized media files (required)
	TMDBID        int    `json:"tmdb_id,omitempty"`        // TMDB unique identifier (required for TMDB scraper)
	DoubanID      string `json:"douban_id,omitempty"`      // Douban subject ID (required for Douban scraper)
	MediaType     string `json:"media_type,omitempty"`     // Media type: movie / tv / anime (optional, default: movie)
	Title         string `json:"title,omitempty"`          // Search or display title (optional)
	Year          int    `json:"year,omitempty"`           // Release year (optional)
	Language      string `json:"language,omitempty"`       // Language code, e.g. "zh-CN" (default: "zh-CN")
	DisableNFO    bool   `json:"disable_nfo,omitempty"`    // Whether to disable generating .nfo files (default: false, meaning NFO is generated)
	DisableImages bool   `json:"disable_images,omitempty"` // Whether to disable downloading poster and backdrop images (default: false)
}

// ScrapeResult represents the final outcome of a scraping operation.
type ScrapeResult struct {
	Success      bool           `json:"success"`
	TargetDir    string         `json:"target_dir"`
	Metadata     *MediaMetadata `json:"metadata,omitempty"`
	CreatedFiles []string       `json:"created_files,omitempty"` // Paths of generated files (e.g. movie.nfo, poster.jpg)
	Error        string         `json:"error,omitempty"`
}
