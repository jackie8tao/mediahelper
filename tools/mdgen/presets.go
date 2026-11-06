package main

// MockFile represents a single mock file to be created.
type MockFile struct {
	RelPath       string // Relative path inside output directory or item directory
	SimulatedSize int64  // Simulated size for sparse file mode (in bytes)
	IsNonMedia    bool   // If true, represents auxiliary non-media files (e.g. .nfo, .srt)
}

// PresetItem represents a test case item (single media file or packaged directory).
type PresetItem struct {
	ID          string     // Unique identifier
	Name        string     // Human readable name
	Category    string     // movie / tv / anime / chinese
	Description string     // Description of the test scenario
	IsDir       bool       // True if this preset is a folder
	DirName     string     // Folder name if IsDir is true
	Files       []MockFile // Files inside this item
}

const (
	Size100MB = 100 * 1024 * 1024
	Size1GB   = 1024 * 1024 * 1024
	Size2GB   = 2 * 1024 * 1024 * 1024
	Size8GB   = 8 * 1024 * 1024 * 1024
	Size15GB  = 15 * 1024 * 1024 * 1024
	Size25GB  = 25 * 1024 * 1024 * 1024
)

// DefaultPresets contains rich, representative test cases covering common BT/PT release naming conventions.
var DefaultPresets = []PresetItem{
	// 1. Movies (Western)
	{
		ID:          "movie-inception-single",
		Name:        "Inception 4K UHD Single File",
		Category:    "movie",
		Description: "Standalone 4K UHD Remux movie file with rich technical tags",
		IsDir:       false,
		Files: []MockFile{
			{RelPath: "Inception.2010.2160p.UHD.BluRay.x265.TrueHD.7.1.Atmos-GROUP.mkv", SimulatedSize: Size25GB},
		},
	},
	{
		ID:          "movie-fight-club-single",
		Name:        "Fight Club 1080p Single File",
		Category:    "movie",
		Description: "Standalone 1080p BluRay movie file with WiKi release naming",
		IsDir:       false,
		Files: []MockFile{
			{RelPath: "Fight.Club.1999.1080p.BluRay.x264.DTS-WiKi.mkv", SimulatedSize: Size8GB},
		},
	},
	{
		ID:          "movie-oppenheimer-packaged",
		Name:        "Oppenheimer Packaged Directory",
		Category:    "movie",
		Description: "Movie released inside a directory with subtitles, sample video, and NFO file",
		IsDir:       true,
		DirName:     "Oppenheimer.2023.IMAX.2160p.UHD.BluRay.x265.TrueHD.7.1.Atmos-TERMiNAL",
		Files: []MockFile{
			{RelPath: "Oppenheimer.2023.IMAX.2160p.UHD.BluRay.x265.TrueHD.7.1.Atmos-TERMiNAL.mkv", SimulatedSize: Size25GB},
			{RelPath: "Sample/sample.mkv", SimulatedSize: Size100MB},
			{RelPath: "Subs/chinese.srt", SimulatedSize: 64 * 1024, IsNonMedia: true},
			{RelPath: "terminal-oppenheimer.nfo", SimulatedSize: 8 * 1024, IsNonMedia: true},
		},
	},

	// 2. TV Series (Western)
	{
		ID:          "tv-breaking-bad-s01",
		Name:        "Breaking Bad Season 1 Batch Pack",
		Category:    "tv",
		Description: "TV series season 1 full season directory with multi-episode S01E0x naming",
		IsDir:       true,
		DirName:     "Breaking.Bad.S01.1080p.BluRay.x265-FLUX",
		Files: []MockFile{
			{RelPath: "Breaking.Bad.S01E01.Pilot.1080p.BluRay.x265.mkv", SimulatedSize: Size2GB},
			{RelPath: "Breaking.Bad.S01E02.Cat's.in.the.Bag.1080p.BluRay.x265.mkv", SimulatedSize: Size2GB},
			{RelPath: "Breaking.Bad.S01E03....And.the.Bag's.in.the.River.1080p.BluRay.x265.mkv", SimulatedSize: Size2GB},
			{RelPath: "Breaking.Bad.S01E04.Cancer.Man.1080p.BluRay.x265.mkv", SimulatedSize: Size2GB},
		},
	},
	{
		ID:          "tv-got-single-episode",
		Name:        "Game of Thrones S08E03 Single Episode",
		Category:    "tv",
		Description: "Standalone single TV episode with streaming platform tags (AMZN WEB-DL)",
		IsDir:       false,
		Files: []MockFile{
			{RelPath: "Game.of.Thrones.S08E03.The.Long.Night.1080p.AMZN.WEB-DL.DDP5.1.H.264-NTb.mkv", SimulatedSize: Size4GB},
		},
	},

	// 3. Anime (Japanese Animation)
	{
		ID:          "anime-frieren-batch",
		Name:        "Sousou no Frieren SubsPlease Batch Pack",
		Category:    "anime",
		Description: "Japanese anime batch pack with release group brackets, CRC32 hash and episode numbering",
		IsDir:       true,
		DirName:     "[SubsPlease] Sousou no Frieren (01-04) (1080p) [Batch]",
		Files: []MockFile{
			{RelPath: "[SubsPlease] Sousou no Frieren - 01 (1080p) [72B6800E].mkv", SimulatedSize: Size1GB},
			{RelPath: "[SubsPlease] Sousou no Frieren - 02 (1080p) [A3D8B091].mkv", SimulatedSize: Size1GB},
			{RelPath: "[SubsPlease] Sousou no Frieren - 03 (1080p) [C9E4D120].mkv", SimulatedSize: Size1GB},
			{RelPath: "[SubsPlease] Sousou no Frieren - 04 (1080p) [F1023847].mkv", SimulatedSize: Size1GB},
		},
	},
	{
		ID:          "anime-spy-family-single",
		Name:        "Spy x Family S02E01 Baha Single Episode",
		Category:    "anime",
		Description: "Anime standalone file with multi-bracket tags and season/episode notation",
		IsDir:       false,
		Files: []MockFile{
			{RelPath: "[Lilith-Raws] Spy x Family - S02E01 [Baha][WEB-DL][1080p][AVC AAC][CHT].mp4", SimulatedSize: Size1GB},
		},
	},

	// 4. Chinese Drama (国产影视)
	{
		ID:          "chinese-knockout-pack",
		Name:        "狂飙 4K WEB-DL Packaged Directory",
		Category:    "chinese",
		Description: "Chinese drama season directory with Chinese title, EPxx episode notation and 4K tags",
		IsDir:       true,
		DirName:     "狂飙.The.Knockout.2023.2160p.WEB-DL.H265.AAC-HDCTV",
		Files: []MockFile{
			{RelPath: "狂飙.EP01.4K.mp4", SimulatedSize: Size2GB},
			{RelPath: "狂飙.EP02.4K.mp4", SimulatedSize: Size2GB},
			{RelPath: "狂飙.EP03.4K.mp4", SimulatedSize: Size2GB},
		},
	},
	{
		ID:          "chinese-long-season-single",
		Name:        "漫长的季节 S01E01 Single File",
		Category:    "chinese",
		Description: "Chinese drama single episode with English & Chinese bilingual title and S01E01 notation",
		IsDir:       false,
		Files: []MockFile{
			{RelPath: "漫长的季节.The.Long.Season.2023.S01E01.1080p.WEB-DL.H264.AAC-MoGu.mp4", SimulatedSize: Size2GB},
		},
	},
}

const Size4GB = 4 * 1024 * 1024 * 1024
