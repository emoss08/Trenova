package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/holdreason"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/ratequote"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeShipmentOperator struct {
	guard      *writeGuard
	refusal    error
	existing   *shipment.Shipment
	uncanceled *repositories.UncancelShipmentRequest
	owner      *repositories.TransferOwnershipRequest
	rerated    *serviceports.AutoRateShipmentRequest
	unpriced   bool
	measured   pulid.ID
	duplicated *repositories.BulkDuplicateShipmentRequest
}

func (f *fakeShipmentOperator) shipment() *shipment.Shipment {
	if f.existing == nil {
		f.existing = &shipment.Shipment{
			ID:                  pulid.MustNew("shp_"),
			ProNumber:           "PRO-7001",
			Status:              shipment.StatusCanceled,
			CancelReason:        "Customer pulled the load",
			OwnerID:             pulid.MustNew("usr_"),
			CustomerID:          pulid.MustNew("cus_"),
			FreightChargeAmount: decimal.NewNullDecimal(decimal.RequireFromString("1000")),
			TotalChargeAmount:   decimal.NewNullDecimal(decimal.RequireFromString("1000")),
			Version:             4,
		}
	}

	return f.existing
}

func (f *fakeShipmentOperator) Uncancel(
	_ context.Context,
	req *repositories.UncancelShipmentRequest,
	_ *serviceports.RequestActor,
) (*shipment.Shipment, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.uncanceled = req

	return f.shipment(), nil
}

func (f *fakeShipmentOperator) PreviewUncancel(
	context.Context,
	*repositories.UncancelShipmentRequest,
) (*serviceports.ShipmentChangePreview, error) {
	if f.refusal != nil {
		return nil, f.refusal
	}
	before := f.shipment()
	after := *before
	after.ApplyUncancel()

	return &serviceports.ShipmentChangePreview{Before: before, After: &after}, nil
}

func (f *fakeShipmentOperator) TransferOwnership(
	_ context.Context,
	req *repositories.TransferOwnershipRequest,
	_ *serviceports.RequestActor,
) (*shipment.Shipment, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.owner = req

	return f.shipment(), nil
}

func (f *fakeShipmentOperator) PreviewTransferOwnership(
	_ context.Context,
	req *repositories.TransferOwnershipRequest,
	_ *serviceports.RequestActor,
) (*serviceports.ShipmentChangePreview, error) {
	before := f.shipment()
	after := *before
	after.OwnerID = req.OwnerID

	return &serviceports.ShipmentChangePreview{Before: before, After: &after}, nil
}

func (f *fakeShipmentOperator) AutoRate(
	_ context.Context,
	req *serviceports.AutoRateShipmentRequest,
	_ *serviceports.RequestActor,
) (*shipment.Shipment, *serviceports.ContractRateApplication, error) {
	if err := f.guard.write(); err != nil {
		return nil, nil, err
	}
	f.rerated = req

	return f.shipment(), &serviceports.ContractRateApplication{Applied: !f.unpriced}, nil
}

func (f *fakeShipmentOperator) PreviewAutoRate(
	context.Context,
	*serviceports.AutoRateShipmentRequest,
	*serviceports.RequestActor,
) (*serviceports.ShipmentAutoRatePreview, error) {
	before := f.shipment()
	if f.unpriced {
		return &serviceports.ShipmentAutoRatePreview{
			Before:      before,
			After:       before,
			Application: &serviceports.ContractRateApplication{Outcome: ratequote.Outcome("NoMatch")},
		}, nil
	}
	after := *before
	after.FreightChargeAmount = decimal.NewNullDecimal(decimal.RequireFromString("1250"))
	after.TotalChargeAmount = decimal.NewNullDecimal(decimal.RequireFromString("1250"))

	return &serviceports.ShipmentAutoRatePreview{
		Before:      before,
		After:       &after,
		Application: &serviceports.ContractRateApplication{Applied: true, AgreementName: "Acme 2026"},
	}, nil
}

func (f *fakeShipmentOperator) RecalculateDistance(
	_ context.Context,
	shipmentID pulid.ID,
	_ pagination.TenantInfo,
) (*serviceports.DistanceCalculationResponse, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.measured = shipmentID

	return &serviceports.DistanceCalculationResponse{TotalDistance: 412.5}, nil
}

func (f *fakeShipmentOperator) PreviewRecalculateDistance(
	context.Context,
	pulid.ID,
	pagination.TenantInfo,
) (*serviceports.ShipmentDistancePreview, error) {
	oldMiles, newMiles := 300.0, 412.5
	moveID := pulid.MustNew("sm_")
	before := *f.shipment()
	before.Moves = []*shipment.ShipmentMove{{ID: moveID, Distance: &oldMiles, Version: 2}}
	after := before
	after.Moves = []*shipment.ShipmentMove{{ID: moveID, Distance: &newMiles, Version: 2}}

	return &serviceports.ShipmentDistancePreview{
		Before:   &before,
		After:    &after,
		Distance: &serviceports.DistanceCalculationResponse{TotalDistance: newMiles},
	}, nil
}

func (f *fakeShipmentOperator) Duplicate(
	_ context.Context,
	req *repositories.BulkDuplicateShipmentRequest,
) (*repositories.ShipmentDuplicateWorkflowResponse, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.duplicated = req

	return &repositories.ShipmentDuplicateWorkflowResponse{WorkflowID: "wf-1"}, nil
}

func (f *fakeShipmentOperator) PreviewDuplicate(
	_ context.Context,
	req *repositories.BulkDuplicateShipmentRequest,
) (*serviceports.ShipmentDuplicatePreview, error) {
	source := f.shipment()
	copies := make([]*shipment.Shipment, 0, req.Count)
	for range req.Count {
		copied := &shipment.Shipment{
			CustomerID: source.CustomerID,
			BOL:        "BOL-COPY-01",
			Moves: []*shipment.ShipmentMove{{Stops: []*shipment.Stop{
				{ScheduledWindowStart: 1_900_000_000},
				{ScheduledWindowStart: 1_900_050_000},
			}}},
		}
		copies = append(copies, copied)
	}

	return &serviceports.ShipmentDuplicatePreview{Source: source, Copies: copies}, nil
}

func TestUncancelShipment_PreviewsTheReopenedShipmentAndWrites(t *testing.T) {
	t.Parallel()

	shipments := &fakeShipmentOperator{guard: &writeGuard{}}
	tool := newUncancelShipmentTool(shipments)
	params := executeParams(map[string]any{paramShipmentID: shipments.shipment().ID.String()})

	preview := previewWithoutWrites(t, shipments.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	change := previewChange(t, preview, 0)
	assert.Equal(t, "New", fieldByPath(t, change, fieldStatus).After)
	assert.Contains(t, preview.Summary, "PRO-7001")

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, shipments.uncanceled)
	assert.Equal(t, params.OrganizationID, shipments.uncanceled.TenantInfo.OrgID)

	policy := tool.Policy()
	assert.Equal(t, permission.ResourceShipment, policy.Resource)
	assert.Equal(t, permission.OpUpdate, policy.Operation)
	assert.Equal(t, agent.TierActWithApproval, policy.MaxTier)
	target, ok := tool.(serviceports.TargetedTool).Target(params.Params)
	require.True(t, ok)
	assert.Equal(t, permission.ResourceShipment, target.Resource)
}

func TestUncancelShipment_AShipmentThatIsNotCanceledIsAWouldFail(t *testing.T) {
	t.Parallel()

	shipments := &fakeShipmentOperator{refusal: errortypes.NewBusinessError("shipment is not canceled")}
	tool := newUncancelShipmentTool(shipments)
	params := executeParams(map[string]any{paramShipmentID: pulid.MustNew("shp_").String()})

	preview, err := tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	requireWarning(t, preview, agent.PreviewWarningWouldFail)
	require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(), params))
	require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(),
		executeParams(map[string]any{paramShipmentID: "not-an-id"})))
}

func TestTransferShipmentOwnership_NamesTheNewOwner(t *testing.T) {
	t.Parallel()

	shipments := &fakeShipmentOperator{guard: &writeGuard{}}
	tool := newTransferShipmentOwnershipTool(shipments)
	owner := pulid.MustNew("usr_")
	params := executeParams(map[string]any{
		paramShipmentID: shipments.shipment().ID.String(),
		paramOwnerID:    owner.String(),
	})

	preview := previewWithoutWrites(t, shipments.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	field := fieldByPath(t, previewChange(t, preview, 0), paramOwnerID)
	require.NotNil(t, field.AfterRef)
	assert.Equal(t, permission.ResourceUser, field.AfterRef.Resource)

	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, owner, shipments.owner.OwnerID)
	require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(),
		executeParams(map[string]any{paramShipmentID: owner.String()})))
}

func TestRerateShipment_ShowsTheMoneyAndIsAMoneyWrite(t *testing.T) {
	t.Parallel()

	shipments := &fakeShipmentOperator{guard: &writeGuard{}}
	tool := newRerateShipmentTool(shipments)
	params := executeParams(map[string]any{paramShipmentID: shipments.shipment().ID.String()})

	preview := previewWithoutWrites(t, shipments.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "Acme 2026")
	change := previewChange(t, preview, 0)
	require.NotNil(t, change.Money)

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, shipments.rerated)
	assert.Equal(t, []agent.EgressClass{agent.EgressMoney}, tool.Policy().Egress)
}

func TestRerateShipment_NoContractChangesNothing(t *testing.T) {
	t.Parallel()

	shipments := &fakeShipmentOperator{unpriced: true}
	tool := newRerateShipmentTool(shipments)
	params := executeParams(map[string]any{paramShipmentID: shipments.shipment().ID.String()})

	preview, err := tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	assert.Empty(t, preview.Changes)
	assert.Contains(t, preview.Summary, "NoMatch")

	result, err := tool.(serviceports.ToolResultReporter).ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)
	assert.Contains(t, result.Action, "left unchanged")
}

func TestRecalculateShipmentDistance_ShowsEachMoveThatChanges(t *testing.T) {
	t.Parallel()

	shipments := &fakeShipmentOperator{guard: &writeGuard{}}
	tool := newRecalculateShipmentDistanceTool(shipments)
	params := executeParams(map[string]any{paramShipmentID: shipments.shipment().ID.String()})

	preview := previewWithoutWrites(t, shipments.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	change := previewChange(t, preview, 0)
	assert.Equal(t, permission.ResourceShipmentMove, change.Resource)
	assert.Contains(t, preview.Summary, "412.5")

	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, shipments.shipment().ID, shipments.measured)
	assert.Equal(t, agent.TierAutoExecute, tool.Policy().MaxTier)
}

func TestDuplicateShipment_CopiesServerSideWithTheNewDates(t *testing.T) {
	t.Parallel()

	shipments := &fakeShipmentOperator{guard: &writeGuard{}}
	tool := newDuplicateShipmentTool(duplicateShipmentDeps{Shipments: shipments})
	params := executeParams(map[string]any{
		paramShipmentID:    shipments.shipment().ID.String(),
		paramCopyCount:     float64(2),
		paramFirstPickupAt: "2030-03-17T12:46:40",
	})
	params.Timezone = "America/Chicago"

	preview := previewWithoutWrites(t, shipments.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	require.Len(t, preview.Changes, 2)
	created := previewChange(t, preview, 0)
	assert.Equal(t, agent.PreviewOperationCreate, created.Operation)
	assert.Equal(t, "BOL-COPY-01", fieldByPath(t, created, fieldBol).After)
	assert.Contains(t, preview.Summary, "2 new shipments")

	result, err := tool.(serviceports.ToolResultReporter).ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)
	require.NotNil(t, shipments.duplicated)
	assert.Equal(t, 2, shipments.duplicated.Count)
	require.NotNil(t, shipments.duplicated.FirstPickupAt)
	assert.Equal(t, int64(1_900_000_000), *shipments.duplicated.FirstPickupAt)
	assert.Equal(t, "wf-1", result.IDs["workflowId"])
	assert.Equal(t, permission.OpDuplicate, tool.Policy().Operation)
}

func TestDuplicateShipment_RefusesBadArguments(t *testing.T) {
	t.Parallel()

	tool := newDuplicateShipmentTool(duplicateShipmentDeps{Shipments: &fakeShipmentOperator{}})
	id := pulid.MustNew("shp_").String()
	for name, raw := range map[string]map[string]any{
		"too many copies":     {paramShipmentID: id, paramCopyCount: float64(21)},
		"unix seconds":        {paramShipmentID: id, paramFirstPickupAt: float64(1_900_000_000)},
		"unix seconds text":   {paramShipmentID: id, paramFirstPickupAt: "1900000000"},
		"a utc offset":        {paramShipmentID: id, paramFirstPickupAt: "2030-03-17T12:46:40Z"},
		"a date without time": {paramShipmentID: id, paramFirstPickupAt: "2030-03-17"},
		"no shipment":         {paramCopyCount: float64(1)},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(),
				executeParams(raw)))
		})
	}
}

type fakeShipmentSources struct {
	source *shipment.Shipment
	asked  *repositories.GetShipmentByIDRequest
}

func (f *fakeShipmentSources) Get(
	_ context.Context,
	req *repositories.GetShipmentByIDRequest,
) (*shipment.Shipment, error) {
	f.asked = req

	return f.source, nil
}

type fakeLocationZones struct {
	zones map[pulid.ID]string
}

func (f fakeLocationZones) GetByIDs(
	_ context.Context,
	req repositories.GetLocationsByIDsRequest,
) ([]*location.Location, error) {
	found := make([]*location.Location, 0, len(req.LocationIDs))
	for _, id := range req.LocationIDs {
		if zone, ok := f.zones[id]; ok {
			found = append(found, &location.Location{ID: id, Timezone: zone})
		}
	}

	return found, nil
}

func TestDuplicateShipment_ReadsTheFirstPickupWhereItIs(t *testing.T) {
	t.Parallel()

	shipments := &fakeShipmentOperator{guard: &writeGuard{}}
	pickup, delivery := pulid.MustNew("loc_"), pulid.MustNew("loc_")
	sources := &fakeShipmentSources{source: &shipment.Shipment{Moves: []*shipment.ShipmentMove{{
		Sequence: 0,
		Stops: []*shipment.Stop{
			{LocationID: delivery, Sequence: 1},
			{LocationID: pickup, Sequence: 0},
		},
	}}}}
	tool := newDuplicateShipmentTool(duplicateShipmentDeps{
		Shipments: shipments,
		Sources:   sources,
		Locations: fakeLocationZones{zones: map[pulid.ID]string{
			pickup:   "America/Los_Angeles",
			delivery: "America/New_York",
		}},
	})
	params := executeParams(map[string]any{
		paramShipmentID:    shipments.shipment().ID.String(),
		paramFirstPickupAt: "2030-03-17T10:46:40",
	})
	params.Timezone = "America/Chicago"

	_, err := tool.(serviceports.ToolResultReporter).ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)

	require.NotNil(t, shipments.duplicated.FirstPickupAt)
	assert.Equal(t, int64(1_900_000_000), *shipments.duplicated.FirstPickupAt,
		"10:46:40 at a pickup in Los Angeles is 12:46:40 in Chicago")
	require.NotNil(t, sources.asked)
	assert.True(t, sources.asked.ExpandShipmentDetails)
}

type fakeHoldUpdater struct {
	guard   *writeGuard
	current *shipment.ShipmentHold
	updated *repositories.UpdateShipmentHoldRequest
}

func (f *fakeHoldUpdater) GetByID(
	context.Context,
	*repositories.GetShipmentHoldByIDRequest,
) (*shipment.ShipmentHold, error) {
	copied := *f.current

	return &copied, nil
}

func (f *fakeHoldUpdater) apply(req *repositories.UpdateShipmentHoldRequest) *shipment.ShipmentHold {
	next := *f.current
	next.Severity = req.Severity
	next.Notes = req.Notes
	next.BlocksDispatch = req.BlocksDispatch
	next.BlocksDelivery = req.BlocksDelivery
	next.BlocksBilling = req.BlocksBilling
	next.VisibleToCustomer = req.VisibleToCustomer

	return &next
}

func (f *fakeHoldUpdater) Update(
	_ context.Context,
	req *repositories.UpdateShipmentHoldRequest,
	_ *serviceports.RequestActor,
) (*shipment.ShipmentHold, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.updated = req

	return f.apply(req), nil
}

func (f *fakeHoldUpdater) PreviewUpdate(
	_ context.Context,
	req *repositories.UpdateShipmentHoldRequest,
	_ *serviceports.RequestActor,
) (*shipment.ShipmentHold, error) {
	return f.apply(req), nil
}

func activeHold() *shipment.ShipmentHold {
	return &shipment.ShipmentHold{
		ID:             pulid.MustNew("shh_"),
		ShipmentID:     pulid.MustNew("shp_"),
		Type:           holdreason.HoldType("Compliance"),
		Severity:       holdreason.HoldSeverityAdvisory,
		ReasonCode:     "DOCS",
		Notes:          "Waiting on the BOL",
		BlocksDispatch: true,
		StartedAt:      1_800_000_000,
		Version:        3,
	}
}

func TestUpdateShipmentHold_ChangesOnlyWhatIsSentAndKeepsTheVersion(t *testing.T) {
	t.Parallel()

	holds := &fakeHoldUpdater{guard: &writeGuard{}, current: activeHold()}
	tool := newUpdateShipmentHoldTool(holds)
	params := executeParams(map[string]any{
		paramShipmentID:     holds.current.ShipmentID.String(),
		paramHoldID:         holds.current.ID.String(),
		paramBlocksBilling:  true,
		paramHoldSeverity:   "Blocking",
		fieldNotes:          " BOL received, billing waits on the POD ",
		paramBlocksDelivery: false,
	})

	preview := previewWithoutWrites(t, holds.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	change := previewChange(t, preview, 0)
	assert.Equal(t, true, fieldByPath(t, change, paramBlocksBilling).After)
	assert.Equal(t, "Blocking", fieldByPath(t, change, fieldSeverity).After)

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, holds.updated)
	assert.True(t, holds.updated.BlocksDispatch, "an unsent flag keeps its value")
	assert.True(t, holds.updated.BlocksBilling)
	assert.Equal(t, "BOL received, billing waits on the POD", holds.updated.Notes)
	assert.Equal(t, int64(3), holds.updated.Version)
	assert.Equal(t, holds.current.StartedAt, holds.updated.StartedAt)
}

func TestUpdateShipmentHold_ShowingItToTheCustomerIsCustomerVisible(t *testing.T) {
	t.Parallel()

	tool := newUpdateShipmentHoldTool(&fakeHoldUpdater{current: activeHold()})
	policy := tool.Policy()
	assert.Equal(t, permission.ResourceShipmentHold, policy.Resource)
	assert.Equal(t, agent.EgressCustomerVisible, policy.Classify(executeParams(map[string]any{
		paramVisibleToCustomer: true,
	})).Egress)
	assert.Equal(t, agent.EgressInternal, policy.Classify(executeParams(map[string]any{
		paramBlocksBilling: true,
	})).Egress)

	require.ErrorIs(t, tool.(serviceports.ToolValidator).Validate(t.Context(),
		executeParams(map[string]any{
			paramShipmentID: pulid.MustNew("shp_").String(),
			paramHoldID:     pulid.MustNew("shh_").String(),
		})), errNothingToChange)
}

type fakeMoveSplitter struct {
	guard *writeGuard
	split *repositories.SplitMoveRequest
}

func splitSourceMove() *shipment.ShipmentMove {
	return &shipment.ShipmentMove{
		ID:       pulid.MustNew("sm_"),
		Sequence: 0,
		Status:   shipment.MoveStatusAssigned,
		Version:  5,
		Stops: []*shipment.Stop{
			{ID: pulid.MustNew("stp_"), Type: shipment.StopTypePickup, LocationID: pulid.MustNew("loc_")},
			{ID: pulid.MustNew("stp_"), Type: shipment.StopTypeDelivery, LocationID: pulid.MustNew("loc_")},
		},
	}
}

func (f *fakeMoveSplitter) SplitMove(
	_ context.Context,
	req *repositories.SplitMoveRequest,
) (*repositories.SplitMoveResponse, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.split = req

	return &repositories.SplitMoveResponse{NewMove: &shipment.ShipmentMove{ID: pulid.MustNew("sm_")}}, nil
}

func (f *fakeMoveSplitter) PreviewSplitMove(
	_ context.Context,
	req *repositories.SplitMoveRequest,
) (*serviceports.MoveSplitPlan, error) {
	move := splitSourceMove()
	move.ID = req.MoveID

	return &serviceports.MoveSplitPlan{Move: move, Split: shipment.PlanMoveSplit(move, req.Spec())}, nil
}

func TestSplitMoveAtRelay_AlwaysProposesAndShowsTheNewMove(t *testing.T) {
	t.Parallel()

	moves := &fakeMoveSplitter{guard: &writeGuard{}}
	tool := newSplitMoveTool(moves)
	destination := pulid.MustNew("loc_")
	params := executeParams(map[string]any{
		paramMoveID:                pulid.MustNew("sm_").String(),
		paramNewDeliveryLocationID: destination.String(),
		paramRelayPickupStart:      "2030-03-17T17:46:40Z",
		paramNewDeliveryStart:      "2030-03-18T07:40:00Z",
		fieldWeight:                float64(42000),
	})

	preview := previewWithoutWrites(t, moves.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	require.Len(t, preview.Changes, 2)
	assert.Equal(t, string(shipment.StopTypeSplitDelivery),
		fieldByPath(t, previewChange(t, preview, 0), fieldType).After)
	created := previewChange(t, preview, 1)
	assert.Equal(t, agent.PreviewOperationCreate, created.Operation)

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, moves.split)
	assert.Equal(t, destination, moves.split.NewDeliveryLocationID)
	require.NotNil(t, moves.split.Weight)
	assert.Equal(t, int64(1_900_000_000), moves.split.SplitPickupTimes.ScheduledWindowStart)
	assert.Equal(t, agent.TierPropose, tool.Policy().MaxTier)

	require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(),
		executeParams(map[string]any{
			paramMoveID:                pulid.MustNew("sm_").String(),
			paramNewDeliveryLocationID: destination.String(),
		})))
}
