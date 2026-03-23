package hooks

import "sort"

// Registry holds a collection of hooks.
type Registry struct {
	hooks map[string]*Hook
}

// NewRegistry creates an empty hook registry.
func NewRegistry() *Registry {
	return &Registry{hooks: make(map[string]*Hook)}
}

// Register adds a hook to the registry.
// Panics on duplicate names (programming error).
func (r *Registry) Register(h *Hook) {
	if _, exists := r.hooks[h.Name]; exists {
		panic("duplicate hook: " + h.Name)
	}
	r.hooks[h.Name] = h
}

// Get returns a hook by name, or nil if not found.
func (r *Registry) Get(name string) *Hook {
	return r.hooks[name]
}

// All returns all hooks, sorted by name.
func (r *Registry) All() []*Hook {
	hooks := make([]*Hook, 0, len(r.hooks))
	for _, h := range r.hooks {
		hooks = append(hooks, h)
	}
	sort.Slice(hooks, func(i, j int) bool {
		return hooks[i].Name < hooks[j].Name
	})
	return hooks
}

// BySource returns all hooks with the given source, sorted by name.
func (r *Registry) BySource(s Source) []*Hook {
	var hooks []*Hook
	for _, h := range r.hooks {
		if h.Source == s {
			hooks = append(hooks, h)
		}
	}
	sort.Slice(hooks, func(i, j int) bool {
		return hooks[i].Name < hooks[j].Name
	})
	return hooks
}

// Names returns all hook names, sorted.
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.hooks))
	for name := range r.hooks {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// DefaultRegistry returns a registry pre-loaded with built-in hooks.
func DefaultRegistry() *Registry {
	reg := NewRegistry()
	for _, h := range builtinHooks() {
		reg.Register(h)
	}
	return reg
}
