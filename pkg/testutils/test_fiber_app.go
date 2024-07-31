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
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/emoss08/trenova/config"
	"github.com/emoss08/trenova/internal/server"
	"github.com/emoss08/trenova/migrate/migrations"
	"github.com/emoss08/trenova/pkg/audit"
	"github.com/emoss08/trenova/pkg/gen"
	"github.com/emoss08/trenova/pkg/models"
	"github.com/emoss08/trenova/pkg/redis"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bundebug"
	"github.com/uptrace/bun/migrate"

	"github.com/gofiber/fiber/v2"
)

// SetupTestServer initializes a new server for testing.
func SetupTestServer(t *testing.T) (*server.Server, func()) {
	t.Helper()
	// Initialize test database
	databaseURL, closeDatabase, err := initDatabase()
	if err != nil {
		t.Fatalf("Failed to initialize test database: %v", err)
	}

	// Create a new DB connection
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(databaseURL)))
	db := bun.NewDB(sqldb, pgdialect.New())
	db.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose(true)))

	// Register models
	db.RegisterModel(
		(*models.GeneralLedgerAccountTag)(nil),
		// Add other models as needed
	)

	// Run migrations
	migrator := migrate.NewMigrator(db, migrations.Migrations)
	if err := migrator.Init(context.Background()); err != nil {
		closeDatabase()
		t.Fatalf("Failed to initialize migrator: %v", err)
	}

	if err := migrator.Lock(context.Background()); err != nil {
		closeDatabase()
		t.Fatalf("Failed to lock migrations: %v", err)
	}
	defer migrator.Unlock(context.Background()) //nolint:errcheck

	group, err := migrator.Migrate(context.Background())
	if err != nil {
		closeDatabase()
		t.Fatalf("Failed to run migrations: %v", err)
	}
	if !group.IsZero() {
		fmt.Printf("Migrated to %s\n", group)
	}
	// Load configuration
	cfg := config.Server{
		// Initialize with test-specific configurations
		Fiber:       config.FiberServer{},
		DB:          config.Database{},
		Auth:        config.Auth{},
		Cors:        config.Cors{},
		Logger:      config.LoggerConfig{},
		Minio:       config.Minio{},
		Kafka:       config.KafkaServer{},
		Integration: config.Integration{},
		Casbin:      config.CasbinConfig{},
		Audit:       config.AuditConfig{},
		Cache:       config.Cache{},
	}

	// Initialize server
	s := server.NewServer(context.Background(), cfg)

	// Set the DB in the server
	s.DB = db

	// Initialize logger
	s.InitLogger()

	// Initialize Fiber app
	s.Fiber = fiber.New()

	// Initialize Casbin (if needed for tests)
	_ = s.InitCasbin()

	// Initialize Audit Service
	s.AuditService = audit.NewAuditService(s.DB, s.Logger, cfg.Audit.QueueSize, cfg.Audit.WorkerCount)

	// Initialize Code Generation System
	s.CounterManager = gen.NewCounterManager()
	s.CodeChecker = &gen.CodeChecker{DB: s.DB}
	s.CodeGenerator = gen.NewCodeGenerator(s.CounterManager, s.CodeChecker)
	s.CodeInitializer = &gen.CodeInitializer{DB: s.DB}

	// Initialize Cache
	s.Cache = redis.NewClient(&redis.Options{
		Addr: cfg.Cache.Addr,
	}, s.Logger)

	// Generate and set up temporary RSA keys for the test
	privateKey, publicKey := SetTestKeys(t)
	s.Config.Auth.PrivateKey = privateKey
	s.Config.Auth.PublicKey = publicKey

	// Initialize Code Generation System
	err = s.InitCodeGenerationSystem(context.Background())
	if err != nil {
		t.Fatalf("Failed to initialize code generation system: %v", err)
	}

	cleanup := func() {
		// Close DB connection
		if err := s.DB.Close(); err != nil {
			t.Errorf("Failed to close database connection: %v", err)
		}
		closeDatabase()

		// Close other resources
		_ = s.Cache.Close()
		_ = s.AuditService.Shutdown(context.Background())
		// Add any other necessary cleanup
	}

	return s, cleanup
}
