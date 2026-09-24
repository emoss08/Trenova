package migrator_test

import (
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun/migrate"
)

const insertTemplateRow = `INSERT INTO "agent_definitions" ` +
	`("id", "business_unit_id", "organization_id", "name", "template") `

func TestSQLiteTemplateCheckAcceptsEveryDeclaredTemplate(t *testing.T) {
	ctx := t.Context()
	db := newSQLiteDB(t)

	migrator := migrate.NewMigrator(db, sqliteMigrations(t))
	require.NoError(t, migrator.Init(ctx))
	_, err := migrator.Migrate(ctx)
	require.NoError(t, err)

	for idx, template := range agentdefinition.AllTemplates() {
		_, err = db.NewRaw(
			insertTemplateRow+`VALUES (?, 'bu_sqlite', 'org_sqlite', ?, ?)`,
			fmt.Sprintf("agdef_sqlite_%d", idx), string(template), string(template),
		).Exec(ctx)
		require.NoErrorf(t, err, "SQLite refuses the %s template", template)
	}

	_, err = db.NewRaw(
		insertTemplateRow +
			`VALUES ('agdef_sqlite_unknown', 'bu_sqlite', 'org_sqlite', 'Unknown', 'Spreadsheet')`,
	).Exec(ctx)
	require.Error(t, err, "the rebuilt table still checks the template")
}
