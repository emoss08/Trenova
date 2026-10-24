package main

import (
	"github.com/emoss08/keel/keelcli"
	"github.com/emoss08/trenova/internal/bootstrap/edition"
	"github.com/emoss08/trenova/internal/infrastructure/database/dbcli"
)

func main() {
	e := edition.Current()
	keelcli.Main(dbcli.App(dbcli.Options{
		Name:          "trenova-devdb",
		Description:   "Trenova's development database: migrations, templates and reset.",
		Editions:      e.Migrations,
		ConfigOptions: e.LoaderOptions(),
		Development:   true,
		Dump:          []string{"docker", "exec", "db", "pg_dump", "-U", "postgres", "-d", "{db}"},
	}))
}
