package schemaversion

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/emoss08/keel/migrate"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/pgxdsn"
	"github.com/emoss08/trenova/schema"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	DefaultWait     = 10 * time.Minute
	pollInterval    = 2 * time.Second
	reportInterval  = 30 * time.Second
	applicationName = "trenova-schema-check"
)

var Module = fx.Module("schemaversion", fx.Invoke(requireAtStartup))

type Params struct {
	fx.In

	Config *config.Config
	Logger *zap.Logger
}

func requireAtStartup(p Params) error {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultWait)
	defer cancel()

	return Require(ctx, RequireParams{
		DSN:     p.Config.GetDSN(p.Config.Database.Password),
		Version: schema.Version,
		Logger:  p.Logger,
	})
}

type RequireParams struct {
	DSN     string
	Version string
	Logger  *zap.Logger
}

func Require(ctx context.Context, p RequireParams) error {
	dsn, err := pgxdsn.Normalize(p.DSN, pgxdsn.Options{ApplicationName: applicationName})
	if err != nil {
		return err
	}

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return fmt.Errorf("schemaversion: parse the database DSN: %w", err)
	}
	poolConfig.MaxConns = 1
	poolConfig.MinConns = 0

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return fmt.Errorf("schemaversion: open the database: %w", err)
	}
	defer pool.Close()

	logger := p.Logger.With(zap.String("component", "schemaversion"), zap.String("want", p.Version))

	first := migrate.Require(ctx, pool, p.Version)
	if first == nil {
		logger.Debug("the database schema is current")
		return nil
	}
	logger.Info(
		"waiting for trenova-db to migrate the database to this build's schema version",
		zap.Error(first),
		zap.Duration("timeout", remaining(ctx)),
	)

	start := time.Now()
	stop := reportWhileWaiting(ctx, logger, start)
	err = migrate.RequireWait(ctx, pool, p.Version, pollInterval)
	stop()

	if err != nil {
		var behind *migrate.BehindError
		if errors.As(err, &behind) {
			return fmt.Errorf(
				"the database schema is at %q and this build needs %s; run trenova-db up "+
					"before starting the API or worker (waited %s): %w",
				behind.Have,
				p.Version,
				time.Since(start).Round(time.Second),
				err,
			)
		}
		return fmt.Errorf(
			"could not confirm the database has schema version %s after %s: %w",
			p.Version,
			time.Since(start).Round(time.Second),
			err,
		)
	}

	logger.Info("the database reached this build's schema version",
		zap.Duration("waited", time.Since(start).Round(time.Millisecond)))

	return nil
}

func reportWhileWaiting(ctx context.Context, logger *zap.Logger, start time.Time) func() {
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(reportInterval)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				logger.Info("still waiting for the database schema",
					zap.Duration("waited", time.Since(start).Round(time.Second)))
			}
		}
	}()

	return func() { close(done) }
}

func remaining(ctx context.Context) time.Duration {
	deadline, ok := ctx.Deadline()
	if !ok {
		return 0
	}

	return time.Until(deadline).Round(time.Second)
}
