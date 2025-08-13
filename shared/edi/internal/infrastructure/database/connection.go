package database

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type DB struct {
	*bun.DB
	logger *zap.Logger
}

type Params struct {
	fx.In
	Config *Config
	Logger *zap.Logger
}

func NewDatabase(lc fx.Lifecycle, params Params) (*DB, error) {
	sqldb := sql.OpenDB(pgdriver.NewConnector(
		pgdriver.WithDSN(params.Config.DSN()),
		pgdriver.WithTimeout(5*params.Config.MaxIdleTime),
		pgdriver.WithReadTimeout(params.Config.MaxIdleTime),
		pgdriver.WithWriteTimeout(params.Config.MaxIdleTime),
	))

	sqldb.SetMaxOpenConns(params.Config.MaxConnections)
	sqldb.SetMaxIdleConns(params.Config.MaxConnections / 2)
	sqldb.SetConnMaxIdleTime(params.Config.MaxIdleTime)
	sqldb.SetConnMaxLifetime(params.Config.ConnMaxLifetime)

	bunDB := bun.NewDB(sqldb, pgdialect.New())

	if err := bunDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db := &DB{
		DB:     bunDB,
		logger: params.Logger,
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			params.Logger.Info("database connection established",
				zap.String("host", params.Config.Host),
				zap.String("database", params.Config.Database),
			)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			params.Logger.Info("closing database connection")
			return bunDB.Close()
		},
	})

	return db, nil
}

func (db *DB) Transaction(ctx context.Context, fn func(tx bun.Tx) error) error {
	return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		return fn(tx)
	})
}