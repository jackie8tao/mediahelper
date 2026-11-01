package downloader

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

// Downloader defines the universal downloader client interface.
type Downloader interface {
	// Name returns the downloader client name, e.g. "qBittorrent"
	Name() string
}
