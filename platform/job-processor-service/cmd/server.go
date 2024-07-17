// COPYRIGHT(c) 2024 Trenova
//
// This file is part of Trenova.
//
// The Trenova software is licensed under the Business Source License 1.1. You are granted the right
// to copy, modify, and redistribute the software, but only for non-production use or with a total
// of less than three server instances. Starting from the Change Date (November 16, 2026), the
// software will be made available under version 2 or later of the GNU General Public License.
// If you use the software in violation of this license, your rights under the license will be
// terminated automatically. The software is provided "as is," and the Licensor disclaims all
// warranties and conditions. If you use this license's text or the "Business Source License" name
// and trademark, you must comply with the Licensor's covenants, which include specifying the
// Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
// Grant, and not modifying the license in any other way.

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
