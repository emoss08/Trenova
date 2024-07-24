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

package testutils

import (
	"testing"

	"github.com/emoss08/trenova/config"
	"github.com/emoss08/trenova/internal/server"
	"github.com/emoss08/trenova/pkg/gen"
	"github.com/rs/zerolog/log"

	"github.com/gofiber/fiber/v2"
)

// SetupTestServer initializes a new server for testing.
func SetupTestServer(t *testing.T) *server.Server {
	t.Helper()
	// Load configuration
	cfg, err := config.DefaultServiceConfigFromEnv(true)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load server configuration")
	}

	// Initialize logger
	logConfig := config.LoggerConfig{
		Level:              0,
		PrettyPrintConsole: true,
		LogToFile:          true,
		LogFilePath:        "/var/log/myapp.log",
		LogMaxSize:         100,
		LogMaxBackups:      3,
		LogMaxAge:          28,
		LogCompress:        true,
	}
	logger := config.NewLogger(logConfig)

	// Initialize Fiber app
	fiberApp := fiber.New()

	// Initialize TestDB
	testDB, cleanup := SetupTestCase(t)
	t.Cleanup(cleanup)

	// Initialize server
	s := &server.Server{
		Fiber:  fiberApp,
		Config: cfg,
		Logger: logger,
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
