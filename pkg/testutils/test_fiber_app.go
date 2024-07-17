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

package testutils

import (
	"testing"

	"github.com/emoss08/trenova/config"
	"github.com/emoss08/trenova/internal/server"
	"github.com/emoss08/trenova/pkg/gen"
	"github.com/rs/zerolog/log"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
)

// SetupTestServer initializes a new server for testing.
func SetupTestServer(t *testing.T) *server.Server {
	t.Helper()
	// Load configuration
	cfg, err := config.DefaultServiceConfigFromEnv()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load server configuration")
	}

	// Initialize logger
	logger := zerolog.New(log.Logger).With().Timestamp().Logger()
	if cfg.Logger.PrettyPrintConsole {
		logger = log.Output(zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
			w.TimeFormat = "15:04:05"
		}))
	}

	// Initialize Fiber app
	fiberApp := fiber.New()

	// Initialize TestDB
	testDB, cleanup := SetupTestCase(t)
	t.Cleanup(cleanup)

	// Initialize server
	s := &server.Server{
		Fiber:  fiberApp,
		Config: cfg,
		Logger: &logger,
		DB:     testDB.GetDB(),
	}

	// Initialize CodeGenerator and related components
	s.CounterManager = gen.NewCounterManager()
	s.CodeChecker = &gen.CodeChecker{DB: s.DB}
	s.CodeGenerator = gen.NewCodeGenerator(s.CounterManager, s.CodeChecker)
	s.CodeInitializer = &gen.CodeInitializer{DB: s.DB}

	// Generate and set up temporary RSA keys for the test
	privateKey, publicKey := SetTestKeys(t)

	// Update server config to use temporary RSA keys
	s.Config.Auth.PrivateKey = privateKey
	s.Config.Auth.PublicKey = publicKey

	return s
}
