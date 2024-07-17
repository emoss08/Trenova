package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/config"
	"github.com/emoss08/trenova/fixtures"
	"github.com/emoss08/trenova/migrate/migrations"
	"github.com/emoss08/trenova/pkg/models"
	"github.com/spf13/cobra"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bundebug"
	"github.com/uptrace/bun/migrate"
)

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Manage database migrations",
	Long:  `Perform database migration operations such as init, migrate, rollback, etc.`,
}

func init() {
	rootCmd.AddCommand(dbCmd)

	// Add subcommands
	dbCmd.AddCommand(initCmd)
	dbCmd.AddCommand(migrateCmd)
	dbCmd.AddCommand(rollbackCmd)
	dbCmd.AddCommand(lockCmd)
	dbCmd.AddCommand(unlockCmd)
	dbCmd.AddCommand(createGoCmd)
	dbCmd.AddCommand(createSQLCmd)
	dbCmd.AddCommand(createTxSQLCmd)
	dbCmd.AddCommand(statusCmd)
	dbCmd.AddCommand(resetCmd)
	dbCmd.AddCommand(markAppliedCmd)
	dbCmd.AddCommand(dbSeedCmd)
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create migration tables",
	Run: func(cmd *cobra.Command, args []string) {
		migrator := getMigrator()
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
		migrator := getMigrator()
		if err := migrator.Lock(context.Background()); err != nil {
			fmt.Printf("Failed to lock migrations: %v\n", err)
			return
		}
		defer migrator.Unlock(context.Background()) //nolint:errcheck

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

var rollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "Rollback the last migration group",
	Run: func(cmd *cobra.Command, args []string) {
		migrator := getMigrator()
		if err := migrator.Lock(context.Background()); err != nil {
			fmt.Printf("Failed to lock migrations: %v\n", err)
			return
		}
		defer migrator.Unlock(context.Background()) //nolint:errcheck

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

var lockCmd = &cobra.Command{
	Use:   "lock",
	Short: "Lock migrations",
	Run: func(cmd *cobra.Command, args []string) {
		migrator := getMigrator()
		if err := migrator.Lock(context.Background()); err != nil {
			fmt.Printf("Failed to lock migrations: %v\n", err)
		} else {
			fmt.Println("Migrations locked successfully")
		}
	},
}

var unlockCmd = &cobra.Command{
	Use:   "unlock",
	Short: "Unlock migrations",
	Run: func(cmd *cobra.Command, args []string) {
		migrator := getMigrator()
		if err := migrator.Unlock(context.Background()); err != nil {
			fmt.Printf("Failed to unlock migrations: %v\n", err)
		} else {
			fmt.Println("Migrations unlocked successfully")
		}
	},
}

var createGoCmd = &cobra.Command{
	Use:   "create_go [name]",
	Short: "Create Go migration",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		migrator := getMigrator()
		name := strings.Join(args, "_")
		mf, err := migrator.CreateGoMigration(context.Background(), name)
		if err != nil {
			fmt.Printf("Failed to create Go migration: %v\n", err)
		} else {
			fmt.Printf("Created migration %s (%s)\n", mf.Name, mf.Path)
		}
	},
}

var createSQLCmd = &cobra.Command{
	Use:   "create_sql [name]",
	Short: "Create up and down SQL migrations",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		migrator := getMigrator()
		name := strings.Join(args, "_")
		files, err := migrator.CreateSQLMigrations(context.Background(), name)
		if err != nil {
			fmt.Printf("Failed to create SQL migrations: %v\n", err)
		} else {
			for _, mf := range files {
				fmt.Printf("Created migration %s (%s)\n", mf.Name, mf.Path)
			}
		}
	},
}

var createTxSQLCmd = &cobra.Command{
	Use:   "create_tx_sql [name]",
	Short: "Create up and down transactional SQL migrations",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		migrator := getMigrator()
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

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Print migrations status",
	Run: func(cmd *cobra.Command, args []string) {
		migrator := getMigrator()
		ms, err := migrator.MigrationsWithStatus(context.Background())
		if err != nil {
			fmt.Printf("Failed to get migrations status: %v\n", err)
		} else {
			fmt.Printf("Migrations: %s\n", ms)
			fmt.Printf("Unapplied migrations: %s\n", ms.Unapplied())
			fmt.Printf("Last migration group: %s\n", ms.LastGroup())
		}
	},
}

var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset the database by rolling back all migrations and then running them again",
	Run: func(cmd *cobra.Command, args []string) {
		migrator := getMigrator()
		if err := migrator.Lock(context.Background()); err != nil {
			fmt.Printf("Failed to lock migrations: %v\n", err)
			return
		}
		defer migrator.Unlock(context.Background()) //nolint:errcheck

		if err := migrator.Reset(context.Background()); err != nil {
			fmt.Printf("Failed to reset migrations: %v\n", err)
			return
		}

		if _, err := migrator.Migrate(context.Background()); err != nil {
			fmt.Printf("Failed to re-run migrations: %v\n", err)
		} else {
			fmt.Println("Database reset and migrations re-run successfully")
		}
	},
}

var markAppliedCmd = &cobra.Command{
	Use:   "mark_applied",
	Short: "Mark migrations as applied without actually running them",
	Run: func(cmd *cobra.Command, args []string) {
		migrator := getMigrator()
		group, err := migrator.Migrate(context.Background(), migrate.WithNopMigration())
		if err != nil {
			fmt.Printf("Failed to mark migrations as applied: %v\n", err)
		} else if group.IsZero() {
			fmt.Println("There are no new migrations to mark as applied")
		} else {
			fmt.Printf("Marked as applied %s\n", group)
		}
	},
}

func getMigrator() *migrate.Migrator {
	serverConfig, err := config.DefaultServiceConfigFromEnv()
	if err != nil {
		panic(err)
	}

	db := getDB(serverConfig)
	return migrate.NewMigrator(db, migrations.Migrations)
}

func getDB(serverConfig config.Server) *bun.DB {
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(serverConfig.DB.DSN())))
	db := bun.NewDB(sqldb, pgdialect.New())
	db.AddQueryHook(bundebug.NewQueryHook(
		bundebug.WithVerbose(true),
	))

	// Register many to many model so bun can better recognize m2m relation.
	// This should be done before you use the model for the first time.
	db.RegisterModel(
		(*models.GeneralLedgerAccountTag)(nil),
	)

	return db
}

var dbSeedCmd = &cobra.Command{
	Use:   "db_seed",
	Short: "Seed the database with fixtures",
	Run: func(_ *cobra.Command, _ []string) {
		if err := fixtures.LoadFixtures(); err != nil {
			panic(err)
		}
	},
}
