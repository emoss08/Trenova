package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/emoss08/trenova/config"
	"github.com/emoss08/trenova/fixtures"
	"github.com/emoss08/trenova/internal/api/router"
	"github.com/emoss08/trenova/internal/server"
	"github.com/emoss08/trenova/migrate/migrations"
	"github.com/emoss08/trenova/pkg/models"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"

	"github.com/rs/zerolog/log"

	"github.com/uptrace/bun/extra/bundebug"
	"github.com/uptrace/bun/migrate"
	"github.com/urfave/cli/v2"

	"github.com/uptrace/bun"
)

func main() {
	serverConfig := config.DefaultServiceConfigFromEnv()

	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(serverConfig.DB.DSN())))
	db := bun.NewDB(sqldb, pgdialect.New())
	db.AddQueryHook(bundebug.NewQueryHook(
		bundebug.WithVerbose(true),
	))

	// Register many to many model so bun can better recognize m2m relation.
	// This should be done before you use the model for the first time.
	db.RegisterModel(
		(*models.RolePermission)(nil),
		(*models.UserRole)(nil),
		(*models.GeneralLedgerAccountTag)(nil),
	)

	app := &cli.App{
		Name: "Trenova",

		Commands: []*cli.Command{
			newDBCommand(migrate.NewMigrator(db, migrations.Migrations)),
			newServerCommand(),
			newSeederCommand(),
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatal().Err(err).Msg("Failed to run app")
	}
}

func newDBCommand(migrator *migrate.Migrator) *cli.Command {
	return &cli.Command{
		Name:  "db",
		Usage: "database migrations",
		Subcommands: []*cli.Command{
			{
				Name:  "init",
				Usage: "create migration tables",
				Action: func(c *cli.Context) error {
					return migrator.Init(c.Context)
				},
			},
			{
				Name:  "migrate",
				Usage: "migrate database",
				Action: func(c *cli.Context) error {
					if err := migrator.Lock(c.Context); err != nil {
						return err
					}
					defer migrator.Unlock(c.Context) //nolint:errcheck

					group, err := migrator.Migrate(c.Context)
					if err != nil {
						return err
					}
					if group.IsZero() {
						fmt.Printf("there are no new migrations to run (database is up to date)\n")
						return nil
					}
					fmt.Printf("migrated to %s\n", group)
					return nil
				},
			},
			{
				Name:  "rollback",
				Usage: "rollback the last migration group",
				Action: func(c *cli.Context) error {
					if err := migrator.Lock(c.Context); err != nil {
						return err
					}
					defer migrator.Unlock(c.Context) //nolint:errcheck

					group, err := migrator.Rollback(c.Context)
					if err != nil {
						return err
					}
					if group.IsZero() {
						fmt.Printf("there are no groups to roll back\n")
						return nil
					}
					fmt.Printf("rolled back %s\n", group)
					return nil
				},
			},
			{
				Name:  "lock",
				Usage: "lock migrations",
				Action: func(c *cli.Context) error {
					return migrator.Lock(c.Context)
				},
			},
			{
				Name:  "unlock",
				Usage: "unlock migrations",
				Action: func(c *cli.Context) error {
					return migrator.Unlock(c.Context)
				},
			},
			{
				Name:  "create_go",
				Usage: "create Go migration",
				Action: func(c *cli.Context) error {
					name := strings.Join(c.Args().Slice(), "_")
					mf, err := migrator.CreateGoMigration(c.Context, name)
					if err != nil {
						return err
					}
					fmt.Printf("created migration %s (%s)\n", mf.Name, mf.Path)
					return nil
				},
			},
			{
				Name:  "create_sql",
				Usage: "create up and down SQL migrations",
				Action: func(c *cli.Context) error {
					name := strings.Join(c.Args().Slice(), "_")
					files, err := migrator.CreateSQLMigrations(c.Context, name)
					if err != nil {
						return err
					}

					for _, mf := range files {
						fmt.Printf("created migration %s (%s)\n", mf.Name, mf.Path)
					}

					return nil
				},
			},
			{
				Name:  "create_tx_sql",
				Usage: "create up and down transactional SQL migrations",
				Action: func(c *cli.Context) error {
					name := strings.Join(c.Args().Slice(), "_")
					files, err := migrator.CreateTxSQLMigrations(c.Context, name)
					if err != nil {
						return err
					}

					for _, mf := range files {
						fmt.Printf("created transaction migration %s (%s)\n", mf.Name, mf.Path)
					}

					return nil
				},
			},
			{
				Name:  "status",
				Usage: "print migrations status",
				Action: func(c *cli.Context) error {
					ms, err := migrator.MigrationsWithStatus(c.Context)
					if err != nil {
						return err
					}
					fmt.Printf("migrations: %s\n", ms)
					fmt.Printf("unapplied migrations: %s\n", ms.Unapplied())
					fmt.Printf("last migration group: %s\n", ms.LastGroup())
					return nil
				},
			},
			{
				Name:  "reset",
				Usage: "reset the database by rolling back all migrations and then running them again",
				Action: func(c *cli.Context) error {
					if err := migrator.Lock(c.Context); err != nil {
						return err
					}
					defer migrator.Unlock(c.Context) //nolint:errcheck

					if err := migrator.Reset(c.Context); err != nil {
						return err
					}

					if _, err := migrator.Migrate(c.Context); err != nil {
						return err
					}
					return nil
				},
			},
			{
				Name:  "mark_applied",
				Usage: "mark migrations as applied without actually running them",
				Action: func(c *cli.Context) error {
					group, err := migrator.Migrate(c.Context, migrate.WithNopMigration())
					if err != nil {
						return err
					}
					if group.IsZero() {
						fmt.Printf("there are no new migrations to mark as applied\n")
						return nil
					}
					fmt.Printf("marked as applied %s\n", group)
					return nil
				},
			},
		},
	}
}

func newSeederCommand() *cli.Command {
	return &cli.Command{
		Name:  "seeder",
		Usage: "load the fixtures",
		Action: func(c *cli.Context) error {
			return fixtures.LoadFixtures()
		},
	}
}

func newServerCommand() *cli.Command {
	return &cli.Command{
		Name:  "server",
		Usage: "start the server",
		Action: func(c *cli.Context) error {
			runServer()
			return nil
		},
	}
}

func runServer() {
	ctx := context.Background()
	serverConfig := config.DefaultServiceConfigFromEnv()

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
