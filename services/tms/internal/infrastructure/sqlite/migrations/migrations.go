package migrations

import (
	"embed"
	"fmt"

	"github.com/uptrace/bun/migrate"
)

var Migrations = migrate.NewMigrations()

//go:embed *.sql
var sqlMigrations embed.FS

func init() {
	if err := Migrations.Discover(sqlMigrations); err != nil {
		panic(err)
	}
}

func Setup() *migrate.Migrations {
	migrations := migrate.NewMigrations()

	if err := migrations.Discover(sqlMigrations); err != nil {
		panic(fmt.Errorf("failed to discover migrations: %w", err))
	}

	return migrations
}
