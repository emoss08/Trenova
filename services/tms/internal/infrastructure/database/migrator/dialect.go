package migrator

import (
	"errors"
	"fmt"
	"io/fs"

	"github.com/emoss08/trenova/internal/bootstrap/edition"
	pgmigrations "github.com/emoss08/trenova/internal/infrastructure/postgres/migrations"
	sqlitemigrations "github.com/emoss08/trenova/internal/infrastructure/sqlite/migrations"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect"
	"github.com/uptrace/bun/migrate"
)

var ErrDuplicateMigration = errors.New("migration is defined more than once")

func migrationsFor(db *bun.DB) (*migrate.Migrations, error) {
	if db != nil && db.Dialect().Name() == dialect.SQLite {
		return sqlitemigrations.Migrations()
	}

	return PostgresMigrations(edition.Current().PostgresMigrations...)
}

func PostgresMigrations(extra ...fs.FS) (*migrate.Migrations, error) {
	migrations, err := pgmigrations.Migrations()
	if err != nil {
		return nil, err
	}

	known := make(map[string]struct{})
	for _, migration := range migrations.Sorted() {
		known[migration.Name] = struct{}{}
	}

	for idx, fsys := range extra {
		set := migrate.NewMigrations()
		if err = set.Discover(fsys); err != nil {
			return nil, fmt.Errorf("discover edition migration set %d: %w", idx, err)
		}

		for _, migration := range set.Sorted() {
			if _, duplicate := known[migration.Name]; duplicate {
				return nil, fmt.Errorf("%w: %s", ErrDuplicateMigration, migration.Name)
			}
			known[migration.Name] = struct{}{}
			migrations.Add(migration)
		}
	}

	return migrations, nil
}
