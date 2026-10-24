package dbcli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/keelcli"
	"github.com/emoss08/keel/migrate"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/database/bunadopt"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/pgxdsn"
	"github.com/emoss08/trenova/schema/migrations"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	StampPath      = "schema/version_gen.go"
	configPath     = "config"
	adoptTimeout   = 2 * time.Minute
	revisionLength = 12

	sqlStateInvalidCatalog = "3D000"
)

var extensions = []string{
	"pg_trgm",
	"unaccent",
	"postgis",
	"btree_gist",
	"pg_stat_statements",
	"pgstattuple",
	"pg_prewarm",
}

type Options struct {
	Name          string
	Description   string
	Editions      []migrate.Source
	ConfigOptions []config.LoaderOption
	Development   bool
	Dump          []string
	Commands      []keelcli.Command
}

func App(o Options) keelcli.App {
	set := migrations.Set(o.Editions...)
	opts := migrations.Options()

	app := keelcli.App{
		Name:        o.Name,
		Description: o.Description,
		Version:     buildVersion(),
		Config: DatabaseConfig(ConfigParams{
			Name:          o.Name,
			ConfigOptions: o.ConfigOptions,
			Migrations:    set,
			Options:       opts,
			Log:           os.Stderr,
		}),
		Migrations: set,
		Migrate:    opts,
		Stamp:      StampPath,
		Lint:       migrations.LintRules(),
		Extensions: extensions,
		Commands:   append([]keelcli.Command{EnableVectorCommand}, o.Commands...),
	}
	if o.Development {
		app.Dirs = migrations.Dirs()
		app.Dump = o.Dump
	}

	return app
}

type ConfigParams struct {
	Name          string
	ConfigOptions []config.LoaderOption
	Migrations    migrate.Set
	Options       migrate.Options
	Log           io.Writer
}

func DatabaseConfig(p ConfigParams) func() (keel.Config, error) {
	return func() (keel.Config, error) {
		cfg, err := config.NewLoader(append(
			[]config.LoaderOption{config.WithConfigPath(configPath)},
			p.ConfigOptions...,
		)...).Load()
		if err != nil {
			return keel.Config{}, fmt.Errorf(
				"%w: load the Trenova configuration: %w",
				keelcli.ErrNoDatabase,
				err,
			)
		}

		dsn, err := pgxdsn.Normalize(
			cfg.GetMigrationDSN(),
			pgxdsn.Options{ApplicationName: p.Name},
		)
		if err != nil {
			return keel.Config{}, err
		}

		ctx, cancel := context.WithTimeout(context.Background(), adoptTimeout)
		defer cancel()
		if err = adoptBunDatabase(ctx, dsn, p); err != nil {
			return keel.Config{}, err
		}

		return keel.Config{DSN: dsn}, nil
	}
}

func adoptBunDatabase(ctx context.Context, dsn string, p ConfigParams) error {
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == sqlStateInvalidCatalog {
			return nil
		}
		return fmt.Errorf("connect to adopt a Bun-migrated database: %w", err)
	}
	defer func() { _ = conn.Close(context.WithoutCancel(ctx)) }()

	result, err := bunadopt.Adopt(ctx, conn, p.Migrations, p.Options)
	if err != nil {
		return err
	}
	if result.Adopted() && p.Log != nil {
		_, _ = fmt.Fprintf(
			p.Log,
			"%s: adopted the database Bun migrated: recorded %d migrations up to %s as applied\n",
			p.Name,
			result.Recorded,
			result.Version,
		)
	}

	return nil
}

func buildVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "devel"
	}

	for _, setting := range info.Settings {
		if setting.Key == "vcs.revision" && setting.Value != "" {
			return setting.Value[:min(len(setting.Value), revisionLength)]
		}
	}

	return "devel"
}
