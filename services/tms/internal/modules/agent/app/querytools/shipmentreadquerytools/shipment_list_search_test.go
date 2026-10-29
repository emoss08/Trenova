package shipmentreadquerytools

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytooltest"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/shipmentquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/graphqlreadports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmenthold"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentcommentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity"
	"github.com/emoss08/trenova/internal/modules/reference/domain/holdreason"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type namedCustomers struct {
	customerports.CustomerRepository

	items []*customer.Customer
}

func (f *namedCustomers) List(
	_ context.Context,
	req *customerports.ListCustomerRequest,
) (*pagination.ListResult[*customer.Customer], error) {
	matched := make([]*customer.Customer, 0, len(f.items))
	for _, item := range f.items {
		if strings.Contains(strings.ToLower(item.Name), strings.ToLower(req.Filter.Query)) {
			matched = append(matched, item)
		}
	}

	return &pagination.ListResult[*customer.Customer]{Items: matched}, nil
}

type namedOrders struct {
	items []shipmentports.OrderReferenceMatch
}

func (f *namedOrders) MatchOrderReferences(
	_ context.Context,
	req *shipmentports.MatchOrderReferencesRequest,
) ([]shipmentports.OrderReferenceMatch, error) {
	matched := make([]shipmentports.OrderReferenceMatch, 0, len(f.items))
	for _, item := range f.items {
		if strings.EqualFold(item.PONumber, req.Query) ||
			strings.EqualFold(item.OrderNumber, req.Query) {
			matched = append(matched, item)
		}
	}

	return matched, nil
}

type fakeActiveHolds struct {
	holds []*shipmenthold.ShipmentHold
	calls int
	asked []pulid.ID
}

func (f *fakeActiveHolds) ActiveHolds(
	_ context.Context,
	req *shipmentports.ActiveHoldsRequest,
) ([]*shipmenthold.ShipmentHold, error) {
	f.calls++
	f.asked = append(f.asked, req.ShipmentIDs...)

	out := make([]*shipmenthold.ShipmentHold, 0, len(f.holds))
	for _, hold := range f.holds {
		if slices.Contains(req.ShipmentIDs, hold.ShipmentID) {
			out = append(out, hold)
		}
	}

	return out, nil
}

/*
"did the acme steel load deliver yet" searched "Acme Steel". "steel" names a
customer (Cascade Steel) as well as the commodity Steel Coils, and that one
ambiguous word turned the whole search into an OR: ten Acme loads came back
that carried no steel. Each word still narrows; a word with several readings
narrows to any of them.
*/
func TestSearchShipments_AWordWithSeveralReadingsStillNarrowsTheOthers(t *testing.T) {
	t.Parallel()

	acme := &customer.Customer{ID: pulid.MustNew("cus_"), Name: "Acme Foods"}
	cascade := &customer.Customer{ID: pulid.MustNew("cus_"), Name: "Cascade Steel"}
	steel := &commodity.Commodity{ID: pulid.MustNew("com_"), Name: "Steel Coils"}
	repo := &recordingShipmentRepo{}
	tool := shipmentquerytools.NewShipmentListTool(repo, shipmentquerytools.PartySources{
		Customers:   &namedCustomers{items: []*customer.Customer{acme, cascade}},
		Commodities: &namedCommodities{items: []*commodity.Commodity{steel}},
	})

	result, err := tool.Query(
		t.Context(),
		querytooltest.TestParams(map[string]any{"query": "acme steel"}),
	)
	require.NoError(t, err)

	assert.Equal(t, []shipmentports.ShipmentPartyGroup{
		{CustomerIDs: []pulid.ID{acme.ID}},
		{CustomerIDs: []pulid.ID{cascade.ID}, CommodityIDs: []pulid.ID{steel.ID}},
	}, partyGroupsOf(repo.requests))

	outcome, ok := result.(querytool.SearchOutcome)
	require.True(t, ok)
	assert.Contains(t, outcome.SearchedFor, "customer Acme Foods")
	assert.Contains(t, outcome.SearchedFor,
		`"steel" as customer Cascade Steel; or carrying Steel Coils`)
}

func TestSearchShipments_TwoCustomersNamedAreEitherOfThem(t *testing.T) {
	t.Parallel()

	acme := &customer.Customer{ID: pulid.MustNew("cus_"), Name: "Acme Foods"}
	sunbelt := &customer.Customer{ID: pulid.MustNew("cus_"), Name: "Sunbelt Produce"}
	repo := &recordingShipmentRepo{}
	tool := shipmentquerytools.NewShipmentListTool(repo, shipmentquerytools.PartySources{
		Customers: &namedCustomers{items: []*customer.Customer{acme, sunbelt}},
	})

	_, err := tool.Query(
		t.Context(),
		querytooltest.TestParams(map[string]any{"query": "acme sunbelt"}),
	)
	require.NoError(t, err)

	assert.Equal(t, []shipmentports.ShipmentPartyGroup{
		{CustomerIDs: []pulid.ID{acme.ID, sunbelt.ID}},
	}, partyGroupsOf(repo.requests))
}

/*
A customer's email asked about "PO 77Q-4182" and every search for the number
came back empty: the words matched a pro number, a BOL or a party, never the
PO on the load's order.
*/
func TestSearchShipments_FindsALoadByTheCustomersPONumber(t *testing.T) {
	t.Parallel()

	ord := shipmentports.OrderReferenceMatch{
		OrderID:     pulid.MustNew("ord_"),
		OrderNumber: "ORD-1042",
		PONumber:    "77Q-4182",
	}
	repo := &recordingShipmentRepo{}
	tool := shipmentquerytools.NewShipmentListTool(repo, shipmentquerytools.PartySources{
		Orders: &namedOrders{items: []shipmentports.OrderReferenceMatch{ord}},
	})

	result, err := tool.Query(
		t.Context(),
		querytooltest.TestParams(map[string]any{"query": "77Q-4182"}),
	)
	require.NoError(t, err)

	require.NotEmpty(t, repo.requests)
	assert.Equal(t, "77Q-4182", repo.requests[0].Filter.Query,
		"the shipment's own numbers, its external reference among them, are searched first")
	assert.Equal(t, []shipmentports.ShipmentPartyGroup{{OrderIDs: []pulid.ID{ord.OrderID}}},
		partyGroupsOf(repo.requests))

	outcome, ok := result.(querytool.SearchOutcome)
	require.True(t, ok)
	assert.Contains(t, outcome.SearchedFor, "on order ORD-1042 (PO 77Q-4182)")
}

func TestShipmentListSchema_SaysPOAndReferenceNumbersMatch(t *testing.T) {
	t.Parallel()

	tool := shipmentquerytools.NewShipmentListTool(
		&fakeShipmentRepo{},
		shipmentquerytools.PartySources{},
	)
	properties, ok := tool.ParamSchema()["properties"].(map[string]any)
	require.True(t, ok)
	query, ok := properties["query"].(map[string]any)
	require.True(t, ok)

	assert.Contains(t, query["description"], "PO or reference number")
	assert.Contains(t, tool.Description(), "PO or reference number")
}

func TestSearchShipments_EachRowSaysWhatItCarries(t *testing.T) {
	t.Parallel()

	row := shipmentquerytools.ToRow(&shipment.Shipment{
		ID: pulid.MustNew("shp_"),
		Commodities: []*shipment.ShipmentCommodity{
			{Commodity: &commodity.Commodity{Name: "Steel Coils"}},
			{Commodity: &commodity.Commodity{Name: "Steel Coils"}},
			{Commodity: &commodity.Commodity{Name: "Lumber"}},
			{},
		},
	})

	assert.Equal(t, "Steel Coils, Lumber", row.Carrying)

	repo := &recordingShipmentRepo{}
	_, err := shipmentquerytools.NewShipmentListTool(repo, shipmentquerytools.PartySources{}).
		Query(t.Context(), querytooltest.TestParams(map[string]any{"query": "acme"}))
	require.NoError(t, err)
	require.NotEmpty(t, repo.requests)
	for _, req := range repo.requests {
		assert.True(t, req.ShipmentOptions.IncludeCommodities)
	}
}

func heldShipments() ([]*shipment.Shipment, *fakeActiveHolds) {
	held := &shipment.Shipment{
		ID: pulid.MustNew("shp_"), ProNumber: "S-1", Status: shipment.StatusAssigned,
	}
	free := &shipment.Shipment{
		ID: pulid.MustNew("shp_"), ProNumber: "S-2", Status: shipment.StatusAssigned,
	}
	holds := &fakeActiveHolds{holds: []*shipmenthold.ShipmentHold{{
		ID:            pulid.MustNew("shh_"),
		ShipmentID:    held.ID,
		Type:          holdreason.HoldTypeFinance,
		Severity:      holdreason.HoldSeverityBlocking,
		ReasonCode:    "CREDIT_HOLD",
		BlocksBilling: true,
		HoldReason:    &holdreason.HoldReason{Label: "Customer over credit limit"},
	}}}

	return []*shipment.Shipment{held, free}, holds
}

/*
"pull the credit holds off all their denver loads" listed nine loads, then
opened each with shipment_get to see which was held, and the twelve-call
budget ran out before the third release. The rows carry the holds in force,
read for the whole page in one call, with the holdId the release takes.
*/
func TestShipmentList_RowsCarryTheHoldsInForceReadOnceForThePage(t *testing.T) {
	t.Parallel()

	items, holds := heldShipments()
	tool := shipmentquerytools.NewShipmentListTool(&fakeShipmentRepo{items: items}, shipmentquerytools.PartySources{}).
		WithHolds(holds, &querytooltest.FakePermissions{Allowed: true})

	result, err := tool.Query(t.Context(), querytooltest.TestParams(map[string]any{}))
	require.NoError(t, err)

	assert.Equal(t, 1, holds.calls, "one read for the page, never one per row")
	assert.ElementsMatch(t, []pulid.ID{items[0].ID, items[1].ID}, holds.asked)

	document := querytooltest.EncodedDocument(t, result)
	rows, ok := document["items"].([]any)
	require.True(t, ok)
	require.Len(t, rows, 2)
	hold := querytooltest.ObjectAt(t, rows, 0, "activeHolds", 0)
	assert.Equal(t, holds.holds[0].ID.String(), hold["holdId"])
	assert.Equal(t, "Customer over credit limit", hold["reason"])
	assert.Equal(t, "CREDIT_HOLD", hold["reasonCode"])
	assert.Equal(t, "billing", hold["blocks"])
	assert.NotContains(t, querytooltest.ObjectAt(t, rows, 1), "activeHolds")
}

func TestShipmentList_ListsOnlyHeldLoads(t *testing.T) {
	t.Parallel()

	items, holds := heldShipments()
	repo := &fakeShipmentRepo{items: items[:1]}
	tool := shipmentquerytools.NewShipmentListTool(repo, shipmentquerytools.PartySources{}).
		WithHolds(holds, &querytooltest.FakePermissions{Allowed: true})

	result, err := tool.Query(t.Context(), querytooltest.TestParams(map[string]any{
		"onHold":          true,
		"holdReasonCodes": []any{"CREDIT_HOLD"},
		"status":          []any{"Assigned"},
	}))
	require.NoError(t, err)

	require.NotNil(t, repo.captured)
	require.NotNil(t, repo.captured.ShipmentOptions.ActiveHold)
	assert.True(t, *repo.captured.ShipmentOptions.ActiveHold)
	assert.Equal(t, []string{"CREDIT_HOLD"}, repo.captured.ShipmentOptions.HoldReasonCodes)
	assert.Equal(t, []shipment.Status{shipment.StatusAssigned},
		repo.captured.ShipmentOptions.Statuses)

	outcome, ok := result.(querytool.SearchOutcome)
	require.True(t, ok)
	assert.Contains(t, outcome.SearchedFor, "with a hold in force for CREDIT_HOLD")
}

func TestShipmentList_RefusesHoldReasonsWithLoadsThatAreNotHeld(t *testing.T) {
	t.Parallel()

	repo := &fakeShipmentRepo{}
	tool := shipmentquerytools.NewShipmentListTool(repo, shipmentquerytools.PartySources{}).
		WithHolds(&fakeActiveHolds{}, &querytooltest.FakePermissions{Allowed: true})

	_, err := tool.Query(t.Context(), querytooltest.TestParams(map[string]any{
		"onHold":          false,
		"holdReasonCodes": []any{"CREDIT_HOLD"},
	}))
	require.ErrorContains(t, err, "onHold")
	assert.Nil(t, repo.captured, "nothing is read on a refused argument")
}

func TestShipmentList_HoldsFollowTheReadersAccess(t *testing.T) {
	t.Parallel()

	items, holds := heldShipments()
	tool := shipmentquerytools.NewShipmentListTool(&fakeShipmentRepo{items: items}, shipmentquerytools.PartySources{}).
		WithHolds(holds, querytooltest.PersonReaching(
			permission.ResourceShipment,
			permission.SensitivityRestricted,
		))

	_, err := tool.Query(t.Context(), querytooltest.ChatParams(map[string]any{"onHold": true}, ""))
	var refused *graphqlreadports.ReadRefusedError
	require.True(t, errors.As(err, &refused), "a hold filter is refused, not ignored")
	assert.Equal(t, permission.ResourceShipmentHold, refused.Resource)

	result, err := tool.Query(t.Context(), querytooltest.ChatParams(map[string]any{}, ""))
	require.NoError(t, err)
	outcome, ok := result.(querytool.SearchOutcome)
	require.True(t, ok)
	assert.Contains(t, outcome.Note, "Holds are left out")
	assert.Zero(t, holds.calls)
}

type fakeCommentLister struct {
	shipmentcommentports.ShipmentCommentRepository

	comments []*shipment.ShipmentComment
}

func (f *fakeCommentLister) ListByShipmentID(
	context.Context,
	*shipmentcommentports.ListShipmentCommentsRequest,
) (*pagination.CursorListResult[*shipment.ShipmentComment], error) {
	return &pagination.CursorListResult[*shipment.ShipmentComment]{Items: f.comments}, nil
}

/*
ServiceFailureDesk read a load twice and never saw the dispatcher's note that
the dock had closed: its access did not reach comments, shipment_get dropped
them without a word, and the desk reported that the record did not say why.
*/
func TestGetShipment_SaysWhenCommentsAndHoldsAreLeftOutForAccess(t *testing.T) {
	t.Parallel()

	fixture := realisticShipment()
	comments := &fakeCommentLister{comments: []*shipment.ShipmentComment{{
		ID:      pulid.MustNew("shc_"),
		Comment: "Chicago DC closed the dock for an inventory count",
	}}}
	_, holds := heldShipments()
	newTool := func(permissions *querytooltest.FakePermissions) agenttoolports.AgentQueryTool {
		return newGetShipmentTool(
			&fakeShipmentGetter{entity: fixture.entity},
			comments,
			&fakeHoldLister{holds: holds.holds},
			permissions,
		)
	}

	result, err := newTool(&querytooltest.FakePermissions{Allowed: true}).
		Query(t.Context(), querytooltest.TestParams(shipmentArgs(fixture, "")))
	require.NoError(t, err)
	document := querytooltest.EncodedDocument(t, result)
	assert.NotContains(t, document, "leftOutForAccess")
	assert.NotEmpty(t, document["recentComments"])

	result, err = newTool(querytooltest.PersonReaching(permission.ResourceShipment,
		permission.SensitivityRestricted)).
		Query(t.Context(), querytooltest.ChatParams(shipmentArgs(fixture, ""), ""))
	require.NoError(t, err)
	document = querytooltest.EncodedDocument(t, result)
	note, ok := document["leftOutForAccess"].(string)
	require.True(t, ok)
	assert.Contains(t, note, "Comments and holds are left out")
	assert.Contains(t, note, "the person you are working for")
	assert.Empty(t, document["recentComments"])

	result, err = newTool(&querytooltest.FakePermissions{Allowed: true}).
		Query(t.Context(), querytooltest.AgentParams(shipmentArgs(fixture, "full"),
			permission.SensitivityInternal))
	require.NoError(t, err)
	note, ok = querytooltest.EncodedDocument(t, result)["leftOutForAccess"].(string)
	require.True(t, ok)
	assert.Contains(t, note, "this agent's unattended access")
}
