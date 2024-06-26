package cmd

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
	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Starts the Trenova Server",
	Long: `Starts the stateless RESTful JSON server
	
	
	Requires configuration throguh ENV
	and a fully migrated PostgreSQL database.`,
	Run: func(cmd *cobra.Command, args []string) {
		runServer()
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)
}

func runServer() {
	ctx := context.Background()
	serverConfig := config.DefaultServiceConfigFromEnv()

	zerolog.TimeFieldFormat = time.RFC3339Nano
	zerolog.SetGlobalLevel(serverConfig.Logger.Level)
	if serverConfig.Logger.PrettyPrintConsole {
		log.Logger = log.Output(zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
			w.TimeFormat = "15:04:05"
		}))
	}

	s := api.NewServer(serverConfig)

	if err := s.LoadRSAKeys(); err != nil {
		log.Fatal().Err(err).Msg("Failed to load RSA keys")
	}

	if err := s.InitClient(ctx); err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize entity client")
	}

	if err := s.InitSessionStore(); err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize session store")
	}

	if err := s.InitLogger(); err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize logger")
	}

	if err := s.InitKafkaClient(); err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize Kafka client")
	}

	if err := s.InitRedisClient(ctx); err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize Redis client")
	}

	if err := s.InitMinioClient(ctx); err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize Minio client")
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
