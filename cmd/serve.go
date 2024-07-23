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
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/emoss08/trenova/config"
	"github.com/emoss08/trenova/internal/api/router"
	"github.com/emoss08/trenova/internal/server"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the Trenova server",
	Long: `Start the Trenova server. This command will start the Trenova server and listen for incoming requests.

	The server will start on the port specified in the environment variable
	TRENOVA_PORT. If the environment variable is not set, the server will start on port 3001.`,
	Run: func(_ *cobra.Command, _ []string) {
		runServer()
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}

func runServer() {
	ctx := context.Background()
	serverConfig, err := config.DefaultServiceConfigFromEnv(false)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load server configuration")
	}

	s := server.NewServer(ctx, serverConfig)

	// Load the RSA keys.
	if err = s.LoadRSAKeys(); err != nil {
		log.Fatal().Err(err).Msg("Failed to load RSA keys")
	}

	// Initialize the Minio client.
	if err = s.InitMinioClient(); err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize Minio client")
	}

	s.InitLogger()
	s.InitDB()
	s.InitCache()

	// Initialize the Kafka client.
	if err = s.InitKafkaClient(); err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize Kafka client")
	}

	if err = s.InitAuditService(); err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize audit service")
	}

	if err = s.InitCasbin(); err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize Casbin")
	}

	// Initialize the code generator.
	if err = s.InitCodeGenerationSystem(ctx); err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize code generator")
	}

	// Initialize the Fiber server and routes.
	router.Init(s)

	go func() {
		if err := s.Start(); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				log.Debug().Msg("Server shutdown")
			} else {
				log.Fatal().Err(err).Msg("Failed to start server")
			}
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	<-c
	log.Debug().Msg("Shutting down server")
	_ = s.Shutdown()

	log.Debug().Msg("Server shutdown complete")
}
