package organizer

// LinkMode represents the file linking or transfer strategy for library organization.
type LinkMode string

const (
	LinkModeHardlink LinkMode = "hardlink" // Hardlink (recommended, zero extra storage, preserves seeding)
	LinkModeCopy     LinkMode = "copy"     // Copy file
	LinkModeMove     LinkMode = "move"     // Move file
	LinkModeSymlink  LinkMode = "symlink"  // Symbolic link
)

// EpisodeFileItem represents an explicit file-to-episode mapping for multi-file season pack organization.
type EpisodeFileItem struct {
	File         string `json:"file"`                    // Relative file path within SourcePath (or absolute file path)
	Season       int    `json:"season"`                  // Season number (1+)
	Episode      int    `json:"episode"`                 // Episode number (1+)
	EpisodeTitle string `json:"episode_title,omitempty"` // Optional episode title
}

// OrganizeTask represents an incoming media file organization task request.
type OrganizeTask struct {
	SourcePath   string            `json:"source_path"`             // Absolute local path to the source media file or directory
	MediaType    string            `json:"media_type"`              // Media category: movie / tv / anime
	Title        string            `json:"title"`                   // Primary media title used for directory and file naming
	Year         int               `json:"year"`                    // Release year
	Season       int               `json:"season,omitempty"`        // Season number (for single-episode TV / anime)
	Episode      int               `json:"episode,omitempty"`       // Episode number (for single-episode TV / anime)
	EpisodeTitle string            `json:"episode_title,omitempty"` // Optional episode title
	Resolution   string            `json:"resolution,omitempty"`    // Optional video resolution tag (e.g. 1080p, 4K)
	Files        []EpisodeFileItem `json:"files,omitempty"`         // LLM-parsed file-to-episode mappings for multi-file directories
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
	Error       string     `json:"error,omitempty"`
}
