package tmdb

import "encoding/xml"

// TMDB API response structures

type searchMovieResponse struct {
	Page    int               `json:"page"`
	Results []tmdbMovieResult `json:"results"`
}

type tmdbMovieResult struct {
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

type searchTVResponse struct {
	Page    int            `json:"page"`
	Results []tmdbTVResult `json:"results"`
}

type tmdbTVResult struct {
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

type tmdbCredits struct {
	Cast []struct {
		Name        string `json:"name"`
		Character   string `json:"character"`
		ProfilePath string `json:"profile_path"`
	} `json:"cast"`
	Crew []struct {
		Name string `json:"name"`
		Job  string `json:"job"`
	} `json:"crew"`
}

type tmdbMovieDetails struct {
	ID                  int     `json:"id"`
	Title               string  `json:"title"`
	OriginalTitle       string  `json:"original_title"`
	ReleaseDate         string  `json:"release_date"`
	Overview            string  `json:"overview"`
	VoteAverage         float64 `json:"vote_average"`
	VoteCount           int     `json:"vote_count"`
	PosterPath          string  `json:"poster_path"`
	BackdropPath        string  `json:"backdrop_path"`
	IMDbID              string  `json:"imdb_id"`
	Genres              []struct{ Name string } `json:"genres"`
	ProductionCompanies []struct{ Name string } `json:"production_companies"`
	Credits             tmdbCredits             `json:"credits"`
}

type tmdbTVDetails struct {
	ID                  int     `json:"id"`
	Name                string  `json:"name"`
	OriginalName        string  `json:"original_name"`
	FirstAirDate        string  `json:"first_air_date"`
	Overview            string  `json:"overview"`
	VoteAverage         float64 `json:"vote_average"`
	VoteCount           int     `json:"vote_count"`
	PosterPath          string  `json:"poster_path"`
	BackdropPath        string  `json:"backdrop_path"`
	Genres              []struct{ Name string } `json:"genres"`
	ProductionCompanies []struct{ Name string } `json:"production_companies"`
	Credits             tmdbCredits             `json:"credits"`
	Seasons             []struct {
		SeasonNumber int `json:"season_number"`
		EpisodeCount int `json:"episode_count"`
	} `json:"seasons"`
}

type tmdbSeasonDetails struct {
	SeasonNumber int               `json:"season_number"`
	Episodes     []tmdbEpisodeItem `json:"episodes"`
}

type tmdbEpisodeItem struct {
	EpisodeNumber int     `json:"episode_number"`
	Name          string  `json:"name"`
	Overview      string  `json:"overview"`
	VoteAverage   float64 `json:"vote_average"`
	AirDate       string  `json:"air_date"`
	StillPath     string  `json:"still_path"`
}

// NFO XML structures for Kodi / Jellyfin / Emby

type nfoUniqueID struct {
	XMLName xml.Name `xml:"uniqueid"`
	Type    string   `xml:"type,attr"`
	Default string   `xml:"default,attr,omitempty"`
	Value   string   `xml:",chardata"`
}

type nfoActor struct {
	XMLName xml.Name `xml:"actor"`
	Name    string   `xml:"name"`
	Role    string   `xml:"role,omitempty"`
	Thumb   string   `xml:"thumb,omitempty"`
}

type movieNFO struct {
	XMLName       xml.Name      `xml:"movie"`
	Title         string        `xml:"title"`
	OriginalTitle string        `xml:"originaltitle,omitempty"`
	SortTitle     string        `xml:"sorttitle,omitempty"`
	Year          int           `xml:"year,omitempty"`
	Premiered     string        `xml:"premiered,omitempty"`
	Plot          string        `xml:"plot,omitempty"`
	Rating        float64       `xml:"rating,omitempty"`
	Votes         int           `xml:"votes,omitempty"`
	UniqueIDs     []nfoUniqueID `xml:"uniqueid"`
	Genres        []string      `xml:"genre,omitempty"`
	Studios       []string      `xml:"studio,omitempty"`
	Directors     []string      `xml:"director,omitempty"`
	Actors        []nfoActor    `xml:"actor,omitempty"`
}

type tvshowNFO struct {
	XMLName       xml.Name      `xml:"tvshow"`
	Title         string        `xml:"title"`
	OriginalTitle string        `xml:"originaltitle,omitempty"`
	SortTitle     string        `xml:"sorttitle,omitempty"`
	Year          int           `xml:"year,omitempty"`
	Premiered     string        `xml:"premiered,omitempty"`
	Plot          string        `xml:"plot,omitempty"`
	Rating        float64       `xml:"rating,omitempty"`
	Votes         int           `xml:"votes,omitempty"`
	UniqueIDs     []nfoUniqueID `xml:"uniqueid"`
	Genres        []string      `xml:"genre,omitempty"`
	Studios       []string      `xml:"studio,omitempty"`
	Actors        []nfoActor    `xml:"actor,omitempty"`
}

type episodeNFO struct {
	XMLName   xml.Name      `xml:"episodedetails"`
	Title     string        `xml:"title"`
	Season    int           `xml:"season"`
	Episode   int           `xml:"episode"`
	Plot      string        `xml:"plot,omitempty"`
	Rating    float64       `xml:"rating,omitempty"`
	Aired     string        `xml:"aired,omitempty"`
	UniqueIDs []nfoUniqueID `xml:"uniqueid,omitempty"`
}
