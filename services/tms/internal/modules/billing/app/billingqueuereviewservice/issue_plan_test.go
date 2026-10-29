package billingqueuereviewservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanResolveIssueShowsTheChargeADropRemovesAndWritesNothing(t *testing.T) {
	t.Parallel()

	w := newReviewWorld(t, billingqueue.StatusInReview)
	issueID := checkOf(w.read(t), billingqueue.CheckCharges).IssueID
	events := len(w.review.eventTexts())

	plan, err := w.reviews().
		PlanResolveIssue(t.Context(), &billingqueueports.ResolveBillingQueueIssueRequest{
			ItemID: w.item.ID, IssueID: issueID, OptionKey: "drop", TenantInfo: w.tenant,
		})
	require.NoError(t, err)

	require.NotNil(t, plan.Option)
	assert.Equal(t, "drop", plan.Option.Key)
	require.NotNil(t, plan.ChargeBefore)
	assert.Equal(t, w.lumperID, plan.ChargeBefore.ID)
	assert.Nil(t, plan.ChargeAfter, "a removed charge has no after")
	assert.False(t, plan.AsksDriver)

	require.Len(t, w.shp.AdditionalCharges, 1, "planning leaves the shipment alone")
	assert.True(t, w.review.issues[0].IsOpen(), "planning leaves the check open")
	assert.Len(t, w.review.eventTexts(), events, "planning records nothing")
}

func TestPlanResolveIssueRefusesWhatResolveRefuses(t *testing.T) {
	t.Parallel()

	w := newReviewWorld(t, billingqueue.StatusInReview)
	issueID := checkOf(w.read(t), billingqueue.CheckCharges).IssueID

	_, err := w.reviews().
		PlanResolveIssue(t.Context(), &billingqueueports.ResolveBillingQueueIssueRequest{
			ItemID: w.item.ID, IssueID: issueID, OptionKey: "ratecon", TenantInfo: w.tenant,
		})
	require.Error(t, err, "the lumper check offers no rate con choice")

	_, err = w.reviews().
		ResolveIssue(t.Context(), &billingqueueports.ResolveBillingQueueIssueRequest{
			ItemID: w.item.ID, IssueID: issueID, OptionKey: "keep", TenantInfo: w.tenant,
		}, w.actor)
	require.NoError(t, err)

	_, err = w.reviews().
		PlanResolveIssue(t.Context(), &billingqueueports.ResolveBillingQueueIssueRequest{
			ItemID: w.item.ID, IssueID: issueID, OptionKey: "drop", TenantInfo: w.tenant,
		})
	require.Error(t, err, "a settled check cannot be settled again")
}

func TestPlanUndoIssueShowsTheRemovedChargeComingBack(t *testing.T) {
	t.Parallel()

	w := newReviewWorld(t, billingqueue.StatusInReview)
	issueID := checkOf(w.read(t), billingqueue.CheckCharges).IssueID
	_, err := w.reviews().
		ResolveIssue(t.Context(), &billingqueueports.ResolveBillingQueueIssueRequest{
			ItemID: w.item.ID, IssueID: issueID, OptionKey: "drop", TenantInfo: w.tenant,
		}, w.actor)
	require.NoError(t, err)
	require.Empty(t, w.shp.AdditionalCharges)

	plan, err := w.reviews().
		PlanUndoIssue(t.Context(), &billingqueueports.UndoBillingQueueIssueRequest{
			ItemID: w.item.ID, IssueID: issueID, TenantInfo: w.tenant,
		})
	require.NoError(t, err)

	assert.Nil(t, plan.ChargeBefore, "the charge is not on the shipment now")
	require.NotNil(t, plan.ChargeAfter)
	assert.True(t, decimal.RequireFromString("110").Equal(plan.ChargeAfter.Amount))
	assert.Empty(t, w.shp.AdditionalCharges, "planning puts nothing back")
	assert.True(t, w.review.issues[0].Settled(), "planning leaves the settlement in place")
}

func TestPlanUndoIssueRefusesACheckNobodySettled(t *testing.T) {
	t.Parallel()

	w := newReviewWorld(t, billingqueue.StatusInReview)
	issueID := checkOf(w.read(t), billingqueue.CheckCharges).IssueID

	_, err := w.reviews().
		PlanUndoIssue(t.Context(), &billingqueueports.UndoBillingQueueIssueRequest{
			ItemID: w.item.ID, IssueID: issueID, TenantInfo: w.tenant,
		})
	require.Error(t, err)
}

func TestPlanChargeEffectRemembersTheChargeAsItWasBeforeRepricing(t *testing.T) {
	t.Parallel()

	chargeID := pulid.MustNew("ac_")
	item := &billingqueue.BillingQueueItem{Shipment: &shipment.Shipment{
		AdditionalCharges: []*shipment.AdditionalCharge{{
			ID:     chargeID,
			Method: accessorialcharge.MethodPerUnit,
			Amount: decimal.RequireFromString("75"),
			Unit:   3,
		}},
	}}

	edit, err := planChargeEffect(item, billingqueue.IssueEffect{
		Kind:     billingqueue.EffectSet,
		ChargeID: chargeID,
		Amount:   decimal.NewNullDecimal(decimal.RequireFromString("150")),
	})
	require.NoError(t, err)

	require.NotNil(t, edit.before)
	assert.Equal(t, accessorialcharge.MethodPerUnit, edit.before.Method,
		"undo restores the method the charge had")
	assert.True(t, decimal.RequireFromString("75").Equal(edit.before.Amount),
		"undo restores the amount the charge had, not the repriced one")
	assert.Equal(t, int16(3), edit.before.Unit)

	require.NotNil(t, edit.after)
	assert.Equal(t, accessorialcharge.MethodFlat, edit.after.Method)
	assert.True(t, decimal.RequireFromString("150").Equal(edit.after.Amount))
	assert.Equal(t, int16(1), edit.after.Unit)
	require.Len(t, edit.charges, 1)
	assert.Same(t, edit.after, edit.charges[0])

	snapshot := chargeSnapshot(edit.before, item)
	assert.True(t, decimal.RequireFromString("75").Equal(snapshot.Amount))
	assert.Equal(t, string(accessorialcharge.MethodPerUnit), snapshot.Method)
}
