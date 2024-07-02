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
	// Load configuration
	cfg := config.DefaultServiceConfigFromEnv()

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
