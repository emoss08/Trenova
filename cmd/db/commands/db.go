package commands

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/lib/pq"
	"github.com/spf13/cobra"
	"github.com/trenova-app/transport/internal/infrastructure/database/postgres/migrations"
	"github.com/trenova-app/transport/internal/pkg/config"
	"github.com/trenova-app/transport/internal/pkg/registry"
	"github.com/trenova-app/transport/test/fixtures"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dbfixture"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bundebug"
	"github.com/uptrace/bun/migrate"
)

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Database CLI",
	Long:  `A complete CLI for managing the database.`,
}

func init() {
	rootCmd.AddCommand(dbCmd)

	// Subcommands for the database.
	dbCmd.AddCommand(createTxSQLCmd)
	dbCmd.AddCommand(initCmd)
	dbCmd.AddCommand(resetCmd)
	dbCmd.AddCommand(rollbackCmd)
	dbCmd.AddCommand(migrateCmd)
	dbCmd.AddCommand(dbSeedCmd)
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create migration tables",
	Run: func(_ *cobra.Command, _ []string) {
		migrator := getMigrator(GetDB())
		if err := migrator.Init(context.Background()); err != nil {
			fmt.Printf("Failed to initialize migration tables: %v\n", err)
		} else {
			fmt.Println("Migration tables created successfully")
		}
	},
}

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate database",
	Run: func(cmd *cobra.Command, args []string) {
		migrator := getMigrator(GetDB())

		if err := migrator.Lock(context.Background()); err != nil {
			fmt.Printf("Failed to lock migrations: %v\n", err)
			return
		}

		defer func(mig *migrate.Migrator, ctx context.Context) {
			err := mig.Unlock(ctx)
			if err != nil {
				fmt.Printf("Failed to unlock migrations: %v\n", err)
			}
		}(migrator, context.Background())

		group, err := migrator.Migrate(context.Background())
		if err != nil {
			fmt.Printf("Migration failed: %v\n", err)
			return
		}
		if group.IsZero() {
			fmt.Println("There are no new migrations to run (database is up to date)")
		} else {
			fmt.Printf("Migrated to %s\n", group)
		}
	},
}

var createTxSQLCmd = &cobra.Command{
	Use:   "create_tx_sql [name]",
	Short: "Create up and down transactional SQL migrations",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		migrator := getMigrator(GetDB())

		name := strings.Join(args, "_")
		files, err := migrator.CreateTxSQLMigrations(context.Background(), name)
		if err != nil {
			fmt.Printf("Failed to create transactional SQL migrations: %v\n", err)
		} else {
			for _, mf := range files {
				fmt.Printf("Created transaction migration %s (%s)\n", mf.Name, mf.Path)
			}
		}
	},
}

var rollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "Rollback the last migration group",
	Run: func(cmd *cobra.Command, args []string) {
		migrator := getMigrator(GetDB())

		if err := migrator.Lock(context.Background()); err != nil {
			fmt.Printf("Failed to lock migrations: %v\n", err)
			return
		}
		defer func(mig *migrate.Migrator, ctx context.Context) {
			err := mig.Unlock(ctx)
			if err != nil {
				fmt.Printf("Failed to unlock migrations: %v\n", err)
			}
		}(migrator, context.Background())

		group, err := migrator.Rollback(context.Background())
		if err != nil {
			fmt.Printf("Rollback failed: %v\n", err)
			return
		}
		if group.IsZero() {
			fmt.Println("There are no groups to roll back")
		} else {
			fmt.Printf("Rolled back %s\n", group)
		}
	},
}

var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset the database by dropping it and creating a new one",
	Run: func(_ *cobra.Command, args []string) {
		manager, _ := GetConfigManager()

		conn, err := pq.NewConnector(manager.GetDSN())
		if err != nil {
			log.Fatalf("Failed to create database connector: %v", err)
		}

		db := sql.OpenDB(conn)
		defer db.Close()

		// Drop the public schema and recreate it
		query := "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
		if _, err = db.ExecContext(context.Background(), query); err != nil {
			log.Fatalf("Failed to drop database: %v", err)
		}

		fmt.Println("Schema dropped and recreated successfully")
	},
}

var dbSeedCmd = &cobra.Command{
	Use:   "seed",
	Short: "Seed the database with fixtures",
	Run: func(_ *cobra.Command, _ []string) {
		ctx := context.Background()
		db := GetDB()

		// Register models before loading fixtures.
		db.RegisterModel(registry.RegisterEntities()...)

		// initialize fixture helpers
		helpers := fixtures.NewFixtureHelpers()

		// Load fixtures
		fixture := dbfixture.New(db, dbfixture.WithTemplateFuncs(helpers.GetTemplateFuncs()))
		if err := fixture.Load(ctx, os.DirFS("./test/fixtures"), "fixtures.yml"); err != nil {
			// Log the full directory structure
			log.Printf("Failed to load fixtures: %v", err)
			log.Printf("Full directory structure: %v", os.DirFS("./test/fixtures"))
			log.Fatalf("Failed to load fixtures: %v", err)
		}

		fmt.Println("Fixtures loaded successfully")

		// Load additional fixtures via seeder
		if err := fixtures.LoadFixtures(ctx, fixture, db); err != nil {
			log.Fatalf("Failed to load additional fixtures: %v", err)
		}

		fmt.Println("Additional fixtures loaded successfully")
	},
}

func GetDB() *bun.DB {
	manager, _ := GetConfigManager()

	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(manager.GetDSN())))
	db := bun.NewDB(sqldb, pgdialect.New())
	db.AddQueryHook(bundebug.NewQueryHook(
		bundebug.WithVerbose(false),
		bundebug.FromEnv("BUNDEBUG"),
	))

	db.RegisterModel(registry.RegisterEntities()...)

	return db
}

func GetConfigManager() (*config.Manager, *config.Config) {
	manager := config.NewManager()

	cfg, err := manager.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	return manager, cfg
}

func getMigrator(db *bun.DB) *migrate.Migrator {
	return migrate.NewMigrator(db, migrations.Migrations)
}
