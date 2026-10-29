package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/databaseconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/reportingconfig"
	"github.com/emoss08/trenova/internal/infrastructure/observability"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/infrastructure/observability/platformmetrics"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbpool"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/keeltables"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/dbscope"
	goredis "github.com/redis/go-redis/v9"
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

	Lifecycle       fx.Lifecycle
	AppConfig       *appconfig.Config
	DatabaseConfig  *databaseconfig.Config
	ReportingConfig *reportingconfig.Config
	Logger          *zap.Logger
	Metrics         *metrics.Registry `optional:"true"`
	Redis           *goredis.Client   `optional:"true"`
	Observers       []keel.Observer   `optional:"true"`
}

type Connection struct {
	tenantKeel     *keel.DB
	systemKeel     *keel.DB
	appConfig      *appconfig.Config
	databaseConfig *databaseconfig.Config
	logger         *observability.ContextLogger
	metrics        *metrics.Registry
	redis          *goredis.Client
	observers      []keel.Observer
	settings       connectionSettings
	rls            *rlsRuntime
}

type connectionSettings struct {
	component         string
	maxOpenConns      int
	connMaxLifetime   time.Duration
	connMaxIdleTime   time.Duration
	connParams        map[string]string
	tenantConcurrency int
	resultCache       bool
	registerStats     bool
}

type txContextKey struct{}

type txBinding struct {
	db    *keel.DB
	scope dbscope.Scope
}

type detachedFromTx struct{}

func NewConnection(p ConnectionParams) (*Connection, error) {
	return newConnection(p, oltpSettings(p.DatabaseConfig))
}

type ReportingConnection struct {
	*Connection
}

func NewReportingConnection(p ConnectionParams) (*ReportingConnection, error) {
	conn, err := newConnection(p, reportingSettings(p.DatabaseConfig, p.ReportingConfig))
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
	if rlsConfigured(p.DatabaseConfig) {
		runtime, err := newRLSRuntime(p.DatabaseConfig, logger, p.Metrics)
		if err != nil {
			return nil, err
		}
		rls = runtime
	}

	conn := &Connection{
		appConfig:      p.AppConfig,
		databaseConfig: p.DatabaseConfig,
		logger:         logger,
		metrics:        p.Metrics,
		redis:          p.Redis,
		observers:      p.Observers,
		settings:       settings,
		rls:            rls,
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

func oltpSettings(cfg *databaseconfig.Config) connectionSettings {
	return connectionSettings{
		component:       "postgres",
		maxOpenConns:    cfg.MaxOpenConns,
		connMaxLifetime: cfg.ConnMaxLifetime,
		connMaxIdleTime: cfg.ConnMaxIdleTime,
		connParams: sessionParams(
			cfg.GetStatementTimeout(),
			cfg.GetLockTimeout(),
			cfg.GetIdleTxTimeout(),
		),
		tenantConcurrency: cfg.GetTenantConcurrency(cfg.MaxOpenConns),
		resultCache:       cfg.Cache.Enabled,
		registerStats:     true,
	}
}

func reportingSettings(
	databaseConfig *databaseconfig.Config,
	reportingConfig *reportingconfig.Config,
) connectionSettings {
	reporting := reportingConfig

	return connectionSettings{
		component:       "postgres-reporting",
		maxOpenConns:    reporting.GetPoolMaxOpenConns(),
		connMaxLifetime: databaseConfig.ConnMaxLifetime,
		connMaxIdleTime: databaseConfig.ConnMaxIdleTime,
		connParams: sessionParams(
			reporting.GetStatementTimeout(),
			databaseConfig.GetLockTimeout(),
			databaseConfig.GetIdleTxTimeout(),
		),
		tenantConcurrency: databaseConfig.GetTenantConcurrency(reporting.GetPoolMaxOpenConns()),
		resultCache:       false,
		registerStats:     false,
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
	c.logger.Debug(ctx, "Connecting to database", zap.String("dsn", c.databaseConfig.DSNMasked()))

	var guard dbpool.Guard
	if c.rls != nil {
		guard = c.rls.guard(scopePoolTenant, !c.databaseConfig.System.Configured())
	}

	var (
		cache *keel.CacheConfig
		err   error
	)
	if c.settings.resultCache {
		if cache, err = c.resultCache(); err != nil {
			return err
		}
	}

	tenant, err := c.openPool(ctx, &dbpool.Options{
		DSN:               c.databaseConfig.DSN(c.appConfig.Name, c.databaseConfig.Password),
		MaxConns:          c.settings.maxOpenConns,
		MaxConnLifetime:   c.settings.connMaxLifetime,
		MaxConnIdleTime:   c.settings.connMaxIdleTime,
		RuntimeParams:     c.settings.connParams,
		Guard:             guard,
		TenantConcurrency: c.settings.tenantConcurrency,
		Cache:             cache,
		SlowWaits:         c.slowWaits(scopePoolTenant),
	})
	if err != nil {
		return err
	}
	c.tenantKeel = tenant

	if c.settings.registerStats && c.metrics != nil && c.metrics.Database != nil {
		pool := tenant.Pool()
		c.metrics.Database.RegisterPoolStats(
			func() platformmetrics.PoolStats { return poolStats(pool) },
		)
	}

	if err = c.HealthCheck(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	if err = c.connectSystem(ctx); err != nil {
		return err
	}

	if c.settings.registerStats && c.metrics != nil && c.metrics.Database != nil {
		c.metrics.Database.RegisterRuntimeStats(c.runtimeStats)
	}

	if err = c.verifyRLS(ctx); err != nil {
		return err
	}

	c.logger.Info(
		ctx, "Database connection established",
		zap.String("database", c.databaseConfig.Name),
		zap.Int32("max_conns", tenant.Pool().Config().MaxConns),
		zap.Int("tenant_concurrency", c.settings.tenantConcurrency),
		zap.Bool("result_cache", cache != nil),
		zap.Bool("result_cache_shared", cache != nil && cache.Shared != nil),
		zap.String("row_level_security", c.rlsMode()),
	)

	if c.appConfig.IsProduction() && !c.rls.enforced() {
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
	opts.Observers = c.poolObservers(slow)
	opts.ObserveArgs = queryLog(c.appConfig, c.databaseConfig) != nil
	opts.Tables = keeltables.All()
	opts.MaxWait = c.databaseConfig.MaxWait
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

func (c *Connection) poolObservers(slow *slowQueryLog) []keel.Observer {
	observers := make([]keel.Observer, 0, 3+len(c.observers))
	if log := queryLog(c.appConfig, c.databaseConfig); log != nil {
		observers = append(observers, log)
	}

	observers = append(observers,
		&slowQueryObserver{log: slow},
		newTracingObserver(c.databaseConfig.Name),
	)

	return append(observers, c.observers...)
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
		return databaseconfig.RLSModeOff
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
		if opts.ReadOnly {
			ctx = kernelports.WithReadOnly(ctx)
		}

		settings := txSettings(opts.Settings)
		if opts.LockTimeout > 0 {
			settings = append(settings, keel.Setting(
				"lock_timeout",
				strconv.FormatInt(max(opts.LockTimeout.Milliseconds(), 1), 10)+"ms",
			))
		}
		if len(settings) > 0 {
			return keel.Read(ctx, db, fn, settings...)
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
	case sql.LevelDefault:
	case sql.LevelReadCommitted:
		txOpts = append(txOpts, keel.ReadCommitted)
	case sql.LevelRepeatableRead:
		txOpts = append(txOpts, keel.RepeatableRead)
	case sql.LevelSerializable:
		txOpts = append(txOpts, keel.Serializable)
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
