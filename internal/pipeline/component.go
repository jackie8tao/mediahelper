package pipeline

import (
	"context"
	"fmt"
	"sync"
)

// Component represents a pluggable, reusable workflow step unit.
type Component interface {
	// ID returns the unique component identifier, e.g. "parser", "scraper", "organizer", "notify".
	ID() string
	// Name returns the friendly display name.
	Name() string
	// Description describes the component capability and input/output contracts.
	Description() string
	// Execute performs the step logic and returns the output dictionary.
	Execute(ctx context.Context, sCtx *StepContext) (map[string]any, error)
}

// ComponentRegistry is a thread-safe registry for pipeline components.
type ComponentRegistry struct {
	mu         sync.RWMutex
	components map[string]Component
}

// NewComponentRegistry creates a new component registry.
func NewComponentRegistry() *ComponentRegistry {
	return &ComponentRegistry{
		components: make(map[string]Component),
	}
}

// Register registers a component. Returns an error if the ID is duplicate or invalid.
func (r *ComponentRegistry) Register(c Component) error {
	if c == nil {
		return fmt.Errorf("cannot register nil component")
	}
	id := c.ID()
	if id == "" {
		return fmt.Errorf("component ID cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.components[id]; exists {
		return fmt.Errorf("component with ID '%s' is already registered", id)
	}

	r.components[id] = c
	return nil
}

// Get retrieves a component by its ID.
func (r *ComponentRegistry) Get(id string) (Component, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	c, ok := r.components[id]
	return c, ok
}

// List returns all registered components.
func (r *ComponentRegistry) List() []Component {
	r.mu.RLock()
	defer r.mu.RUnlock()

	list := make([]Component, 0, len(r.components))
	for _, c := range r.components {
		list = append(list, c)
	}
	return list
}

// Unregister removes a component by its ID.
func (r *ComponentRegistry) Unregister(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.components, id)
}

// FuncComponent is a functional wrapper around Component, convenient for tests and inline functions.
type FuncComponent struct {
	id          string
	name        string
	description string
	execFn      func(ctx context.Context, sCtx *StepContext) (map[string]any, error)
}

// NewFuncComponent creates a new functional component.
func NewFuncComponent(
	id string,
	name string,
	description string,
	fn func(ctx context.Context, sCtx *StepContext) (map[string]any, error),
) Component {
	return &FuncComponent{
		id:          id,
		name:        name,
		description: description,
		execFn:      fn,
	}
}

func (f *FuncComponent) ID() string          { return f.id }
func (f *FuncComponent) Name() string        { return f.name }
func (f *FuncComponent) Description() string { return f.description }
func (f *FuncComponent) Execute(ctx context.Context, sCtx *StepContext) (map[string]any, error) {
	if f.execFn == nil {
		return nil, nil
	}
	return f.execFn(ctx, sCtx)
}
