package cmd

import (
	"context"
	"database/sql"
	"log"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/emoss08/trenova/cmd/migratedata"
	"github.com/emoss08/trenova/internal/config"
	"github.com/emoss08/trenova/internal/ent"
	_ "github.com/emoss08/trenova/internal/ent/runtime" // ent codegen
	_ "github.com/jackc/pgx/v5/stdlib"                  // pgx driver
	"github.com/spf13/cobra"
)

// seederCmd represents the seeder command
var seederCmd = &cobra.Command{
	Use:   "seeder",
	Short: "Inserts or updates fixtures in the database.",
	Long: `The seeder command is used to insert or update fixtures in the database. 
	
	This command is useful for populating the database with
	initial data or updating existing data.`,
	Run: seedCmdFunc,
}

func init() {
	rootCmd.AddCommand(seederCmd)
}

func seedCmdFunc(_ *cobra.Command, _ []string) {
	if err := applyFixtures(); err != nil {
		log.Fatalf("failed to apply fixtures: %v", err)
		return
	}

	log.Print("fixtures applied successfully\n")
}

func initClient() (*ent.Client, error) {
	config := config.DefaultServiceConfigFromEnv()
	// Initialize the new db connection
	db, err := sql.Open("pgx", config.DB.ConnectionString())
	if err != nil {
		log.Printf("failed opening connection to postgres: %v\n", err)
		return nil, err
	}

	db.SetMaxIdleConns(10)
	db.SetMaxOpenConns(100)
	db.SetConnMaxLifetime(time.Hour)
	db.SetConnMaxIdleTime(time.Minute * 30)

	drv := entsql.OpenDB(dialect.Postgres, db)
	client := ent.NewClient(ent.Driver(drv))
	return client, nil
}

func applyFixtures() error {
	ctx := context.Background()

	// Initialize the new db connection
	client, err := initClient()
	if err != nil {
		return err
	}

	// Seed Resources
	if err = migratedata.SeedResources(ctx, client); err != nil {
		return err
	}

	// Check if the business unit already exists.
	bu, err := migratedata.SeedBusinessUnits(ctx, client)
	if err != nil {
		return err
	}

	// Check if the organization already exists.
	org, err := migratedata.SeedOrganization(ctx, client, bu)
	if err != nil {
		return err
	}

	// Seed permissions for each resource for the created organization and business unit.
	if err = migratedata.SeedPermissions(ctx, client, org, bu); err != nil {
		return err
	}

	// Seed roles for the created organization and business unit.
	if err = migratedata.SeedRoles(ctx, client, org, bu); err != nil {
		return err
	}

	// Seed the States in the US.
	if err = migratedata.SeedUSStates(ctx, client); err != nil {
		return err
	}

	// Seed the feature flags for the created organization.
	if err = migratedata.SeedFeatureFlags(ctx, client, org); err != nil {
		return err
	}

	if err = migratedata.SeedAccountingControl(ctx, client, org, bu); err != nil {
		return err
	}

	if err = migratedata.SeedBillingControl(ctx, client, org, bu); err != nil {
		return err
	}

	if err = migratedata.SeedDispatchControl(ctx, client, org, bu); err != nil {
		return err
	}

	if err = migratedata.SeedShipmentControl(ctx, client, org, bu); err != nil {
		return err
	}

	if err = migratedata.SeedInvoiceControl(ctx, client, org, bu); err != nil {
		return err
	}

	if err = migratedata.SeedRouteControl(ctx, client, org, bu); err != nil {
		return err
	}

	if err = migratedata.SeedCustomers(ctx, client, org, bu); err != nil {
		return err
	}

	if err = migratedata.SeedEmailControl(ctx, client, org, bu); err != nil {
		return err
	}

	if err = migratedata.SeedFeasibilityToolControl(ctx, client, org, bu); err != nil {
		return err
	}

	if err = migratedata.SeedGoogleAPI(ctx, client, org, bu); err != nil {
		return err
	}

	if err = migratedata.SeedRevenueCodes(ctx, client, org, bu); err != nil {
		return err
	}

	if err = migratedata.SeedGeneralLedgerAccounts(ctx, client, org, bu); err != nil {
		return err
	}

	if err = migratedata.SeedEquipmentTypes(ctx, client, org, bu); err != nil {
		return err
	}

	if err = migratedata.SeedEquipmentManufacturers(ctx, client, org, bu); err != nil {
		return err
	}

	if err = migratedata.SeedCommentTypes(ctx, client, org, bu); err != nil {
		return err
	}

	if err = migratedata.SeedAdminAccount(ctx, client, org, bu); err != nil {
		return err
	}

	if err = migratedata.SeedNormalAccount(ctx, client, org, bu); err != nil {
		return err
	}

	if err = migratedata.SeedRates(ctx, client, org, bu); err != nil {
		return err
	}

	return nil
}
