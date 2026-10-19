package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/operationstools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/tendertools"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
		"customerId":        customerID.String(),
		"serviceTypeId":     serviceTypeID.String(),
		"shipmentTypeId":    pulid.MustNew("sht_").String(),
		"formulaTemplateId": pulid.MustNew("fmt_").String(),
		"bol":               "BOL-778",
		"pieces":            12,
		"weight":            18000,
		"moves": []any{map[string]any{
			"loaded": true,
			"stops": []any{
				map[string]any{
					"locationId":           originID.String(),
					"type":                 "Pickup",
					"sequence":             0,
					"scheduledWindowStart": "2026-10-01T08:00",
				},
				map[string]any{
					"locationId":           destID.String(),
					"type":                 "Delivery",
					"sequence":             1,
					"scheduledWindowStart": "2026-10-02T08:00",
				},
			},
		}},
	}
}

func TestCreateShipment_EntersTheShipmentUnderTheActorTenant(t *testing.T) {
	t.Parallel()

	writer := &agenttoolstest.FakeShipmentWriter{}
	imports := &fakeImportCompleter{}
	tool := operationstools.NewCreateShipmentTool(operationstools.CreateShipmentDeps{Shipments: writer, Imports: imports})
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

	params := agenttoolstest.ExecuteParams(map[string]any{"shipment": payload, "sourceDocumentId": docID.String()})
	params.IdempotencyKey = "idem-1"
	require.NoError(t, tool.Execute(t.Context(), params))

	require.NotNil(t, writer.Created)
	created := writer.Created
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
	assert.Same(t, params.Actor, writer.Actor)

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

	writer := &agenttoolstest.FakeShipmentWriter{}
	tool := operationstools.NewCreateShipmentTool(operationstools.CreateShipmentDeps{Shipments: writer})
	payload := shipmentPayload(
		pulid.MustNew("cust_"),
		pulid.MustNew("st_"),
		pulid.MustNew("loc_"),
		pulid.MustNew("loc_"),
	)

	params := agenttoolstest.ExecuteParams(map[string]any{"shipment": payload})
	require.ErrorIs(t, tool.Execute(t.Context(), params), common.ErrMissingIdempotencyKey)

	params.IdempotencyKey = "idem-2"
	params.Actor.BusinessUnitID = pulid.MustNew("bu_")
	require.ErrorIs(t, tool.Execute(t.Context(), params), common.ErrTenantMismatch)
	assert.Nil(t, writer.Created)

	params = agenttoolstest.ExecuteParams(map[string]any{"shipment": payload, "sourceDocumentId": "not-an-id"})
	params.IdempotencyKey = "idem-3"
	require.Error(t, tool.Execute(t.Context(), params))
	assert.Nil(t, writer.Created)
}

// The create plan is what decides the write; Validate runs it, so a BOL the
// organization already has, or anything else the service refuses, is
// refused to the model before a person is asked to approve it.
func TestCreateShipment_ValidateRefusesWhatTheServiceWould(t *testing.T) {
	t.Parallel()

	writer := &agenttoolstest.FakeShipmentWriter{PreviewErr: errortypes.NewValidationError(
		"bol", errortypes.ErrDuplicate, "A shipment with this BOL already exists",
	)}
	tool := operationstools.NewCreateShipmentTool(operationstools.CreateShipmentDeps{
		Shipments: writer,
		Imports:   &fakeImportCompleter{},
	})
	validator, ok := tool.(agentports.ToolValidator)
	require.True(t, ok, "shipment_create checks a call before it is filed")

	payload := shipmentPayload(
		pulid.MustNew("cust_"), pulid.MustNew("st_"), pulid.MustNew("loc_"), pulid.MustNew("loc_"),
	)
	params := agenttoolstest.ExecuteParams(map[string]any{"shipment": payload})

	err := validator.Validate(t.Context(), params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "BOL already exists")
	assert.Equal(t, 1, writer.Previews, "the plan ran once")
	assert.Nil(t, writer.Created, "nothing was written")

	writer.PreviewErr = nil
	require.NoError(t, validator.Validate(t.Context(), params))
}

func TestUpdateShipment_PatchesOnlyTheNamedFields(t *testing.T) {
	t.Parallel()

	shipmentID := pulid.MustNew("shp_")
	original := &shipment.Shipment{
		ID: shipmentID, CustomerID: pulid.MustNew("cust_"), ServiceTypeID: pulid.MustNew("st_"),
		BOL: "OLD", Version: 3,
	}
	writer := &agenttoolstest.FakeShipmentWriter{Existing: original}
	tool := operationstools.NewUpdateShipmentTool(writer, nil)
	assert.Equal(t, permission.OpUpdate, tool.Policy().Operation)

	newCustomer := pulid.MustNew("cust_")
	params := agenttoolstest.ExecuteParams(map[string]any{
		"shipmentId": shipmentID.String(),
		"customerId": newCustomer.String(),
		"weight":     22000,
	})
	require.NoError(t, tool.Execute(t.Context(), params))

	require.NotNil(t, writer.Updated)
	assert.Equal(t, newCustomer, writer.Updated.CustomerID)
	assert.EqualValues(t, 22000, *writer.Updated.Weight)
	assert.Equal(t, "OLD", writer.Updated.BOL, "a field not named is left alone")
	assert.Equal(t, original.ServiceTypeID, writer.Updated.ServiceTypeID)
	assert.True(
		t,
		writer.LastGet.ExpandShipmentDetails,
		"the update carries the moves it read, so none are lost",
	)
	assert.Equal(t, params.OrganizationID, writer.LastGet.TenantInfo.OrgID)

	target, ok := tool.(agentports.TargetedTool).Target(params.Params)
	require.True(t, ok)
	assert.Equal(t, shipmentID, target.ID)
	assert.Equal(t, permission.ResourceShipment, target.Resource)
}

func TestUpdateShipment_RefusesAnEmptyPatchAndABadID(t *testing.T) {
	t.Parallel()

	writer := &agenttoolstest.FakeShipmentWriter{Existing: &shipment.Shipment{}}
	tool := operationstools.NewUpdateShipmentTool(writer, nil)

	err := tool.Execute(
		t.Context(),
		agenttoolstest.ExecuteParams(map[string]any{"shipmentId": pulid.MustNew("shp_").String()}),
	)
	require.ErrorContains(t, err, "Nothing to change")
	assert.Nil(t, writer.Updated)

	err = tool.Execute(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		"shipmentId": pulid.MustNew("shp_").String(),
		"customerId": "acme",
	}))
	require.ErrorContains(t, err, `"customerId" is not an id`)
	assert.Nil(t, writer.Updated)
}

func TestTenderToRoutingGuide_StartsAWaterfallForTheMove(t *testing.T) {
	t.Parallel()

	tenders := &agenttoolstest.FakeTenders{}
	tool := tendertools.NewTenderToRoutingGuideTool(tenders)
	assert.Equal(t, permission.ResourceTender, tool.Policy().Resource)
	assert.Equal(t, permission.OpCreate, tool.Policy().Operation)

	moveID, guideID := pulid.MustNew("smv_"), pulid.MustNew("rg_")
	params := agenttoolstest.ExecuteParams(
		map[string]any{"shipmentMoveId": moveID.String(), "routingGuideId": guideID.String()},
	)
	params.IdempotencyKey = "idem-1"
	require.NoError(t, tool.Execute(t.Context(), params))

	require.NotNil(t, tenders.Waterfall)
	assert.Equal(t, moveID, tenders.Waterfall.ShipmentMoveID)
	assert.Equal(t, guideID, *tenders.Waterfall.RoutingGuideID)
	assert.Equal(t, params.OrganizationID, tenders.Waterfall.TenantInfo.OrgID)

	target, ok := tool.(agentports.TargetedTool).Target(params.Params)
	require.True(t, ok)
	assert.Equal(t, permission.ResourceShipmentMove, target.Resource)
}

func TestTenderToCarriers_BuildsEachLineAndRefusesJunk(t *testing.T) {
	t.Parallel()

	tenders := &agenttoolstest.FakeTenders{}
	tool := tendertools.NewTenderToCarriersTool(tenders)

	moveID, carrierA, carrierB := pulid.MustNew(
		"smv_",
	), pulid.MustNew(
		"carr_",
	), pulid.MustNew(
		"carr_",
	)
	params := agenttoolstest.ExecuteParams(map[string]any{
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

	require.NotNil(t, tenders.Spot)
	assert.Equal(t, tender.ModeSpotSequential, tenders.Spot.Mode)
	require.Len(t, tenders.Spot.Lines, 2)
	assert.Equal(t, carrierA, tenders.Spot.Lines[0].CarrierID)
	assert.Equal(t, "1450", tenders.Spot.Lines[0].Rate.String())
	assert.Equal(t, shipment.CarrierRateMethodFlat, tenders.Spot.Lines[0].RateMethod)
	assert.EqualValues(t, 1800, tenders.Spot.Lines[0].OfferTTLSeconds)
	assert.Equal(t, tender.ChannelEmail, tenders.Spot.Lines[0].Channel)
	assert.Equal(t, shipment.CarrierRateMethodPerMile, tenders.Spot.Lines[1].RateMethod)
	assert.False(t, tenders.Spot.OverrideInsuranceWarnings)

	bad := agenttoolstest.ExecuteParams(map[string]any{
		"shipmentMoveId": moveID.String(),
		"mode":           "SpotBroadcast",
		"lines":          []any{map[string]any{"carrierId": carrierA.String(), "rate": "a lot"}},
	})
	bad.IdempotencyKey = "idem-2"
	require.ErrorContains(t, tool.Execute(t.Context(), bad), "lines[0].rate")

	empty := agenttoolstest.ExecuteParams(
		map[string]any{
			"shipmentMoveId": moveID.String(),
			"mode":           "SpotBroadcast",
			"lines":          []any{},
		},
	)
	empty.IdempotencyKey = "idem-3"
	require.ErrorContains(t, tool.Execute(t.Context(), empty), "between 1 and")
}
