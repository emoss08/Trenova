package migrations

import (
	"io/fs"
	"regexp"
	"testing"

	"github.com/emoss08/keel/migrate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const lintFloor = "20261231019000"

var migrationFileRE = regexp.MustCompile(`^\d{14}_[0-9a-z_]+\.sql$`)

func TestEveryEmbeddedFileIsAMigration(t *testing.T) {
	t.Parallel()

	entries, err := fs.ReadDir(files, ".")
	require.NoError(t, err)

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		require.Truef(t, migrationFileRE.MatchString(entry.Name()),
			"%s is not named <14-digit version>_<name>.sql", entry.Name())
		names = append(names, entry.Name())
	}

	ms := coreMigrations(t)
	require.Len(t, ms, len(names), "keel reads every file as one migration")
	for _, m := range ms {
		assert.Equal(t, SourceName, m.Source)
		assert.Equal(t, migrate.KindVersioned, m.Kind, m.ID())
	}
}

func TestOptionsReconcileRowLevelSecurityAfterEveryRun(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []string{"SELECT trenova_rls.reconcile()"}, Options().After)
}

func TestSetMergesEditionSourcesIntoOneVersionLine(t *testing.T) {
	t.Parallel()

	edition := migrate.Source{Name: "edition", FS: mapFS(map[string]string{
		"29991231000000_edition_table.sql": "CREATE TABLE edition_table (id text);",
	})}

	set := Set(edition)
	require.NoError(t, set.Err())
	assert.Equal(t, "29991231000000", set.Version())

	ms, err := set.Migrations()
	require.NoError(t, err)
	assert.Equal(t, "edition", ms[len(ms)-1].Source)
}

func TestSetRefusesAnEditionVersionTheCoreAlreadyHas(t *testing.T) {
	t.Parallel()

	core := coreMigrations(t)
	clash := migrate.Source{Name: "edition", FS: mapFS(map[string]string{
		core[0].Version + "_clash.sql": "SELECT 1;",
	})}

	require.Error(t, Set(clash).Err())
}

func TestLatestConstraintDefinitionReturnsTheNewestDefinition(t *testing.T) {
	t.Parallel()

	definition, err := latestConstraintDefinition("ck_organization_onboarding_status")
	require.NoError(t, err)
	assert.Contains(t, definition, "ck_organization_onboarding_status")
}

func TestLatestConstraintDefinitionRefusesAnUnknownConstraint(t *testing.T) {
	t.Parallel()

	_, err := latestConstraintDefinition("ck_does_not_exist")
	require.ErrorIs(t, err, errConstraintNotFound)
}

func TestMigrationsSinceTheMoveToKeelPassLint(t *testing.T) {
	t.Parallel()

	var checked []migrate.Migration
	for _, m := range coreMigrations(t) {
		if m.Version > lintFloor {
			checked = append(checked, m)
		}
	}

	for _, finding := range migrate.Lint(checked, LintRules()...) {
		if finding.Severity == migrate.SeverityError {
			t.Errorf("%s:%d %s: %s", finding.Migration.ID(), finding.Line, finding.Rule,
				finding.Message)
		}
	}
}
