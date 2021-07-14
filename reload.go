package reload

import (
	"context"
)

// Reloader knows how to reload a resource.
type Reloader interface {
	Reload(ctx context.Context, id string) error
}

//go:generate mockery --case underscore --output internal/reloadmock --outpkg reloadmock --name Reloader

// ReloaderFunc is a helper to create reloaders based on functions.
type ReloaderFunc func(ctx context.Context, id string) error

// Reload satisifies Reloader interface.
func (r ReloaderFunc) Reload(ctx context.Context, id string) error { return r(ctx, id) }

// Notifier knows how to trigger a reload process.
type Notifier interface {
	Notify(ctx context.Context) (string, error)
}

// NotifierFunc is a helper to create notifiers from functions.
type NotifierFunc func(ctx context.Context) (string, error)

// Notify satisifies Notifier interface.
func (n NotifierFunc) Notify(ctx context.Context) (string, error) { return n(ctx) }
