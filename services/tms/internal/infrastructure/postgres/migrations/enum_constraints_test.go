package migrations

import (
	"io/fs"
	"regexp"
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/stretchr/testify/require"
)

type enumConstraint struct {
	name   string
	values []string
}

func stringsOf[T ~string](values []T) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, string(value))
	}

	return out
}

/*
Every value a Go enum declares is one the column's CHECK constraint accepts.

A template added to the domain and forgotten here is a row the database
refuses. The seed for the intake desk hit exactly that: the check still listed
the original eight templates, so a fresh install could not seed at all, and an
administrator could not create any of the desk agents.

The constraint is read from the newest up migration that adds it, which is the
one a migrated database is left with.
*/
func TestEnumCheckConstraintsAcceptEveryDeclaredValue(t *testing.T) {
	t.Parallel()

	constraints := []enumConstraint{
		{
			name:   "ck_agent_definitions_template",
			values: stringsOf(agentdefinition.AllTemplates()),
		},
		{
			name:   "ck_assistant_artifacts_kind",
			values: stringsOf(assistantartifact.AllKinds()),
		},
	}

	files := embeddedMigrationFiles(t)
	slices.SortFunc(files, func(a, b migrationFile) int {
		if a.version < b.version {
			return -1
		}
		if a.version > b.version {
			return 1
		}
		return 0
	})

	for _, constraint := range constraints {
		adds := regexp.MustCompile(
			`(?s)ADD CONSTRAINT "` + constraint.name + `"(.*?);|CONSTRAINT "` +
				constraint.name + `" CHECK(.*?)\)\s*,?\s*\n`,
		)

		var latest string
		for _, file := range files {
			if file.direction != "up" {
				continue
			}
			body, err := fs.ReadFile(sqlMigrations, file.name)
			require.NoError(t, err)
			if match := adds.FindSubmatch(body); match != nil {
				latest = string(match[0])
			}
		}
		require.NotEmpty(t, latest, "no migration adds %s", constraint.name)

		for _, value := range constraint.values {
			require.Contains(t, latest, "'"+value+"'",
				"%s does not accept %q; add a migration that widens it", constraint.name, value)
		}
	}
}
