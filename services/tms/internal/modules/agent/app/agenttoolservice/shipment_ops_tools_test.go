package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/movetools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/shipmenttools"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratequote"
	"github.com/emoss08/trenova/internal/modules/reference/domain/holdreason"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeShipmentOperator struct {
	guard      *agenttoolstest.WriteGuard
	refusal    error
	existing   *shipment.Shipment
	uncanceled *shipmentports.UncancelShipmentRequest
	owner      *shipmentports.TransferOwnershipRequest
	rerated    *shipmentports.AutoRateShipmentRequest
	unpriced   bool
	measured   pulid.ID
	duplicated *shipmentports.BulkDuplicateShipmentRequest
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
	req *shipmentports.UncancelShipmentRequest,
	_ *permissionports.RequestActor,
) (*shipment.Shipment, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.uncanceled = req

	return f.shipment(), nil
}

func (f *fakeShipmentOperator) PreviewUncancel(
	context.Context,
	*shipmentports.UncancelShipmentRequest,
) (*shipmentports.ShipmentChangePreview, error) {
	if f.refusal != nil {
		return nil, f.refusal
	}
	before := f.shipment()
	after := *before
	after.ApplyUncancel()

	return &shipmentports.ShipmentChangePreview{Before: before, After: &after}, nil
}

func (f *fakeShipmentOperator) TransferOwnership(
	_ context.Context,
	req *shipmentports.TransferOwnershipRequest,
	_ *permissionports.RequestActor,
) (*shipment.Shipment, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.owner = req

	return f.shipment(), nil
}

func (f *fakeShipmentOperator) PreviewTransferOwnership(
	_ context.Context,
	req *shipmentports.TransferOwnershipRequest,
	_ *permissionports.RequestActor,
) (*shipmentports.ShipmentChangePreview, error) {
	before := f.shipment()
	after := *before
	after.OwnerID = req.OwnerID

	return &shipmentports.ShipmentChangePreview{Before: before, After: &after}, nil
}

func (f *fakeShipmentOperator) AutoRate(
	_ context.Context,
	req *shipmentports.AutoRateShipmentRequest,
	_ *permissionports.RequestActor,
) (*shipment.Shipment, *shipmentports.ContractRateApplication, error) {
	if err := f.guard.Write(); err != nil {
		return nil, nil, err
	}
	f.rerated = req

	return f.shipment(), &shipmentports.ContractRateApplication{Applied: !f.unpriced}, nil
}

func (f *fakeShipmentOperator) PreviewAutoRate(
	context.Context,
	*shipmentports.AutoRateShipmentRequest,
	*permissionports.RequestActor,
) (*shipmentports.ShipmentAutoRatePreview, error) {
	before := f.shipment()
	if f.unpriced {
		return &shipmentports.ShipmentAutoRatePreview{
			Before: before,
			After:  before,
			Application: &shipmentports.ContractRateApplication{
				Outcome: ratequote.Outcome("NoMatch"),
			},
		}, nil
	}
	after := *before
	after.FreightChargeAmount = decimal.NewNullDecimal(decimal.RequireFromString("1250"))
	after.TotalChargeAmount = decimal.NewNullDecimal(decimal.RequireFromString("1250"))

	return &shipmentports.ShipmentAutoRatePreview{
		Before: before,
		After:  &after,
		Application: &shipmentports.ContractRateApplication{
			Applied:       true,
			AgreementName: "Acme 2026",
		},
	}, nil
}

func (f *fakeShipmentOperator) RecalculateDistance(
	_ context.Context,
	shipmentID pulid.ID,
	_ pagination.TenantInfo,
) (*shipmentports.DistanceCalculationResponse, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.measured = shipmentID

	return &shipmentports.DistanceCalculationResponse{TotalDistance: 412.5}, nil
}

func (f *fakeShipmentOperator) PreviewRecalculateDistance(
	context.Context,
	pulid.ID,
	pagination.TenantInfo,
) (*shipmentports.ShipmentDistancePreview, error) {
	oldMiles, newMiles := 300.0, 412.5
	moveID := pulid.MustNew("sm_")
	before := *f.shipment()
	before.Moves = []*shipment.ShipmentMove{{ID: moveID, Distance: &oldMiles, Version: 2}}
	after := before
	after.Moves = []*shipment.ShipmentMove{{ID: moveID, Distance: &newMiles, Version: 2}}

	return &shipmentports.ShipmentDistancePreview{
		Before:   &before,
		After:    &after,
		Distance: &shipmentports.DistanceCalculationResponse{TotalDistance: newMiles},
	}, nil
}

func (f *fakeShipmentOperator) Duplicate(
	_ context.Context,
	req *shipmentports.BulkDuplicateShipmentRequest,
) (*shipmentports.ShipmentDuplicateWorkflowResponse, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.duplicated = req

	return &shipmentports.ShipmentDuplicateWorkflowResponse{WorkflowID: "wf-1"}, nil
}

func (f *fakeShipmentOperator) PreviewDuplicate(
	_ context.Context,
	req *shipmentports.BulkDuplicateShipmentRequest,
) (*shipmentports.ShipmentDuplicatePreview, error) {
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

	return &shipmentports.ShipmentDuplicatePreview{Source: source, Copies: copies}, nil
}

func TestUncancelShipment_PreviewsTheReopenedShipmentAndWrites(t *testing.T) {
	t.Parallel()

	shipments := &fakeShipmentOperator{guard: &agenttoolstest.WriteGuard{}}
	tool := shipmenttools.NewUncancelShipmentTool(shipments)
	params := agenttoolstest.ExecuteParams(
		map[string]any{common.ParamShipmentID: shipments.shipment().ID.String()},
	)

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		shipments.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	change := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, "New", agenttoolstest.FieldByPath(t, change, common.FieldStatus).After)
	assert.Contains(t, preview.Summary, "PRO-7001")

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, shipments.uncanceled)
	assert.Equal(t, params.OrganizationID, shipments.uncanceled.TenantInfo.OrgID)

	policy := tool.Policy()
	assert.Equal(t, permission.ResourceShipment, policy.Resource)
	assert.Equal(t, permission.OpUpdate, policy.Operation)
	assert.Equal(t, agent.TierActWithApproval, policy.MaxTier)
	target, ok := tool.(agenttoolports.TargetedTool).Target(params.Params)
	require.True(t, ok)
	assert.Equal(t, permission.ResourceShipment, target.Resource)
}

func TestUncancelShipment_AShipmentThatIsNotCanceledIsAWouldFail(t *testing.T) {
	t.Parallel()

	shipments := &fakeShipmentOperator{
		refusal: errortypes.NewBusinessError("shipment is not canceled"),
	}
	tool := shipmenttools.NewUncancelShipmentTool(shipments)
	params := agenttoolstest.ExecuteParams(
		map[string]any{common.ParamShipmentID: pulid.MustNew("shp_").String()},
	)

	preview, err := tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	agenttoolstest.RequireWarning(t, preview, agent.PreviewWarningWouldFail)
	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(), params))
	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(),
		agenttoolstest.ExecuteParams(map[string]any{common.ParamShipmentID: "not-an-id"})))
}

func TestTransferShipmentOwnership_NamesTheNewOwner(t *testing.T) {
	t.Parallel()

	shipments := &fakeShipmentOperator{guard: &agenttoolstest.WriteGuard{}}
	tool := shipmenttools.NewTransferShipmentOwnershipTool(shipments)
	owner := pulid.MustNew("usr_")
	params := agenttoolstest.ExecuteParams(map[string]any{
		common.ParamShipmentID: shipments.shipment().ID.String(),
		common.ParamOwnerID:    owner.String(),
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		shipments.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	field := agenttoolstest.FieldByPath(
		t,
		agenttoolstest.PreviewChange(t, preview, 0),
		common.ParamOwnerID,
	)
	require.NotNil(t, field.AfterRef)
	assert.Equal(t, permission.ResourceUser, field.AfterRef.Resource)

	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, owner, shipments.owner.OwnerID)
	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(),
		agenttoolstest.ExecuteParams(map[string]any{common.ParamShipmentID: owner.String()})))
}

func TestRerateShipment_ShowsTheMoneyAndIsAMoneyWrite(t *testing.T) {
	t.Parallel()

	shipments := &fakeShipmentOperator{guard: &agenttoolstest.WriteGuard{}}
	tool := shipmenttools.NewRerateShipmentTool(shipments)
	params := agenttoolstest.ExecuteParams(
		map[string]any{common.ParamShipmentID: shipments.shipment().ID.String()},
	)

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		shipments.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Contains(t, preview.Summary, "Acme 2026")
	change := agenttoolstest.PreviewChange(t, preview, 0)
	require.NotNil(t, change.Money)

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, shipments.rerated)
	assert.Equal(t, []agent.EgressClass{agent.EgressMoney}, tool.Policy().Egress)
}

func TestRerateShipment_NoContractChangesNothing(t *testing.T) {
	t.Parallel()

	shipments := &fakeShipmentOperator{unpriced: true}
	tool := shipmenttools.NewRerateShipmentTool(shipments)
	params := agenttoolstest.ExecuteParams(
		map[string]any{common.ParamShipmentID: shipments.shipment().ID.String()},
	)

	preview, err := tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	assert.Empty(t, preview.Changes)
	assert.Contains(t, preview.Summary, "NoMatch")

	result, err := tool.(agenttoolports.ToolResultReporter).ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)
	assert.Contains(t, result.Action, "left unchanged")
}

func TestRecalculateShipmentDistance_ShowsEachMoveThatChanges(t *testing.T) {
	t.Parallel()

	shipments := &fakeShipmentOperator{guard: &agenttoolstest.WriteGuard{}}
	tool := shipmenttools.NewRecalculateShipmentDistanceTool(shipments)
	params := agenttoolstest.ExecuteParams(
		map[string]any{common.ParamShipmentID: shipments.shipment().ID.String()},
	)

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		shipments.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	change := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, permission.ResourceShipmentMove, change.Resource)
	assert.Contains(t, preview.Summary, "412.5")

	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, shipments.shipment().ID, shipments.measured)
	assert.Equal(t, agent.TierAutoExecute, tool.Policy().MaxTier)
}

func TestDuplicateShipment_CopiesServerSideWithTheNewDates(t *testing.T) {
	t.Parallel()

	shipments := &fakeShipmentOperator{guard: &agenttoolstest.WriteGuard{}}
	tool := shipmenttools.NewDuplicateShipmentTool(
		shipmenttools.DuplicateShipmentDeps{Shipments: shipments},
	)
	params := agenttoolstest.ExecuteParams(map[string]any{
		common.ParamShipmentID:           shipments.shipment().ID.String(),
		shipmenttools.ParamCopyCount:     float64(2),
		shipmenttools.ParamFirstPickupAt: "2030-03-17T12:46:40",
	})
	params.Timezone = "America/Chicago"

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		shipments.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	require.Len(t, preview.Changes, 2)
	created := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, agent.PreviewOperationCreate, created.Operation)
	assert.Equal(t, "BOL-COPY-01", agenttoolstest.FieldByPath(t, created, common.FieldBol).After)
	assert.Contains(t, preview.Summary, "2 new shipments")

	result, err := tool.(agenttoolports.ToolResultReporter).ExecuteWithResult(t.Context(), params)
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

	tool := shipmenttools.NewDuplicateShipmentTool(
		shipmenttools.DuplicateShipmentDeps{Shipments: &fakeShipmentOperator{}},
	)
	id := pulid.MustNew("shp_").String()
	for name, raw := range map[string]map[string]any{
		"too many copies":     {common.ParamShipmentID: id, shipmenttools.ParamCopyCount: float64(21)},
		"unix seconds":        {common.ParamShipmentID: id, shipmenttools.ParamFirstPickupAt: float64(1_900_000_000)},
		"unix seconds text":   {common.ParamShipmentID: id, shipmenttools.ParamFirstPickupAt: "1900000000"},
		"a utc offset":        {common.ParamShipmentID: id, shipmenttools.ParamFirstPickupAt: "2030-03-17T12:46:40Z"},
		"a date without time": {common.ParamShipmentID: id, shipmenttools.ParamFirstPickupAt: "2030-03-17"},
		"no shipment":         {shipmenttools.ParamCopyCount: float64(1)},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(),
				agenttoolstest.ExecuteParams(raw)))
		})
	}
}

type fakeShipmentSources struct {
	source *shipment.Shipment
	asked  *shipmentports.GetShipmentByIDRequest
}

func (f *fakeShipmentSources) Get(
	_ context.Context,
	req *shipmentports.GetShipmentByIDRequest,
) (*shipment.Shipment, error) {
	f.asked = req

	return f.source, nil
}

func TestDuplicateShipment_ReadsTheFirstPickupWhereItIs(t *testing.T) {
	t.Parallel()

	shipments := &fakeShipmentOperator{guard: &agenttoolstest.WriteGuard{}}
	pickup, delivery := pulid.MustNew("loc_"), pulid.MustNew("loc_")
	sources := &fakeShipmentSources{source: &shipment.Shipment{Moves: []*shipment.ShipmentMove{{
		Sequence: 0,
		Stops: []*shipment.Stop{
			{LocationID: delivery, Sequence: 1},
			{LocationID: pickup, Sequence: 0},
		},
	}}}}
	tool := shipmenttools.NewDuplicateShipmentTool(shipmenttools.DuplicateShipmentDeps{
		Shipments: shipments,
		Sources:   sources,
		Locations: agenttoolstest.FakeLocationZones{Zones: map[pulid.ID]string{
			pickup:   "America/Los_Angeles",
			delivery: "America/New_York",
		}},
	})
	params := agenttoolstest.ExecuteParams(map[string]any{
		common.ParamShipmentID:           shipments.shipment().ID.String(),
		shipmenttools.ParamFirstPickupAt: "2030-03-17T10:46:40",
	})
	params.Timezone = "America/Chicago"

	_, err := tool.(agenttoolports.ToolResultReporter).ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)

	require.NotNil(t, shipments.duplicated.FirstPickupAt)
	assert.Equal(t, int64(1_900_000_000), *shipments.duplicated.FirstPickupAt,
		"10:46:40 at a pickup in Los Angeles is 12:46:40 in Chicago")
	require.NotNil(t, sources.asked)
	assert.True(t, sources.asked.ExpandShipmentDetails)
}

type fakeHoldUpdater struct {
	guard   *agenttoolstest.WriteGuard
	current *shipment.ShipmentHold
	updated *shipmentports.UpdateShipmentHoldRequest
}

func (f *fakeHoldUpdater) GetByID(
	context.Context,
	*shipmentports.GetShipmentHoldByIDRequest,
) (*shipment.ShipmentHold, error) {
	copied := *f.current

	return &copied, nil
}

func (f *fakeHoldUpdater) apply(
	req *shipmentports.UpdateShipmentHoldRequest,
) *shipment.ShipmentHold {
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
	req *shipmentports.UpdateShipmentHoldRequest,
	_ *permissionports.RequestActor,
) (*shipment.ShipmentHold, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.updated = req

	return f.apply(req), nil
}

func (f *fakeHoldUpdater) PreviewUpdate(
	_ context.Context,
	req *shipmentports.UpdateShipmentHoldRequest,
	_ *permissionports.RequestActor,
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

	holds := &fakeHoldUpdater{guard: &agenttoolstest.WriteGuard{}, current: activeHold()}
	tool := shipmenttools.NewUpdateShipmentHoldTool(holds)
	params := agenttoolstest.ExecuteParams(map[string]any{
		common.ParamShipmentID:            holds.current.ShipmentID.String(),
		shipmenttools.ParamHoldID:         holds.current.ID.String(),
		shipmenttools.ParamBlocksBilling:  true,
		shipmenttools.ParamHoldSeverity:   "Blocking",
		common.ParamNote:                  " BOL received, billing waits on the POD ",
		shipmenttools.ParamBlocksDelivery: false,
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		holds.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	change := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(
		t,
		true,
		agenttoolstest.FieldByPath(t, change, shipmenttools.ParamBlocksBilling).After,
	)
	assert.Equal(t, "Blocking", agenttoolstest.FieldByPath(t, change, common.FieldSeverity).After)

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

	tool := shipmenttools.NewUpdateShipmentHoldTool(&fakeHoldUpdater{current: activeHold()})
	policy := tool.Policy()
	assert.Equal(t, permission.ResourceShipmentHold, policy.Resource)
	assert.Equal(
		t,
		agent.EgressCustomerVisible,
		policy.Classify(agenttoolstest.ExecuteParams(map[string]any{
			shipmenttools.ParamVisibleToCustomer: true,
		})).Egress,
	)
	assert.Equal(
		t,
		agent.EgressInternal,
		policy.Classify(agenttoolstest.ExecuteParams(map[string]any{
			shipmenttools.ParamBlocksBilling: true,
		})).Egress,
	)

	require.ErrorIs(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(),
		agenttoolstest.ExecuteParams(map[string]any{
			common.ParamShipmentID:    pulid.MustNew("shp_").String(),
			shipmenttools.ParamHoldID: pulid.MustNew("shh_").String(),
		})), common.ErrNothingToChange)
}

type fakeMoveSplitter struct {
	guard *agenttoolstest.WriteGuard
	split *shipmentports.SplitMoveRequest
}

func splitSourceMove() *shipment.ShipmentMove {
	return &shipment.ShipmentMove{
		ID:       pulid.MustNew("sm_"),
		Sequence: 0,
		Status:   shipment.MoveStatusAssigned,
		Version:  5,
		Stops: []*shipment.Stop{
			{
				ID:         pulid.MustNew("stp_"),
				Type:       shipment.StopTypePickup,
				LocationID: pulid.MustNew("loc_"),
			},
			{
				ID:         pulid.MustNew("stp_"),
				Type:       shipment.StopTypeDelivery,
				LocationID: pulid.MustNew("loc_"),
			},
		},
	}
}

func (f *fakeMoveSplitter) SplitMove(
	_ context.Context,
	req *shipmentports.SplitMoveRequest,
) (*shipmentports.SplitMoveResponse, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.split = req

	return &shipmentports.SplitMoveResponse{
		NewMove: &shipment.ShipmentMove{ID: pulid.MustNew("sm_")},
	}, nil
}

func (f *fakeMoveSplitter) PreviewSplitMove(
	_ context.Context,
	req *shipmentports.SplitMoveRequest,
) (*shipmentports.MoveSplitPlan, error) {
	move := splitSourceMove()
	move.ID = req.MoveID

	return &shipmentports.MoveSplitPlan{
		Move:  move,
		Split: shipment.PlanMoveSplit(move, req.Spec()),
	}, nil
}

func TestSplitMoveAtRelay_AlwaysProposesAndShowsTheNewMove(t *testing.T) {
	t.Parallel()

	moves := &fakeMoveSplitter{guard: &agenttoolstest.WriteGuard{}}
	tool := movetools.NewSplitMoveTool(moves)
	destination := pulid.MustNew("loc_")
	params := agenttoolstest.ExecuteParams(map[string]any{
		movetools.ParamMoveID:                pulid.MustNew("sm_").String(),
		movetools.ParamNewDeliveryLocationID: destination.String(),
		movetools.ParamRelayPickupStart:      "2030-03-17T17:46:40Z",
		movetools.ParamNewDeliveryStart:      "2030-03-18T07:40:00Z",
		common.FieldWeight:                   float64(42000),
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		moves.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	require.Len(t, preview.Changes, 2)
	assert.Equal(
		t,
		string(shipment.StopTypeSplitDelivery),
		agenttoolstest.FieldByPath(
			t,
			agenttoolstest.PreviewChange(t, preview, 0),
			common.FieldType,
		).After,
	)
	created := agenttoolstest.PreviewChange(t, preview, 1)
	assert.Equal(t, agent.PreviewOperationCreate, created.Operation)

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, moves.split)
	assert.Equal(t, destination, moves.split.NewDeliveryLocationID)
	require.NotNil(t, moves.split.Weight)
	assert.Equal(t, int64(1_900_000_000), moves.split.SplitPickupTimes.ScheduledWindowStart)
	assert.Equal(t, agent.TierPropose, tool.Policy().MaxTier)

	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(),
		agenttoolstest.ExecuteParams(map[string]any{
			movetools.ParamMoveID:                pulid.MustNew("sm_").String(),
			movetools.ParamNewDeliveryLocationID: destination.String(),
		})))
}
