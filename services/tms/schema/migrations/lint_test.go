package migrations

import (
	"testing"
	"testing/fstest"

	"github.com/emoss08/keel/migrate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mapFS(files map[string]string) fstest.MapFS {
	out := make(fstest.MapFS, len(files))
	for name, body := range files {
		out[name] = &fstest.MapFile{Data: []byte(body)}
	}

	return out
}

func lintOne(t *testing.T, body string) []migrate.Finding {
	t.Helper()

	set := migrate.Sources(migrate.Source{
		Name: "test",
		FS:   mapFS(map[string]string{"20270101000000_change.sql": body}),
	})
	ms, err := set.Migrations()
	require.NoError(t, err)

	var out []migrate.Finding
	for _, rule := range LintRules() {
		for _, finding := range rule.Check(ms[0]) {
			finding.Rule = rule.Name
			out = append(out, finding)
		}
	}

	return out
}

func rulesOf(findings []migrate.Finding) []string {
	out := make([]string, 0, len(findings))
	for _, f := range findings {
		out = append(out, f.Rule)
	}

	return out
}


func TestLintWarnsAboutAReconcileCallTheRunMakesAnyway(t *testing.T) {
	t.Parallel()

	findings := lintOne(
		t,
		"ALTER TABLE shipments ADD COLUMN note text;\n\nSELECT trenova_rls.reconcile();\n",
	)

	require.Len(t, findings, 1)
	assert.Equal(t, RuleReconcile, findings[0].Rule)
	assert.Equal(t, migrate.SeverityWarning, findings[0].Severity)
	assert.Equal(t, 3, findings[0].Line)
}

func TestLintIgnoresAReconcileCallInAComment(t *testing.T) {
	t.Parallel()

	assert.Empty(t, lintOne(t, "-- SELECT trenova_rls.reconcile();\nSELECT 1;\n"))
}

func TestLintRequiresNotxIndexStatementsToBeRepeatable(t *testing.T) {
	t.Parallel()

	findings := lintOne(t, "--keel:notx\n"+
		"CREATE INDEX CONCURRENTLY idx_a ON a (b);\n"+
		"--keel:split\n"+
		"CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS idx_c ON a (c);\n"+
		"--keel:split\n"+
		"DROP INDEX CONCURRENTLY idx_d;\n"+
		"--keel:split\n"+
		"DROP INDEX CONCURRENTLY IF EXISTS idx_e;\n")

	assert.Equal(t, []string{RuleNoTxIdempotent, RuleNoTxIdempotent}, rulesOf(findings))
	assert.Equal(t, 2, findings[0].Line)
	assert.Equal(t, 6, findings[1].Line)
}

func TestLintAcceptsAStandardTenantTable(t *testing.T) {
	t.Parallel()

	assert.Empty(t, lintOne(t, `CREATE TABLE IF NOT EXISTS "holds" (
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "reason" text NOT NULL,
    CONSTRAINT "pk_holds" PRIMARY KEY ("id", "organization_id", "business_unit_id")
);
`))
}

func TestLintWarnsAboutATableReconcileCannotProtect(t *testing.T) {
	t.Parallel()

	findings := lintOne(t, `CREATE TABLE "partner_catalog" (
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100),
    "code" text NOT NULL
);
`)

	require.Len(t, findings, 1)
	assert.Equal(t, RuleTenantRLS, findings[0].Rule)
	assert.Contains(t, findings[0].Message, "partner_catalog")
}

func TestLintAcceptsATableThatIsGlobalOrHasItsOwnPolicy(t *testing.T) {
	t.Parallel()

	assert.Empty(t, lintOne(t, `CREATE TABLE "partner_catalog" ("id" text NOT NULL);

INSERT INTO trenova_rls.global_tables ("table_name", "reason")
VALUES ('partner_catalog', 'Catalog shared by every organization');
`))
	assert.Empty(
		t,
		lintOne(t, `CREATE TABLE "partner_links" ("id" text NOT NULL, "organization_id" text);

SELECT trenova_rls.apply_policy('public.partner_links', 'organization_id IS NULL', 'false');
`),
	)
}

func TestLintSkipsTablesOutsideThePublicSchema(t *testing.T) {
	t.Parallel()

	assert.Empty(t, lintOne(t, `CREATE TABLE trenova_rls.scope_keys ("key_id" text NOT NULL);`))
}
