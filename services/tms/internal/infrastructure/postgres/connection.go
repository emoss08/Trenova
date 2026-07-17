package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/observability"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/pkg/domainregistry"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
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
	Metrics   *metrics.Registry `optional:"true"`
}

type Connection struct {
	db       *bun.DB
	cfg      *config.Config
	logger   *observability.ContextLogger
	metrics  *metrics.Registry
	settings connectionSettings
}

type connectionSettings struct {
	component       string
	maxOpenConns    int
	maxIdleConns    int
	connMaxLifetime time.Duration
	connMaxIdleTime time.Duration
	connParams      map[string]any
	registerStats   bool
}

type txContextKey struct{}

func NewConnection(p ConnectionParams) (*Connection, error) {
	return newConnection(p, oltpSettings(p.Config))
}

type ReportingConnection struct {
	*Connection
}

func NewReportingConnection(p ConnectionParams) (*ReportingConnection, error) {
	conn, err := newConnection(p, reportingSettings(p.Config))
	if err != nil {
		return nil, err
	}

	return &ReportingConnection{Connection: conn}, nil
}

func newConnection(p ConnectionParams, settings connectionSettings) (*Connection, error) {
	logger := observability.NewContextLogger(
		p.Logger.With(zap.String("component", settings.component)),
	)

	conn := &Connection{
		cfg:      p.Config,
		logger:   logger,
		metrics:  p.Metrics,
		settings: settings,
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

func oltpSettings(cfg *config.Config) connectionSettings {
	return connectionSettings{
		component:       "postgres",
		maxOpenConns:    cfg.Database.MaxOpenConns,
		maxIdleConns:    cfg.Database.MaxIdleConns,
		connMaxLifetime: cfg.Database.ConnMaxLifetime,
		connMaxIdleTime: cfg.Database.ConnMaxIdleTime,
		connParams: map[string]any{
			"statement_timeout":                   fmt.Sprintf("%dms", max(cfg.Database.GetStatementTimeout().Milliseconds(), 1)),
			"lock_timeout":                        fmt.Sprintf("%dms", max(cfg.Database.GetLockTimeout().Milliseconds(), 1)),
			"idle_in_transaction_session_timeout": fmt.Sprintf("%dms", max(cfg.Database.GetIdleTxTimeout().Milliseconds(), 1)),
		},
		registerStats: true,
	}
}

func reportingSettings(cfg *config.Config) connectionSettings {
	reporting := cfg.GetReportingConfig()

	return connectionSettings{
		component:       "postgres-reporting",
		maxOpenConns:    reporting.GetPoolMaxOpenConns(),
		maxIdleConns:    reporting.GetPoolMaxIdleConns(),
		connMaxLifetime: cfg.Database.ConnMaxLifetime,
		connMaxIdleTime: cfg.Database.ConnMaxIdleTime,
		connParams: map[string]any{
			"statement_timeout":                   fmt.Sprintf("%dms", max(reporting.GetStatementTimeout().Milliseconds(), 1)),
			"lock_timeout":                        fmt.Sprintf("%dms", max(cfg.Database.GetLockTimeout().Milliseconds(), 1)),
			"idle_in_transaction_session_timeout": fmt.Sprintf("%dms", max(cfg.Database.GetIdleTxTimeout().Milliseconds(), 1)),
			"default_transaction_read_only":       "on",
		},
		registerStats: false,
	}
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

	sqldb := sql.OpenDB(pgdriver.NewConnector(
		pgdriver.WithDSN(dsn),
		pgdriver.WithConnParams(c.settings.connParams),
	))

	if c.settings.maxOpenConns > 0 {
		sqldb.SetMaxOpenConns(c.settings.maxOpenConns)
	}
	if c.settings.maxIdleConns > 0 {
		sqldb.SetMaxIdleConns(c.settings.maxIdleConns)
	}
	if c.settings.connMaxLifetime > 0 {
		sqldb.SetConnMaxLifetime(c.settings.connMaxLifetime)
	}
	if c.settings.connMaxIdleTime > 0 {
		sqldb.SetConnMaxIdleTime(c.settings.connMaxIdleTime)
	}

	c.db = bun.NewDB(sqldb, pgdialect.New())
	if c.settings.registerStats && c.metrics != nil && c.metrics.Database != nil {
		c.metrics.Database.RegisterSQLStats(c.db.Stats)
	}

	c.setupHooks()

	c.db.RegisterModel(domainregistry.RegisterManyToManyEntities()...)
	c.db.RegisterModel(domainregistry.RegisterEntities()...)

	if err := c.HealthCheck(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	c.logger.Info(ctx, "PostgreSQL connection established",
		zap.String("database", c.cfg.Database.Name),
		zap.Int("max_open_conns", c.settings.maxOpenConns),
		zap.Int("max_idle_conns", c.settings.maxIdleConns),
	)

	return nil
}

func (c *Connection) setupHooks() {
	if c.cfg.App.IsDevelopment() && c.cfg.App.Debug {
		c.db = c.db.WithQueryHook(bundebug.NewQueryHook(
			bundebug.WithVerbose(c.cfg.Database.Verbose),
			bundebug.FromEnv("BUNDEBUG"),
		))
	}

	c.db = c.db.WithQueryHook(newSlowQueryHook(time.Second, c.logger))
	c.db = c.db.WithQueryHook(
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
	db.RegisterModel(domainregistry.RegisterManyToManyEntities()...)
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

//nolint:govet // existing scoped variable reuse is local and behavior-preserving
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
