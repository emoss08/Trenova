package reportquerytools

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytooltest"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A built-in report that reads an add-on's tables is still listed, marked
// unavailable with the add-on it needs, so the model answers "that report is
// part of Accounting & Cash" rather than running it or calling it missing.
func TestListReports_MarksABuiltInReportOffThePlanWithItsAddOn(t *testing.T) {
	t.Parallel()

	service, _, tools := reportingTools(t)
	service.OffPlan = map[string]permissionports.PlanDenial{
		"ar_aging_by_customer": permissionports.NewPlanDenial(platformcatalog.FeatureAccounting),
	}

	result, err := tools["report_list"].Query(
		t.Context(),
		querytooltest.TestParams(map[string]any{}),
	)
	require.NoError(t, err)

	outcome, ok := result.(querytool.SearchOutcome)
	require.True(t, ok)
	rows, ok := outcome.Items.([]reportCatalogRow)
	require.True(t, ok)
	require.Len(t, rows, 1)
	assert.False(t, rows[0].Available)
	assert.Equal(t, "Accounting & Cash", rows[0].Requires)
	assert.Contains(t, rows[0].Note,
		"is part of the Accounting & Cash add-on, which is not on your plan")
}

func TestListReports_WithEveryAddOnOnMarksEveryReportAvailable(t *testing.T) {
	t.Parallel()

	_, _, tools := reportingTools(t)

	result, err := tools["report_list"].Query(
		t.Context(),
		querytooltest.TestParams(map[string]any{}),
	)
	require.NoError(t, err)

	rows, ok := result.(querytool.SearchOutcome).Items.([]reportCatalogRow)
	require.True(t, ok)
	for _, row := range rows {
		assert.True(t, row.Available, row.Name)
		assert.Empty(t, row.Requires, row.Name)
	}
}
