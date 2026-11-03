package parser

// MediaType represents the category of a media resource.
type MediaType string

const (
	MediaTypeMovie MediaType = "movie"
	MediaTypeTV    MediaType = "tv"
	MediaTypeAnime MediaType = "anime"
)

// MediaInfo holds structured metadata extracted from a media filename or torrent title.
type MediaInfo struct {
	OriginalText string    `json:"original_text"` // Raw input filename or torrent title
	MediaType    MediaType `json:"media_type"`    // Media type: movie / tv / anime

	// Title metadata
	Title   string `json:"title"`    // Primary title (Chinese preferred; falls back to English)
	CnTitle string `json:"cn_title"` // Cleaned Chinese title (if present)
	EnTitle string `json:"en_title"` // Cleaned English title (if present)
	Year    int    `json:"year"`     // Release year (e.g. 2023, or 0 if unrecognized)

	// Season and episode metadata (for TV shows / anime)
	Season  int `json:"season"`  // Season number (0 for movies, defaults to 1 for TV/anime)
	Episode int `json:"episode"` // Episode number (0 for movies)

	// Technical specification metadata
	Resolution string `json:"resolution"`  // 4K, 1080p, 720p, 480p
	Source     string `json:"source"`      // BluRay, WEB-DL, HDTV, Remux, DVD
	VideoCodec string `json:"video_codec"` // HEVC, H.264, AV1, VC-1
	AudioCodec string `json:"audio_codec"` // Atmos, TrueHD, DTS-HD MA, DTS, DDP5.1, AAC, FLAC
}
