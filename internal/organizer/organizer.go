package organizer

import "context"

// Organizer defines the universal media library organizer interface.
type Organizer interface {
	// Name returns the organizer implementation name.
	Name() string
	// Organize transfers or links media files from SourcePath into the standardized library directory.
	Organize(ctx context.Context, task OrganizeTask) (*OrganizeResult, error)
}
