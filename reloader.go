package reload

import (
	"context"
)

// Reloader knows how to reload a resource.
type Reloader interface {
	Reload(ctx context.Context, id string) error
}

//go:generate mockery --case underscore --output reloadmock --outpkg reloadmock --name Reloader

// ReloaderFunc is a helper to create reloaders based on functions.
type ReloaderFunc func(ctx context.Context, id string) error

// Reload satisifies Reloader interface.
func (r ReloaderFunc) Reload(ctx context.Context, id string) error { return r(ctx, id) }
