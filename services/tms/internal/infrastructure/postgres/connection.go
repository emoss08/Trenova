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
	"github.com/emoss08/trenova/pkg/dbdialect"
	"github.com/emoss08/trenova/pkg/domainregistry"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bundebug"
	"github.com/uptrace/bun/extra/bunotel"
	"github.com/uptrace/bun/schema"
	"go.opentelemetry.io/otel"
	"go.uber.org/fx"
	"go.uber.org/zap"

	_ "modernc.org/sqlite"
)

const (
	sqliteDriverName = "sqlite"

	sqliteMaxOpenConns = 8
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
			"statement_timeout": fmt.Sprintf(
				"%dms",
				max(cfg.Database.GetStatementTimeout().Milliseconds(), 1),
			),
			"lock_timeout": fmt.Sprintf(
				"%dms",
				max(cfg.Database.GetLockTimeout().Milliseconds(), 1),
			),
			"idle_in_transaction_session_timeout": fmt.Sprintf(
				"%dms",
				max(cfg.Database.GetIdleTxTimeout().Milliseconds(), 1),
			),
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
			"statement_timeout": fmt.Sprintf(
				"%dms",
				max(reporting.GetStatementTimeout().Milliseconds(), 1),
			),
			"lock_timeout": fmt.Sprintf(
				"%dms",
				max(cfg.Database.GetLockTimeout().Milliseconds(), 1),
			),
			"idle_in_transaction_session_timeout": fmt.Sprintf(
				"%dms",
				max(cfg.Database.GetIdleTxTimeout().Milliseconds(), 1),
			),
		},
		registerStats: false,
	}
}

func (c *Connection) connect(ctx context.Context) error {
	dsn := c.cfg.GetDSN(c.cfg.Database.Password)
	maskedDSN := c.cfg.GetDSNMasked()
	dialect := c.cfg.Database.GetDialect()

	c.logger.Debug(
		ctx, "Connecting to database",
		zap.String("driver", dialect.String()),
		zap.String("dsn", maskedDSN),
	)

	sqldb, bunDialect, err := c.openDB(dialect, dsn)
	if err != nil {
		return err
	}

	c.applyPoolSettings(dialect, sqldb)

	c.db = bun.NewDB(sqldb, bunDialect)
	if c.settings.registerStats && c.metrics != nil && c.metrics.Database != nil {
		c.metrics.Database.RegisterSQLStats(c.db.Stats)
	}

	c.setupHooks()

	c.db.RegisterModel(domainregistry.RegisterManyToManyEntities()...)
	c.db.RegisterModel(domainregistry.RegisterEntities()...)

	if err = c.HealthCheck(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	c.logger.Info(
		ctx, "Database connection established",
		zap.String("driver", dialect.String()),
		zap.String("database", c.databaseName()),
		zap.Int("max_open_conns", c.settings.maxOpenConns),
		zap.Int("max_idle_conns", c.settings.maxIdleConns),
	)

	return nil
}

func (c *Connection) openDB(
	dialect dbdialect.Kind,
	dsn string,
) (*sql.DB, schema.Dialect, error) {
	if dialect.IsSQLite() {
		sqldb, err := openSQLiteDB(dsn)
		if err != nil {
			return nil, nil, err
		}

		return sqldb, sqlitedialect.New(), nil
	}

	sqldb := sql.OpenDB(pgdriver.NewConnector(
		pgdriver.WithDSN(dsn),
		pgdriver.WithConnParams(c.settings.connParams),
	))

	return sqldb, pgdialect.New(), nil
}

func (c *Connection) applyPoolSettings(dialect dbdialect.Kind, sqldb *sql.DB) {
	maxOpenConns := c.settings.maxOpenConns
	maxIdleConns := c.settings.maxIdleConns

	if dialect.IsSQLite() {
		maxOpenConns = min(maxOpenConns, sqliteMaxOpenConns)
		maxIdleConns = min(maxIdleConns, maxOpenConns)
	}

	if maxOpenConns > 0 {
		sqldb.SetMaxOpenConns(maxOpenConns)
	}
	if maxIdleConns > 0 {
		sqldb.SetMaxIdleConns(maxIdleConns)
	}
	if c.settings.connMaxLifetime > 0 {
		sqldb.SetConnMaxLifetime(c.settings.connMaxLifetime)
	}
	if c.settings.connMaxIdleTime > 0 {
		sqldb.SetConnMaxIdleTime(c.settings.connMaxIdleTime)
	}

	c.settings.maxOpenConns = maxOpenConns
	c.settings.maxIdleConns = maxIdleConns
}

func (c *Connection) databaseName() string {
	if c.cfg.Database.GetDialect().IsSQLite() {
		return c.cfg.Database.SQLite.GetPath()
	}

	return c.cfg.Database.Name
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
			bunotel.WithDBName(c.databaseName()),
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

	supportsLockTimeout := c.cfg == nil || c.cfg.Database.GetDialect().IsPostgres()

	if existingTx, ok := ctx.Value(txContextKey{}).(bun.Tx); ok {
		if opts.LockTimeout > 0 && supportsLockTimeout {
			if err := applyLockTimeout(ctx, existingTx, opts.LockTimeout); err != nil {
				return err
			}
		}

		return fn(ctx, existingTx)
	}

	tx, err := c.db.BeginTx(ctx, c.txOptions(opts))
	if err != nil {
		return err
	}

	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if opts.LockTimeout > 0 && supportsLockTimeout {
		if err = applyLockTimeout(ctx, tx, opts.LockTimeout); err != nil {
			return err
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

func applyLockTimeout(ctx context.Context, tx bun.Tx, timeout time.Duration) error {
	lockTimeoutMS := max(timeout.Milliseconds(), 1)
	query := fmt.Sprintf("SET LOCAL lock_timeout = '%dms'", lockTimeoutMS)

	if _, err := tx.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("set local lock_timeout: %w", err)
	}

	return nil
}

func (c *Connection) txOptions(opts ports.TxOptions) *sql.TxOptions {
	if c.cfg != nil && c.cfg.Database.GetDialect().IsSQLite() {
		return &sql.TxOptions{ReadOnly: opts.ReadOnly}
	}

	return &sql.TxOptions{
		Isolation: opts.Isolation,
		ReadOnly:  opts.ReadOnly,
	}
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

// NowEpoch is the current-Unix-timestamp SQL for the connection's dialect.
func (c *Connection) NowEpoch() string {
	if c.cfg == nil {
		return dbdialect.DefaultKind.NowEpoch()
	}

	return c.cfg.Database.GetDialect().NowEpoch()
}
