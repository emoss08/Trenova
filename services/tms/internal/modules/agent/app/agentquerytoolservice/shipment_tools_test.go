package agentquerytoolservice

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/trailer"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/modules/places/ports/locationports"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity"
	"github.com/emoss08/trenova/internal/modules/reference/domain/holdreason"
	"github.com/emoss08/trenova/internal/modules/reference/domain/servicetype"
	"github.com/emoss08/trenova/internal/modules/reference/domain/shipmenttype"
	"github.com/emoss08/trenova/internal/modules/reference/ports/commodityports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/shipmenttypeports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeShipmentRepo struct {
	shipmentports.ShipmentRepository

	captured *shipmentports.ListShipmentsRequest
	items    []*shipment.Shipment
}

func (f *fakeShipmentRepo) List(
	_ context.Context,
	req *shipmentports.ListShipmentsRequest,
) (*pagination.CursorListResult[*shipment.Shipment], error) {
	if f.captured == nil {
		f.captured = req
	}

	return &pagination.CursorListResult[*shipment.Shipment]{Items: f.items}, nil
}

// shipment_list carried the same required-query defect as search_worker, so
// "show me everything delivered yesterday" would have failed the same way. It
// already took an optional status, which is the shape the query should have had
// from the start.
func TestSearchShipments_ListsWithoutAQuery(t *testing.T) {
	t.Parallel()

	repo := &fakeShipmentRepo{items: []*shipment.Shipment{{ID: pulid.MustNew("shp_")}}}
	tool := newShipmentListTool(repo, shipmentPartySources{})

	_, err := tool.Query(t.Context(), testParams(map[string]any{}))
	require.NoError(t, err)
	require.NotNil(t, repo.captured)

	assert.Empty(t, repo.captured.Filter.Query)
}

func TestSearchShipments_KeepsTheStatusFilter(t *testing.T) {
	t.Parallel()

	repo := &fakeShipmentRepo{}
	tool := newShipmentListTool(repo, shipmentPartySources{})

	// Completed, not "Delivered". These fixtures used to say Delivered, copied
	// from the tool's own description, which named a status the system does not
	// have — the documentation was wrong for long enough to mislead its tests.
	_, err := tool.Query(t.Context(), testParams(map[string]any{"status": "Completed"}))
	require.NoError(t, err)

	assert.Equal(t, []string{"Completed"}, statusFilterOf(t, repo.captured))
}

func statusFilterOf(t *testing.T, req *shipmentports.ListShipmentsRequest) []string {
	t.Helper()

	require.NotNil(t, req)
	for _, filter := range req.Filter.FieldFilters {
		if filter.Field != "status" {
			continue
		}
		values, ok := filter.Value.([]string)
		if !ok {
			raw, isAny := filter.Value.([]any)
			require.True(t, isAny, "the status filter holds a list, not %T", filter.Value)
			for _, value := range raw {
				values = append(values, fmt.Sprint(value))
			}
		}
		assert.Equal(t, "in", string(filter.Operator))
		return values
	}
	if len(req.ShipmentOptions.Statuses) > 0 {
		values := make([]string, 0, len(req.ShipmentOptions.Statuses))
		for _, status := range req.ShipmentOptions.Statuses {
			values = append(values, string(status))
		}
		return values
	}

	return nil
}

func TestSearchShipments_EmptyResultNamesEveryFilterItApplied(t *testing.T) {
	t.Parallel()

	repo := &fakeShipmentRepo{items: nil}
	tool := newShipmentListTool(repo, shipmentPartySources{})

	result, err := tool.Query(
		t.Context(),
		testParams(map[string]any{"query": "PRO 12345", "status": "Completed"}),
	)
	require.NoError(t, err)

	outcome, ok := result.(searchOutcome)
	require.True(t, ok)
	assert.Len(t, outcome.SearchedFor, 2, "both the text and the status are reported back")
	assert.NotEmpty(t, outcome.Note)
}

// An unfiltered list that comes back empty means something different from a
// filtered one that does, and the note has to say which — otherwise the model
// reads a short page as the whole population, or an empty organization as a
// near miss.
func TestSearchShipments_UnfilteredEmptyResultSaysSo(t *testing.T) {
	t.Parallel()

	repo := &fakeShipmentRepo{items: nil}
	tool := newShipmentListTool(repo, shipmentPartySources{})

	result, err := tool.Query(t.Context(), testParams(map[string]any{}))
	require.NoError(t, err)

	outcome, ok := result.(searchOutcome)
	require.True(t, ok)
	assert.Contains(t, outcome.Note, "whole set")
}

// shipment_list returned the stored entity for the same reason
// search_worker did, and pays the same cost: a shipment carries its moves,
// stops, commodities and charges, none of which a list of matches needs.
func TestSearchShipments_ReturnsTheCuratedRowNotTheStoredEntity(t *testing.T) {
	t.Parallel()

	repo := &fakeShipmentRepo{items: []*shipment.Shipment{{
		ID:        pulid.MustNew("shp_"),
		ProNumber: "S-1001",
		Status:    shipment.StatusNew,
	}}}

	result, err := newShipmentListTool(repo, shipmentPartySources{}).Query(t.Context(), testParams(map[string]any{}))
	require.NoError(t, err)

	outcome, ok := result.(searchOutcome)
	require.True(t, ok)
	rows, ok := outcome.Items.([]any)
	require.True(t, ok)
	require.Len(t, rows, 1)
	row, ok := rows[0].(shipmentRow)
	require.True(t, ok, "the list must return the curated row, not the entity")

	assert.Equal(t, "S-1001", row.ProNumber)
}

// A shipment that has not arrived is in transit, not missing paperwork. The
// two absences read differently and have to stay that way through search.
func TestSearchShipments_DistinguishesInTransitFromUnrecorded(t *testing.T) {
	t.Parallel()

	repo := &fakeShipmentRepo{items: []*shipment.Shipment{{
		ID: pulid.MustNew("shp_"), ProNumber: "S-1001", Status: shipment.StatusNew,
	}}}

	result, err := newShipmentListTool(repo, shipmentPartySources{}).Query(t.Context(), testParams(map[string]any{}))
	require.NoError(t, err)

	outcome, _ := result.(searchOutcome)
	encoded, err := sonic.Marshal(outcome.Items)
	require.NoError(t, err)

	assert.Contains(t, string(encoded), "not delivered yet")
	assert.NotContains(t, string(encoded), "none on file",
		"an undelivered load is not a record with a gap in it")
}

/*
The schema was telling the model to use a status that does not exist.

"Optional status filter, such as New, Assigned, InTransit, Delivered, or
Canceled" — and there is no Delivered. A delivered load is Completed. The
parameter was unvalidated, so the invented status went to the repository,
matched nothing, and came back as an empty page the model reports as "there are
no delivered shipments". The tool was instructing its caller to produce exactly
the failure the search outcome exists to prevent.
*/
func TestSearchShipments_RefusesAStatusThatDoesNotExist(t *testing.T) {
	t.Parallel()

	repo := &fakeShipmentRepo{}
	_, err := newShipmentListTool(repo, shipmentPartySources{}).Query(t.Context(), testParams(map[string]any{
		"status": "Delivered",
	}))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Completed", "the refusal has to name what to use instead")
	assert.Nil(t, repo.captured, "nothing may be queried on a refused argument")
}

func TestSearchShipments_AcceptsAStatusThatExists(t *testing.T) {
	t.Parallel()

	repo := &fakeShipmentRepo{}
	_, err := newShipmentListTool(repo, shipmentPartySources{}).Query(t.Context(), testParams(map[string]any{
		"status": "InTransit",
	}))

	require.NoError(t, err)
	assert.Equal(t, []string{"InTransit"}, statusFilterOf(t, repo.captured))
}

/*
Asked for "the LA to chicago one that hasn't been picked up yet", gpt-6-luna
searched New, then Assigned, then PartiallyAssigned, one call each. The
statuses are taken together, so the search is one call.
*/
func TestSearchShipments_TakesSeveralStatusesInOneCall(t *testing.T) {
	t.Parallel()

	repo := &fakeShipmentRepo{}
	_, err := newShipmentListTool(repo, shipmentPartySources{}).Query(t.Context(), testParams(map[string]any{
		"status": []any{"New", "PartiallyAssigned", "Assigned", "New"},
	}))

	require.NoError(t, err)
	assert.Equal(t, []string{"New", "PartiallyAssigned", "Assigned"},
		statusFilterOf(t, repo.captured))

	repo = &fakeShipmentRepo{}
	_, err = newShipmentListTool(repo, shipmentPartySources{}).Query(t.Context(), testParams(map[string]any{
		"query":  "chicago",
		"status": []any{"New", "Assigned"},
	}))
	require.NoError(t, err)
	assert.Equal(t, []shipment.Status{shipment.StatusNew, shipment.StatusAssigned},
		repo.captured.ShipmentOptions.Statuses, "a search by words narrows by status itself")
}

// Every status the schema offers has to be one the tool accepts, or the model
// is being handed a value that fails on arrival.
func TestSearchShipmentsSchema_OffersOnlyStatusesTheToolAccepts(t *testing.T) {
	t.Parallel()

	properties, ok := newShipmentListTool(&fakeShipmentRepo{}, shipmentPartySources{}).
		ParamSchema()["properties"].(map[string]any)
	require.True(t, ok)
	items, ok := properties["status"].(map[string]any)["items"].(map[string]any)
	require.True(t, ok)
	offered, ok := items["enum"].([]string)
	require.True(t, ok)
	require.NotEmpty(t, offered)

	for _, value := range offered {
		_, err := shipmentStatusFilter(map[string]any{"status": value})
		assert.NoError(t, err, "schema offers %q", value)
	}
}

// "Not yet billed" is a shipment billing has never received: its transfer
// stage is empty. The filter says so, and isnull reaches the repository as
// the condition that finds them.
func TestListShipments_FindsShipmentsNeverTransferredToBilling(t *testing.T) {
	t.Parallel()

	repo := &fakeShipmentRepo{}
	tool := newShipmentListTool(repo, shipmentPartySources{})

	filters, ok := tool.ParamSchema()["properties"].(map[string]any)["filters"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, filters["description"], "isnull means never transferred")

	_, err := tool.Query(t.Context(), testParams(map[string]any{
		"filters": []any{map[string]any{
			"field": "billingTransferStatus", "operator": "isnull",
		}},
	}))
	require.NoError(t, err)
	require.NotNil(t, repo.captured)
	require.Len(t, repo.captured.Filter.FieldFilters, 1)
	assert.Equal(t, "billingTransferStatus", repo.captured.Filter.FieldFilters[0].Field)
	assert.Equal(t, "isnull", string(repo.captured.Filter.FieldFilters[0].Operator))
}

type shipmentFixture struct {
	entity    *shipment.Shipment
	primary   *worker.Worker
	secondary *worker.Worker
	pickup    *shipment.Stop
	delivery  *shipment.Stop
	move      *shipment.ShipmentMove
	owner     *tenant.User
	canceler  *tenant.User
}

func realisticShipment() shipmentFixture {
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	illinois := &usstate.UsState{ID: pulid.MustNew("us_"), Name: "Illinois", Abbreviation: "IL"}
	ohio := &usstate.UsState{ID: pulid.MustNew("us_"), Name: "Ohio", Abbreviation: "OH"}
	lat, lng := 41.8781, -87.6298
	chicago := &location.Location{
		ID:             pulid.MustNew("loc_"),
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Code:           "CHI-DC",
		Name:           "Chicago Distribution Center",
		Description:    "Cross-dock serving the Midwest region, doors 1-40 inbound only",
		AddressLine1:   "2200 S Western Ave",
		City:           "Chicago",
		PostalCode:     "60608",
		Timezone:       "America/Chicago",
		Latitude:       &lat,
		Longitude:      &lng,
		StateID:        illinois.ID,
		State:          illinois,
	}
	columbus := &location.Location{
		ID:             pulid.MustNew("loc_"),
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Code:           "CMH-RX",
		Name:           "Columbus Receiving",
		AddressLine1:   "4100 Alum Creek Dr",
		City:           "Columbus",
		PostalCode:     "43207",
		StateID:        ohio.ID,
		State:          ohio,
	}
	windowEnd := int64(1_790_866_800)
	arrived := int64(1_790_861_400)
	departed := int64(1_790_866_800)
	pickup := &shipment.Stop{
		ID:                   pulid.MustNew("stp_"),
		OrganizationID:       orgID,
		BusinessUnitID:       buID,
		LocationID:           chicago.ID,
		Status:               shipment.StopStatusCompleted,
		Type:                 shipment.StopTypePickup,
		Sequence:             1,
		ScheduledWindowStart: 1_790_859_600,
		ScheduledWindowEnd:   &windowEnd,
		ActualArrival:        &arrived,
		ActualDeparture:      &departed,
		AddressLine:          "2200 S Western Ave, Chicago, IL 60608",
		Location:             chicago,
	}
	delivery := &shipment.Stop{
		ID:                   pulid.MustNew("stp_"),
		OrganizationID:       orgID,
		BusinessUnitID:       buID,
		LocationID:           columbus.ID,
		Status:               shipment.StopStatusNew,
		Type:                 shipment.StopTypeDelivery,
		Sequence:             2,
		ScheduledWindowStart: 1_790_942_400,
		AddressLine:          "4100 Alum Creek Dr, Columbus, OH 43207",
		Location:             columbus,
	}
	primary := &worker.Worker{
		ID:                    pulid.MustNew("wrk_"),
		OrganizationID:        orgID,
		BusinessUnitID:        buID,
		FirstName:             "Dana",
		LastName:              "Whitfield",
		AddressLine1:          "18 Maple Row",
		City:                  "Joliet",
		PostalCode:            "60431",
		Email:                 "dana.whitfield@example.com",
		PhoneNumber:           "815-555-0142",
		EmergencyContactName:  "Sam Whitfield",
		EmergencyContactPhone: "815-555-0199",
		Profile: &worker.WorkerProfile{
			ID:            pulid.MustNew("wpr_"),
			LicenseNumber: "W123-4567-8901",
			DOB:           315_532_800,
		},
	}
	secondary := &worker.Worker{
		ID:          pulid.MustNew("wrk_"),
		FirstName:   "Luis",
		LastName:    "Ortega",
		Email:       "luis.ortega@example.com",
		PhoneNumber: "815-555-0178",
	}
	tractorID := pulid.MustNew("tr_")
	trailerID := pulid.MustNew("trl_")
	move := &shipment.ShipmentMove{
		ID:             pulid.MustNew("sm_"),
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Status:         shipment.MoveStatusInTransit,
		Loaded:         true,
		Sequence:       1,
		Stops:          []*shipment.Stop{delivery, pickup},
		Assignment: &shipment.Assignment{
			ID:                pulid.MustNew("a_"),
			Status:            shipment.AssignmentStatusInProgress,
			TractorID:         &tractorID,
			TrailerID:         &trailerID,
			PrimaryWorkerID:   &primary.ID,
			SecondaryWorkerID: &secondary.ID,
			Tractor:           &tractor.Tractor{ID: tractorID, Code: "T-104"},
			Trailer:           &trailer.Trailer{ID: trailerID, Code: "TRL-2201"},
			PrimaryWorker:     primary,
			SecondaryWorker:   secondary,
		},
		CarrierAssignment: &shipment.CarrierAssignment{
			ID:                  pulid.MustNew("casn_"),
			Status:              shipment.CarrierAssignmentStatusConfirmed,
			ExternalDriverName:  "Pat Carrier",
			ExternalDriverPhone: "312-555-0111",
			Carrier: &carrier.Carrier{
				ID:   pulid.MustNew("car_"),
				Name: "Lakeshore Freight",
			},
		},
	}
	move.CarrierAssignment.CarrierID = move.CarrierAssignment.Carrier.ID
	templateID := pulid.MustNew("ft_")
	customerID := pulid.MustNew("cus_")
	entity := &shipment.Shipment{
		ID:                  pulid.MustNew("shp_"),
		OrganizationID:      orgID,
		BusinessUnitID:      buID,
		CustomerID:          customerID,
		FormulaTemplateID:   templateID,
		Status:              shipment.StatusInTransit,
		ProNumber:           "S-1001",
		BOL:                 "BOL-77812",
		BaseRate:            decimal.NewNullDecimal(decimal.RequireFromString("1850.0000")),
		FreightChargeAmount: decimal.NewNullDecimal(decimal.RequireFromString("1850.0000")),
		OtherChargeAmount:   decimal.NewNullDecimal(decimal.RequireFromString("225.5000")),
		TotalChargeAmount:   decimal.NewNullDecimal(decimal.RequireFromString("2075.5000")),
		RateLocked:          true,
		RatingDetail: &shipment.RatingDetail{
			FormulaTemplateID:   templateID.String(),
			FormulaTemplateName: "Midwest dry van",
			Expression:          "max(minCharge, distance * ratePerMile) + stops * stopCharge",
			ResolvedVariables: map[string]any{
				"distance": 355.2, "ratePerMile": 5.2, "stops": 2, "stopCharge": 50,
			},
			Result:        1850,
			Source:        "Agreement",
			AgreementName: "Acme 2026 contract",
			Breakdown: []shipment.RatingBreakdownItem{
				{Name: "linehaul", Label: "Linehaul", Amount: 1750},
				{Name: "stops", Label: "Stop charges", Amount: 100},
			},
		},
		Customer: &customer.Customer{
			ID:           customerID,
			Code:         "ACME",
			Name:         "Acme Foods",
			AddressLine1: "1 Acme Plaza",
			City:         "Chicago",
		},
		ServiceType:  &servicetype.ServiceType{ID: pulid.MustNew("st_"), Code: "STD"},
		ShipmentType: &shipmenttype.ShipmentType{ID: pulid.MustNew("sht_"), Code: "FTL"},
		FormulaTemplate: &formulatemplate.FormulaTemplate{
			ID:          templateID,
			Name:        "Midwest dry van",
			Description: "Distance-based dry van rating for the Midwest lanes",
			Expression:  "max(minCharge, distance * ratePerMile) + stops * stopCharge",
		},
		Moves: []*shipment.ShipmentMove{move},
		Commodities: []*shipment.ShipmentCommodity{{
			ID:          pulid.MustNew("sc_"),
			CommodityID: pulid.MustNew("com_"),
			Pieces:      24,
			Weight:      38_000,
			Commodity:   &commodity.Commodity{Name: "Canned goods"},
		}},
		AdditionalCharges: []*shipment.AdditionalCharge{{
			ID:                  pulid.MustNew("ac_"),
			AccessorialChargeID: pulid.MustNew("acc_"),
			Method:              accessorialcharge.MethodFlat,
			Amount:              decimal.RequireFromString("225.50"),
			Unit:                1,
			AccessorialCharge: &accessorialcharge.AccessorialCharge{
				Code:        "LUMP",
				Description: "Lumper fee",
			},
		}},
	}
	move.ShipmentID = entity.ID
	owner := &tenant.User{
		ID:           pulid.MustNew(tenant.UserIDPrefix),
		Name:         "Avery Chen",
		Username:     "achen",
		EmailAddress: "avery.chen@example.com",
		Timezone:     "America/Chicago",
	}
	canceler := &tenant.User{
		ID:           pulid.MustNew(tenant.UserIDPrefix),
		Name:         "Riley Stone",
		Username:     "rstone",
		EmailAddress: "riley.stone@example.com",
		Timezone:     "America/New_York",
	}
	entity.OwnerID = owner.ID
	entity.Owner = owner
	entity.CanceledByID = canceler.ID
	entity.CanceledBy = canceler

	return shipmentFixture{
		owner:     owner,
		canceler:  canceler,
		entity:    entity,
		primary:   primary,
		secondary: secondary,
		pickup:    pickup,
		delivery:  delivery,
		move:      move,
	}
}

func queryShipment(
	t *testing.T,
	fixture shipmentFixture,
	permissions *fakePermissions,
	params *agentports.QueryToolParams,
) map[string]any {
	t.Helper()

	tool := newGetShipmentTool(
		&fakeShipmentGetter{entity: fixture.entity},
		nil,
		&fakeHoldLister{holds: []*shipment.ShipmentHold{{
			ID:             pulid.MustNew("shh_"),
			Type:           holdreason.HoldType("Compliance"),
			Severity:       holdreason.HoldSeverityBlocking,
			Source:         shipment.HoldSourceUser,
			BlocksDelivery: true,
		}}},
		permissions,
	)
	result, err := tool.Query(t.Context(), params)
	require.NoError(t, err)

	return encodedDocument(t, result)
}

func encodedDocument(t *testing.T, result any) map[string]any {
	t.Helper()

	encoded, err := sonic.Marshal(result)
	require.NoError(t, err)
	var document map[string]any
	require.NoError(t, sonic.Unmarshal(encoded, &document))

	return document
}

func shipmentArgs(fixture shipmentFixture, detail string) map[string]any {
	args := map[string]any{"shipmentId": fixture.entity.ID.String()}
	if detail != "" {
		args["detail"] = detail
	}

	return args
}

func objectAt(t *testing.T, document any, path ...any) map[string]any {
	t.Helper()

	node := document
	for _, step := range path {
		switch key := step.(type) {
		case string:
			object, ok := node.(map[string]any)
			require.True(t, ok, "expected an object before %q", key)
			node = object[key]
		case int:
			list, ok := node.([]any)
			require.True(t, ok, "expected a list before [%d]", key)
			require.Greater(t, len(list), key)
			node = list[key]
		}
	}
	object, ok := node.(map[string]any)
	require.True(t, ok, "expected an object at %v", path)

	return object
}

func TestGetShipment_SummaryIsTheDefault(t *testing.T) {
	t.Parallel()

	fixture := realisticShipment()
	params := testParamsIn("America/New_York", shipmentArgs(fixture, ""))
	document := queryShipment(t, fixture, &fakePermissions{allowed: true}, params)

	assert.Equal(t, fixture.entity.ID.String(), document["id"])
	assert.Equal(t, "S-1001", document["proNumber"])
	assert.Equal(t, "BOL-77812", document["bol"])
	assert.Equal(t, "InTransit", document["status"])
	assert.Equal(t, map[string]any{
		"id": fixture.entity.CustomerID.String(), "name": "Acme Foods",
	}, document["customer"])
	assert.Equal(t, "STD", document["serviceType"])
	assert.Equal(t, "FTL", document["shipmentType"])

	rating := objectAt(t, document, "rating")
	assert.Equal(t, fixture.entity.FormulaTemplateID.String(), rating["formulaTemplateId"])
	assert.Equal(t, "Midwest dry van", rating["formulaTemplate"])
	assert.Equal(t, "Agreement", rating["ratingMethod"])
	assert.Equal(t, "1850", rating["baseRate"])
	assert.Equal(t, "225.50", rating["otherChargeAmount"])
	assert.Equal(t, "2075.50", rating["totalChargeAmount"])
	assert.Equal(t, true, rating["rateLocked"])

	move := objectAt(t, document, "moves", 0)
	assert.Equal(t, fixture.move.ID.String(), move["id"])
	pickup := objectAt(t, move, "stops", 0)
	delivery := objectAt(t, move, "stops", 1)
	assert.Equal(t, fixture.pickup.ID.String(), pickup["id"], "stops run in sequence")
	assert.Equal(t, "Pickup", pickup["type"])
	assert.Equal(t, "Chicago Distribution Center", pickup["location"])
	assert.Equal(t, "Chicago", pickup["city"])
	assert.Equal(t, "IL", pickup["state"])
	assert.Equal(t, "Delivery", delivery["type"])
	assert.Equal(t, "Columbus", delivery["city"])

	assignment := objectAt(t, move, "assignment")
	assert.Equal(t, "T-104", assignment["tractor"])
	assert.Equal(t, "TRL-2201", assignment["trailer"])
	assert.Equal(t, "Dana Whitfield", assignment["primaryWorker"])
	assert.Equal(t, fixture.primary.ID.String(), assignment["primaryWorkerId"])
	assert.Equal(t, "Luis Ortega", assignment["secondaryWorker"])
	assert.Equal(t, "Lakeshore Freight", objectAt(t, move, "carrier")["carrier"])

	commodity := objectAt(t, document, "commodities", 0)
	assert.Equal(t, "Canned goods", commodity["name"])
	assert.InDelta(t, 24, commodity["pieces"], 0)
	assert.InDelta(t, 38_000, commodity["weight"], 0)

	charge := objectAt(t, document, "additionalCharges", 0)
	assert.Equal(t, "LUMP", charge["code"])
	assert.Equal(t, "225.5", charge["amount"])

	hold := objectAt(t, document, "activeHolds", 0)
	assert.Equal(t, true, hold["blocksDelivery"])

	encoded, err := sonic.Marshal(document)
	require.NoError(t, err)
	for _, contact := range []string{
		"dana.whitfield@example.com", "815-555-0142", "18 Maple Row", "W123-4567-8901",
		"312-555-0111",
	} {
		assert.NotContains(
			t,
			string(encoded),
			contact,
			"the summary names people, never reaches them",
		)
	}
}

func TestGetShipment_SummaryIsAFractionOfTheFullRecord(t *testing.T) {
	t.Parallel()

	fixture := realisticShipment()
	permissions := &fakePermissions{allowed: true}
	summary, err := sonic.Marshal(queryShipment(t, fixture, permissions,
		testParams(shipmentArgs(fixture, "summary"))))
	require.NoError(t, err)
	full, err := sonic.Marshal(queryShipment(t, realisticShipment(), permissions,
		testParams(shipmentArgs(fixture, "full"))))
	require.NoError(t, err)

	assert.Less(t, len(summary)*3, len(full),
		"summary is %d bytes against %d for the full record", len(summary), len(full))
}

func TestGetShipment_FullKeepsTheStoredRecord(t *testing.T) {
	t.Parallel()

	fixture := realisticShipment()
	document := queryShipment(t, fixture, &fakePermissions{allowed: true},
		testParams(shipmentArgs(fixture, "full")))

	assert.Equal(t, fixture.entity.ID.String(), document["id"])
	assert.Equal(t, fixture.entity.FormulaTemplateID.String(), document["formulaTemplateId"])
	assert.Contains(t, objectAt(t, document, "ratingDetail"), "resolvedVariables")
	assert.Contains(t, objectAt(t, document, "formulaTemplate"), "expression")
	assignment := objectAt(t, document, "moves", 0, "assignment")
	assert.Equal(t, "T-104", objectAt(t, assignment, "tractor")["code"])
	assert.Equal(t, "Dana", objectAt(t, assignment, "primaryWorker")["firstName"])
	stop := objectAt(t, document, "moves", 0, "stops", 0)
	assert.Contains(t, stop, "addressLine")
	assert.Contains(t, document, "activeHolds")
}

func TestGetShipment_SummaryWindowsAreLocalToTheStop(t *testing.T) {
	t.Parallel()

	fixture := realisticShipment()
	document := queryShipment(t, fixture, &fakePermissions{allowed: true},
		testParamsIn("America/New_York", shipmentArgs(fixture, "")))

	pickup := objectAt(t, document, "moves", 0, "stops", 0)
	assert.Equal(t, "America/Chicago", pickup["timezone"])
	assert.Equal(t, "2026-10-01T08:00", pickup["scheduledWindowStart"])
	assert.Equal(t, "2026-10-01T10:00", pickup["scheduledWindowEnd"])
	assert.Equal(t, "2026-10-01T08:30", pickup["actualArrival"])
	assert.Equal(t, "2026-10-01T10:00", pickup["actualDeparture"])

	delivery := objectAt(t, document, "moves", 0, "stops", 1)
	assert.Equal(t, "America/New_York", delivery["timezone"],
		"a location with no zone reads in the organization's")
	assert.Equal(t, "2026-10-02T08:00", delivery["scheduledWindowStart"])
	assert.NotContains(t, delivery, "actualArrival")

	utc := queryShipment(t, realisticShipment(), &fakePermissions{allowed: true},
		testParams(shipmentArgs(fixture, "")))
	delivery = objectAt(t, utc, "moves", 0, "stops", 1)
	assert.Equal(t, "UTC", delivery["timezone"])
	assert.Equal(t, "2026-10-02T12:00", delivery["scheduledWindowStart"])
}

func TestGetShipment_WorkerContactFollowsTheCeiling(t *testing.T) {
	t.Parallel()

	for _, detail := range []string{"summary", "full"} {
		t.Run(detail+" below the ceiling", func(t *testing.T) {
			t.Parallel()

			fixture := realisticShipment()
			document := queryShipment(t, fixture, &fakePermissions{allowed: true},
				agentParams(shipmentArgs(fixture, detail), permission.SensitivityInternal))

			encoded, err := sonic.Marshal(document)
			require.NoError(t, err)
			for _, contact := range []string{
				"dana.whitfield@example.com", "815-555-0142", "luis.ortega@example.com",
				"815-555-0178", "18 Maple Row", "Sam Whitfield", "W123-4567-8901",
				"312-555-0111",
			} {
				assert.NotContains(t, string(encoded), contact)
			}
			assert.Contains(t, string(encoded), "Dana")
			if detail == "full" {
				assert.Subset(t, document["withheldByAccess"], []any{
					"worker.email", "worker.phoneNumber", "worker.addressLine1",
					"shipmentMove.externalDriverPhone",
				})
				carrier := objectAt(t, document, "moves", 0, "carrierAssignment")
				assert.NotContains(t, carrier, "externalDriverPhone")
				assert.Equal(t, "Pat Carrier", carrier["externalDriverName"],
					"the external driver's name is Internal, like a worker's")
				assert.NotContains(t, document["withheldByAccess"], "worker.licenseNumber",
					"a confidential field is never named")
			}
		})

		t.Run(detail+" within the ceiling", func(t *testing.T) {
			t.Parallel()

			fixture := realisticShipment()
			document := queryShipment(t, fixture,
				personReachingEach(permission.SensitivityRestricted,
					permission.ResourceWorker, permission.ResourceShipmentMove),
				chatParams(shipmentArgs(fixture, detail), ""))

			encoded, err := sonic.Marshal(document)
			require.NoError(t, err)
			assert.NotContains(t, string(encoded), "W123-4567-8901",
				"a confidential field never reaches a model")
			assert.NotContains(t, document, "withheldByAccess")
			if detail == "full" {
				primary := objectAt(t, document, "moves", 0, "assignment", "primaryWorker")
				assert.Equal(t, "dana.whitfield@example.com", primary["email"])
				assert.Equal(t, "815-555-0142", primary["phoneNumber"])
				assert.NotContains(t, objectAt(t, primary, "profile"), "dob")
				carrier := objectAt(t, document, "moves", 0, "carrierAssignment")
				assert.Equal(t, "312-555-0111", carrier["externalDriverPhone"])
			}
		})
	}
}

func TestGetShipment_SummaryWithholdsNamesFromSomeoneWhoCannotSeeWorkers(t *testing.T) {
	t.Parallel()

	fixture := realisticShipment()
	document := queryShipment(t, fixture,
		personReaching(permission.ResourceShipment, permission.SensitivityRestricted),
		chatParams(shipmentArgs(fixture, ""), ""))

	assignment := objectAt(t, document, "moves", 0, "assignment")
	assert.NotContains(t, assignment, "primaryWorker")
	assert.Equal(t, fixture.primary.ID.String(), assignment["primaryWorkerId"])
	assert.Subset(t, document["withheldByAccess"], []any{"worker.firstName", "worker.lastName"})
}

func TestGetShipmentSchema_RefusesAnUnknownDetail(t *testing.T) {
	t.Parallel()

	fixture := realisticShipment()
	getter := &fakeShipmentGetter{entity: fixture.entity}
	tool := newGetShipmentTool(getter, nil, nil, &fakePermissions{allowed: true})
	schema := tool.ParamSchema()

	require.NoError(t, toolschema.Validate(schema, shipmentArgs(fixture, "full")))
	require.NoError(t, toolschema.Validate(schema, shipmentArgs(fixture, "summary")))
	require.NoError(t, toolschema.Validate(schema, shipmentArgs(fixture, "")))
	require.Error(t, toolschema.Validate(schema, shipmentArgs(fixture, "brief")))

	_, err := tool.Query(t.Context(), testParams(shipmentArgs(fixture, "brief")))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "summary")
	assert.Zero(t, getter.reads, "nothing is read on a refused argument")
}

func TestGetShipment_FullReducesTheUsersOnTheRecord(t *testing.T) {
	t.Parallel()

	fixture := realisticShipment()
	document := queryShipment(t, fixture, &fakePermissions{allowed: true},
		agentParams(shipmentArgs(fixture, "full"), permission.SensitivityRestricted))

	assert.Equal(t, map[string]any{
		"id": fixture.owner.ID.String(), "name": "Avery Chen",
	}, document["owner"])
	assert.Equal(t, map[string]any{
		"id": fixture.canceler.ID.String(), "name": "Riley Stone",
	}, document["canceledBy"])
	assert.Equal(t, fixture.owner.ID.String(), document["ownerId"])

	encoded, err := sonic.Marshal(document)
	require.NoError(t, err)
	for _, contact := range []string{"avery.chen@example.com", "riley.stone@example.com"} {
		assert.NotContains(t, string(encoded), contact, "no user's email reaches a model")
	}
}

func TestGetShipment_SummaryIsUnchangedByTheUsersOnTheRecord(t *testing.T) {
	t.Parallel()

	fixture := realisticShipment()
	document := queryShipment(t, fixture, &fakePermissions{allowed: true},
		agentParams(shipmentArgs(fixture, ""), permission.SensitivityInternal))

	assert.Equal(t, fixture.owner.ID.String(), document["ownerId"])
	assert.NotContains(t, document, "owner")
	assert.Equal(t, "Lakeshore Freight", objectAt(t, document, "moves", 0, "carrier")["carrier"])
	assert.NotContains(t, document, "withheldByAccess",
		"the summary names people and withholds nothing it did not name")
}

type recordingShipmentRepo struct {
	shipmentports.ShipmentRepository

	requests []*shipmentports.ListShipmentsRequest
}

func (f *recordingShipmentRepo) List(
	_ context.Context,
	req *shipmentports.ListShipmentsRequest,
) (*pagination.CursorListResult[*shipment.Shipment], error) {
	f.requests = append(f.requests, req)

	return &pagination.CursorListResult[*shipment.Shipment]{}, nil
}

type namedCommodities struct {
	commodityports.CommodityRepository

	items []*commodity.Commodity
}

func (f *namedCommodities) List(
	_ context.Context,
	req *commodityports.ListCommodityRequest,
) (*pagination.ListResult[*commodity.Commodity], error) {
	matched := make([]*commodity.Commodity, 0, len(f.items))
	for _, item := range f.items {
		if strings.Contains(strings.ToLower(item.Name), strings.ToLower(req.Filter.Query)) {
			matched = append(matched, item)
		}
	}

	return &pagination.ListResult[*commodity.Commodity]{Items: matched}, nil
}

/*
"did the acme steel load deliver yet?" found nothing: shipment_list matched
customers, places and drivers but not what a load carries. SEED-SHP-005 is
Acme's load of steel coils.
*/
func TestSearchShipments_MatchesWhatALoadCarries(t *testing.T) {
	t.Parallel()

	steel := &commodity.Commodity{ID: pulid.MustNew("com_"), Name: "Steel Coils"}
	repo := &recordingShipmentRepo{}
	tool := newShipmentListTool(repo, shipmentPartySources{
		commodities: &namedCommodities{items: []*commodity.Commodity{steel}},
	})

	_, err := tool.Query(t.Context(), agentParams(map[string]any{"query": "steel"},
		permission.SensitivityRestricted))
	require.NoError(t, err)

	var carried []pulid.ID
	for _, req := range repo.requests {
		if len(req.ShipmentOptions.CommodityIDs) > 0 {
			carried = req.ShipmentOptions.CommodityIDs
		}
	}
	assert.Equal(t, []pulid.ID{steel.ID}, carried)
}

type namedShipmentTypes struct {
	shipmenttypeports.ShipmentTypeRepository

	items []*shipmenttype.ShipmentType
}

func (f *namedShipmentTypes) List(
	_ context.Context,
	req *shipmenttypeports.ListShipmentTypesRequest,
) (*pagination.ListResult[*shipmenttype.ShipmentType], error) {
	matched := make([]*shipmenttype.ShipmentType, 0, len(f.items))
	for _, item := range f.items {
		if strings.EqualFold(item.Code, req.Filter.Query) {
			matched = append(matched, item)
		}
	}

	return &pagination.ListResult[*shipmenttype.ShipmentType]{Items: matched}, nil
}

// "the peak ltl going to chicago" found nothing tagged LTL: search matched no
// shipment type, though LTL is one.
func TestSearchShipments_MatchesTheShipmentType(t *testing.T) {
	t.Parallel()

	ltl := &shipmenttype.ShipmentType{ID: pulid.MustNew("sht_"), Code: "LTL"}
	repo := &recordingShipmentRepo{}
	tool := newShipmentListTool(repo, shipmentPartySources{
		types: &namedShipmentTypes{items: []*shipmenttype.ShipmentType{ltl}},
	})

	_, err := tool.Query(t.Context(), agentParams(map[string]any{"query": "ltl"},
		permission.SensitivityRestricted))
	require.NoError(t, err)

	var typed []pulid.ID
	for _, req := range repo.requests {
		if len(req.ShipmentOptions.ShipmentTypeIDs) > 0 {
			typed = req.ShipmentOptions.ShipmentTypeIDs
		}
	}
	assert.Equal(t, []pulid.ID{ltl.ID}, typed)
}

type namedLocations struct {
	locationports.LocationRepository

	items []*location.Location
}

func (f *namedLocations) List(
	_ context.Context,
	req *locationports.ListLocationRequest,
) (*pagination.ListResult[*location.Location], error) {
	matched := make([]*location.Location, 0, len(f.items))
	for _, item := range f.items {
		if strings.Contains(strings.ToLower(item.City), strings.ToLower(req.Filter.Query)) {
			matched = append(matched, item)
		}
	}

	return &pagination.ListResult[*location.Location]{Items: matched}, nil
}

/*
"invoice the peak denver to phoenix load" matched Peak and a stop at Denver or
Phoenix, so Peak's newest loads into Denver filled the page and SEED-SHP-009,
Denver to Phoenix, was never seen. Two words naming places each have to be a
stop on the load.
*/
func TestSearchShipments_RequiresAStopAtEachPlaceNamed(t *testing.T) {
	t.Parallel()

	denver := &location.Location{ID: pulid.MustNew("loc_"), Name: "Denver Drop Point", City: "Denver"}
	phoenix := &location.Location{
		ID: pulid.MustNew("loc_"), Name: "Phoenix Cold Storage", City: "Phoenix",
	}
	repo := &recordingShipmentRepo{}
	tool := newShipmentListTool(repo, shipmentPartySources{
		locations: &namedLocations{items: []*location.Location{denver, phoenix}},
	})

	_, err := tool.Query(t.Context(), agentParams(map[string]any{"query": "denver phoenix"},
		permission.SensitivityRestricted))
	require.NoError(t, err)

	var each [][]pulid.ID
	for _, req := range repo.requests {
		if len(req.ShipmentOptions.StopEachOf) > 0 {
			each = req.ShipmentOptions.StopEachOf
		}
		assert.Empty(t, req.ShipmentOptions.StopLocationIDs,
			"either place alone is not the load meant")
	}
	assert.Equal(t, [][]pulid.ID{{denver.ID}, {phoenix.ID}}, each)
}

type statusShipmentRepo struct {
	shipmentports.ShipmentRepository

	items []*shipment.Shipment
}

func (f *statusShipmentRepo) List(
	_ context.Context,
	req *shipmentports.ListShipmentsRequest,
) (*pagination.CursorListResult[*shipment.Shipment], error) {
	out := make([]*shipment.Shipment, 0, len(f.items))
	for _, item := range f.items {
		statuses := req.ShipmentOptions.Statuses
		if len(statuses) > 0 && !slices.Contains(statuses, item.Status) {
			continue
		}
		out = append(out, item)
		if limit := req.Filter.Pagination.Limit; limit > 0 && len(out) >= limit {
			break
		}
	}

	return &pagination.CursorListResult[*shipment.Shipment]{Items: out}, nil
}

/*
"email freshhaul that the reefer to LA is running behind" was searched across
New through Delayed. Ten new FreshHaul loads to Los Angeles filled the page,
the one delayed load never appeared, and Haiku asked which load was meant. A
full page across several statuses also carries the newest match in each status
it does not show.
*/
func TestSearchShipments_AFullPageStillShowsEachStatusAskedFor(t *testing.T) {
	t.Parallel()

	items := make([]*shipment.Shipment, 0, 11)
	for range defaultSearchLimit {
		items = append(items, &shipment.Shipment{
			ID: pulid.MustNew("shp_"), ProNumber: "NEW", Status: shipment.StatusNew,
		})
	}
	delayed := &shipment.Shipment{
		ID: pulid.MustNew("shp_"), ProNumber: "SEED-SHP-010", Status: shipment.StatusDelayed,
	}
	items = append(items, delayed)

	result, err := newShipmentListTool(&statusShipmentRepo{items: items}, shipmentPartySources{}).
		Query(t.Context(), testParams(map[string]any{
			"query":  "freshhaul los angeles",
			"status": []any{"New", "InTransit", "Delayed"},
		}))
	require.NoError(t, err)

	outcome, ok := result.(searchOutcome)
	require.True(t, ok)
	rows, ok := outcome.Items.([]shipmentRow)
	require.True(t, ok)
	pros := make([]string, 0, len(rows))
	for idx := range rows {
		pros = append(pros, rows[idx].ProNumber)
	}
	assert.Contains(t, pros, "SEED-SHP-010")
	assert.Contains(t, outcome.Note, "Delayed")
	assert.True(t, outcome.HasMore)
}

/*
Asked "who's on the second one", gpt-6-luna read SEED-SHP-006, found no
assignment on its second move, could not tell nobody from not shown, and went
looking on the dispatch board. A move with neither a driver nor a carrier says
it needs a driver.
*/
func TestMoveSummary_SaysAnUncoveredMoveNeedsADriver(t *testing.T) {
	t.Parallel()

	input := &shipmentSummaryInput{}
	open := moveSummary(&shipment.ShipmentMove{ID: pulid.MustNew("sm_")}, input)
	assert.Equal(t, coverageNone, open.Coverage)

	covered := moveSummary(&shipment.ShipmentMove{
		ID:         pulid.MustNew("sm_"),
		Assignment: &shipment.Assignment{ID: pulid.MustNew("a_")},
	}, input)
	assert.Empty(t, covered.Coverage)
}

type proShipmentRepo struct {
	shipmentports.ShipmentRepository

	items []*shipment.Shipment
}

func (f *proShipmentRepo) List(
	_ context.Context,
	_ *shipmentports.ListShipmentsRequest,
) (*pagination.CursorListResult[*shipment.Shipment], error) {
	return &pagination.CursorListResult[*shipment.Shipment]{Items: f.items}, nil
}

func TestGetShipment_ReadsTheShipmentAPersonNamedByItsProNumber(t *testing.T) {
	t.Parallel()

	fixture := realisticShipment()
	other := &shipment.Shipment{ID: pulid.MustNew("shp_"), ProNumber: fixture.entity.ProNumber + "1"}
	repo := &proShipmentRepo{items: []*shipment.Shipment{other, fixture.entity}}
	tool := newGetShipmentTool(repo, nil, nil, &fakePermissions{allowed: true})

	require.NoError(t, toolschema.Validate(tool.ParamSchema(),
		map[string]any{"proNumber": fixture.entity.ProNumber}))
	result, err := tool.Query(t.Context(), testParams(map[string]any{
		"proNumber": strings.ToLower(fixture.entity.ProNumber),
	}))
	require.NoError(t, err)
	encoded, err := sonic.Marshal(result)
	require.NoError(t, err)
	assert.Contains(t, string(encoded), fixture.entity.ID.String())

	_, err = tool.Query(t.Context(), testParams(map[string]any{}))
	require.ErrorContains(t, err, "shipmentId or proNumber")
}

func TestShipmentList_NarrowsASearchByWordsWithTheFiltersAndPagesIt(t *testing.T) {
	t.Parallel()

	repo := &fakeShipmentRepo{}
	_, err := newShipmentListTool(repo, shipmentPartySources{}).Query(t.Context(),
		testParams(map[string]any{
			"query": "acme",
			"filters": []any{map[string]any{
				"field": "billingTransferStatus", "operator": "isnull",
			}},
			"sortBy": "createdAt",
			"offset": float64(10),
			"limit":  float64(5),
		}))
	require.NoError(t, err)
	require.NotNil(t, repo.captured)
	assert.Equal(t, "acme", repo.captured.Filter.Query)
	require.Len(t, repo.captured.Filter.FieldFilters, 1)
	assert.Equal(t, "billingTransferStatus", repo.captured.Filter.FieldFilters[0].Field)
	assert.Equal(t, 10, repo.captured.Filter.Pagination.Offset)
	assert.Equal(t, 5, repo.captured.Filter.Pagination.Limit)
	assert.NotEmpty(t, repo.captured.Filter.Sort)
}

func TestShipmentListSchema_OffersStatusOnceAndNotAgainAsAFilter(t *testing.T) {
	t.Parallel()

	properties, ok := newShipmentListTool(&fakeShipmentRepo{}, shipmentPartySources{}).
		ParamSchema()["properties"].(map[string]any)
	require.True(t, ok)
	filters, ok := properties["filters"].(map[string]any)
	require.True(t, ok)
	fields := toolschema.EnumValues(
		filters["items"].(map[string]any)["properties"].(map[string]any)["field"].(map[string]any))
	assert.NotContains(t, fields, "status")
	assert.Contains(t, fields, "billingTransferStatus")
}
