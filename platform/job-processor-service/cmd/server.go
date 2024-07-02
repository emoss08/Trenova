package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/emoss08/trenova-bg-jobs/internal/config"
	"github.com/emoss08/trenova-bg-jobs/internal/server"
	"github.com/emoss08/trenova-bg-jobs/internal/util"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Starts the Trenova job processor service",
	Long: `Starts the Trenova job processor service
	
	
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

	s := server.NewServer(serverConfig)

	if err := s.InitDB(ctx); err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize database")
	}

	if err := s.InitalizeLogger(); err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize logger")
	}

	if err := s.InitRedisClient(ctx); err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize Redis client")
	}

	if err := s.InitAsynqClient(); err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize Asynq client")
	}

	// Example: Enqueue a Send Report Task
	reportID := 123
	if err := s.Enqueuer.EnqueueSendReportTask(reportID); err != nil {
		log.Fatal().Err(err).Msg("Failed to enqueue send report task:")
	}

	log.Info().Msg("Send report task enqueued successfully")

	// Inspect queue
	if err := util.InspectQueue(serverConfig.Redis.Addr); err != nil {
		log.Fatal().Err(err).Msg("Failed to inspect queue")
	}

	go func() {
		if err := s.Start(); err != nil {
			log.Fatal().Err(err).Msg("Failed to start server")
		}
	}()

	// Wait for termination signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Received shutdown signal, shutting down gracefully...")

	// Create a context with timeout for the shutdown process
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.Shutdown(shutdownCtx); err != nil {
		log.Fatal().Err(err).Msg("Failed to shut down server gracefully")
	}

	log.Info().Msg("Server shut down gracefully")
}
