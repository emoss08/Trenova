package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/observability"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbpool"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var (
	_ kernelports.DBConnection = (*Connection)(nil)
	_ kernelports.Transactor   = (*Connection)(nil)
	_ kernelports.Savepointer  = (*Connection)(nil)
)

type ConnectionParams struct {
	fx.In

	Lifecycle fx.Lifecycle
	Config    *config.Config
	Logger    *zap.Logger
	Metrics   *metrics.Registry `optional:"true"`
}

type Connection struct {
	tenantKeel *keel.DB
	systemKeel *keel.DB
	cfg        *config.Config
	logger     *observability.ContextLogger
	metrics    *metrics.Registry
	settings   connectionSettings
	rls        *rlsRuntime
}

type connectionSettings struct {
	component       string
	maxOpenConns    int
	connMaxLifetime time.Duration
	connMaxIdleTime time.Duration
	connParams      map[string]string
	registerStats   bool
}

type txContextKey struct{}

type txBinding struct {
	db    *keel.DB
	scope dbscope.Scope
}

type detachedFromTx struct{}

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

	var rls *rlsRuntime
	if rlsConfigured(p.Config) {
		runtime, err := newRLSRuntime(p.Config, logger, p.Metrics)
		if err != nil {
			return nil, err
		}
		rls = runtime
	}

	conn := &Connection{
		cfg:      p.Config,
		logger:   logger,
		metrics:  p.Metrics,
		settings: settings,
		rls:      rls,
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
		connMaxLifetime: cfg.Database.ConnMaxLifetime,
		connMaxIdleTime: cfg.Database.ConnMaxIdleTime,
		connParams: sessionParams(
			cfg.Database.GetStatementTimeout(),
			cfg.Database.GetLockTimeout(),
			cfg.Database.GetIdleTxTimeout(),
		),
		registerStats: true,
	}
}

func reportingSettings(cfg *config.Config) connectionSettings {
	reporting := cfg.GetReportingConfig()

	return connectionSettings{
		component:       "postgres-reporting",
		maxOpenConns:    reporting.GetPoolMaxOpenConns(),
		connMaxLifetime: cfg.Database.ConnMaxLifetime,
		connMaxIdleTime: cfg.Database.ConnMaxIdleTime,
		connParams: sessionParams(
			reporting.GetStatementTimeout(),
			cfg.Database.GetLockTimeout(),
			cfg.Database.GetIdleTxTimeout(),
		),
		registerStats: false,
	}
}

func sessionParams(statement, lock, idleInTx time.Duration) map[string]string {
	return map[string]string{
		"statement_timeout":                   milliseconds(statement),
		"lock_timeout":                        milliseconds(lock),
		"idle_in_transaction_session_timeout": milliseconds(idleInTx),
	}
}

func milliseconds(d time.Duration) string {
	return fmt.Sprintf("%dms", max(d.Milliseconds(), 1))
}

func (c *Connection) connect(ctx context.Context) error {
	c.logger.Debug(ctx, "Connecting to database", zap.String("dsn", c.cfg.GetDSNMasked()))

	var guard dbpool.Guard
	if c.rls != nil {
		guard = c.rls.guard(scopePoolTenant, !c.cfg.Database.System.Configured())
	}

	tenant, err := c.openPool(ctx, &dbpool.Options{
		DSN:             c.cfg.GetDSN(c.cfg.Database.Password),
		MaxConns:        c.settings.maxOpenConns,
		MaxConnLifetime: c.settings.connMaxLifetime,
		MaxConnIdleTime: c.settings.connMaxIdleTime,
		RuntimeParams:   c.settings.connParams,
		Guard:           guard,
	})
	if err != nil {
		return err
	}
	c.tenantKeel = tenant

	if c.settings.registerStats && c.metrics != nil && c.metrics.Database != nil {
		pool := tenant.Pool()
		c.metrics.Database.RegisterPoolStats(func() metrics.PoolStats { return poolStats(pool) })
	}

	if err = c.HealthCheck(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	if err = c.connectSystem(ctx); err != nil {
		return err
	}

	if err = c.verifyRLS(ctx); err != nil {
		return err
	}

	c.logger.Info(
		ctx, "Database connection established",
		zap.String("database", c.cfg.Database.Name),
		zap.Int32("max_conns", tenant.Pool().Config().MaxConns),
		zap.String("row_level_security", c.rlsMode()),
	)

	if c.cfg.App.IsProduction() && !c.rls.enforced() {
		c.logger.Warn(
			ctx,
			"Row-level security is not enforced; tenant isolation rests on application filters alone",
			zap.String("row_level_security", c.rlsMode()),
		)
	}

	return nil
}

func (c *Connection) openPool(ctx context.Context, opts *dbpool.Options) (*keel.DB, error) {
	slow := newSlowQueryLog(c.logger)
	opts.Observers = c.observers(slow)
	opts.VersionErrors = func(err error) {
		c.logger.Warn(context.Background(), "Failed to record data versions", zap.Error(err))
	}

	db, err := dbpool.Open(ctx, opts)
	if err != nil {
		return nil, err
	}

	slow.pool = db.Pool()

	return db, nil
}

func (c *Connection) observers(slow *slowQueryLog) []keel.Observer {
	observers := make([]keel.Observer, 0, 3)
	if c.cfg.App.IsDevelopment() && c.cfg.App.Debug {
		observers = append(observers, &debugObserver{
			logger:  c.logger.Named("keel"),
			verbose: c.cfg.Database.Verbose,
		})
	}

	return append(observers,
		&slowQueryObserver{log: slow},
		newTracingObserver(c.cfg.Database.Name),
	)
}

func (c *Connection) RunDetached(
	ctx context.Context,
	readOnly bool,
	fn func(context.Context) error,
) error {
	return c.RunScoped(context.WithValue(ctx, txContextKey{}, detachedFromTx{}), readOnly, fn)
}

func (c *Connection) RunScoped(
	ctx context.Context,
	readOnly bool,
	fn func(context.Context) error,
) error {
	return c.Transact(ctx, kernelports.TxOptions{ReadOnly: readOnly}, fn)
}

func (c *Connection) rlsMode() string {
	if c.rls == nil {
		return config.RLSModeOff
	}

	return c.rls.mode
}

func (c *Connection) shutdown(ctx context.Context) error {
	c.logger.Info(ctx, "Closing PostgreSQL connection")
	c.closePools()
	c.logger.Info(ctx, "PostgreSQL connection closed successfully")
	return nil
}

func (c *Connection) closePools() {
	for _, db := range []*keel.DB{c.systemKeel, c.tenantKeel} {
		if db != nil {
			db.Close()
		}
	}
}

func WrapDB(db *keel.DB) *Connection {
	return &Connection{tenantKeel: db}
}

func NewTestConnection(db *keel.DB) *Connection {
	return WrapDB(db)
}

func (c *Connection) Keel(ctx context.Context) *keel.DB {
	if c.systemKeel != nil && dbscope.IsSystem(ctx) {
		return c.systemKeel
	}

	return c.tenantKeel
}

func (c *Connection) joins(ctx context.Context, db *keel.DB) bool {
	binding, ok := ctx.Value(txContextKey{}).(*txBinding)
	if !ok || binding.db != db || !keel.InTx(ctx, db) {
		return false
	}

	return binding.scope.Matches(dbscope.From(ctx))
}

func (c *Connection) Transact(
	ctx context.Context,
	opts kernelports.TxOptions,
	fn func(context.Context) error,
) error {
	db := c.Keel(ctx)
	if db == nil {
		return ErrDatabaseConnectionNotInitialized
	}

	if c.joins(ctx, db) {
		if opts.LockTimeout > 0 {
			if err := applyLockTimeout(ctx, db, opts.LockTimeout); err != nil {
				return err
			}
		}

		if opts.ReadOnly {
			ctx = kernelports.WithReadOnly(ctx)
		}

		if len(opts.Settings) > 0 {
			return keel.Read(ctx, db, fn, txSettings(opts.Settings)...)
		}

		return fn(ctx)
	}

	txOpts, err := keelTxOptions(opts)
	if err != nil {
		return err
	}

	baseCtx := ctx
	if opts.ReadOnly {
		baseCtx = kernelports.WithReadOnly(baseCtx)
	}
	txCtx, hooks := kernelports.WithAfterCommitHooks(baseCtx)
	txCtx = context.WithValue(txCtx, txContextKey{}, &txBinding{db: db, scope: dbscope.From(ctx)})

	run := keel.Write
	if opts.ReadOnly {
		run = keel.Read
	}

	if err = run(txCtx, db, fn, txOpts...); err != nil {
		return err
	}

	hooks.Run(ctx)
	return nil
}

func keelTxOptions(opts kernelports.TxOptions) ([]keel.TxOption, error) {
	txOpts := []keel.TxOption{keel.Detached()}

	switch opts.Isolation {
	case sql.LevelDefault, sql.LevelReadCommitted:
	case sql.LevelRepeatableRead:
		txOpts = append(txOpts, keel.RepeatableRead())
	case sql.LevelSerializable:
		txOpts = append(txOpts, keel.Serializable())
	case sql.LevelReadUncommitted,
		sql.LevelWriteCommitted,
		sql.LevelSnapshot,
		sql.LevelLinearizable:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedIsolation, opts.Isolation)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedIsolation, opts.Isolation)
	}

	if opts.LockTimeout > 0 {
		txOpts = append(txOpts, keel.LockTimeout(opts.LockTimeout))
	}

	return append(txOpts, txSettings(opts.Settings)...), nil
}

func txSettings(settings []kernelports.TxSetting) []keel.TxOption {
	out := make([]keel.TxOption, len(settings))
	for i, s := range settings {
		out[i] = keel.Setting(s.Name, s.Value)
	}

	return out
}

func (c *Connection) Savepoint(ctx context.Context, fn func(context.Context) error) error {
	return dbtx.Savepoint(ctx, c, fn)
}

func applyLockTimeout(ctx context.Context, db *keel.DB, timeout time.Duration) error {
	lockTimeoutMS := max(timeout.Milliseconds(), 1)
	query := fmt.Sprintf("SET LOCAL lock_timeout = '%dms'", lockTimeoutMS)

	if _, err := keel.Exec(ctx, db, query); err != nil {
		return fmt.Errorf("set local lock_timeout: %w", err)
	}

	return nil
}

func (c *Connection) HealthCheck(ctx context.Context) error {
	if c.tenantKeel == nil {
		return ErrDatabaseConnectionNotInitialized
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return c.tenantKeel.Ping(ctx)
}

func (c *Connection) IsHealthy(ctx context.Context) bool {
	return c.HealthCheck(ctx) == nil
}

func (c *Connection) Close() error {
	c.closePools()
	return nil
}
