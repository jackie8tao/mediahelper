package tmdb

import "encoding/xml"

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
