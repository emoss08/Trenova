package validationframework

import (
	"context"
	"errors"

	"github.com/uptrace/bun"
)

var errDBNotInitialized = errors.New("database connection is not initialized")

type ScopedDB interface {
	DBForContext(ctx context.Context) bun.IDB
	RunScoped(ctx context.Context, readOnly bool, fn func(context.Context) error) error
}

type dbSource struct {
	db     bun.IDB
	scoped ScopedDB
}

func (s dbSource) read(ctx context.Context, fn func(context.Context, bun.IDB) error) error {
	if s.scoped != nil {
		return s.scoped.RunScoped(ctx, true, func(ctx context.Context) error {
			db := s.scoped.DBForContext(ctx)
			if db == nil {
				return errDBNotInitialized
			}
			return fn(ctx, db)
		})
	}

	if s.db == nil {
		return errDBNotInitialized
	}

	return fn(ctx, s.db)
}
