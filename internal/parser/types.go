package parser

import "encoding/json"

// MediaType represents the category of a media resource.
type MediaType string

const (
	MediaTypeMovie MediaType = "movie"
	MediaTypeTV    MediaType = "tv"
	MediaTypeAnime MediaType = "anime"
)

// FileItem holds a parsed media file path with its season and episode numbers.
type FileItem struct {
	FilePath string `json:"file_path"` // File path (relative to the scanned directory in LLM output, absolute after resolution)
	Season   int    `json:"season"`    // Season number (1+)
	Episode  int    `json:"episode"`   // Episode number (1+)
}

// ParseTask represents incoming parameters for a media title or directory parsing step.
type ParseTask struct {
	RawTitle string   `json:"raw_title"` // Raw filename or torrent title to parse
	Path     string   `json:"path"`      // Local filesystem directory or file path
	Files    []string `json:"files"`     // Explicit list of relative video file paths (optional)
}

// UnmarshalJSON supports unmarshaling from both "raw_title" and legacy "title".
func (t *ParseTask) UnmarshalJSON(data []byte) error {
	type alias ParseTask
	var aux struct {
		alias
		LegacyTitle string `json:"title"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*t = ParseTask(aux.alias)
	if t.RawTitle == "" && aux.LegacyTitle != "" {
		t.RawTitle = aux.LegacyTitle
	}
	return nil
}

// MediaInfo holds structured metadata extracted from a media filename, torrent title, or directory file list.
type MediaInfo struct {
	OriginalText string    `json:"original_text"` // Raw input filename or torrent title
	MediaType    MediaType `json:"media_type"`    // Media type: movie / tv / anime

	// Title metadata
	Title   string `json:"title"`    // Primary title (Chinese preferred; falls back to English)
	CnTitle string `json:"cn_title"` // Cleaned Chinese title (if present)
	EnTitle string `json:"en_title"` // Cleaned English title (if present)
	Year    int    `json:"year"`     // Release year (e.g. 2023, or 0 if unrecognized)

	// Technical specification metadata
	Resolution string `json:"resolution"`  // 4K, 1080p, 720p, 480p
	Source     string `json:"source"`      // BluRay, WEB-DL, HDTV, Remux, DVD
	VideoCodec string `json:"video_codec"` // HEVC, H.264, AV1, VC-1
	AudioCodec string `json:"audio_codec"` // Atmos, TrueHD, DTS-HD MA, DTS, DDP5.1, AAC, FLAC

	// Multi-file episode mappings (populated when parsing a directory / season pack)
	Files []FileItem `json:"files,omitempty"`
}
