package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/emoss08/gtc/internal/adapters/primary/postgres"
	"github.com/emoss08/gtc/internal/adapters/primary/wal"
	gtcmeili "github.com/emoss08/gtc/internal/adapters/secondary/meilisearch"
	gtcredis "github.com/emoss08/gtc/internal/adapters/secondary/redis"
	"github.com/emoss08/gtc/internal/core/domain"
	"github.com/emoss08/gtc/internal/core/ports"
	"github.com/emoss08/gtc/internal/core/services"
	"github.com/emoss08/gtc/internal/infrastructure/config"
	"github.com/emoss08/gtc/internal/infrastructure/server"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	_ = godotenv.Load()

	logger := newLogger()
	if err := runCLI(logger, os.Args[1:]); err != nil {
		logger.Error("gateway exited with error", zap.Error(err))
		os.Exit(1)
	}
}

func runCLI(logger *zap.Logger, args []string) error {
	command := "run"
	if len(args) > 0 {
		command = args[0]
		args = args[1:]
	}

	switch command {
	case "run":
		return runServer(logger)
	case "validate-config":
		return validateConfig(logger)
	case "backfill":
		return runBackfill(logger, args)
	case "replay-dlq":
		return runReplayDLQ(logger, args)
	default:
		return fmt.Errorf("unknown command %q", command)
	}
}

func runServer(logger *zap.Logger) error {
	app, ctx, stop, err := buildApplication(logger)
	if err != nil {
		return err
	}
	defer stop()
	defer app.close()

	health := server.NewHealthStatus()
	stopMonitor := server.StartHealthMonitor(health, app.runtime, app.cfg.HealthPollInterval)
	defer stopMonitor()

	httpServer := server.New(server.ServerParams{
		Config:  server.Config{Port: app.cfg.HTTPPort},
		Checker: health,
		Logger:  logger,
	})

	serverErrCh := make(chan error, 1)
	go func() {
		serverErrCh <- httpServer.Start()
	}()

	runtimeErrCh := make(chan error, 1)
	go func() {
		runtimeErrCh <- app.runtime.Start(ctx)
	}()

	select {
	case err := <-serverErrCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case err := <-runtimeErrCh:
		if err != nil && !errors.Is(err, context.Canceled) {
			return err
		}
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	stop()

	var shutdownErr error
	if err := httpServer.Stop(shutdownCtx); err != nil && shutdownErr == nil {
		shutdownErr = err
	}
	if err := app.runtime.Stop(shutdownCtx); err != nil && shutdownErr == nil {
		shutdownErr = err
	}

	return shutdownErr
}

func validateConfig(logger *zap.Logger) error {
	app, _, _, err := buildApplication(logger)
	if err != nil {
		return err
	}
	defer app.close()
	defer func() { _ = app.runtime.Stop(context.Background()) }()

	if err := app.runtime.Validate(context.Background()); err != nil {
		return err
	}

	logger.Info("configuration validated successfully")
	return nil
}

func runBackfill(logger *zap.Logger, args []string) error {
	flags := flag.NewFlagSet("backfill", flag.ContinueOnError)
	var projectionArg string
	var tableArg string
	flags.StringVar(&projectionArg, "projection", "", "comma-separated projection names")
	flags.StringVar(&tableArg, "table", "", "comma-separated full table names")
	if err := flags.Parse(args); err != nil {
		return err
	}

	app, _, _, err := buildApplication(logger)
	if err != nil {
		return err
	}
	defer app.close()
	defer func() { _ = app.runtime.Stop(context.Background()) }()

	if err := app.runtime.Backfill(context.Background(), csvList(projectionArg), csvList(tableArg)); err != nil {
		return err
	}

	logger.Info("backfill completed", zap.String("projections", projectionArg), zap.String("tables", tableArg))
	return nil
}

func runReplayDLQ(logger *zap.Logger, args []string) error {
	flags := flag.NewFlagSet("replay-dlq", flag.ContinueOnError)
	var limit int
	var deleteOnSuccess bool
	flags.IntVar(&limit, "limit", 100, "maximum dead-letter entries to replay")
	flags.BoolVar(&deleteOnSuccess, "delete", true, "delete successfully replayed entries")
	if err := flags.Parse(args); err != nil {
		return err
	}

	app, _, _, err := buildApplication(logger)
	if err != nil {
		return err
	}
	defer app.close()
	defer func() { _ = app.runtime.Stop(context.Background()) }()

	entries, err := app.dlq.Read(context.Background(), int64(limit))
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		logger.Info("no dead-letter entries found")
		return nil
	}

	records := make([]domain.DeadLetterRecord, 0, len(entries))
	ids := make([]string, 0, len(entries))
	for _, entry := range entries {
		records = append(records, entry.Record)
		ids = append(ids, entry.ID)
	}

	if err := app.runtime.ReplayDeadLetters(context.Background(), records); err != nil {
		return err
	}
	if deleteOnSuccess {
		if err := app.dlq.Delete(context.Background(), ids...); err != nil {
			return err
		}
	}

	logger.Info("replayed dead-letter entries", zap.Int("count", len(entries)))
	return nil
}

type application struct {
	cfg     *config.Config
	pool    interface{ Close() }
	dlq     *gtcredis.DeadLetterWriter
	runtime *services.Runtime
}

func buildApplication(logger *zap.Logger) (*application, context.Context, context.CancelFunc, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, nil, nil, err
	}

	projections, err := config.LoadProjections(cfg.ProjectionConfigFile)
	if err != nil {
		return nil, nil, nil, err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL, logger)
	if err != nil {
		stop()
		return nil, nil, nil, err
	}

	checkpoints := postgres.NewCheckpointStore(pool, cfg.CheckpointSchema, cfg.CheckpointTable, logger)
	metadataStore := postgres.NewMetadataStore(pool, logger)
	snapshotter := postgres.NewSnapshotReader(
		pool,
		checkpoints,
		cfg.SnapshotBatchSize,
		cfg.SnapshotConcurrency,
		logger,
	)

	tailer := wal.NewReader(wal.Config{
		DatabaseURL:        cfg.DatabaseURL,
		SlotName:           cfg.SlotName,
		PublicationName:    cfg.PublicationName,
		PublicationTables:  projectionTables(projections),
		StandbyTimeout:     cfg.StandbyTimeout,
		AutoCreateSlot:     cfg.AutoCreateSlot,
		AutoCreatePub:      cfg.AutoCreatePublication,
		InactiveSlotAction: cfg.InactiveSlotAction,
		MaxLagBytes:        cfg.MaxLagBytes,
		SlotRetryInterval:  5 * time.Second,
		SlotRetryTimeout:   time.Minute,
	}, logger)

	redisJSONSink, err := gtcredis.NewJSONSink(cfg.RedisURL, logger)
	if err != nil {
		stop()
		pool.Close()
		return nil, nil, nil, err
	}

	redisStreamSink, err := gtcredis.NewStreamSink(cfg.RedisURL, logger)
	if err != nil {
		stop()
		pool.Close()
		return nil, nil, nil, err
	}

	dlqWriter, err := gtcredis.NewDeadLetterWriter(cfg.RedisURL, cfg.DLQStream, logger)
	if err != nil {
		stop()
		pool.Close()
		return nil, nil, nil, err
	}

	meiliSink := gtcmeili.NewSink(cfg.MeilisearchURL, cfg.MeilisearchAPIKey, logger)

	tcaStreamSink, err := gtcredis.NewTCAStreamSink(cfg.RedisURL, logger)
	if err != nil {
		stop()
		pool.Close()
		return nil, nil, nil, err
	}

	runtime, err := services.NewRuntime(services.RuntimeParams{
		TailReader:    tailer,
		Snapshotter:   snapshotter,
		Checkpoints:   checkpoints,
		MetadataStore: metadataStore,
		DeadLetter:    dlqWriter,
		Projections:   projections,
		Sinks: []ports.Sink{
			redisJSONSink,
			redisStreamSink,
			meiliSink,
			tcaStreamSink,
		},
		ProcessTimeout:  cfg.ProcessTimeout,
		WorkerCount:     cfg.WorkerCount,
		WorkerQueueSize: cfg.WorkerQueueSize,
		RetryMax:        cfg.RetryMaxAttempts,
		RetryBackoff:    cfg.RetryBackoff,
		Logger:          logger,
	})
	if err != nil {
		stop()
		pool.Close()
		_ = dlqWriter.Close()
		return nil, nil, nil, err
	}

	return &application{
		cfg:     cfg,
		pool:    pool,
		dlq:     dlqWriter,
		runtime: runtime,
	}, ctx, stop, nil
}

func (a *application) close() {
	_ = a.dlq.Close()
	a.pool.Close()
}

func projectionTables(projections []domain.Projection) []string {
	tables := make([]string, 0, len(projections))
	for _, projection := range projections {
		fullName := projection.FullTableName()
		if !slices.Contains(tables, fullName) {
			tables = append(tables, fullName)
		}
	}
	return tables
}

func csvList(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	parts := strings.Split(raw, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			items = append(items, trimmed)
		}
	}
	return items
}

func newLogger() *zap.Logger {
	level := parseLogLevel(os.Getenv("LOG_LEVEL"))

	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevelAt(level)
	cfg.Encoding = "json"
	cfg.DisableStacktrace = true
	cfg.EncoderConfig.TimeKey = "time"
	cfg.EncoderConfig.LevelKey = "level"
	cfg.EncoderConfig.MessageKey = "msg"
	cfg.EncoderConfig.NameKey = "component"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	if level == zapcore.DebugLevel {
		cfg.EncoderConfig.CallerKey = "caller"
	} else {
		cfg.EncoderConfig.CallerKey = ""
	}

	logger, err := cfg.Build()
	if err != nil {
		panic(err)
	}

	return logger
}

func parseLogLevel(s string) zapcore.Level {
	switch strings.ToUpper(s) {
	case "DEBUG":
		return zapcore.DebugLevel
	case "INFO", "":
		return zapcore.InfoLevel
	case "WARN", "WARNING":
		return zapcore.WarnLevel
	case "ERROR":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}
