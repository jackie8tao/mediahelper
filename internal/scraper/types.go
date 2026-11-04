package scraper

// Actor represents a cast member in a media item.
type Actor struct {
	Name  string `json:"name"`  // Actor full name
	Role  string `json:"role"`  // Character role name
	Thumb string `json:"thumb"` // Profile image URL
}

// MediaMetadata holds standardized multimedia metadata scraped from external providers.
type MediaMetadata struct {
	TMDBID        int      `json:"tmdb_id"`        // TMDB ID
	IMDbID        string   `json:"imdb_id"`        // IMDb ID (e.g. tt15398776)
	MediaType     string   `json:"media_type"`     // Media type: movie / tv / anime
	Title         string   `json:"title"`          // Official localized title
	OriginalTitle string   `json:"original_title"` // Original language title
	SortTitle     string   `json:"sort_title"`     // Sort title
	Year          int      `json:"year"`           // Release year
	Premiered     string   `json:"premiered"`      // Premiere date (YYYY-MM-DD)
	Plot          string   `json:"plot"`           // Overview / plot summary
	Rating        float64  `json:"rating"`         // Rating score (0.0 - 10.0)
	Votes         int      `json:"votes"`          // Rating vote count
	Genres        []string `json:"genres"`         // Genre tags
	Studios       []string `json:"studios"`        // Production companies / studios
	Directors     []string `json:"directors"`      // Director names
	Actors        []Actor  `json:"actors"`         // Principal cast list
	PosterURL     string   `json:"poster_url"`     // Full poster image URL
	BackdropURL   string   `json:"backdrop_url"`   // Full backdrop / fanart URL

	// Episode-specific fields for TV shows and anime
	Season        int     `json:"season"`         // Season number
	Episode       int     `json:"episode"`        // Episode number
	EpisodeTitle  string  `json:"episode_title"`  // Episode title
	EpisodePlot   string  `json:"episode_plot"`   // Episode overview
	EpisodeRating float64 `json:"episode_rating"` // Episode rating score
	EpisodeAired  string  `json:"episode_aired"`  // Episode air date
	EpisodeThumb  string  `json:"episode_thumb"`  // Episode still / thumbnail URL
}
