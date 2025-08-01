/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/infrastructure/telemetry"
	"github.com/emoss08/trenova/internal/pkg/config"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/metrics"
	"github.com/emoss08/trenova/internal/pkg/middleware"
	"github.com/emoss08/trenova/internal/pkg/registry"
	"github.com/emoss08/trenova/internal/pkg/utils/intutils"
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

	Config           *config.Manager
	Logger           *logger.Logger
	LC               fx.Lifecycle
	TelemetryMetrics *telemetry.Metrics `name:"telemetryMetrics" optional:"true"`
}

var (
	ErrDBConnStringEmpty = eris.New("database connection string is empty")
	ErrDBConfigNil       = eris.New("database config is nil")
	ErrAppConfigNil      = eris.New("application config is nil")
)

type connection struct {
	cfg               *config.Manager
	log               *zerolog.Logger
	db                *bun.DB
	sql               *sql.DB
	pool              *pgxpool.Pool
	connectionPool    *ConnectionPool
	mu                sync.RWMutex
	healthCheckCancel context.CancelFunc
	telemetryMetrics  *telemetry.Metrics
}

type readReplica struct {
	name      string
	db        *bun.DB
	sql       *sql.DB
	pool      *pgxpool.Pool
	weight    int
	healthy   bool
	lastCheck time.Time
	mu        sync.RWMutex
}

func NewConnection(p ConnectionParams) db.Connection {
	log := p.Logger.With().
		Str("component", "postgres").
		Str("service", "connection").
		Logger()

	var telemetryMetrics *telemetry.Metrics
	if p.TelemetryMetrics != nil {
		telemetryMetrics = p.TelemetryMetrics
	}

	conn := &connection{
		cfg:              p.Config,
		log:              &log,
		telemetryMetrics: telemetryMetrics,
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
	return c.WriteDB(ctx)
}

func (c *connection) WriteDB(ctx context.Context) (*bun.DB, error) {
	return c.initializePrimaryIfNeeded(ctx)
}

func (c *connection) initializePrimaryIfNeeded(ctx context.Context) (*bun.DB, error) {
	c.mu.RLock()
	if c.db != nil {
		defer c.mu.RUnlock()
		return c.db, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

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

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, eris.Wrap(err, "failed to parse database config")
	}

	cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, eris.Wrap(err, "failed to create database pool")
	}
	c.pool = pool

	sqldb := stdlib.OpenDBFromPool(pool)
	c.sql = sqldb

	bunDB := bun.NewDB(sqldb, pgdialect.New(), bun.WithDiscardUnknownColumns())

	if appCfg.Environment == "development" && dbCfg.Debug {
		bunDB.AddQueryHook(bundebug.NewQueryHook(
			bundebug.WithVerbose(true),
			bundebug.FromEnv("BUNDEBUG"),
		))
	}

	bunDB.AddQueryHook(middleware.NewDatabaseQueryHook(c.log, "primary", true))

	if c.telemetryMetrics != nil {
		appCfg := c.cfg.App()
		if appCfg != nil {
			_, err := telemetry.InstrumentDatabase(bunDB, appCfg.Name, c.telemetryMetrics)
			if err != nil {
				c.log.Error().Err(err).Msg("failed to instrument database for telemetry")
			} else {
				c.log.Info().Msg("Database telemetry instrumentation added")
			}
		}
	}

	bunDB.RegisterModel(registry.RegisterEntities()...)

	sqldb.SetConnMaxIdleTime(time.Duration(dbCfg.ConnMaxIdleTime) * time.Second)
	sqldb.SetMaxOpenConns(dbCfg.MaxConnections)
	sqldb.SetMaxIdleConns(dbCfg.MaxIdleConns)
	sqldb.SetConnMaxLifetime(time.Duration(dbCfg.ConnMaxLifetime) * time.Second)

	if err = bunDB.PingContext(ctx); err != nil {
		metrics.RecordConnectionAttempt("primary", false)
		return nil, eris.Wrap(err, "failed to ping database")
	}

	metrics.RecordConnectionAttempt("primary", true)

	c.db = bunDB
	c.log.Info().Msg("🚀 Established connection to primary Postgres database!")

	go c.monitorConnectionPoolStats(ctx)

	c.connectionPool = NewConnectionPool(c.db, c.log)

	if dbCfg.EnableReadWriteSeparation && len(dbCfg.ReadReplicas) > 0 {
		if err = c.initializeReadReplicas(ctx); err != nil {
			c.log.Error().
				Err(err).
				Msg("failed to initialize read replicas, continuing with primary only")
		}
	}

	return c.db, nil
}

func (c *connection) ReadDB(ctx context.Context) (*bun.DB, error) {
	start := time.Now()

	if _, err := c.initializePrimaryIfNeeded(ctx); err != nil {
		return nil, err
	}

	c.mu.RLock()
	if c.connectionPool == nil {
		c.mu.RUnlock()
		return c.db, nil
	}
	pool := c.connectionPool
	c.mu.RUnlock()

	conn, connName := pool.GetReadConnection()

	c.log.Debug().
		Str("connection", connName).
		Dur("duration", time.Since(start)).
		Msg("read connection acquired")

	return conn, nil
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

	if c.healthCheckCancel != nil {
		c.healthCheckCancel()
	}

	defer func() {
		c.db = nil
		c.sql = nil
		c.pool = nil
		c.connectionPool = nil
		c.log.Info().Msg("PostgreSQL database connection resources cleared")
	}()

	closeCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	done := make(chan struct{})

	go func() {
		defer close(done)

		if c.db != nil {
			c.log.Debug().Msg("closing database connection")
			if err := c.db.Close(); err != nil {
				c.log.Error().Err(err).Msg("error closing database connection")
			} else {
				c.log.Debug().Msg("database connection closed successfully")
			}
		}

		if c.pool != nil {
			c.log.Debug().Msg("closing connection pool")
			c.pool.Close()
			c.log.Debug().Msg("connection pool closed")
		}

		if c.connectionPool != nil {
			c.log.Debug().Msg("closing connection pool manager")
			if err := c.connectionPool.Close(); err != nil {
				c.log.Error().Err(err).Msg("error closing connection pool manager")
			}
		}
	}()

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

func (c *connection) initializeReadReplicas( //nolint:funlen // we need to keep this function long
	ctx context.Context,
) error {
	dbCfg := c.cfg.Database()
	appCfg := c.cfg.App()

	successCount := 0
	for _, replicaCfg := range dbCfg.ReadReplicas {
		c.log.Info().Str("replica", replicaCfg.Name).Msg("initializing read replica")

		start := time.Now()

		hostPort := net.JoinHostPort(replicaCfg.Host, strconv.Itoa(replicaCfg.Port))
		dsn := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=%s",
			dbCfg.Username,
			dbCfg.Password,
			hostPort,
			dbCfg.Database,
			dbCfg.SSLMode,
		)

		cfg, err := pgxpool.ParseConfig(dsn)
		if err != nil {
			c.log.Error().
				Err(err).
				Str("replica", replicaCfg.Name).
				Msg("failed to parse replica config")
			continue
		}

		cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

		pool, err := pgxpool.NewWithConfig(ctx, cfg)
		if err != nil {
			c.log.Error().
				Err(err).
				Str("replica", replicaCfg.Name).
				Msg("failed to create replica pool")
			continue
		}

		sqldb := stdlib.OpenDBFromPool(pool)
		bunDB := bun.NewDB(sqldb, pgdialect.New(), bun.WithDiscardUnknownColumns())

		if appCfg.Environment == "development" && dbCfg.Debug {
			bunDB.AddQueryHook(bundebug.NewQueryHook(
				bundebug.WithVerbose(true),
				bundebug.FromEnv("BUNDEBUG"),
			))
		}

		bunDB.AddQueryHook(middleware.NewDatabaseQueryHook(c.log, replicaCfg.Name, true))

		bunDB.RegisterModel(registry.RegisterEntities()...)

		maxConns := replicaCfg.MaxConnections
		if maxConns == 0 {
			maxConns = dbCfg.MaxConnections
		}
		maxIdleConns := replicaCfg.MaxIdleConns
		if maxIdleConns == 0 {
			maxIdleConns = dbCfg.MaxIdleConns
		}

		sqldb.SetConnMaxIdleTime(time.Duration(dbCfg.ConnMaxIdleTime) * time.Second)
		sqldb.SetMaxOpenConns(maxConns)
		sqldb.SetMaxIdleConns(maxIdleConns)
		sqldb.SetConnMaxLifetime(time.Duration(dbCfg.ConnMaxLifetime) * time.Second)

		if err = bunDB.PingContext(ctx); err != nil {
			c.log.Error().Err(err).Str("replica", replicaCfg.Name).Msg("failed to ping replica")
			pool.Close()
			metrics.RecordConnectionAttempt(replicaCfg.Name, false)
			continue
		}

		weight := replicaCfg.Weight
		if weight <= 0 {
			weight = 1
		}

		replica := &readReplica{
			name:      replicaCfg.Name,
			db:        bunDB,
			sql:       sqldb,
			pool:      pool,
			weight:    weight,
			healthy:   true,
			lastCheck: time.Now(),
		}

		c.connectionPool.AddReplica(replica)
		successCount++

		metrics.RecordConnectionAttempt(replicaCfg.Name, true)
		metrics.RecordDatabaseOperation("replica_init", replicaCfg.Name, time.Since(start))
		c.log.Info().
			Str("replica", replicaCfg.Name).
			Dur("duration", time.Since(start)).
			Msg("🚀 Established connection to read replica!")
	}

	if successCount == 0 {
		return eris.New("no read replicas could be initialized")
	}

	ctx, cancel := context.WithCancel(context.Background())
	c.healthCheckCancel = cancel
	go c.monitorReplicaHealth(ctx)

	return nil
}

func (c *connection) monitorReplicaHealth(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	c.performHealthCheck(ctx)

	for {
		select {
		case <-ctx.Done():
			c.log.Info().Msg("stopping health check monitor")
			return
		case <-ticker.C:
			c.performHealthCheck(ctx)
		}
	}
}

func (c *connection) performHealthCheck(ctx context.Context) {
	c.mu.RLock()
	dbCfg := c.cfg.Database()
	if c.connectionPool == nil {
		c.mu.RUnlock()
		return
	}
	pool := c.connectionPool
	c.mu.RUnlock()

	lagThreshold := time.Duration(dbCfg.ReplicaLagThreshold) * time.Second
	pool.HealthCheck(ctx, lagThreshold)

	pool.GetPoolStats()
}

func (c *connection) monitorConnectionPoolStats(ctx context.Context) {
	time.Sleep(2 * time.Second)

	c.updatePoolStats()

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.updatePoolStats()
		}
	}
}

func (c *connection) updatePoolStats() {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.sql != nil {
		stats := c.sql.Stats()

		c.log.Info().
			Int("max_open", stats.MaxOpenConnections).
			Int("open", stats.OpenConnections).
			Int("in_use", stats.InUse).
			Int("idle", stats.Idle).
			Int64("wait_count", stats.WaitCount).
			Dur("wait_duration", stats.WaitDuration).
			Int64("max_idle_closed", stats.MaxIdleClosed).
			Int64("max_lifetime_closed", stats.MaxLifetimeClosed).
			Msg("Database connection pool stats")

		metrics.UpdateConnectionPoolStats("primary",
			intutils.SafeInt32(stats.MaxOpenConnections),
			intutils.SafeInt32(stats.Idle),
			intutils.SafeInt32(stats.InUse))
	}

	if c.connectionPool != nil {
		c.connectionPool.GetPoolStats()
	}
}
