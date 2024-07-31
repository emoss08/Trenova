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

package fixtures

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/casbin/casbin/v2"
	"github.com/emoss08/trenova/config"
	"github.com/emoss08/trenova/pkg/audit"
	tCasbin "github.com/emoss08/trenova/pkg/casbin"
	"github.com/emoss08/trenova/pkg/gen"
	"github.com/emoss08/trenova/pkg/models"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bundebug"
)

func LoadFixtures() error {
	ctx := context.Background()

	serverConfig, err := config.DefaultServiceConfigFromEnv(false)
	if err != nil {
		log.Fatalf("Failed to load server configuration: %v", err)
		return err
	}

	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(serverConfig.DB.DSN())))
	db := bun.NewDB(sqldb, pgdialect.New())
	db.AddQueryHook(bundebug.NewQueryHook(
		bundebug.WithVerbose(true),
	))

	// Register the code generator.
	counterManager := gen.NewCounterManager()
	codeChecker := &gen.CodeChecker{DB: db}
	codeGen := gen.NewCodeGenerator(counterManager, codeChecker)
	codeInitializer := &gen.CodeInitializer{DB: db}

	// Register a new logger.
	logConfig := config.LoggerConfig{
		Level:              serverConfig.Logger.Level,
		PrettyPrintConsole: true,
		LogToFile:          true,
		LogFilePath:        serverConfig.Logger.LogFilePath,
		LogMaxSize:         100,
		LogMaxBackups:      3,
		LogMaxAge:          28,
		LogCompress:        true,
	}
	logger := config.NewLogger(logConfig)

	// Register the Audit Service.
	auditService := audit.NewAuditService(db, logger, serverConfig.Audit.QueueSize, serverConfig.Audit.WorkerCount)

	// Initialize the counter manager with existing codes
	err = codeInitializer.Initialize(ctx, counterManager, &models.Worker{})
	if err != nil {
		return err
	}

	// Initialize the casbin enforcer.

	enforcer, err := initCasbin(db, serverConfig.Casbin.ModelPath)
	if err != nil {
		return err
	}

	// Register many-to-many model so bun can better recognize m2m relation.
	// This should be done before you use the model for the first time.
	db.RegisterModel(
		(*models.GeneralLedgerAccountTag)(nil),
	)

	if err = loadResources(ctx, db); err != nil {
		log.Fatalf("Failed to load resources: %v", err)
		return err
	}

	if err = loadUSStates(ctx, db); err != nil {
		log.Fatalf("Failed to load US States: %v", err)
		return err
	}

	// Load the business unit
	bu, err := loadBusinessUnit(ctx, db)
	if err != nil {
		log.Fatalf("Failed to load business unit: %v", err)
		return err
	}

	// Load the organization
	org, err := loadOrganization(ctx, db, bu)
	if err != nil {
		log.Fatalf("Failed to load organization: %v", err)
		return err
	}

	// Load the Shipment Control
	if err = loadShipmentControl(ctx, db, org.ID, bu.ID); err != nil {
		log.Fatalf("Failed to load shipment control: %v", err)
	}

	// Load the master key generation
	mkg, err := LoadMasterKeyGeneration(ctx, db, org.ID, bu.ID)
	if err != nil {
		log.Fatalf("Failed to load master key generation: %v", err)
		return err
	}

	// Load the worker master key generation
	if err = LoadWorkerMasterKeyGeneration(ctx, db, mkg); err != nil {
		log.Fatalf("Failed to load worker master key generation: %v", err)
		return err
	}

	// Load the location master key generation
	if err = LoadLocationMasterKeyGeneration(ctx, db, mkg); err != nil {
		log.Fatalf("Failed to load location master key generation: %v", err)
		return err
	}

	// Load the customer master key generation
	if err = LoadCustomerMasterKeyGeneration(ctx, db, mkg); err != nil {
		log.Fatalf("Failed to load equipment master key generation: %v", err)
		return err
	}

	user, err := LoadAdminAccount(ctx, db, enforcer, org, bu)
	if err != nil {
		log.Fatalf("Failed to load admin account: %v", err)
		return err
	}

	// Load the workers
	if err = loadWorkers(ctx, db, codeGen, auditService, user, org.ID, bu.ID); err != nil {
		log.Fatalf("Failed to load workers: %v", err)
		return err
	}

	// Load the customers
	if err = loadCustomers(ctx, db, codeGen, auditService, user, org.ID, bu.ID); err != nil {
		log.Fatalf("Failed to load customers: %v", err)
		return err
	}

	// Load the shipments
	if err = loadShipments(ctx, db, codeGen, auditService, user, org.ID, bu.ID); err != nil {
		log.Fatalf("Failed to load shipments: %v", err)
		return err
	}

	// Load the equipment types
	if err = loadEquipmentTypes(ctx, db, org.ID, bu.ID); err != nil {
		log.Fatalf("Failed to load equipment types: %v", err)
		return err
	}

	// Load the admin account
	if err = InitializeCasbinPolicies(ctx, db, enforcer, org, bu); err != nil {
		log.Fatalf("Failed to load admin account: %v", err)
		return err
	}

	return nil
}

func InitializeCasbinPolicies(ctx context.Context, db *bun.DB, enforcer *casbin.Enforcer, org *models.Organization, bu *models.BusinessUnit) error {
	if err := loadPermissions(ctx, db, enforcer); err != nil {
		return err
	}

	if err := LoadNormalAccount(ctx, db, org, bu); err != nil {
		return err
	}

	return nil
}

func initCasbin(db *bun.DB, modelPath string) (*casbin.Enforcer, error) {
	adapter, err := tCasbin.NewBunAdapter(db)
	if err != nil {
		return nil, err
	}

	enforcer, err := casbin.NewEnforcer(modelPath, adapter)
	if err != nil {
		return nil, err
	}

	// Load the policy from the adapter
	if err = enforcer.LoadPolicy(); err != nil {
		return nil, fmt.Errorf("failed to load Casbin policy: %w", err)
	}

	return enforcer, nil
}
