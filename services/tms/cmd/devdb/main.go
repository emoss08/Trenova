package main

import (
	"github.com/emoss08/keel/keelcli"
	"github.com/emoss08/trenova/internal/bootstrap/edition"
	"github.com/emoss08/trenova/internal/infrastructure/database/dbcli"
	"github.com/emoss08/trenova/internal/seeds/fixtures"
	"github.com/emoss08/trenova/internal/seeds/volume"
)

func main() {
	e := edition.Current()
	keelcli.Main(dbcli.App(dbcli.Options{
		Name:          "trenova-devdb",
		Description:   "Trenova's development database: migrations, seeds, templates, volume data and reset.",
		Editions:      e.Migrations,
		ConfigOptions: e.LoaderOptions(),
		Development:   true,
		Dump:          []string{"docker", "exec", "db", "pg_dump", "-U", "postgres", "-d", "{db}"},
		Fixtures:      fixtures.Seeds,
		Commands:      []keelcli.Command{volume.Command},
	}))
}
