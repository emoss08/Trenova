package carriersettlementcostservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatchPlans_ResolveOnlyWhatTheQueueAllows(t *testing.T) {
	t.Parallel()

	userID := pulid.MustNew("usr_")
	const now = int64(1_790_000_000)

	matched := &carriersettlement.InvoiceMatch{Status: carriersettlement.InvoiceMatchStatusMatched}
	require.NoError(t, PlanAcceptMatch(matched, "Looks right", userID, now))
	assert.Equal(t, carriersettlement.InvoiceMatchStatusResolved, matched.Status)

	variance := &carriersettlement.InvoiceMatch{
		Status:        carriersettlement.InvoiceMatchStatusVariance,
		VarianceMinor: 2500,
	}
	require.Error(t, PlanAcceptMatch(variance, "", userID, now))
	require.NoError(t, PlanAcceptWithVariance(variance, &VarianceResolution{
		Note:         "Detention agreed",
		UserID:       userID,
		ResolvedAt:   now,
		AdjustmentID: pulid.MustNew("cce_"),
	}))
	require.NotNil(t, variance.AdjustmentCostEventID)

	flat := &carriersettlement.InvoiceMatch{Status: carriersettlement.InvoiceMatchStatusVariance}
	require.Error(t, CheckAcceptWithVariance(flat))

	open := &carriersettlement.InvoiceMatch{Status: carriersettlement.InvoiceMatchStatusVariance}
	require.Error(t, PlanRejectMatch(open, "", userID, now))
	require.NoError(t, PlanRejectMatch(open, "Billed the wrong load", userID, now))
	assert.Equal(t, carriersettlement.InvoiceMatchStatusRejected, open.Status)
	require.Error(t, PlanRejectMatch(open, "Again", userID, now))
}

func TestVarianceAdjustmentEvent_AccruesTheVarianceForTheCarrier(t *testing.T) {
	t.Parallel()

	match := &carriersettlement.InvoiceMatch{
		ID:                  pulid.MustNew("cim_"),
		CarrierID:           pulid.MustNew("car_"),
		CarrierAssignmentID: pulid.MustNew("ca_"),
		InvoiceNumber:       "INV-9",
		VarianceMinor:       -1200,
		CurrencyCode:        "USD",
	}
	event := VarianceAdjustmentEvent(pagination.TenantInfo{}, match, 1_790_000_000)

	assert.Equal(t, carriersettlement.CostEventTypeAdjustment, event.EventType)
	assert.Equal(t, int64(-1200), event.AmountMinor)
	assert.Equal(t, "Invoice variance (INV-9)", event.Description)
	assert.Equal(t, "carrier-invoice-variance:"+match.ID.String(), event.IdempotencyKey)
}
