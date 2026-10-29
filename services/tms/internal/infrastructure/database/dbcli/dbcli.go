package dbcli

import (
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"sync"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/keelcli"
	"github.com/emoss08/keel/migrate"
	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/config/databaseconfig"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbpool"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/pgxdsn"
	"github.com/emoss08/trenova/internal/keeltables"
	"github.com/emoss08/trenova/internal/seeds/reference"
	"github.com/emoss08/trenova/internal/seeds/seedenv"
	"github.com/emoss08/trenova/schema/migrations"
)

const (
	StampPath      = "schema/version_gen.go"
	configPath     = "config"
	revisionLength = 12
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

	dbConfig := DatabaseConfig(ConfigParams{
		Name:       o.Name,
		Settings:   settings,
		Migrations: set,
		Options:    opts,
		Log:        os.Stderr,
	})
	opts.AfterFunc = markMigratedTables(dbConfig)

	app := keelcli.App{
		Name:        o.Name,
		Description: o.Description,
		Version:     buildVersion(),
		Config:      dbConfig,
		Migrations:  set,
		Migrate:     opts,
		Stamp:       StampPath,
		Lint:        migrations.LintRules(),
		Seeds:       Seeds(settings, o.Fixtures),
		Tenants:     env.Tenants,
		System:      env.System,
		Extensions:  extensions,
		Commands:    append([]keelcli.Command{EnableVectorCommand}, o.Commands...),
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
	seeds.Add(scopeKeySeed(settings.RLS))

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

func (s *Settings) RLS() databaseconfig.RLSConfig {
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
			cfg.Database.MigrationDSN(cfg.App.Name),
			pgxdsn.Options{ApplicationName: p.Name},
		)
		if err != nil {
			return keel.Config{}, err
		}

		kc := dbpool.KeelConfig(nil, nil)
		kc.DSN = dsn
		kc.Tables = keeltables.All()
		kc.DataVersions = keel.DataVersionsConfig{Enabled: true}

		return kc, nil
	}
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
