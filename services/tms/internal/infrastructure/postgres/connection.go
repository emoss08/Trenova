package postgres

import (
	"context"
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
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var _ ports.DBConnection = (*Connection)(nil)

type ConnectionParams struct {
	fx.In

	Config  *config.Config
	Logger  *zap.Logger
	Tracer  *observability.TracerProvider
	Metrics *observability.MetricsRegistry
	LC      fx.Lifecycle
}

type Connection struct {
	db      *bun.DB
	cfg     *config.Config
	logger  *zap.Logger
	tracer  *observability.TracerProvider
	metrics *observability.MetricsRegistry
}

const (
	slowQueryThresholdWarn  = 100 * time.Millisecond
	slowQueryThresholdError = 1 * time.Second
)

func NewConnection(p ConnectionParams) (*Connection, error) {
	ctx := context.Background()
	logger := p.Logger.With(zap.String("component", "postgres"))

	password, err := loadPassword(p.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to load database password: %w", err)
	}

	dsn := p.Config.GetDSN(password)
	maskedDSN := p.Config.GetDSNMasked()

	logger.Debug("Connecting to PostgreSQL",
		zap.String("dsn", maskedDSN),
		zap.String("host", p.Config.Database.Host),
		zap.Int("port", p.Config.Database.Port),
		zap.String("database", p.Config.Database.Name),
	)

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database configuration: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create database pool: %w", err)
	}

	sqldb := stdlib.OpenDBFromPool(pool)
	db := bun.NewDB(sqldb, pgdialect.New())

	if p.Config.IsDevelopment() && p.Config.App.Debug {
		db.AddQueryHook(bundebug.NewQueryHook(
			bundebug.WithVerbose(p.Config.Database.Verbose),
			bundebug.FromEnv("BUNDEBUG"),
		))
	}

	if p.Tracer != nil && p.Tracer.IsEnabled() {
		db.AddQueryHook(bunotel.NewQueryHook(
			bunotel.WithDBName(p.Config.Database.Name),
			bunotel.WithFormattedQueries(true),
		))
	}

	db.RegisterModel(domainregistry.RegisterEntities()...)
	conn := &Connection{
		db:      db,
		cfg:     p.Config,
		logger:  logger,
		tracer:  p.Tracer,
		metrics: p.Metrics,
	}

	conn.addSlowQueryHook()
	conn.addMetricsHook()

	if err = conn.ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	p.LC.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("PostgreSQL connection established",
				zap.String("database", p.Config.Database.Name),
				zap.Int("max_open_conns", p.Config.Database.MaxOpenConns),
				zap.Int("max_idle_conns", p.Config.Database.MaxIdleConns),
			)

			go conn.monitorConnectionPool(ctx)

			return nil
		},
		OnStop: func(context.Context) error {
			logger.Info("Closing PostgreSQL connection")
			if err = conn.Close(); err != nil {
				logger.Error("Failed to close database connection", zap.Error(err))
				return err
			}
			logger.Info("PostgreSQL connection closed successfully")
			return nil
		},
	})

	return conn, nil
}

func (c *Connection) DB(ctx context.Context) (*bun.DB, error) {
	if c.db == nil {
		return nil, ErrDatabaseConnectionNotInitialized
	}

	if err := c.ping(ctx); err != nil {
		c.logger.Error("Database health check failed", zap.Error(err))
		if c.metrics != nil {
			c.metrics.RecordError("database_ping_failed", "postgres")
		}
		return nil, fmt.Errorf("database is not healthy: %w", err)
	}

	return c.db, nil
}

func (c *Connection) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

func (c *Connection) ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return c.db.PingContext(ctx)
}

func (c *Connection) addSlowQueryHook() {
	c.db.AddQueryHook(&slowQueryHook{
		logger:  c.logger,
		tracer:  c.tracer,
		metrics: c.metrics,
	})
}

func (c *Connection) addMetricsHook() {
	if c.metrics != nil && c.metrics.IsEnabled() {
		c.db.AddQueryHook(&metricsHook{
			metrics: c.metrics,
		})
	}
}

func (c *Connection) monitorConnectionPool(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			stats := c.db.Stats()

			c.logger.Debug("Database connection pool stats",
				zap.Int("max_open_connections", stats.MaxOpenConnections),
				zap.Int("open_connections", stats.OpenConnections),
				zap.Int("in_use", stats.InUse),
				zap.Int("idle", stats.Idle),
				zap.Int64("wait_count", stats.WaitCount),
				zap.Duration("wait_duration", stats.WaitDuration),
				zap.Int64("max_idle_closed", stats.MaxIdleClosed),
				zap.Int64("max_idle_time_closed", stats.MaxIdleTimeClosed),
				zap.Int64("max_lifetime_closed", stats.MaxLifetimeClosed),
			)

			if c.metrics != nil {
				c.metrics.UpdateDBConnections(stats.InUse, stats.Idle)
			}
		}
	}
}

func loadPassword(cfg *config.Config) (string, error) {
	switch cfg.Database.PasswordSource {
	case "env":
		if cfg.Database.Password == "" {
			return "", config.ErrDatabasePasswordNotSet
		}
		return cfg.Database.Password, nil
	case "file":
		return "", ErrFilePasswordSourceNotImplemented
	case "secret":
		return "", ErrSecretPasswordSourceNotImplemented
	default:
		return "", fmt.Errorf("unknown password source: %s", cfg.Database.PasswordSource)
	}
}

type slowQueryHook struct {
	logger  *zap.Logger
	tracer  *observability.TracerProvider
	metrics *observability.MetricsRegistry
}

func (h *slowQueryHook) BeforeQuery(ctx context.Context, _ *bun.QueryEvent) context.Context {
	return ctx
}

func (h *slowQueryHook) AfterQuery(ctx context.Context, event *bun.QueryEvent) {
	duration := time.Since(event.StartTime)
	query := event.Query

	if duration >= slowQueryThresholdError { //nolint:nestif // This is a slow query error
		h.logger.Error("Very slow query detected",
			zap.String("query", query),
			zap.Duration("duration", duration),
			zap.String("operation", event.Operation()),
			zap.Error(event.Err),
		)

		if h.tracer != nil && h.tracer.IsEnabled() {
			span := trace.SpanFromContext(ctx)
			if span.IsRecording() {
				span.SetAttributes(
					attribute.Bool("db.slow_query", true),
					attribute.String("db.slow_query.level", "error"),
					attribute.Float64(
						"db.slow_query.duration_ms",
						float64(duration.Milliseconds()),
					),
				)
				span.SetStatus(codes.Error, "Very slow query detected")
			}
		}

		if h.metrics != nil {
			h.metrics.RecordError("slow_query_error", "database")
		}
	} else if duration >= slowQueryThresholdWarn {
		h.logger.Warn("Slow query detected",
			zap.String("query", query),
			zap.Duration("duration", duration),
			zap.String("operation", event.Operation()),
		)

		if h.tracer != nil && h.tracer.IsEnabled() {
			span := trace.SpanFromContext(ctx)
			if span.IsRecording() {
				span.SetAttributes(
					attribute.Bool("db.slow_query", true),
					attribute.String("db.slow_query.level", "warning"),
					attribute.Float64("db.slow_query.duration_ms", float64(duration.Milliseconds())),
				)
				span.AddEvent("Slow query detected",
					trace.WithAttributes(
						attribute.String("query", query),
						attribute.Float64("duration_ms", float64(duration.Milliseconds())),
					),
				)
			}
		}
	}
}

type metricsHook struct {
	metrics *observability.MetricsRegistry
}

func (h *metricsHook) BeforeQuery(ctx context.Context, _ *bun.QueryEvent) context.Context {
	return ctx
}

func (h *metricsHook) AfterQuery(_ context.Context, event *bun.QueryEvent) {
	duration := time.Since(event.StartTime).Seconds()
	operation := event.Operation()

	var tableName string
	if event.Model != nil {
		tableName = fmt.Sprintf("%T", event.Model)
	} else {
		tableName = "unknown"
	}

	status := "success"
	if event.Err != nil {
		status = "error"
	}

	h.metrics.RecordDBQuery(operation, tableName, status, duration)
}
