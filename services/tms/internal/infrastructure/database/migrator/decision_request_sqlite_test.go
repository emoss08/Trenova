package migrator_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun/migrate"
)

const decisionRequestMigration = "20261231006950"

const insertDecisionArtifact = `INSERT INTO "assistant_artifacts" ` +
	`("id", "business_unit_id", "organization_id", "thread_id", "kind", "title") `

func TestSQLiteAssistantArtifactsAcceptADecisionRequest(t *testing.T) {
	ctx := t.Context()
	db := newSQLiteDB(t)

	migrator := migrate.NewMigrator(db, sqliteMigrations(t))
	require.NoError(t, migrator.Init(ctx))
	_, err := migrator.Migrate(ctx)
	require.NoError(t, err)

	_, err = db.NewRaw(insertPageThread +
		`VALUES ('athr_1', 'bu_1', 'org_1', 'usr_1', 'agdef_1', 'Active', 'Desk', NULL, NULL)`,
	).Exec(ctx)
	require.NoError(t, err)

	_, err = db.NewRaw(insertDecisionArtifact +
		`VALUES ('aart_1', 'bu_1', 'org_1', 'athr_1', 'decision_request', 'Approve the hold')`,
	).Exec(ctx)
	require.NoError(t, err, "a decision request is an artifact kind the table accepts")

	_, err = db.NewRaw(insertDecisionArtifact +
		`VALUES ('aart_2', 'bu_1', 'org_1', 'athr_1', 'draft_edit', 'Set the customer')`,
	).Exec(ctx)
	require.NoError(t, err, "the kinds accepted before are still accepted")

	_, err = db.NewRaw(insertDecisionArtifact +
		`VALUES ('aart_3', 'bu_1', 'org_1', 'athr_1', 'spreadsheet', 'Nope')`,
	).Exec(ctx)
	require.Error(t, err, "the rebuilt table still checks the kind")

	var kept int
	require.NoError(t, db.NewRaw(
		`SELECT count(*) FROM "assistant_artifacts" WHERE "thread_id" = 'athr_1'`,
	).Scan(ctx, &kept))
	require.Equal(t, 2, kept)
}

func sqliteMigrationsThrough(t *testing.T, last string, inclusive bool) *migrate.Migrations {
	t.Helper()

	out := migrate.NewMigrations()
	for _, migration := range sqliteMigrations(t).Sorted() {
		if migration.Name > last || (!inclusive && migration.Name == last) {
			continue
		}
		out.Add(migration)
	}

	return out
}

func TestSQLiteDecisionRequestRollbackNarrowsTheKindCheck(t *testing.T) {
	ctx := t.Context()
	db := newSQLiteDB(t)

	before := migrate.NewMigrator(db, sqliteMigrationsThrough(t, decisionRequestMigration, false))
	require.NoError(t, before.Init(ctx))
	_, err := before.Migrate(ctx)
	require.NoError(t, err)

	widened := migrate.NewMigrator(db, sqliteMigrationsThrough(t, decisionRequestMigration, true))
	group, err := widened.Migrate(ctx)
	require.NoError(t, err)
	require.Len(t, group.Migrations, 1)

	_, err = db.NewRaw(insertPageThread +
		`VALUES ('athr_1', 'bu_1', 'org_1', 'usr_1', 'agdef_1', 'Active', 'Desk', NULL, NULL)`,
	).Exec(ctx)
	require.NoError(t, err)
	_, err = db.NewRaw(insertDecisionArtifact +
		`VALUES ('aart_1', 'bu_1', 'org_1', 'athr_1', 'decision_request', 'Approve the hold'), ` +
		`('aart_2', 'bu_1', 'org_1', 'athr_1', 'draft_edit', 'Set the customer')`,
	).Exec(ctx)
	require.NoError(t, err)

	rolledBack, err := widened.Rollback(ctx)
	require.NoError(t, err)
	require.Len(t, rolledBack.Migrations, 1)

	var kinds []string
	require.NoError(t, db.NewRaw(
		`SELECT "kind" FROM "assistant_artifacts" ORDER BY "id"`,
	).Scan(ctx, &kinds))
	require.Equal(t, []string{"draft_edit"}, kinds, "the rollback keeps every other artifact")

	_, err = db.NewRaw(insertDecisionArtifact +
		`VALUES ('aart_3', 'bu_1', 'org_1', 'athr_1', 'decision_request', 'Approve again')`,
	).Exec(ctx)
	require.Error(t, err, "the rolled-back table refuses a decision request again")
}
