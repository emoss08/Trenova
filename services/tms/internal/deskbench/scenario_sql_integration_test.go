//go:build integration

package deskbench

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const explainPlaceholder = "'deskbench_explain'"

type plannedStatement struct {
	scenario string
	where    string
	sql      string
	args     []any
}

func TestScenarioStatements_PlanAgainstTheMigratedSchema(t *testing.T) {
	dir, err := filepath.Abs(filepath.Join("..", "..", "deskbench", "scenarios"))
	require.NoError(t, err)
	scenarios, err := LoadScenarios([]string{dir})
	require.NoError(t, err)

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	statements := benchStatements(scenarios)
	require.NotEmpty(t, statements)
	for _, statement := range statements {
		_, explainErr := keel.Exec(ctx, db, "EXPLAIN "+statement.sql, statement.args...)
		assert.NoError(t, explainErr, "%s, %s does not plan against the migrated schema:\n%s",
			statement.scenario, statement.where, strings.TrimSpace(statement.sql))
	}
}

func benchStatements(scenarios []*Scenario) []plannedStatement {
	statements := make([]plannedStatement, 0, len(scenarios)*4)
	for idx, check := range dataQueries() {
		statements = append(statements, plannedStatement{
			scenario: "the bench's data profile",
			where:    fmt.Sprintf("check %d", idx+1),
			sql:      check.sql,
			args:     check.args,
		})
	}
	for _, scenario := range scenarios {
		statements = append(statements, scenarioStatements(scenario)...)
	}

	return statements
}

func scenarioStatements(scenario *Scenario) []plannedStatement {
	statements := make([]plannedStatement, 0, 4)
	add := func(where, sql string, args ...any) {
		sql = strings.ReplaceAll(sql, placeholderRun, explainPlaceholder)
		sql = strings.ReplaceAll(sql, placeholderSubject, explainPlaceholder)
		statements = append(statements, plannedStatement{
			scenario: scenario.Name,
			where:    where,
			sql:      sql,
			args:     args,
		})
	}

	for idx := range scenario.Requires {
		sql, args := scenario.Requires[idx].probe()
		add(fmt.Sprintf("requires %d", idx+1), sql, args...)
	}
	for idx, sql := range scenario.Setup {
		add(fmt.Sprintf("setup %d", idx+1), sql)
	}
	for stepIdx := range scenario.Steps {
		step := &scenario.Steps[stepIdx]
		at := fmt.Sprintf("step %d", stepIdx+1)
		for idx, sql := range step.Setup {
			add(fmt.Sprintf("%s setup %d", at, idx+1), sql)
		}
		for idx, sql := range step.mentionFinds() {
			add(fmt.Sprintf("%s mention %d find", at, idx+1), sql)
		}
		if step.Fire != nil && strings.TrimSpace(step.Fire.Subject) != "" {
			add(at+" fire subject", step.Fire.Subject)
		}
		for idx := range step.Expect.Facts {
			add(fmt.Sprintf("%s fact %d", at, idx+1), step.Expect.Facts[idx].SQL)
		}
		for idx := range step.Expect.DB {
			add(fmt.Sprintf("%s db check %d", at, idx+1), step.Expect.DB[idx].SQL)
		}
	}

	return statements
}
