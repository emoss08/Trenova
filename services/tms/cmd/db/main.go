package main

import (
	"github.com/emoss08/keel/keelcli"
	"github.com/emoss08/trenova/internal/bootstrap/edition"
	"github.com/emoss08/trenova/internal/infrastructure/database/dbcli"
)

func main() {
	e := edition.Current()
	keelcli.Main(dbcli.App(dbcli.Options{
		Name:          "trenova-db",
		Description:   "Applies Trenova's migrations: the deploy step before the API and worker start.",
		Editions:      e.Migrations,
		ConfigOptions: e.LoaderOptions(),
	}))
}
