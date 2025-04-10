package queue

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/pkg/config"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivershared/util/slogutil"
	"github.com/riverqueue/river/rivertype"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Logger  *logger.Logger
	LC      fx.Lifecycle
	ConfigM *config.Manager
}

type Client struct {
	riverClient *river.Client[pgx.Tx]
	l           *zerolog.Logger
	dsn         string
	dbPool      *pgxpool.Pool
	lifecycle   fx.Lifecycle
	mu          sync.Mutex // For thread safety
}

func NewClient(p Params) *Client {
	log := p.Logger.With().
		Str("module", "queue").
		Logger()

	client := &Client{
		l:         &log,
		dsn:       p.ConfigM.GetDSN(),
		lifecycle: p.LC,
	}

	return client
}

// SetupWithFx registers the client with the fx lifecycle
func (c *Client) SetupWithFx() {
	c.lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// Create a background context that won't be canceled when the OnStart completes
			bgCtx := context.Background()

			// Do the initialization in a goroutine to avoid blocking
			go func() {
				if err := c.Initialize(bgCtx); err != nil {
					c.l.Error().Err(err).Msg("Failed to initialize River client")
				}
			}()

			// Return immediately to unblock fx startup
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return c.Stop(ctx)
		},
	})
}

// initialize creates and initializes the River client
func (c *Client) Initialize(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Create a new database pool
	dbPool, err := c.createDBPool(ctx)
	if err != nil {
		c.l.Error().Err(err).Msg("Failed to open database pool")
		return err
	}
	c.dbPool = dbPool

	// Create a driver with the pool
	driver := riverpgxv5.New(dbPool)

	// Create workers
	workers := river.NewWorkers()

	// Register workers
	if err = RegisterWorkers(workers); err != nil {
		c.l.Error().Err(err).Msg("Failed to register workers")
		return err
	}

	// Create the River client
	riverClient, err := river.NewClient(driver, &river.Config{
		Logger: slog.New(&slogutil.SlogMessageOnlyHandler{Level: slog.LevelInfo}),
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 10}, // Reduced to avoid resource issues
			"high_priority":    {MaxWorkers: 10}, // Reduced to avoid resource issues
		},
		Workers:           workers,
		PollOnly:          true,
		FetchPollInterval: 5 * time.Second,
	})
	if err != nil {
		c.l.Error().Err(err).Msg("Failed to create River client")
		return err
	}

	c.riverClient = riverClient

	// Start the River client
	c.l.Info().Msg("🚀 Starting River client")
	if err = c.riverClient.Start(ctx); err != nil {
		c.l.Error().Err(err).Msg("Failed to start River client")
		return err
	}

	// Wait a moment to ensure the client is ready
	time.Sleep(5 * time.Second)

	// Schedule jobs
	if err = c.ScheduleJobs(ctx); err != nil {
		c.l.Error().Err(err).Msg("Failed to schedule jobs")
		// Don't return the error, we want to keep the client running
	}

	return nil
}

func (c *Client) createDBPool(ctx context.Context) (*pgxpool.Pool, error) {
	// Parse the config first so we can modify it
	cfg, err := pgxpool.ParseConfig(c.dsn)
	if err != nil {
		return nil, err
	}

	// Set appropriate timeouts to avoid context deadline exceeded errors
	cfg.ConnConfig.ConnectTimeout = 5 * time.Second
	cfg.MaxConnLifetime = 30 * time.Minute
	cfg.MaxConnIdleTime = 5 * time.Minute
	cfg.HealthCheckPeriod = 1 * time.Minute
	cfg.MaxConns = 10 // Limit max connections

	// Create the pool with our modified config
	dbPool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}

	// Test the connection
	if err = dbPool.Ping(ctx); err != nil {
		return nil, err
	}

	return dbPool, nil
}

func (c *Client) Stop(ctx context.Context) error {
	if c.riverClient == nil {
		c.l.Warn().Msg("River client is nil, nothing to stop")
		return nil
	}

	c.l.Info().Msg("🔴 Stopping river client")
	err := c.riverClient.Stop(ctx)

	// Close the database pool when stopping
	if c.dbPool != nil {
		c.l.Info().Msg("🔴 Closing database pool")
		c.dbPool.Close()
	}

	return err
}

// InsertJob inserts a job using the provided arguments
func (c *Client) InsertJob(ctx context.Context, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error) {
	return c.riverClient.Insert(ctx, args, opts)
}

// InsertJobTx inserts a job within a transaction
func (c *Client) InsertJobTx(ctx context.Context, tx pgx.Tx, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error) {
	return c.riverClient.InsertTx(ctx, tx, args, opts)
}

func (c *Client) ScheduleJobs(ctx context.Context) error {
	_, err := c.riverClient.Insert(ctx, ScheduledAliveArgs{
		Message: "Hello, world!",
	}, &river.InsertOpts{
		Queue:       "high_priority",
		ScheduledAt: time.Now().Add(10 * time.Second), // 10 seconds from now
	})
	if err != nil {
		c.l.Error().Err(err).Msg("Failed to schedule job")
		return err
	}

	return nil
}
