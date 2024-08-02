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
	"os"
	"sync"
	"syscall"
	"testing"

	"github.com/gofiber/fiber/v2"

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
)

var (
	sharedDBURL string
	dbOnce      sync.Once
	dbCleanup   func()
)

const (
	dbURLFile = "/tmp/trenova_test_db_url"
	lockFile  = "/tmp/trenova_test.lock"
)

func InitTestEnvironment() {
	dbOnce.Do(func() {
		file, err := os.OpenFile(lockFile, os.O_CREATE|os.O_RDWR, 0666)
		if err != nil {
			panic(fmt.Sprintf("cannot open lock file: %v", err))
		}
		defer func(file *os.File) {
			err := file.Close()
			if err != nil {
				panic(fmt.Sprintf("cannot close lock file: %v", err))
			}
		}(file)

		err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX)
		if err != nil {
			panic(fmt.Sprintf("cannot flock: %v", err))
		}
		defer func(fd int, how int) {
			err := syscall.Flock(fd, how)
			if err != nil {
				panic(fmt.Sprintf("cannot flock: %v", err))
			}
		}(int(file.Fd()), syscall.LOCK_UN)

		// Always try to create a new database
		url, cleanup, err := createTestDatabase()
		if err != nil {
			panic(fmt.Sprintf("cannot create test database: %v", err))
		}
		dbCleanup = cleanup
		sharedDBURL = url

		// Write the URL to the file
		err = os.WriteFile(dbURLFile, []byte(url), 0666)
		if err != nil {
			panic(fmt.Sprintf("cannot write DB URL to file: %v", err))
		}
	})
}

func GetTestDatabaseURL() string {
	InitTestEnvironment()
	return sharedDBURL
}

func CleanupTestEnvironment() {
	if dbCleanup != nil {
		dbCleanup()
	}
	err := os.Remove(dbURLFile)
	if err != nil {
		panic(fmt.Sprintf("cannot remove DB URL file: %v", err))
	}
}

// SetupTestServer initializes a new server for testing.
func SetupTestServer(t *testing.T) (*server.Server, func()) {
	t.Helper()

	// Get the shared database URL
	databaseURL := GetTestDatabaseURL()

	// Create a new DB connection
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(databaseURL)))
	db := bun.NewDB(sqldb, pgdialect.New())
	db.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose(false)))

	// Register models
	db.RegisterModel(
		(*models.GeneralLedgerAccountTag)(nil),
	)

	// Run migrations
	migrator := migrate.NewMigrator(db, migrations.Migrations)
	if err := migrator.Init(context.Background()); err != nil {
		t.Fatalf("Failed to initialize migrator: %v", err)
	}

	if err := migrator.Lock(context.Background()); err != nil {
		t.Fatalf("Failed to lock migrations: %v", err)
	}

	defer func(migrator *migrate.Migrator, ctx context.Context) {
		err := migrator.Unlock(ctx)
		if err != nil {
			t.Fatalf("Failed to unlock migrations: %v", err)
		}
	}(migrator, context.Background())

	group, err := migrator.Migrate(context.Background())
	if err != nil {
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
		Audit: config.AuditConfig{
			QueueSize:   1000, // Increased queue size for tests
			WorkerCount: 10,   // Increased worker count for tests
		},
		Cache: config.Cache{},
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
	s.Cache = redis.NewClient(&redis.Options{Addr: cfg.Cache.Addr}, s.Logger)

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
		if err = s.DB.Close(); err != nil {
			t.Errorf("Failed to close database connection: %v", err)
		}

		// Close other resources
		_ = s.Cache.Close()
		_ = s.AuditService.Shutdown(context.Background())
	}

	return s, cleanup
}
