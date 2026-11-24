package migrations

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeletingAReportKeepsItsRunsInTheirTenant(t *testing.T) {
	t.Parallel()

	key, err := latestConstraintDefinition("fk_report_runs_definition")
	require.NoError(t, err)
	assert.Contains(t, compactSQL(key), `ON DELETE SET NULL ("definition_id")`,
		"a bare SET NULL nulls the run's business unit and organization with its report")

	source, err := latestConstraintDefinition("check_report_runs_source")
	require.NoError(t, err)
	assert.Contains(t, compactSQL(source), `"revision_id" IS NOT NULL`,
		"a run whose report was deleted is still bound to the revision it ran")
}
