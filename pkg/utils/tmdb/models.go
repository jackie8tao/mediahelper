package tmdb

// SearchMovieResponse represents the TMDB search movie response.
type SearchMovieResponse struct {
	Page         int           `json:"page"`
	Results      []MovieResult `json:"results"`
	TotalPages   int           `json:"total_pages,omitempty"`
	TotalResults int           `json:"total_results,omitempty"`
}

// MovieResult represents a single movie summary in TMDB search results.
type MovieResult struct {
	ID            int     `json:"id"`
	Title         string  `json:"title"`
	OriginalTitle string  `json:"original_title"`
	ReleaseDate   string  `json:"release_date"`
	Overview      string  `json:"overview"`
	VoteAverage   float64 `json:"vote_average"`
	VoteCount     int     `json:"vote_count"`
	PosterPath    string  `json:"poster_path"`
	BackdropPath  string  `json:"backdrop_path"`
}

// SearchTVResponse represents the TMDB search TV response.
type SearchTVResponse struct {
	Page         int        `json:"page"`
	Results      []TVResult `json:"results"`
	TotalPages   int        `json:"total_pages,omitempty"`
	TotalResults int        `json:"total_results,omitempty"`
}

// TVResult represents a single TV show summary in TMDB search results.
type TVResult struct {
	ID           int     `json:"id"`
	Name         string  `json:"name"`
	OriginalName string  `json:"original_name"`
	FirstAirDate string  `json:"first_air_date"`
	Overview     string  `json:"overview"`
	VoteAverage  float64 `json:"vote_average"`
	VoteCount    int     `json:"vote_count"`
	PosterPath   string  `json:"poster_path"`
	BackdropPath string  `json:"backdrop_path"`
}

// Genre represents a media genre.
type Genre struct {
	ID   int    `json:"id,omitempty"`
	Name string `json:"name"`
}

// ProductionCompany represents a media production studio.
type ProductionCompany struct {
	ID            int    `json:"id,omitempty"`
	Name          string `json:"name"`
	LogoPath      string `json:"logo_path,omitempty"`
	OriginCountry string `json:"origin_country,omitempty"`
}

// CastMember represents an actor/cast member in credits.
type CastMember struct {
	ID          int    `json:"id,omitempty"`
	Name        string `json:"name"`
	Character   string `json:"character"`
	ProfilePath string `json:"profile_path"`
}

// CrewMember represents a crew member (e.g., director, writer) in credits.
type CrewMember struct {
	ID   int    `json:"id,omitempty"`
	Name string `json:"name"`
	Job  string `json:"job"`
}

// Credits represents cast and crew information.
type Credits struct {
	Cast []CastMember `json:"cast"`
	Crew []CrewMember `json:"crew"`
}

// MovieDetails represents detailed information for a movie from TMDB.
type MovieDetails struct {
	ID                  int                 `json:"id"`
	Title               string              `json:"title"`
	OriginalTitle       string              `json:"original_title"`
	ReleaseDate         string              `json:"release_date"`
	Overview            string              `json:"overview"`
	VoteAverage         float64             `json:"vote_average"`
	VoteCount           int                 `json:"vote_count"`
	PosterPath          string              `json:"poster_path"`
	BackdropPath        string              `json:"backdrop_path"`
	IMDbID              string              `json:"imdb_id"`
	Genres              []Genre             `json:"genres"`
	ProductionCompanies []ProductionCompany `json:"production_companies"`
	Credits             Credits             `json:"credits"`
}

// SeasonSummary represents season summary embedded in TV details.
type SeasonSummary struct {
	SeasonNumber int `json:"season_number"`
	EpisodeCount int `json:"episode_count"`
}

// TVDetails represents detailed information for a TV series from TMDB.
type TVDetails struct {
	ID                  int                 `json:"id"`
	Name                string              `json:"name"`
	OriginalName        string              `json:"original_name"`
	FirstAirDate        string              `json:"first_air_date"`
	Overview            string              `json:"overview"`
	VoteAverage         float64             `json:"vote_average"`
	VoteCount           int                 `json:"vote_count"`
	PosterPath          string              `json:"poster_path"`
	BackdropPath        string              `json:"backdrop_path"`
	Genres              []Genre             `json:"genres"`
	ProductionCompanies []ProductionCompany `json:"production_companies"`
	Credits             Credits             `json:"credits"`
	Seasons             []SeasonSummary     `json:"seasons"`
}

// SeasonDetails represents episode details for a specific season.
type SeasonDetails struct {
	SeasonNumber int           `json:"season_number"`
	Episodes     []EpisodeItem `json:"episodes"`
}

// EpisodeItem represents an individual episode item in a season.
type EpisodeItem struct {
	EpisodeNumber int     `json:"episode_number"`
	Name          string  `json:"name"`
	Overview      string  `json:"overview"`
	VoteAverage   float64 `json:"vote_average"`
	AirDate       string  `json:"air_date"`
	StillPath     string  `json:"still_path"`
}
