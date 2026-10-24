package migrations

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/emoss08/keel/migrate"
	"github.com/stretchr/testify/require"
)

var (
	errConstraintNotFound = errors.New("no migration adds the constraint")
	readCore              = sync.OnceValues(func() ([]migrate.Migration, error) {
		return Set().Migrations()
	})
)

func coreMigrations(t *testing.T) []migrate.Migration {
	t.Helper()

	ms, err := readCore()
	require.NoError(t, err, "read the embedded migrations")

	return ms
}

func migrationByID(t *testing.T, id string) migrate.Migration {
	t.Helper()

	for _, m := range coreMigrations(t) {
		if m.ID() == id {
			return m
		}
	}
	require.Failf(t, "unknown migration", "no migration %s", id)

	return migrate.Migration{}
}

func readMigration(t *testing.T, id string) string {
	t.Helper()

	return migrationByID(t, id).SQL
}

func migrationStatements(t *testing.T, id string) []string {
	t.Helper()

	statements := migrationByID(t, id).Statements()
	out := make([]string, 0, len(statements))
	for _, statement := range statements {
		out = append(out, compactSQL(statement))
	}

	return out
}

func compactSQL(sql string) string {
	return strings.Join(strings.Fields(sql), " ")
}

func latestConstraintDefinition(name string) (string, error) {
	ms, err := readCore()
	if err != nil {
		return "", err
	}

	quoted := regexp.QuoteMeta(name)
	adds := regexp.MustCompile(
		`(?s)ADD CONSTRAINT "` + quoted + `"(.*?);|CONSTRAINT "` + quoted + `" CHECK(.*?)\)\s*,?\s*\n`,
	)

	latest := ""
	needle := `"` + name + `"`
	for _, m := range ms {
		if !strings.Contains(m.SQL, needle) {
			continue
		}
		if match := adds.FindString(m.SQL); match != "" {
			latest = match
		}
	}
	if latest == "" {
		return "", fmt.Errorf("%w: %s", errConstraintNotFound, name)
	}

	return latest, nil
}
