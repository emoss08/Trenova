package migrator_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun/migrate"
)

const insertPageThread = `INSERT INTO "assistant_threads" ` +
	`("id", "business_unit_id", "organization_id", "user_id", "agent_definition_id", ` +
	`"status", "origin", "subject_type", "subject_id") `

func TestSQLitePageThreadsKeepOneLiveConversationPerPersonAndSubject(t *testing.T) {
	ctx := t.Context()
	db := newSQLiteDB(t)

	migrator := migrate.NewMigrator(db, sqliteMigrations(t))
	require.NoError(t, migrator.Init(ctx))
	_, err := migrator.Migrate(ctx)
	require.NoError(t, err)

	_, err = db.NewRaw(insertPageThread +
		`VALUES ('athr_1', 'bu_1', 'org_1', 'usr_1', 'agdef_1', 'Active', 'Import', 'Document', 'doc_1')`,
	).Exec(ctx)
	require.NoError(t, err)

	_, err = db.NewRaw(insertPageThread +
		`VALUES ('athr_2', 'bu_1', 'org_1', 'usr_1', 'agdef_1', 'Active', 'Import', 'Document', 'doc_1')`,
	).Exec(ctx)
	require.Error(t, err, "a second live conversation about the same document is refused")

	_, err = db.NewRaw(insertPageThread +
		`VALUES ('athr_3', 'bu_1', 'org_1', 'usr_2', 'agdef_1', 'Active', 'Import', 'Document', 'doc_1')`,
	).Exec(ctx)
	require.NoError(t, err, "another person has a conversation of their own")

	_, err = db.NewRaw(`UPDATE "assistant_threads" SET "status" = 'Archived' WHERE "id" = 'athr_1'`).
		Exec(ctx)
	require.NoError(t, err)
	_, err = db.NewRaw(insertPageThread +
		`VALUES ('athr_4', 'bu_1', 'org_1', 'usr_1', 'agdef_1', 'Active', 'Import', 'Document', 'doc_1')`,
	).Exec(ctx)
	require.NoError(t, err, "a closed conversation makes room for a new one")

	_, err = db.NewRaw(insertPageThread +
		`VALUES ('athr_5', 'bu_1', 'org_1', 'usr_1', 'agdef_2', 'Active', 'Formula', NULL, NULL), ` +
		`('athr_6', 'bu_1', 'org_1', 'usr_1', 'agdef_2', 'Active', 'Formula', NULL, NULL)`,
	).Exec(ctx)
	require.NoError(t, err, "formulas not yet saved each have their own conversation")

	_, err = db.NewRaw(`INSERT INTO "assistant_artifacts" ` +
		`("id", "business_unit_id", "organization_id", "thread_id", "kind", "title") ` +
		`VALUES ('aart_1', 'bu_1', 'org_1', 'athr_4', 'draft_edit', 'Set the customer')`,
	).Exec(ctx)
	require.NoError(t, err, "a draft edit is an artifact kind the table accepts")

	_, err = db.NewRaw(`INSERT INTO "assistant_artifacts" ` +
		`("id", "business_unit_id", "organization_id", "thread_id", "kind", "title") ` +
		`VALUES ('aart_2', 'bu_1', 'org_1', 'athr_4', 'spreadsheet', 'Nope')`,
	).Exec(ctx)
	require.Error(t, err, "the rebuilt table still checks the kind")
}
