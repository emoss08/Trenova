package ports

import (
	"context"

	"github.com/uptrace/bun"
)

type DBConnection interface {
	DB(ctx context.Context) (*bun.DB, error)
	Close() error
}
