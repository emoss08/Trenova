package migrations

import (
	"embed"
	"fmt"

	"github.com/uptrace/bun/migrate"
)

//go:embed *.sql
var sqlMigrations embed.FS

// Migrations discovers the embedded SQL migrations into a fresh registry.
func Migrations() (*migrate.Migrations, error) {
	migrations := migrate.NewMigrations()
	if err := migrations.Discover(sqlMigrations); err != nil {
		return nil, fmt.Errorf("discover embedded sqlite migrations: %w", err)
	}

	return migrations, nil
}
