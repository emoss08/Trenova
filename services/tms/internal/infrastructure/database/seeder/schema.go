package seeder

import (
	"embed"
	"fmt"
	"strings"
)

// The seed bookkeeping tables are created by the seeder rather than by a
// migration, so their DDL lives here rather than in the migration corpus. It is
// kept in .sql files so the SQL stays reviewable and diffable instead of being
// buried in Go string literals.
//
//go:embed schema/*.sql
var schemaFS embed.FS

const schemaSplitMarker = "--bun:split"

const (
	seedHistorySchema   = "seed_history"
	seedRollbacksSchema = "seed_rollbacks"
)

// loadSchema returns the DDL statements for a bookkeeping table. Statements are
// returned individually so a failure can name what broke.
func loadSchema(name string) ([]string, error) {
	path := "schema/" + name + ".sql"

	contents, err := schemaFS.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded schema %q: %w", path, err)
	}

	rawStatements := strings.Split(string(contents), schemaSplitMarker)
	statements := make([]string, 0, len(rawStatements))

	for _, statement := range rawStatements {
		trimmed := strings.TrimSpace(statement)
		if trimmed == "" {
			continue
		}

		statements = append(statements, trimmed)
	}

	if len(statements) == 0 {
		return nil, fmt.Errorf("embedded schema %q contains no statements", path)
	}

	return statements, nil
}
