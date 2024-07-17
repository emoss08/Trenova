// Copyright (c) 2024 Trenova Technologies, LLC
//
// Licensed under the Business Source License 1.1 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://trenova.app/pricing/
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//
// Key Terms:
// - Non-production use only
// - Change Date: 2026-11-16
// - Change License: GNU General Public License v2 or later
//
// For full license text, see the LICENSE file in the root directory.

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
	serverConfig, err := config.DefaultServiceConfigFromEnv()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load server configuration")
	}

	s := server.NewServer(ctx, serverConfig)

	// Load the RSA keys.
	if err := s.LoadRSAKeys(); err != nil {
		log.Fatal().Err(err).Msg("Failed to load RSA keys")
	}

	// Initialize the Minio client.
	if err := s.InitMinioClient(); err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize Minio client")
	}

	// Initialize the Kafka client.
	if err := s.InitKafkaClient(); err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize Kafka client")
	}

	s.InitLogger()
	s.InitDB()

	if err := s.InitCasbin(); err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize Casbin")
	}

	// Initialize the code generator.
	if err := s.InitCodeGenerationSystem(ctx); err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize code generator")
	}

	// Initialize the Fiber server and routes.
	router.Init(s)

	go func() {
		if err := s.Start(); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				log.Info().Msg("Server shutdown")
			} else {
				log.Fatal().Err(err).Msg("Failed to start server")
			}
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	<-c
	log.Info().Msg("Shutting down server")
	_ = s.Shutdown()

	log.Info().Msg("Server shutdown complete")
}
