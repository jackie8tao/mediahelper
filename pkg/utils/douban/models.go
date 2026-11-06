package douban

// Person represents a director or cast member in Douban.
type Person struct {
	Name      string `json:"name"`
	Character string `json:"character,omitempty"`
	Avatar    struct {
		Large  string `json:"large,omitempty"`
		Normal string `json:"normal,omitempty"`
	} `json:"avatar,omitempty"`
}

// Rating represents rating information in Douban.
type Rating struct {
	Count     int     `json:"count"`
	Max       int     `json:"max"`
	StarCount float64 `json:"star_count"`
	Value     float64 `json:"value"`
}

// ImageDetails represents image dimension and URL metadata.
type ImageDetails struct {
	URL    string `json:"url"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
}

// SubjectDetails represents complete metadata for a Douban movie or TV show.
type SubjectDetails struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	OriginalTitle string   `json:"original_title"`
	Aka           []string `json:"aka"`
	Year          string   `json:"year"`
	Intro         string   `json:"intro"`
	Rating        Rating   `json:"rating"`
	Pic           struct {
		Large  string `json:"large"`
		Normal string `json:"normal"`
	} `json:"pic"`
	Cover struct {
		Image struct {
			Large ImageDetails `json:"large"`
		} `json:"image"`
	} `json:"cover"`
	CoverURL      string   `json:"cover_url"`
	Genres        []string `json:"genres"`
	Countries     []string `json:"countries"`
	Directors     []Person `json:"directors"`
	Actors        []Person `json:"actors"`
	Durations     []string `json:"durations"`
	EpisodesCount int      `json:"episodes_count"`
	Type          string   `json:"type"`
	Subtype       string   `json:"subtype"`
	CardSubtitle  string   `json:"card_subtitle"`
	URL           string   `json:"url"`
}

// PosterURL returns the best available poster image URL for the subject.
func (s *SubjectDetails) PosterURL() string {
	if s == nil {
		return ""
	}
	if s.Cover.Image.Large.URL != "" {
		return s.Cover.Image.Large.URL
	}
	if s.Pic.Large != "" {
		return s.Pic.Large
	}
	return s.CoverURL
}

// PhotoItem represents an individual photo or still.
type PhotoItem struct {
	ID    string `json:"id"`
	Image struct {
		Large ImageDetails `json:"large"`
	} `json:"image"`
}

// PhotosResponse represents the Douban photos API response.
type PhotosResponse struct {
	Total  int         `json:"total"`
	Photos []PhotoItem `json:"photos"`
}

// SearchTarget holds metadata for a search target item.
type SearchTarget struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Year         string `json:"year"`
	CoverURL     string `json:"cover_url"`
	CardSubtitle string `json:"card_subtitle"`
	Rating       Rating `json:"rating"`
}

// SearchItem represents an individual item in Douban search results.
type SearchItem struct {
	TargetType string       `json:"target_type"`
	Target     SearchTarget `json:"target"`
}

// SearchResponse represents the Douban search API response.
type SearchResponse struct {
	Total int          `json:"total"`
	Items []SearchItem `json:"items"`
}

