package provider

import "fmt"

// Registry maps provider names to their implementations.
type Registry struct {
	providers map[string]MusicProvider
}

// NewRegistry creates a new provider registry from the given providers.
func NewRegistry(providers ...MusicProvider) *Registry {
	m := make(map[string]MusicProvider, len(providers))
	for _, p := range providers {
		m[p.Name()] = p
	}
	return &Registry{providers: m}
}

// Get returns the provider for the given name, or an error if not found.
func (r *Registry) Get(name string) (MusicProvider, error) {
	p, ok := r.providers[name]
	if !ok {
		return nil, fmt.Errorf("unsupported music provider: %s", name)
	}
	return p, nil
}

// Names returns all registered provider names.
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}
