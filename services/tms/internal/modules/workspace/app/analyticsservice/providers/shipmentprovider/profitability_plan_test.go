package shipmentprovider

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/app/onplan"
	"github.com/emoss08/trenova/internal/modules/identity/app/refview"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadProfitability_OnABaseOnlyPlanTheCostingFiguresAreLockedAndNotComputed(t *testing.T) {
	t.Parallel()

	registry := permissioncatalog.NewRegistry()
	provider := &Provider{
		views: refview.NewPort(
			refview.New(registry, onplan.New(registry, plantest.BaseOnly())),
			&readerEngine{},
		),
		// The costing service is deliberately absent: a locked card must not reach it.
	}
	opts := &kernelports.AnalyticsRequestOptions{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}

	card, err := provider.readProfitability(t.Context(), opts)
	require.NoError(t, err)

	require.NotNil(t, card.PlanDenial)
	assert.Equal(t, "Accounting & Cash", card.PlanDenial.AddOnName)
	assert.Nil(t, card.AvgMarginPct)
	assert.Nil(t, card.UnprofitableCount)
	assert.Zero(t, card.AvgCPM)
	assert.Zero(t, card.TotalMiles)
}
