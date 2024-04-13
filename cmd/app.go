package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/api/router"
	"github.com/emoss08/trenova/internal/config"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	config := config.DefaultServiceConfigFromEnv()

	zerolog.TimeFieldFormat = time.RFC3339Nano
	zerolog.SetGlobalLevel(config.Logger.Level)
	if config.Logger.PrettyPrintConsole {
		log.Logger = log.Output(zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
			w.TimeFormat = "15:04:05"
		}))
	}

	s := api.NewServer(config)

	if err := s.InitClient(context.Background()); err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize entity client")
	}

	if err := s.InitSessionStore(); err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize session store")
	}

	if err := s.InitLogger(); err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize logger")
	}

	router.Init(s)

	go func() {
		if err := s.Start(); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				log.Info().Msg("Server closed")
			} else {
				log.Fatal().Err(err).Msg("Failed to start server")
			}
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	<-c
	log.Info().Msg("Shutting down server...")
	_ = s.Shutdown()

	log.Info().Msg("Cleaning up...")
	if err := s.Cleanup(); err != nil {
		log.Fatal().Err(err).Msg("Failed to cleanup")
	}

	log.Info().Msg("Goodbye!")
}
