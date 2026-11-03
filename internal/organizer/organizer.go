package organizer

// Organizer defines the universal media library organizer interface.
type Organizer interface {
	// Name returns the organizer implementation name
	Name() string
}
