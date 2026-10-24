package kernelports

import (
	"context"
	"database/sql"
	"time"

	"github.com/uptrace/bun"
)

type TxOptions struct {
	Isolation   sql.IsolationLevel
	ReadOnly    bool
	LockTimeout time.Duration
}

type Transactor interface {
	Transact(ctx context.Context, opts TxOptions, fn func(context.Context) error) error
}

type Savepointer interface {
	Savepoint(ctx context.Context, fn func(context.Context) error) error
}

type DBConnection interface {
	DB() *bun.DB
	DBForContext(ctx context.Context) bun.IDB
	WithTx(ctx context.Context, opts TxOptions, fn func(context.Context, bun.Tx) error) error
	HealthCheck(ctx context.Context) error
	IsHealthy(ctx context.Context) bool
	Close() error
}
