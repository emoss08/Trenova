package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/observability"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbpool"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/domainregistry"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/extra/bundebug"
	"github.com/uptrace/bun/extra/bunotel"
	"go.opentelemetry.io/otel"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var _ kernelports.DBConnection = (*Connection)(nil)

type ConnectionParams struct {
	fx.In

	Lifecycle fx.Lifecycle
	Config    *config.Config
	Logger    *zap.Logger
	Metrics   *metrics.Registry `optional:"true"`
}

type Connection struct {
	db         *bun.DB
	system     *bun.DB
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
	tx    bun.Tx
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
	c.db, c.tenantKeel = tenant.Bun, tenant.Keel

	if c.settings.registerStats && c.metrics != nil && c.metrics.Database != nil {
		pool := tenant.Keel.Pool()
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
		zap.Int32("max_conns", tenant.Keel.Pool().Config().MaxConns),
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

func (c *Connection) openPool(ctx context.Context, opts *dbpool.Options) (dbpool.Pool, error) {
	slow := newSlowQueryLog(c.logger)
	opts.Observers = c.observers(slow)
	opts.VersionErrors = func(err error) {
		c.logger.Warn(context.Background(), "Failed to record data versions", zap.Error(err))
	}

	pool, err := dbpool.Open(ctx, opts)
	if err != nil {
		return dbpool.Pool{}, err
	}

	slow.pool = pool.Keel.Pool()
	pool.Bun = c.withHooks(pool.Bun, slow)
	registerModels(pool.Bun)

	return pool, nil
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

func (c *Connection) ScopedTransactions() bool {
	return c != nil && c.rls != nil
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
	if !c.ScopedTransactions() {
		return fn(ctx)
	}

	return c.WithTx(
		ctx,
		kernelports.TxOptions{ReadOnly: readOnly},
		func(ctx context.Context, _ bun.Tx) error {
			return fn(ctx)
		},
	)
}

func (c *Connection) rlsMode() string {
	if c.rls == nil {
		return config.RLSModeOff
	}

	return c.rls.mode
}

func registerModels(db *bun.DB) {
	db.RegisterModel(domainregistry.RegisterManyToManyEntities()...)
	db.RegisterModel(domainregistry.RegisterEntities()...)
}

func (c *Connection) withHooks(db *bun.DB, slow *slowQueryLog) *bun.DB {
	if c.cfg.App.IsDevelopment() && c.cfg.App.Debug {
		db = db.WithQueryHook(bundebug.NewQueryHook(
			bundebug.WithVerbose(c.cfg.Database.Verbose),
			bundebug.FromEnv("BUNDEBUG"),
		))
	}

	db = db.WithQueryHook(&slowQueryHook{log: slow})

	return db.WithQueryHook(
		bunotel.NewQueryHook(
			bunotel.WithDBName(c.cfg.Database.Name),
			bunotel.WithTracerProvider(otel.GetTracerProvider()),
			bunotel.WithFormattedQueries(c.cfg.App.IsDevelopment()),
		),
	)
}

func (c *Connection) shutdown(ctx context.Context) error {
	c.logger.Info(ctx, "Closing PostgreSQL connection")

	for _, db := range []*bun.DB{c.system, c.db} {
		if db == nil {
			continue
		}
		if err := db.Close(); err != nil {
			c.logger.Error(ctx, "Failed to close database connection", zap.Error(err))
			return err
		}
	}

	for _, db := range []*keel.DB{c.systemKeel, c.tenantKeel} {
		if db != nil {
			db.Close()
		}
	}

	c.logger.Info(ctx, "PostgreSQL connection closed successfully")
	return nil
}

func WrapDB(db *bun.DB) *Connection {
	registerModels(db)
	return &Connection{db: db, tenantKeel: dbpool.KeelOf(db.DB)}
}

func NewTestConnection(db *bun.DB) *Connection {
	return WrapDB(db)
}

func (c *Connection) DB() *bun.DB {
	return c.db
}

func (c *Connection) Keel(ctx context.Context) *keel.DB {
	if c.systemKeel != nil && dbscope.IsSystem(ctx) {
		return c.systemKeel
	}

	return c.tenantKeel
}

func (c *Connection) DBForContext(ctx context.Context) bun.IDB {
	if tx, ok := c.txForContext(ctx); ok {
		return tx
	}

	return c.poolForContext(ctx)
}

func (c *Connection) txForContext(ctx context.Context) (bun.Tx, bool) {
	binding, ok := ctx.Value(txContextKey{}).(*txBinding)
	if !ok {
		return bun.Tx{}, false
	}

	if c.rls != nil && !binding.scope.Matches(dbscope.From(ctx)) {
		return bun.Tx{}, false
	}

	return binding.tx, true
}

func (c *Connection) poolForContext(ctx context.Context) *bun.DB {
	if c.system != nil && dbscope.IsSystem(ctx) {
		return c.system
	}

	return c.db
}

func (c *Connection) WithTx(
	ctx context.Context,
	opts kernelports.TxOptions,
	fn func(context.Context, bun.Tx) error,
) error {
	if c.db == nil {
		return ErrDatabaseConnectionNotInitialized
	}

	if existingTx, ok := c.txForContext(ctx); ok {
		if opts.LockTimeout > 0 {
			if err := applyLockTimeout(ctx, existingTx, opts.LockTimeout); err != nil {
				return err
			}
		}

		if opts.ReadOnly {
			ctx = kernelports.WithReadOnly(ctx)
		}

		return fn(ctx, existingTx)
	}

	baseCtx := ctx
	if opts.ReadOnly {
		baseCtx = kernelports.WithReadOnly(baseCtx)
	}
	txCtx, hooks := kernelports.WithAfterCommitHooks(baseCtx)

	tx, err := c.begin(txCtx, opts)
	if err != nil {
		return err
	}
	defer tx.end(false)

	if opts.LockTimeout > 0 {
		if err = applyLockTimeout(txCtx, tx.bun, opts.LockTimeout); err != nil {
			return err
		}
	}

	if err = fn(tx.ctx, tx.bun); err != nil {
		return err
	}

	if err = txCtx.Err(); err != nil {
		return err
	}

	if err = tx.commit(); err != nil {
		return err
	}

	hooks.Run(ctx)
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

func (c *Connection) HealthCheck(ctx context.Context) error {
	if c.db == nil {
		return ErrDatabaseConnectionNotInitialized
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if c.tenantKeel != nil {
		return c.tenantKeel.Ping(ctx)
	}

	return c.db.PingContext(ctx)
}

func (c *Connection) IsHealthy(ctx context.Context) bool {
	return c.HealthCheck(ctx) == nil
}

func (c *Connection) Close() error {
	for _, db := range []*bun.DB{c.system, c.db} {
		if db == nil {
			continue
		}
		if err := db.Close(); err != nil {
			return err
		}
	}

	return nil
}
