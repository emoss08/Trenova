package dbcli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"sync"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/keelcli"
	"github.com/emoss08/keel/migrate"
	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/database/bunadopt"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbpool"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/pgxdsn"
	"github.com/emoss08/trenova/internal/seeds/reference"
	"github.com/emoss08/trenova/internal/seeds/seedenv"
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
	Fixtures      *seed.Registry
	Commands      []keelcli.Command
}

func App(o Options) keelcli.App {
	set := migrations.Set(o.Editions...)
	opts := migrations.Options()
	settings := NewSettings(o.Name, o.ConfigOptions)
	env := seedenv.Env{Config: settings.MustConfig}

	app := keelcli.App{
		Name:        o.Name,
		Description: o.Description,
		Version:     buildVersion(),
		Config: DatabaseConfig(ConfigParams{
			Name:       o.Name,
			Settings:   settings,
			Migrations: set,
			Options:    opts,
			Log:        os.Stderr,
		}),
		Migrations: set,
		Migrate:    opts,
		Stamp:      StampPath,
		Lint:       migrations.LintRules(),
		Seeds:      Seeds(settings, o.Fixtures),
		Tenants:    env.Tenants,
		System:     env.System,
		Extensions: extensions,
		Commands:   append([]keelcli.Command{EnableVectorCommand}, o.Commands...),
	}
	if o.Development {
		app.Dirs = migrations.Dirs()
		app.Dump = o.Dump
	}

	return app
}

func Seeds(settings *Settings, fixtures *seed.Registry) *seed.Registry {
	registries := []*seed.Registry{reference.Seeds}
	if fixtures != nil {
		registries = append(registries, fixtures)
	}

	seeds := seed.Merge(registries...)
	seeds.Add(reference.ScopeKey(settings.RLS))

	return seeds
}

type Settings struct {
	name string
	load func() (*config.Config, error)
}

func NewSettings(name string, options []config.LoaderOption) *Settings {
	return &Settings{
		name: name,
		load: sync.OnceValues(func() (*config.Config, error) {
			return config.NewLoader(append(
				[]config.LoaderOption{config.WithConfigPath(configPath)},
				options...,
			)...).Load()
		}),
	}
}

func (s *Settings) Config() (*config.Config, error) {
	cfg, err := s.load()
	if err != nil {
		return nil, fmt.Errorf("%w: load the Trenova configuration: %w", keelcli.ErrNoDatabase, err)
	}

	return cfg, nil
}

func (s *Settings) MustConfig() *config.Config {
	cfg, err := s.load()
	if err != nil {
		return &config.Config{}
	}

	return cfg
}

func (s *Settings) RLS() config.RLSConfig {
	return s.MustConfig().Database.RLS
}

type ConfigParams struct {
	Name       string
	Settings   *Settings
	Migrations migrate.Set
	Options    migrate.Options
	Log        io.Writer
}

func DatabaseConfig(p ConfigParams) func() (keel.Config, error) {
	return func() (keel.Config, error) {
		cfg, err := p.Settings.Config()
		if err != nil {
			return keel.Config{}, err
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

		kc := dbpool.KeelConfig(nil, nil)
		kc.DSN = dsn

		return kc, nil
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
