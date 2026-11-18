package databasesize

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/datausageports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"go.uber.org/fx"
)

const (
	timeout       = 10 * time.Second
	databaseBytes = "SELECT pg_database_size(current_database())"
)

type Params struct {
	fx.In

	DB *postgres.Connection
}

type Result struct {
	fx.Out

	Source datausageports.Source `group:"managed_data_sources"`
}

type database struct {
	db *postgres.Connection
}

func New(p Params) Result {
	return Result{Source: &database{db: p.DB}}
}

func (d *database) Figure() datausageports.Figure {
	return datausageports.FigureDatabase
}

func (d *database) Timeout() time.Duration {
	return timeout
}

func (d *database) Measure(ctx context.Context) (int64, error) {
	ctx = dbscope.WithSystem(ctx, "measure the size of this managed instance's database")
	return dbtx.Read(ctx, d.db, func(ctx context.Context) (int64, error) {
		size, err := keel.QueryValue[int64](ctx, d.db.Keel(ctx), databaseBytes)
		if err != nil {
			return 0, fmt.Errorf("measure the database size: %w", err)
		}
		return size, nil
	})
}
