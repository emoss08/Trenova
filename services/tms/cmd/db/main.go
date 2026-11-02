package main

import (
	"github.com/emoss08/keel/keelcli"
	"github.com/emoss08/trenova/internal/infrastructure/database/dbcli"
)

func main() {
	keelcli.Main(dbcli.App(dbcli.Options{
		Name:        "trenova-db",
		Description: "Applies Trenova's migrations and reference data: the deploy step before the API and worker start.",
	}))
}
