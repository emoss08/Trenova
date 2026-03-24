package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/emoss08/trenova/samsara-sim/internal/config"
	"github.com/emoss08/trenova/samsara-sim/internal/sim"
)

func main() {
	os.Exit(run())
}

func run() int {
	configPath := flag.String("config", "", "path to simulator config file")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Error("failed to load config", slog.String("error", err.Error()))
		return 1
	}

	store, err := sim.NewStoreFromFixtureFile(cfg.Seed.FixturePath)
	if err != nil {
		logger.Error("failed to load fixture", slog.String("error", err.Error()))
		return 1
	}
	if err = sim.ApplyRouteDataset(store, cfg.Seed.RouteDatasetPath); err != nil {
		logger.Warn("failed to apply route dataset", slog.String("error", err.Error()))
	}

	scenarios, err := sim.NewScenarioEngine(
		cfg.Seed.DeterministicSeed,
		cfg.Scenario.DefaultProfile,
	)
	if err != nil {
		logger.Error("failed to initialize scenario engine", slog.String("error", err.Error()))
		return 1
	}

	dispatcher := sim.NewDispatcher(cfg.Webhooks, store, logger.With("component", "webhooks"))
	defer dispatcher.Shutdown()

	server := sim.NewServer(
		&cfg,
		store,
		scenarios,
		dispatcher,
		logger.With("component", "http"),
	).HTTPServer()

	logger.Info("starting samsara simulator", slog.String("addr", server.Addr))

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.ListenAndServe()
	}()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err = <-serverErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("simulator server failed", slog.String("error", err.Error()))
			return 1
		}
	case sig := <-signalChan:
		logger.Info("received shutdown signal", slog.String("signal", sig.String()))
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err = server.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", slog.String("error", err.Error()))
		return 1
	}

	logger.Info("samsara simulator stopped")
	return 0
}
