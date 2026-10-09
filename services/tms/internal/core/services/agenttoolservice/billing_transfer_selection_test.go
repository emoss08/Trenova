package agenttoolservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/billingtransfercriteria"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type transferSelection struct {
	planner  *fakeTransferPlanner
	tool     *transferToBillingTool
	ready    pulid.ID
	marked   pulid.ID
	refused  pulid.ID
	returned pulid.ID
}

func newTransferSelection() *transferSelection {
	ready, marked := pulid.MustNew("shp_"), pulid.MustNew("shp_")
	refused, returned := pulid.MustNew("shp_"), pulid.MustNew("shp_")
	planner := &fakeTransferPlanner{
		preview: true,
		candidates: &serviceports.BillingTransferCandidateIDsResponse{
			IDs:        []pulid.ID{ready, marked, refused, returned},
			TotalCount: 4,
		},
		plan: &serviceports.BillingTransferPlan{
			Transfer: 2,
			Refused:  1,
			Returned: 1,
			Decisions: []serviceports.BillingTransferDecision{
				{ShipmentID: ready, Outcome: serviceports.BillingTransferOutcomeTransfer},
				{
					ShipmentID: marked,
					Status:     shipment.StatusCompleted,
					Outcome:    serviceports.BillingTransferOutcomeMarkReadyAndTransfer,
				},
				{ShipmentID: refused, Outcome: serviceports.BillingTransferOutcomeRefused},
				{
					ShipmentID: returned,
					Outcome:    serviceports.BillingTransferOutcomeReturnToOperations,
				},
			},
		},
	}

	return &transferSelection{
		planner:  planner,
		tool:     newTransferToBillingTool(planner, &fakeRunStarter{}).(*transferToBillingTool),
		ready:    ready,
		marked:   marked,
		refused:  refused,
		returned: returned,
	}
}

func TestTransferToBilling_ResolvesEveryTransferableShipmentToIDs(t *testing.T) {
	t.Parallel()

	sel := newTransferSelection()
	customerID := pulid.MustNew("cus_")

	resolved, err := sel.tool.ResolveSelection(t.Context(), executeParams(map[string]any{
		paramAllTransferable:                       true,
		paramMarkCompletedReady:                    true,
		paramBillType:                              "Invoice",
		billingtransfercriteria.ParamStatus:        "Completed",
		billingtransfercriteria.ParamCustomerID:    customerID.String(),
		billingtransfercriteria.ParamQuery:         "acme",
		billingtransfercriteria.ParamDeliveredFrom: "2026-09-01",
		billingtransfercriteria.ParamDeliveredTo:   "2026-09-15",
	}))
	require.NoError(t, err)

	listed := sel.planner.listed
	require.NotNil(t, listed)
	assert.Equal(t, shipment.StatusCompleted, listed.Status)
	assert.Equal(t, "acme", listed.Filter.Query)
	require.Len(t, listed.Filter.FieldFilters, 3)
	assert.Equal(t, "customerId", listed.Filter.FieldFilters[0].Field)
	assert.Equal(t, dbtype.OpGreaterThanOrEqual, listed.Filter.FieldFilters[1].Operator)
	assert.Equal(t, dbtype.OpLessThanOrEqual, listed.Filter.FieldFilters[2].Operator)
	assert.Equal(t,
		[]pulid.ID{sel.ready, sel.marked, sel.refused, sel.returned},
		sel.planner.planned.ShipmentIDs)
	assert.True(t, sel.planner.planned.MarkCompletedReadyToInvoice)

	assert.Equal(t, map[string]any{
		paramShipmentIDs:        []any{sel.ready.String(), sel.marked.String()},
		paramMarkCompletedReady: true,
		paramBillType:           "Invoice",
	}, resolved, "only what would transfer is proposed, and the criteria are spent")
	require.NoError(t, sel.tool.Validate(t.Context(), executeParams(resolved)))
}

func TestTransferToBilling_LeavesAnExplicitSelectionAsItIs(t *testing.T) {
	t.Parallel()

	sel := newTransferSelection()
	params := map[string]any{paramShipmentIDs: []any{sel.ready.String()}}

	resolved, err := sel.tool.ResolveSelection(t.Context(), executeParams(params))
	require.NoError(t, err)

	assert.Equal(t, params, resolved)
	assert.Nil(t, sel.planner.listed, "nothing is read for ids the model named")
}

func TestTransferToBilling_RefusesACriteriaSelectionThatTransfersNothing(t *testing.T) {
	t.Parallel()

	sel := newTransferSelection()
	sel.planner.plan = &serviceports.BillingTransferPlan{
		Refused: 1,
		Decisions: []serviceports.BillingTransferDecision{
			{ShipmentID: sel.refused, Outcome: serviceports.BillingTransferOutcomeRefused},
		},
	}

	_, err := sel.tool.ResolveSelection(t.Context(), executeParams(map[string]any{
		paramAllTransferable: true,
	}))
	require.ErrorContains(t, err, "list_billing_transfer_candidates")

	sel.planner.candidates = &serviceports.BillingTransferCandidateIDsResponse{}
	sel.planner.planned = nil
	_, err = sel.tool.ResolveSelection(t.Context(), executeParams(map[string]any{
		paramAllTransferable: true,
	}))
	require.Error(t, err)
	assert.Nil(t, sel.planner.planned, "no candidate means nothing to plan")
}

func TestTransferToBilling_RefusesACriteriaSelectionPastWhatOneTransferHolds(t *testing.T) {
	t.Parallel()

	sel := newTransferSelection()
	sel.planner.candidates.TotalCount = serviceports.MaxBillingTransferCandidateIDs + 1
	sel.planner.candidates.Truncated = true

	_, err := sel.tool.ResolveSelection(t.Context(), executeParams(map[string]any{
		paramAllTransferable: true,
	}))
	require.ErrorContains(t, err, "narrow")
}

func TestTransferToBilling_PreviewOfACriteriaSelectionShowsTheShipmentsItResolvesTo(t *testing.T) {
	t.Parallel()

	sel := newTransferSelection()

	preview, err := sel.tool.Preview(t.Context(), executeParams(map[string]any{
		paramAllTransferable:    true,
		paramMarkCompletedReady: true,
	}))
	require.NoError(t, err)

	assert.Len(t, sel.planner.planned.ShipmentIDs, 4, "every candidate is checked once")
	assert.Contains(t, preview.Summary, "2 shipments")
	assert.Contains(t, preview.Summary, "2 would transfer")
	assert.NotContains(t, preview.Summary, "refused")
	require.Len(t, preview.Changes, 2)
	assert.ElementsMatch(t,
		[]pulid.ID{sel.ready, sel.marked},
		[]pulid.ID{preview.Changes[0].EntityID, preview.Changes[1].EntityID})
	assert.False(t, preview.Partial)
}

func TestTransferToBilling_ExecutesOnlyTheApprovedShipments(t *testing.T) {
	t.Parallel()

	sel := newTransferSelection()
	sel.planner.preview = false

	err := sel.tool.Execute(t.Context(), executeParams(map[string]any{
		paramAllTransferable: true,
	}))
	require.ErrorContains(t, err, "approved")
	assert.Nil(t, sel.planner.listed, "a criteria call is never evaluated at execution")
	assert.Nil(t, sel.planner.requested)
}

func TestTransferToBilling_RefusesNeitherOrBothSelections(t *testing.T) {
	t.Parallel()

	tool := newTransferToBillingTool(&fakeTransferPlanner{}, &fakeRunStarter{}).(*transferToBillingTool)

	for name, params := range map[string]map[string]any{
		"neither":        {paramBillType: "Invoice"},
		"all turned off": {paramAllTransferable: false},
		"both":           {paramShipmentIDs: shipmentIDs(1), paramAllTransferable: true},
		"bad status":     {paramAllTransferable: true, billingtransfercriteria.ParamStatus: "InTransit"},
		"bad day":        {paramAllTransferable: true, billingtransfercriteria.ParamDeliveredFrom: "soon"},
		"bad customer":   {paramAllTransferable: true, billingtransfercriteria.ParamCustomerID: "acme"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			require.Error(t, tool.Validate(t.Context(), executeParams(params)))
		})
	}

	for name, params := range map[string]map[string]any{
		"all":              {paramAllTransferable: true},
		"all with filters": {paramAllTransferable: true, billingtransfercriteria.ParamStatus: "Completed"},
		"ids":              {paramShipmentIDs: shipmentIDs(2)},
		"ids with the filters they were found by": {
			paramShipmentIDs:                    shipmentIDs(1),
			billingtransfercriteria.ParamQuery:  "SEED-PAY-001",
			billingtransfercriteria.ParamStatus: "Completed",
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			require.NoError(t, tool.Validate(t.Context(), executeParams(params)))
		})
	}
}

func TestTransferToBilling_OffersTheCandidateFiltersToSelectEverythingThatCanGo(t *testing.T) {
	t.Parallel()

	tool := newTransferToBillingTool(&fakeTransferPlanner{}, &fakeRunStarter{})
	schema := tool.ParamSchema()

	properties := schema[toolschema.KeyProperties].(map[string]any)
	for name := range billingtransfercriteria.Properties() {
		assert.Contains(t, properties, name)
	}
	assert.Contains(t, properties, paramAllTransferable)
	assert.NotContains(t, schema, toolschema.KeyRequired,
		"either selection is enough, and Validate refuses neither")
	assert.Contains(t, tool.Description(), "allTransferable")
}

/*
A model that found its shipment with list_billing_transfer_candidates sent
the search and the status back beside the id, and the transfer refused it
for naming filters that only narrow allTransferable, every time, until the
turn's budget was spent. The filters cannot widen what the ids name, so they
are set aside and the proposal holds only the shipments named.
*/
func TestTransferToBilling_SetsAsideTheFiltersANamedSelectionWasFoundBy(t *testing.T) {
	t.Parallel()

	sel := newTransferSelection()
	named := sel.ready.String()

	resolved, err := sel.tool.ResolveSelection(t.Context(), executeParams(map[string]any{
		paramShipmentIDs:                    []any{named},
		paramMarkCompletedReady:             true,
		billingtransfercriteria.ParamQuery:  "SEED-PAY-001",
		billingtransfercriteria.ParamStatus: "Completed",
	}))
	require.NoError(t, err)

	assert.Equal(t, map[string]any{
		paramShipmentIDs:        []any{named},
		paramMarkCompletedReady: true,
	}, resolved)
	assert.Nil(t, sel.planner.listed, "a named selection is never widened by its filters")
}

func TestTransferToBilling_ARefusalForBothSelectionsShowsBothCalls(t *testing.T) {
	t.Parallel()

	sel := newTransferSelection()

	_, err := sel.tool.ResolveSelection(t.Context(), executeParams(map[string]any{
		paramShipmentIDs:     []any{sel.ready.String()},
		paramAllTransferable: true,
	}))

	require.ErrorIs(t, err, errBothSelections)
	assert.Contains(t, err.Error(), "send shipmentIds without allTransferable")
	assert.Contains(t, err.Error(), "send allTransferable true with the filters")
}
