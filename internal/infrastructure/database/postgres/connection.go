package postgres

import (
	"context"
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
	DBConnStringEmpty = eris.New("database connection string is empty")
	DBConfigNil       = eris.New("database config is nil")
	AppConfigNil      = eris.New("application config is nil")
)

type connection struct {
	cfg  *config.Manager
	log  *zerolog.Logger
	db   *bun.DB
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
		return nil, DBConnStringEmpty
	}

	appCfg := c.cfg.App()
	if appCfg == nil {
		return nil, AppConfigNil
	}

	dbCfg := c.cfg.Database()
	if dbCfg == nil {
		return nil, DBConfigNil
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

func (c *connection) ConnectionInfo(ctx context.Context) (*db.ConnectionInfo, error) {
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

	// Close the connection pool
	if c.pool != nil {
		c.pool.Close()
	}

	// Close the database connection
	if c.db != nil {
		if err := c.db.Close(); err != nil {
			return eris.Wrap(err, "failed to close database connection")
		}
	}

	return nil
}
