package postgres

import (
	"context"
	"database/sql"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/pkg/config"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/registry"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/extra/bundebug"
	"go.uber.org/fx"
)

type ConnectionParams struct {
	fx.In

	Config *config.Manager
	Logger *logger.Logger
	LC     fx.Lifecycle
}

var (
	ErrDBConnStringEmpty = eris.New("database connection string is empty")
	ErrDBConfigNil       = eris.New("database config is nil")
	ErrAppConfigNil      = eris.New("application config is nil")
)

type connection struct {
	cfg  *config.Manager
	log  *zerolog.Logger
	db   *bun.DB
	sql  *sql.DB
	pool *pgxpool.Pool
	mu   sync.RWMutex
}

func NewConnection(p ConnectionParams) db.Connection {
	log := p.Logger.With().
		Str("component", "postgres").
		Str("service", "connection").
		Logger()

	conn := &connection{
		cfg: p.Config,
		log: &log,
	}

	p.LC.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			_, err := conn.DB(ctx)
			if err != nil {
				return err
			}

			return nil
		},
		OnStop: func(context.Context) error {
			return conn.Close()
		},
	})

	return conn
}

func (c *connection) DB(ctx context.Context) (*bun.DB, error) {
	c.mu.RLock()
	if c.db != nil {
		defer c.mu.RUnlock()
		return c.db, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if c.db != nil {
		return c.db, nil
	}

	dsn := c.cfg.GetDSN()
	if dsn == "" {
		return nil, ErrDBConnStringEmpty
	}

	appCfg := c.cfg.App()
	if appCfg == nil {
		return nil, ErrAppConfigNil
	}

	dbCfg := c.cfg.Database()
	if dbCfg == nil {
		return nil, ErrDBConfigNil
	}

	// Parse the database connection string
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, eris.Wrap(err, "failed to parse database config")
	}

	cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	// Setup connection pool
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, eris.Wrap(err, "failed to create database pool")
	}
	c.pool = pool

	sqldb := stdlib.OpenDBFromPool(pool)
	c.sql = sqldb

	bunDB := bun.NewDB(sqldb, pgdialect.New(), bun.WithDiscardUnknownColumns())

	// If the environment is development and debug is enabled, add a query hook to the database
	if appCfg.Environment == "development" && dbCfg.Debug {
		bunDB.AddQueryHook(bundebug.NewQueryHook(
			bundebug.WithVerbose(true),
			bundebug.FromEnv("BUNDEBUG"),
		))
	}

	bunDB.RegisterModel(registry.RegisterEntities()...)

	// Configure connection pool settings
	sqldb.SetConnMaxIdleTime(time.Duration(dbCfg.ConnMaxIdleTime) * time.Second)
	sqldb.SetMaxOpenConns(dbCfg.MaxConnections)
	sqldb.SetMaxIdleConns(dbCfg.MaxIdleConns)
	sqldb.SetConnMaxLifetime(time.Duration(dbCfg.ConnMaxLifetime) * time.Second)

	// Verify connection
	if err = bunDB.PingContext(ctx); err != nil {
		return nil, eris.Wrap(err, "failed to ping database")
	}

	c.db = bunDB
	c.log.Info().Msg("🚀 Established connection to Postgres database!")

	return c.db, nil
}

func (c *connection) ConnectionInfo() (*db.ConnectionInfo, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return &db.ConnectionInfo{
		Host:     c.cfg.Database().Host,
		Port:     c.cfg.Database().Port,
		Database: c.cfg.Database().Database,
		Username: c.cfg.Database().Username,
		Password: c.cfg.Database().Password,
		SSLMode:  c.cfg.Database().SSLMode,
	}, nil
}

func (c *connection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.log.Info().Msg("closing PostgreSQL database connection")

	// First, we'll mark everything as nil after we're done to prevent any lingering references
	defer func() {
		c.db = nil
		c.sql = nil
		c.pool = nil
		c.log.Info().Msg("PostgreSQL database connection resources cleared")
	}()

	// Use a very short timeout for the whole close operation
	closeCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Create a channel to signal completion
	done := make(chan struct{})

	// Do all closing operations in a goroutine
	go func() {
		defer close(done)

		// Close the database connection first
		if c.db != nil {
			c.log.Debug().Msg("closing database connection")
			if err := c.db.Close(); err != nil {
				c.log.Error().Err(err).Msg("error closing database connection")
			} else {
				c.log.Debug().Msg("database connection closed successfully")
			}
		}

		// Then close the connection pool
		if c.pool != nil {
			c.log.Debug().Msg("closing connection pool")
			c.pool.Close()
			c.log.Debug().Msg("connection pool closed")
		}
	}()

	// Wait for completion or timeout
	select {
	case <-done:
		c.log.Info().Msg("PostgreSQL database connection closed successfully")
	case <-closeCtx.Done():
		c.log.Warn().Msg("PostgreSQL database connection close timed out, forcing shutdown")
	}

	return nil
}

func (c *connection) SQLDB(_ context.Context) (*sql.DB, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.db == nil {
		return nil, eris.New("database connection is not initialized")
	}

	return c.db.DB, nil
}
