// Package shutdown coordinates graceful termination across multiple
// resources (HTTP server, DB pool, message bus, etc.). Hooks run in LIFO
// order so that resources opened last are closed first.
package shutdown

import (
	"context"
	"errors"
	"slices"
	"sync"
)

type Hook struct {
	Name string
	Fn   func(ctx context.Context) error
}

type Registry struct {
	mu    sync.Mutex
	hooks []Hook
}

func NewRegistry() *Registry { return &Registry{} }

func (r *Registry) Register(name string, fn func(ctx context.Context) error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.hooks = append(r.hooks, Hook{Name: name, Fn: fn})
}

// Run executes every hook with the supplied context. It does not abort on
// the first failure — every hook gets a chance to release its resources —
// and returns the joined errors at the end.
func (r *Registry) Run(ctx context.Context) error {
	r.mu.Lock()
	hooks := slices.Clone(r.hooks)
	r.mu.Unlock()

	var errs []error
	for i := len(hooks) - 1; i >= 0; i-- {
		if err := hooks[i].Fn(ctx); err != nil {
			errs = append(errs, &HookError{Name: hooks[i].Name, Err: err})
		}
	}
	return errors.Join(errs...)
}

type HookError struct {
	Name string
	Err  error
}

func (e *HookError) Error() string { return e.Name + ": " + e.Err.Error() }
func (e *HookError) Unwrap() error { return e.Err }
