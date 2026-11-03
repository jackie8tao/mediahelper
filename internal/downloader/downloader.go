package downloader

import "context"

// Torrent represents general metadata of a torrent download task.
type Torrent struct {
	Hash        string   `json:"hash"`
	Name        string   `json:"name"`
	Category    string   `json:"category"`
	Tags        []string `json:"tags"`
	SavePath    string   `json:"save_path"`    // Download directory path as reported by the downloader
	ContentPath string   `json:"content_path"` // Content root path as reported by the downloader
	Progress    float64  `json:"progress"`     // Download progress (0.0 - 1.0)
	State       string   `json:"state"`        // Torrent state (e.g. pausedUP, uploading, downloading)
	Size        int64    `json:"size"`         // Total size in bytes
	IsCompleted bool     `json:"is_completed"` // Whether the torrent has completed downloading
	DlSpeed     int64    `json:"dl_speed"`     // Download speed (B/s)
	UpSpeed     int64    `json:"up_speed"`     // Upload speed (B/s)
	ETA         int64    `json:"eta"`          // Estimated remaining time in seconds
	AddedOn     int64    `json:"added_on"`     // Unix timestamp when added
}

// TorrentFile represents a file contained within a torrent task.
type TorrentFile struct {
	Index    int     `json:"index"`
	Name     string  `json:"name"`     // Relative file path
	Size     int64   `json:"size"`     // File size in bytes
	Progress float64 `json:"progress"` // File completion progress (0.0 - 1.0)
}

// AddTorrentOptions defines parameters for adding a new torrent task.
type AddTorrentOptions struct {
	FilePath string   `json:"file_path,omitempty"` // Local .torrent file path
	URL      string   `json:"url,omitempty"`       // Magnet URI or remote .torrent URL
	SavePath string   `json:"save_path,omitempty"` // Custom download directory
	Category string   `json:"category,omitempty"`  // Category to assign
	Tags     []string `json:"tags,omitempty"`      // Tags to assign
	Paused   bool     `json:"paused,omitempty"`    // Whether to add the torrent in paused state
}

// GetTorrentsOptions defines filtering parameters for querying torrents.
type GetTorrentsOptions struct {
	Hashes   []string `json:"hashes,omitempty"`   // Specific torrent hashes
	Filter   string   `json:"filter,omitempty"`   // Status filter (e.g. all, completed, downloading, paused, active)
	Category string   `json:"category,omitempty"` // Filter by category
	Tag      string   `json:"tag,omitempty"`      // Filter by tag
	Limit    int      `json:"limit,omitempty"`    // Max number of torrents to return (0 for unlimited)
}

// Downloader defines the universal downloader client interface.
type Downloader interface {
	// Name returns the downloader client name, e.g. "qBittorrent".
	Name() string
	// AddTorrent adds a torrent by local file path or URL/magnet link and returns added torrent hashes if available.
	AddTorrent(ctx context.Context, opts AddTorrentOptions) ([]string, error)
	// DeleteTorrent removes one or more torrents and optionally deletes their downloaded data files.
	DeleteTorrent(ctx context.Context, hashes []string, deleteFiles bool) error
	// GetTorrents retrieves a list of torrents matching the given filter options.
	GetTorrents(ctx context.Context, opts GetTorrentsOptions) ([]Torrent, error)
	// AddTags attaches one or more tags to the specified torrents.
	AddTags(ctx context.Context, hashes []string, tags []string) error
	// RemoveTags removes one or more tags from the specified torrents.
	RemoveTags(ctx context.Context, hashes []string, tags []string) error
	// Pause pauses the specified torrents.
	Pause(ctx context.Context, hashes []string) error
	// Resume resumes the specified torrents.
	Resume(ctx context.Context, hashes []string) error
}
