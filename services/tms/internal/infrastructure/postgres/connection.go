package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/observability"
	"github.com/emoss08/trenova/pkg/domainregistry"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/extra/bundebug"
	"github.com/uptrace/bun/extra/bunotel"
	"go.opentelemetry.io/otel"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var _ ports.DBConnection = (*Connection)(nil)

type ConnectionParams struct {
	fx.In

	Lifecycle fx.Lifecycle
	Config    *config.Config
	Logger    *zap.Logger
}

type Connection struct {
	db     *bun.DB
	cfg    *config.Config
	logger *observability.ContextLogger
}

type txContextKey struct{}

func NewConnection(p ConnectionParams) (*Connection, error) {
	logger := observability.NewContextLogger(p.Logger.With(zap.String("component", "postgres")))

	conn := &Connection{
		cfg:    p.Config,
		logger: logger,
	}

	p.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return conn.connect(ctx)
		},
		OnStop: func(ctx context.Context) error {
			return conn.shutdown(ctx)
		},
	})

	return conn, nil
}

func (c *Connection) connect(ctx context.Context) error {
	dsn := c.cfg.GetDSN(c.cfg.Database.Password)
	maskedDSN := c.cfg.GetDSNMasked()

	c.logger.Debug(ctx, "Connecting to PostgreSQL",
		zap.String("dsn", maskedDSN),
		zap.String("host", c.cfg.Database.Host),
		zap.Int("port", c.cfg.Database.Port),
		zap.String("database", c.cfg.Database.Name),
	)

	poolCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return fmt.Errorf("failed to parse database configuration: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return fmt.Errorf("failed to create database pool: %w", err)
	}

	sqldb := stdlib.OpenDBFromPool(pool)

	if c.cfg.Database.MaxOpenConns > 0 {
		sqldb.SetMaxOpenConns(c.cfg.Database.MaxOpenConns)
	}
	if c.cfg.Database.MaxIdleConns > 0 {
		sqldb.SetMaxIdleConns(c.cfg.Database.MaxIdleConns)
	}
	if c.cfg.Database.ConnMaxLifetime > 0 {
		sqldb.SetConnMaxLifetime(c.cfg.Database.ConnMaxLifetime)
	}
	if c.cfg.Database.ConnMaxIdleTime > 0 {
		sqldb.SetConnMaxIdleTime(c.cfg.Database.ConnMaxIdleTime)
	}

	c.db = bun.NewDB(sqldb, pgdialect.New())

	c.setupHooks()

	c.db.RegisterModel(domainregistry.RegisterEntities()...)

	if err = c.HealthCheck(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	c.logger.Info(ctx, "PostgreSQL connection established",
		zap.String("database", c.cfg.Database.Name),
		zap.Int("max_open_conns", c.cfg.Database.MaxOpenConns),
		zap.Int("max_idle_conns", c.cfg.Database.MaxIdleConns),
	)

	return nil
}

func (c *Connection) setupHooks() {
	if c.cfg.App.IsDevelopment() && c.cfg.App.Debug {
		c.db.AddQueryHook(bundebug.NewQueryHook(
			bundebug.WithVerbose(c.cfg.Database.Verbose),
			bundebug.FromEnv("BUNDEBUG"),
		))
	}

	c.db.WithQueryHook(
		bunotel.NewQueryHook(
			bunotel.WithDBName(c.cfg.Database.Name),
			bunotel.WithTracerProvider(otel.GetTracerProvider()),
			bunotel.WithFormattedQueries(c.cfg.App.IsDevelopment()),
		),
	)
}

func (c *Connection) shutdown(ctx context.Context) error {
	c.logger.Info(ctx, "Closing PostgreSQL connection")

	if c.db != nil {
		if err := c.db.Close(); err != nil {
			c.logger.Error(ctx, "Failed to close database connection", zap.Error(err))
			return err
		}
	}

	c.logger.Info(ctx, "PostgreSQL connection closed successfully")
	return nil
}

func NewTestConnection(db *bun.DB) *Connection {
	db.RegisterModel(domainregistry.RegisterEntities()...)
	return &Connection{db: db}
}

func (c *Connection) DB() *bun.DB {
	return c.db
}

func (c *Connection) DBForContext(ctx context.Context) bun.IDB {
	if tx, ok := ctx.Value(txContextKey{}).(bun.Tx); ok {
		return tx
	}

	return c.db
}

func (c *Connection) WithTx(
	ctx context.Context,
	opts ports.TxOptions,
	fn func(context.Context, bun.Tx) error,
) (err error) {
	if c.db == nil {
		return ErrDatabaseConnectionNotInitialized
	}

	if existingTx, ok := ctx.Value(txContextKey{}).(bun.Tx); ok {
		if opts.LockTimeout > 0 {
			lockTimeoutMS := max(opts.LockTimeout.Milliseconds(), 1)
			query := fmt.Sprintf("SET LOCAL lock_timeout = '%dms'", lockTimeoutMS)
			if _, err := existingTx.ExecContext(ctx, query); err != nil {
				return fmt.Errorf("set local lock_timeout: %w", err)
			}
		}

		return fn(ctx, existingTx)
	}

	tx, err := c.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: opts.Isolation,
		ReadOnly:  opts.ReadOnly,
	})
	if err != nil {
		return err
	}

	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if opts.LockTimeout > 0 {
		lockTimeoutMS := max(opts.LockTimeout.Milliseconds(), 1)
		query := fmt.Sprintf("SET LOCAL lock_timeout = '%dms'", lockTimeoutMS)
		if _, err = tx.ExecContext(ctx, query); err != nil {
			return fmt.Errorf("set local lock_timeout: %w", err)
		}
	}

	ctx = context.WithValue(ctx, txContextKey{}, tx)

	if err = fn(ctx, tx); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	committed = true
	return nil
}

func (c *Connection) HealthCheck(ctx context.Context) error {
	if c.db == nil {
		return ErrDatabaseConnectionNotInitialized
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return c.db.PingContext(ctx)
}

func (c *Connection) IsHealthy(ctx context.Context) bool {
	return c.HealthCheck(ctx) == nil
}

func (c *Connection) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}
