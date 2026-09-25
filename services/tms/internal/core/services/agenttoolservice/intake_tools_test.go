package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/tenderservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeShipmentWriter struct {
	existing *shipment.Shipment
	created  *shipment.Shipment
	updated  *shipment.Shipment
	lastGet  *repositories.GetShipmentByIDRequest
	actor    *serviceports.RequestActor
}

func (f *fakeShipmentWriter) Get(
	_ context.Context,
	req *repositories.GetShipmentByIDRequest,
) (*shipment.Shipment, error) {
	f.lastGet = req

	return f.existing, nil
}

func (f *fakeShipmentWriter) Create(
	_ context.Context,
	entity *shipment.Shipment,
	actor *serviceports.RequestActor,
) (*shipment.Shipment, error) {
	f.created = entity
	f.actor = actor
	entity.ID = pulid.MustNew("shp_")

	return entity, nil
}

func (f *fakeShipmentWriter) Update(
	_ context.Context,
	entity *shipment.Shipment,
	actor *serviceports.RequestActor,
) (*shipment.Shipment, error) {
	f.updated = entity
	f.actor = actor

	return entity, nil
}

type fakeImportCompleter struct {
	completed []string
	tenant    pagination.TenantInfo
}

func (f *fakeImportCompleter) CompleteHistory(
	_ context.Context,
	documentID string,
	tenantInfo pagination.TenantInfo,
) error {
	f.completed = append(f.completed, documentID)
	f.tenant = tenantInfo

	return nil
}

func shipmentPayload(customerID, serviceTypeID, originID, destID pulid.ID) map[string]any {
	return map[string]any{
		"customerId":    customerID.String(),
		"serviceTypeId": serviceTypeID.String(),
		"bol":           "BOL-778",
		"pieces":        12,
		"weight":        18000,
		"moves": []any{map[string]any{
			"loaded": true,
			"stops": []any{
				map[string]any{
					"locationId": originID.String(), "type": "Pickup", "sequence": 0, "scheduledWindowStart": 1_790_000_000,
				},
				map[string]any{
					"locationId": destID.String(), "type": "Delivery", "sequence": 1, "scheduledWindowStart": 1_790_086_400,
				},
			},
		}},
	}
}

func TestCreateShipment_EntersTheShipmentUnderTheActorTenant(t *testing.T) {
	t.Parallel()

	writer := &fakeShipmentWriter{}
	imports := &fakeImportCompleter{}
	tool := newCreateShipmentTool(writer, imports, nil)
	assert.Equal(t, permission.ResourceShipment, tool.Policy().Resource)
	assert.Equal(t, permission.OpCreate, tool.Policy().Operation)
	assert.Equal(t, agent.TierActWithApproval, tool.Policy().DefaultTier)
	assert.True(t, tool.Policy().Idempotent)

	customerID, serviceTypeID := pulid.MustNew("cust_"), pulid.MustNew("st_")
	originID, destID := pulid.MustNew("loc_"), pulid.MustNew("loc_")
	payload := shipmentPayload(customerID, serviceTypeID, originID, destID)
	// A model that names another organization, or an id for the record it is
	// creating, is overruled by the actor and the fact that the record is new.
	payload["organizationId"] = pulid.MustNew("org_").String()
	payload["id"] = pulid.MustNew("shp_").String()
	docID := pulid.MustNew("doc_")

	params := executeParams(map[string]any{"shipment": payload, "sourceDocumentId": docID.String()})
	params.IdempotencyKey = "idem-1"
	require.NoError(t, tool.Execute(t.Context(), params))

	require.NotNil(t, writer.created)
	created := writer.created
	assert.Equal(t, params.OrganizationID, created.OrganizationID)
	assert.Equal(t, params.BusinessUnitID, created.BusinessUnitID)
	assert.Equal(t, params.Actor.UserID, created.EnteredByID)
	assert.Equal(t, customerID, created.CustomerID)
	assert.Equal(t, "BOL-778", created.BOL)
	assert.EqualValues(t, 18000, *created.Weight)
	assert.Equal(t, shipment.StatusNew, created.Status)
	assert.Equal(t, docID.String(), created.SourceDocumentID)
	require.Len(t, created.Moves, 1)
	require.Len(t, created.Moves[0].Stops, 2)
	assert.Equal(t, params.OrganizationID, created.Moves[0].Stops[1].OrganizationID)
	assert.Equal(t, shipment.StopScheduleTypeOpen, created.Moves[0].Stops[1].ScheduleType)
	assert.Equal(t, destID, created.Moves[0].Stops[1].LocationID)
	assert.Same(t, params.Actor, writer.actor)

	assert.Equal(
		t,
		[]string{docID.String()},
		imports.completed,
		"the import conversation closes once the shipment exists",
	)
	assert.Equal(t, params.OrganizationID, imports.tenant.OrgID)
}

func TestCreateShipment_RefusesAWriteWithoutAnIdempotencyKeyOrAMismatchedActor(t *testing.T) {
	t.Parallel()

	writer := &fakeShipmentWriter{}
	tool := newCreateShipmentTool(writer, nil, nil)
	payload := shipmentPayload(
		pulid.MustNew("cust_"),
		pulid.MustNew("st_"),
		pulid.MustNew("loc_"),
		pulid.MustNew("loc_"),
	)

	params := executeParams(map[string]any{"shipment": payload})
	require.ErrorIs(t, tool.Execute(t.Context(), params), ErrMissingIdempotencyKey)

	params.IdempotencyKey = "idem-2"
	params.Actor.BusinessUnitID = pulid.MustNew("bu_")
	require.ErrorIs(t, tool.Execute(t.Context(), params), ErrTenantMismatch)
	assert.Nil(t, writer.created)

	params = executeParams(map[string]any{"shipment": payload, "sourceDocumentId": "not-an-id"})
	params.IdempotencyKey = "idem-3"
	require.Error(t, tool.Execute(t.Context(), params))
	assert.Nil(t, writer.created)
}

func TestUpdateShipment_PatchesOnlyTheNamedFields(t *testing.T) {
	t.Parallel()

	shipmentID := pulid.MustNew("shp_")
	original := &shipment.Shipment{
		ID: shipmentID, CustomerID: pulid.MustNew("cust_"), ServiceTypeID: pulid.MustNew("st_"),
		BOL: "OLD", Version: 3,
	}
	writer := &fakeShipmentWriter{existing: original}
	tool := newUpdateShipmentTool(writer, nil)
	assert.Equal(t, permission.OpUpdate, tool.Policy().Operation)

	newCustomer := pulid.MustNew("cust_")
	params := executeParams(map[string]any{
		"shipmentId": shipmentID.String(),
		"customerId": newCustomer.String(),
		"weight":     22000,
	})
	require.NoError(t, tool.Execute(t.Context(), params))

	require.NotNil(t, writer.updated)
	assert.Equal(t, newCustomer, writer.updated.CustomerID)
	assert.EqualValues(t, 22000, *writer.updated.Weight)
	assert.Equal(t, "OLD", writer.updated.BOL, "a field not named is left alone")
	assert.Equal(t, original.ServiceTypeID, writer.updated.ServiceTypeID)
	assert.True(
		t,
		writer.lastGet.ExpandShipmentDetails,
		"the update carries the moves it read, so none are lost",
	)
	assert.Equal(t, params.OrganizationID, writer.lastGet.TenantInfo.OrgID)

	target, ok := tool.(serviceports.TargetedTool).Target(params.Params)
	require.True(t, ok)
	assert.Equal(t, shipmentID, target.ID)
	assert.Equal(t, permission.ResourceShipment, target.Resource)
}

func TestUpdateShipment_RefusesAnEmptyPatchAndABadID(t *testing.T) {
	t.Parallel()

	writer := &fakeShipmentWriter{existing: &shipment.Shipment{}}
	tool := newUpdateShipmentTool(writer, nil)

	err := tool.Execute(
		t.Context(),
		executeParams(map[string]any{"shipmentId": pulid.MustNew("shp_").String()}),
	)
	require.ErrorContains(t, err, "Nothing to change")
	assert.Nil(t, writer.updated)

	err = tool.Execute(t.Context(), executeParams(map[string]any{
		"shipmentId": pulid.MustNew("shp_").String(),
		"customerId": "acme",
	}))
	require.ErrorContains(t, err, `"customerId" is not an id`)
	assert.Nil(t, writer.updated)
}

type fakeTenders struct {
	waterfall *tenderservice.CreateWaterfallTenderRequest
	spot      *tenderservice.CreateSpotTenderRequest
}

func (f *fakeTenders) CreateWaterfall(
	_ context.Context,
	req *tenderservice.CreateWaterfallTenderRequest,
) (*tenderservice.CreateWaterfallResult, error) {
	f.waterfall = req

	return &tenderservice.CreateWaterfallResult{
		Tender: &tender.Tender{ID: pulid.MustNew("tnd_")},
	}, nil
}

func (f *fakeTenders) CreateSpot(
	_ context.Context,
	req *tenderservice.CreateSpotTenderRequest,
) (*tender.Tender, error) {
	f.spot = req

	return &tender.Tender{ID: pulid.MustNew("tnd_")}, nil
}

func TestTenderToRoutingGuide_StartsAWaterfallForTheMove(t *testing.T) {
	t.Parallel()

	tenders := &fakeTenders{}
	tool := newTenderToRoutingGuideTool(tenders)
	assert.Equal(t, permission.ResourceTender, tool.Policy().Resource)
	assert.Equal(t, permission.OpCreate, tool.Policy().Operation)

	moveID, guideID := pulid.MustNew("smv_"), pulid.MustNew("rg_")
	params := executeParams(
		map[string]any{"shipmentMoveId": moveID.String(), "routingGuideId": guideID.String()},
	)
	params.IdempotencyKey = "idem-1"
	require.NoError(t, tool.Execute(t.Context(), params))

	require.NotNil(t, tenders.waterfall)
	assert.Equal(t, moveID, tenders.waterfall.ShipmentMoveID)
	assert.Equal(t, guideID, *tenders.waterfall.RoutingGuideID)
	assert.Equal(t, params.OrganizationID, tenders.waterfall.TenantInfo.OrgID)

	target, ok := tool.(serviceports.TargetedTool).Target(params.Params)
	require.True(t, ok)
	assert.Equal(t, permission.ResourceShipmentMove, target.Resource)
}

func TestTenderToCarriers_BuildsEachLineAndRefusesJunk(t *testing.T) {
	t.Parallel()

	tenders := &fakeTenders{}
	tool := newTenderToCarriersTool(tenders)

	moveID, carrierA, carrierB := pulid.MustNew(
		"smv_",
	), pulid.MustNew(
		"carr_",
	), pulid.MustNew(
		"carr_",
	)
	params := executeParams(map[string]any{
		"shipmentMoveId": moveID.String(),
		"mode":           "SpotSequential",
		"lines": []any{
			map[string]any{
				"carrierId": carrierA.String(), "rate": "1450.00", "rateMethod": "Flat",
				"offerTtlSeconds": 1800, "channel": "Email",
			},
			map[string]any{"carrierId": carrierB.String(), "rate": "2.15", "rateMethod": "PerMile"},
		},
	})
	params.IdempotencyKey = "idem-1"
	require.NoError(t, tool.Execute(t.Context(), params))

	require.NotNil(t, tenders.spot)
	assert.Equal(t, tender.ModeSpotSequential, tenders.spot.Mode)
	require.Len(t, tenders.spot.Lines, 2)
	assert.Equal(t, carrierA, tenders.spot.Lines[0].CarrierID)
	assert.Equal(t, "1450", tenders.spot.Lines[0].Rate.String())
	assert.Equal(t, shipment.CarrierRateMethodFlat, tenders.spot.Lines[0].RateMethod)
	assert.EqualValues(t, 1800, tenders.spot.Lines[0].OfferTTLSeconds)
	assert.Equal(t, tender.ChannelEmail, tenders.spot.Lines[0].Channel)
	assert.Equal(t, shipment.CarrierRateMethodPerMile, tenders.spot.Lines[1].RateMethod)
	assert.False(t, tenders.spot.OverrideInsuranceWarnings)

	bad := executeParams(map[string]any{
		"shipmentMoveId": moveID.String(),
		"mode":           "SpotBroadcast",
		"lines":          []any{map[string]any{"carrierId": carrierA.String(), "rate": "a lot"}},
	})
	bad.IdempotencyKey = "idem-2"
	require.ErrorContains(t, tool.Execute(t.Context(), bad), "lines[0].rate")

	empty := executeParams(
		map[string]any{
			"shipmentMoveId": moveID.String(),
			"mode":           "SpotBroadcast",
			"lines":          []any{},
		},
	)
	empty.IdempotencyKey = "idem-3"
	require.ErrorContains(t, tool.Execute(t.Context(), empty), "between 1 and")
}
