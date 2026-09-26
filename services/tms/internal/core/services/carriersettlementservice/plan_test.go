package carriersettlementservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/carriersettlement"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func carrierPlanFixture(status carriersettlement.Status) *carriersettlement.CarrierSettlement {
	return &carriersettlement.CarrierSettlement{
		ID:               pulid.MustNew("carstl_"),
		SettlementNumber: "CS-100",
		Status:           status,
		CurrencyCode:     "USD",
		Lines: []*carriersettlement.CarrierSettlementLine{{
			ID:          pulid.MustNew("carstll_"),
			EventType:   carriersettlement.CostEventTypeLinehaulCost,
			Description: "Linehaul",
			AmountMinor: 120000,
		}},
	}
}

func TestCarrierPlanTransitions_FollowTheLifecycle(t *testing.T) {
	t.Parallel()

	userID := pulid.MustNew("usr_")
	const now = int64(1_790_000_000)

	entity := carrierPlanFixture(carriersettlement.StatusDraft)
	require.NoError(t, PlanSubmit(entity, userID, now))
	require.Error(t, PlanSubmit(entity, userID, now))
	require.Error(t, PlanReject(entity, ""))
	require.NoError(t, PlanApprove(entity, userID, now))
	require.Error(t, PlanApprove(entity, userID, now))
	require.Error(t, PlanRecalculate(entity))
	require.NoError(t, PlanPost(entity, userID, now))
	require.Error(t, PlanMarkPaid(entity, &MarkPaidInput{PaidAt: now, UserID: userID}))
	require.NoError(t, PlanMarkPaid(entity, &MarkPaidInput{
		PaymentMethod: "ACH",
		PaidAt:        now,
		UserID:        userID,
	}))
	assert.Equal(t, carriersettlement.StatusPaid, entity.Status)
	assert.Equal(t, "ACH", entity.PaymentMethod)

	rejected := carrierPlanFixture(carriersettlement.StatusPendingApproval)
	require.NoError(t, PlanReject(rejected, "Wrong fuel"))
	assert.Equal(t, carriersettlement.StatusDraft, rejected.Status)
	assert.Equal(t, "Rejected: Wrong fuel", rejected.Notes)

	voided := carrierPlanFixture(carriersettlement.StatusApproved)
	require.Error(t, PlanVoid(voided, "", userID, now))
	require.NoError(t, PlanVoid(voided, "Duplicate", userID, now))
	assert.Equal(t, carriersettlement.StatusVoided, voided.Status)
}

func TestCarrierPlanAdjustments_AddAndRemoveAManualLine(t *testing.T) {
	t.Parallel()

	entity := carrierPlanFixture(carriersettlement.StatusDraft)
	clone := CloneSettlement(entity)
	require.NoError(t, PlanAddAdjustment(clone, &AdjustmentLineInput{
		Description: "Lumper",
		AmountMinor: 5000,
	}))
	require.Len(t, clone.Lines, 2)
	assert.Len(t, entity.Lines, 1)
	assert.Equal(t, int64(125000), clone.NetPayableMinor)

	require.Error(t, PlanRemoveAdjustment(clone, clone.Lines[0].ID))
	clone.Lines[1].ID = pulid.MustNew("carstll_")
	require.NoError(t, PlanRemoveAdjustment(clone, clone.Lines[1].ID))
	assert.Equal(t, int64(120000), clone.NetPayableMinor)

	posted := carrierPlanFixture(carriersettlement.StatusPosted)
	require.Error(t, PlanAddAdjustment(posted, &AdjustmentLineInput{
		Description: "Late",
		AmountMinor: 100,
	}))
}

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
