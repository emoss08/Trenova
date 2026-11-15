package billingqueuemapping

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingvocab"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/identity/app/onplan"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBillingQueueItemToModel_MapsTypedBillingQueueItem(t *testing.T) {
	t.Parallel()

	assignedBillerID := pulid.MustNew("usr_")
	canceledByID := pulid.MustNew("usr_")
	sourceInvoiceID := pulid.MustNew("inv_")
	exceptionReason := billingqueue.ExceptionServiceFailure
	reviewStartedAt := int64(1780415883)
	canceledAt := int64(1780415999)

	model, err := BillingQueueItemToModel(allOn(t), &billingqueue.BillingQueueItem{
		ID:                        pulid.MustNew("bqi_"),
		OrganizationID:            pulid.MustNew("org_"),
		BusinessUnitID:            pulid.MustNew("bu_"),
		ShipmentID:                pulid.MustNew("shp_"),
		AssignedBillerID:          &assignedBillerID,
		Number:                    "BQI-1001",
		Status:                    billingqueue.StatusException,
		BillType:                  billingvocab.BillTypeCreditMemo,
		ExceptionReasonCode:       &exceptionReason,
		ReviewNotes:               "needs review",
		ExceptionNotes:            "unresolved service failure",
		ReviewStartedAt:           &reviewStartedAt,
		CanceledByID:              &canceledByID,
		CanceledAt:                &canceledAt,
		CancelReason:              "duplicate",
		IsAdjustmentOrigin:        true,
		SourceInvoiceID:           &sourceInvoiceID,
		RequiresReplacementReview: true,
		RerateVariancePercent:     decimal.RequireFromString("2.500000"),
		AdjustmentContext: billingqueue.AdjustmentContext{
			Rebill: &billingqueue.AdjustmentRebill{
				ReplacementLines: []*invoice.InvoiceLine{{}, {}},
				SubtotalAmount:   decimal.RequireFromString("1250.5"),
				OtherAmount:      decimal.RequireFromString("185"),
				TotalAmount:      decimal.RequireFromString("1435.5"),
				AccountingDate:   1780416000,
			},
		},
		Version:   7,
		CreatedAt: 1780415777,
		UpdatedAt: 1780415888,
	})
	require.NoError(t, err)

	require.NotNil(t, model)
	assert.Equal(t, "BQI-1001", model.Number)
	assert.Equal(t, billingqueue.StatusException, model.Status)
	assert.Equal(t, billingvocab.BillTypeCreditMemo, model.BillType)
	assert.Equal(t, billingqueue.ExceptionServiceFailure, *model.ExceptionReasonCode)
	assert.Equal(t, assignedBillerID.String(), *model.AssignedBillerID)
	assert.Equal(t, canceledByID.String(), *model.CanceledByID)
	assert.Equal(t, sourceInvoiceID.String(), *model.SourceInvoiceID)
	assert.Equal(t, "2.5", model.RerateVariancePercent)
	require.NotNil(t, model.AdjustmentContext.Rebill)
	assert.Equal(t, gqlmodel.BillingQueueAdjustmentRebill{
		ReplacementLineCount: 2,
		SubtotalAmount:       "1250.50",
		OtherAmount:          "185.00",
		TotalAmount:          "1435.50",
		AccountingDate:       1780416000,
	}, *model.AdjustmentContext.Rebill)
	assert.Nil(t, model.AdjustmentContext.Memo)
	assert.Equal(t, 7, model.Version)
	assert.Equal(t, 1780415777, model.CreatedAt)
	assert.Equal(t, 1780415888, model.UpdatedAt)
}

func TestBillingQueueItemToModel_MapsAMemoAndWhoRaisedIt(t *testing.T) {
	t.Parallel()

	raisedBy := pulid.MustNew("usr_")
	model, err := BillingQueueItemToModel(allOn(t), &billingqueue.BillingQueueItem{
		ID:                    pulid.MustNew("bqi_"),
		OrganizationID:        pulid.MustNew("org_"),
		BusinessUnitID:        pulid.MustNew("bu_"),
		Status:                billingqueue.StatusApproved,
		BillType:              billingvocab.BillTypeCreditMemo,
		RerateVariancePercent: decimal.Zero,
		AdjustmentContext: billingqueue.AdjustmentContext{
			Memo: &billingqueue.AdjustmentMemo{Kind: invoice.MemoKindManual, RaisedByID: raisedBy},
		},
	})
	require.NoError(t, err)

	require.NotNil(t, model.AdjustmentContext.Memo)
	assert.Equal(t, invoice.MemoKindManual, model.AdjustmentContext.Memo.Kind)
	assert.Equal(t, raisedBy.String(), *model.AdjustmentContext.Memo.RaisedByID)
	assert.Nil(t, model.AdjustmentContext.Rebill)
}

func TestBillingQueueItemToModel_DefaultsNilAdjustmentContext(t *testing.T) {
	t.Parallel()

	model, err := BillingQueueItemToModel(allOn(t), &billingqueue.BillingQueueItem{
		ID:                    pulid.MustNew("bqi_"),
		OrganizationID:        pulid.MustNew("org_"),
		BusinessUnitID:        pulid.MustNew("bu_"),
		ShipmentID:            pulid.MustNew("shp_"),
		Status:                billingqueue.StatusReadyForReview,
		BillType:              billingvocab.BillTypeInvoice,
		RerateVariancePercent: decimal.Zero,
	})
	require.NoError(t, err)

	require.NotNil(t, model)
	assert.Equal(t, &gqlmodel.BillingQueueAdjustmentContext{}, model.AdjustmentContext)
}

func TestRequiredBillingQueueItemToModel_RejectsNilItem(t *testing.T) {
	t.Parallel()

	model, err := RequiredBillingQueueItemToModel(t.Context(), nil)

	require.Error(t, err)
	assert.Nil(t, model)
	assert.Contains(t, err.Error(), "Billing queue transfer did not return an item")
}

func allOn(t *testing.T) context.Context {
	t.Helper()

	return gqlctx.WithPlanMemo(t.Context(), plantest.AllOn().Memo())
}

func reviewedItem() *billingqueue.BillingQueueItem {
	verdict := billingqueue.AgentVerdictConcern
	reviewedAt := int64(1_790_000_000)
	return &billingqueue.BillingQueueItem{
		ID:              pulid.MustNew("bqi_"),
		Lane:            billingqueue.LaneNeedsYou,
		AgentVerdict:    &verdict,
		AgentSummary:    "Fuel is 18% above this lane's usual",
		AgentReviewedAt: &reviewedAt,
	}
}

func TestBillingQueueItemConnectionToModel_WithholdsTheAgentReviewWithoutAgentAutomation(t *testing.T) {
	t.Parallel()

	gate := &countingGate{FeatureGate: plantest.BaseOnly()}
	ctx := gqlctx.WithPlanMemo(t.Context(), onplanMemo(gate))
	conn, err := BillingQueueItemConnectionToModel(
		ctx,
		&pagination.CursorListResult[*billingqueue.BillingQueueItem]{
			Items:       []*billingqueue.BillingQueueItem{reviewedItem(), reviewedItem()},
			EdgeCursors: []string{"cursor-1", "cursor-2"},
		},
	)

	require.NoError(t, err)
	require.Len(t, conn.Edges, 2)
	for _, edge := range conn.Edges {
		assert.Nil(t, edge.Node.AgentVerdict)
		assert.Nil(t, edge.Node.AgentSummary)
		assert.Nil(t, edge.Node.AgentReviewedAt)
		assert.Equal(t, billingqueue.LaneNeedsYou, edge.Node.Lane)
	}
	assert.Equal(t, 1, gate.asked(platformcatalog.FeatureAgentAutomation),
		"the plan is asked once per request, not once per row")
}

func TestBillingQueueItemToModel_ShowsTheAgentReviewWithAgentAutomation(t *testing.T) {
	t.Parallel()

	model, err := BillingQueueItemToModel(allOn(t), reviewedItem())

	require.NoError(t, err)
	require.NotNil(t, model.AgentVerdict)
	assert.Equal(t, billingqueue.AgentVerdictConcern, *model.AgentVerdict)
	require.NotNil(t, model.AgentSummary)
	assert.Equal(t, "Fuel is 18% above this lane's usual", *model.AgentSummary)
	require.NotNil(t, model.AgentReviewedAt)
	assert.Equal(t, 1_790_000_000, *model.AgentReviewedAt)
}

func TestBillingQueueItemToModel_AGateErrorIsAnError(t *testing.T) {
	t.Parallel()

	ctx := gqlctx.WithPlanMemo(t.Context(), onplanMemo(failingGate{}))
	model, err := BillingQueueItemToModel(ctx, reviewedItem())

	require.Error(t, err)
	assert.Nil(t, model)
}

type countingGate struct {
	platformcatalogports.FeatureGate

	mu   sync.Mutex
	asks map[platformcatalog.FeatureKey]int
}

func (g *countingGate) Enabled(
	ctx context.Context,
	feature platformcatalog.FeatureKey,
) (bool, error) {
	g.mu.Lock()
	if g.asks == nil {
		g.asks = make(map[platformcatalog.FeatureKey]int, 1)
	}
	g.asks[feature]++
	g.mu.Unlock()

	return g.FeatureGate.Enabled(ctx, feature)
}

func (g *countingGate) asked(feature platformcatalog.FeatureKey) int {
	g.mu.Lock()
	defer g.mu.Unlock()

	return g.asks[feature]
}

type failingGate struct{}

func (failingGate) Enabled(context.Context, platformcatalog.FeatureKey) (bool, error) {
	return false, errors.New("the plan could not be read")
}

func onplanMemo(gate platformcatalogports.FeatureGate) *onplan.Memo {
	return onplan.New(permissioncatalog.Default(), gate).Memo()
}
