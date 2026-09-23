package migrations

import (
	"io/fs"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	evalCasesUp   = "20261231005710_agent_eval_cases.tx.up.sql"
	evalCasesDown = "20261231005710_agent_eval_cases.tx.down.sql"
	retentionUp   = "20261231005720_agent_eval_case_retention.tx.up.sql"
)

func readMigration(t *testing.T, name string) string {
	t.Helper()

	body, err := fs.ReadFile(sqlMigrations, name)
	require.NoError(t, err)

	return string(body)
}

func createTableColumns(t *testing.T, body, table string) map[string]string {
	t.Helper()

	start := strings.Index(body, `CREATE TABLE IF NOT EXISTS "`+table+`" (`)
	require.GreaterOrEqual(t, start, 0, "no CREATE TABLE for %s", table)
	end := strings.Index(body[start:], "\n);")
	require.Positive(t, end)

	column := regexp.MustCompile(`^\s+"([a-z_]+)" ([a-z]+(?: [a-z]+)?(?:\(\d+\))?(?:\[\])?)`)
	columns := make(map[string]string)
	for line := range strings.SplitSeq(body[start:start+end], "\n") {
		if match := column.FindStringSubmatch(line); match != nil {
			columns[match[1]] = match[2]
		}
	}

	return columns
}

func TestAgentEvalCasesMigration_KeysTimesAndComments(t *testing.T) {
	t.Parallel()

	body := readMigration(t, evalCasesUp)
	columns := createTableColumns(t, body, "agent_eval_cases")

	assert.Contains(t, body,
		`CONSTRAINT "pk_agent_eval_cases" PRIMARY KEY ("id", "business_unit_id", "organization_id")`)
	for _, name := range []string{"created_at", "updated_at", "expires_at", "version"} {
		assert.Equal(t, "bigint", columns[name], "%s is a BIGINT", name)
	}
	assert.Equal(t, "jsonb", columns["history"])
	assert.Equal(t, "jsonb", columns["tool_fixtures"])
	assert.Equal(t, "jsonb", columns["expected"])
	assert.Equal(t, "jsonb", columns["redaction"])
	assert.Equal(t, "jsonb", columns["captured_fingerprint"])
	assert.Equal(t, "text[]", columns["held_tools"])

	tenancy := map[string]struct{}{
		"id":               {},
		"business_unit_id": {},
		"organization_id":  {},
		"version":          {},
		"created_at":       {},
		"updated_at":       {},
	}
	for name := range columns {
		if _, skip := tenancy[name]; skip {
			continue
		}
		assert.Contains(t, body, `COMMENT ON COLUMN "agent_eval_cases"."`+name+`"`,
			"column %s has no comment", name)
	}
	assert.Contains(t, body,
		`CREATE UNIQUE INDEX IF NOT EXISTS "uq_agent_eval_cases_content"`)
}

func TestAgentEvaluationsMigration_ExactlyOneSource(t *testing.T) {
	t.Parallel()

	body := readMigration(t, evalCasesUp)

	assert.Contains(t, body, `ALTER COLUMN "source_run_id" DROP NOT NULL`)
	assert.Contains(t, body,
		`ADD CONSTRAINT "ck_agent_evaluations_source" CHECK `+
			`(num_nonnulls("source_run_id", "eval_case_id") = 1)`)
	assert.Contains(t, body,
		`REFERENCES "agent_eval_cases"("id", "business_unit_id", "organization_id")`)
	added := []string{"eval_case_id", "checks", "judge", "case_score", "fingerprint"}
	for _, column := range added {
		assert.Contains(t, body, `ADD COLUMN IF NOT EXISTS "`+column+`"`)
		assert.Contains(t, body, `COMMENT ON COLUMN "agent_evaluations"."`+column+`"`)
	}
}

func TestAgentEvalCasesMigration_DownUndoesUp(t *testing.T) {
	t.Parallel()

	body := readMigration(t, evalCasesDown)

	assert.Contains(t, body, `DROP TABLE IF EXISTS "agent_eval_cases"`)
	assert.Contains(t, body, `ALTER COLUMN "source_run_id" SET NOT NULL`)
	assert.Less(t,
		strings.Index(body, `DELETE FROM "agent_evaluations"`),
		strings.Index(body, `ALTER COLUMN "source_run_id" SET NOT NULL`),
		"case evaluations go before the column is required again")
}

func TestAgentEvalCaseRetentionMigration_DefaultsToAYear(t *testing.T) {
	t.Parallel()

	body := readMigration(t, retentionUp)

	assert.Contains(t, body,
		`ADD COLUMN IF NOT EXISTS "agent_eval_case_retention_period" integer NOT NULL DEFAULT 365`)
	assert.Contains(t, body,
		`COMMENT ON COLUMN "data_retention"."agent_eval_case_retention_period"`)
}
