package organizer

import (
	"github.com/jackie8tao/mediahelper/internal/parser"
	"github.com/jackie8tao/mediahelper/internal/scraper"
)

// LinkMode represents the file linking or transfer strategy for library organization.
type LinkMode string

const (
	LinkModeHardlink LinkMode = "hardlink" // Hardlink (recommended, zero extra storage, preserves seeding)
	LinkModeCopy     LinkMode = "copy"     // Copy file
	LinkModeMove     LinkMode = "move"     // Move file
	LinkModeSymlink  LinkMode = "symlink"  // Symbolic link
)

// OrganizeTask represents an incoming media file organization task request.
type OrganizeTask struct {
	SourcePath string                 `json:"source_path"` // Absolute local path to the source media file or directory
	MediaInfo  *parser.MediaInfo      `json:"media_info"`  // Parsed media specification info
	Metadata   *scraper.MediaMetadata `json:"metadata"`    // Scraped standardized media metadata
}

// FilePair records the source-to-target file mapping after organization.
type FilePair struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

// OrganizeResult holds the execution result report of an organization task.
type OrganizeResult struct {
	Success     bool       `json:"success"`
	TargetDir   string     `json:"target_dir"`   // Target media library directory
	LinkedFiles []FilePair `json:"linked_files"` // List of successfully organized file pairs
	NFOPath     string     `json:"nfo_path"`     // Generated primary NFO file path
	PosterPath  string     `json:"poster_path"`  // Saved poster image path
	FanartPath  string     `json:"fanart_path"`  // Saved backdrop/fanart image path
	Error       string     `json:"error,omitempty"`
}
