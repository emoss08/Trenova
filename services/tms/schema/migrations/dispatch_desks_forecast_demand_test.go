package migrations

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const dispatchDesksForecastUp = "20261231023226_dispatch_desks_forecast_demand"

func TestDispatchDesksForecastMigration_GrantsTheToolOnceToBothDispatchTemplates(t *testing.T) {
	t.Parallel()

	up := compactSQL(readMigration(t, dispatchDesksForecastUp))

	assert.Contains(t, up, `"template" IN ('DispatchAssistant', 'DispatchAssignment')`)
	assert.Contains(t, up, `NOT ('dispatch_forecast_demand' = ANY(`,
		"a desk that already has the tool is left alone, so the migration can run twice")
}
