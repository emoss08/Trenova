package dispatchtools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentcoverage"
	"github.com/emoss08/trenova/internal/modules/operations/ports/assignmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentmoveports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeUnassigner answers from a set of moves it holds on assignments; a move
// it does not hold is refused as the service refuses one no longer fresh.
type fakeUnassigner struct {
	guard     agenttoolstest.WriteGuard
	assigned  map[pulid.ID]*shipment.Shipment
	unassigns []*assignmentports.UnassignShipmentMoveRequest
}

func newFakeUnassigner(moves ...*shipment.Shipment) *fakeUnassigner {
	f := &fakeUnassigner{assigned: map[pulid.ID]*shipment.Shipment{}}
	for _, entity := range moves {
		f.assigned[entity.Moves[0].ID] = entity
	}

	return f
}

func (f *fakeUnassigner) plan(
	req *assignmentports.UnassignShipmentMoveRequest,
) (*assignmentports.AssignmentPlan, error) {
	entity, ok := f.assigned[req.ShipmentMoveID]
	if !ok {
		return nil, errortypes.NewBusinessError(
			"Only fresh assigned shipment moves can be unassigned",
		)
	}
	after := shipmentcoverage.CloneForUpdate(entity)
	after.Status = shipment.StatusNew
	after.Moves[0].Status = shipment.MoveStatusNew
	after.Moves[0].CoverageType = shipment.MoveCoverageTypeUnassigned
	after.Moves[0].Assignment = nil

	return &assignmentports.AssignmentPlan{
		Assignment:     entity.Moves[0].Assignment,
		ShipmentBefore: entity,
		ShipmentAfter:  after,
	}, nil
}

func (f *fakeUnassigner) PreviewUnassign(
	_ context.Context,
	req *assignmentports.UnassignShipmentMoveRequest,
) (*assignmentports.AssignmentPlan, error) {
	return f.plan(req)
}

func (f *fakeUnassigner) Unassign(
	_ context.Context,
	req *assignmentports.UnassignShipmentMoveRequest,
) error {
	if err := f.guard.Write(); err != nil {
		return err
	}
	f.unassigns = append(f.unassigns, req)
	if _, err := f.plan(req); err != nil {
		return err
	}
	delete(f.assigned, req.ShipmentMoveID)

	return nil
}

func driverCoveredShipment(pro string) *shipment.Shipment {
	moveID := pulid.MustNew("smv_")
	workerID := pulid.MustNew("wrk_")
	tractorID := pulid.MustNew("tr_")

	return &shipment.Shipment{
		ID:        pulid.MustNew("shp_"),
		ProNumber: pro,
		Status:    shipment.StatusAssigned,
		Version:   7,
		Moves: []*shipment.ShipmentMove{{
			ID:           moveID,
			Status:       shipment.MoveStatusAssigned,
			CoverageType: shipment.MoveCoverageTypeDriver,
			Version:      3,
			Assignment: &shipment.Assignment{
				ID:              pulid.MustNew("asn_"),
				ShipmentMoveID:  moveID,
				PrimaryWorkerID: &workerID,
				TractorID:       &tractorID,
				Status:          shipment.AssignmentStatusNew,
			},
		}},
	}
}

func moveIDsParam(ids ...pulid.ID) []any {
	out := make([]any, 0, len(ids))
	for _, id := range ids {
		out = append(out, id.String())
	}

	return out
}

func TestUnassignMoves_TakesEachMoveOffItsDriverInTheActorsTenant(t *testing.T) {
	t.Parallel()

	first, second := driverCoveredShipment("S-1001"), driverCoveredShipment("S-1002")
	assignments := newFakeUnassigner(first, second)
	params := agenttoolstest.ExecuteParams(map[string]any{
		fieldMoveIDs: moveIDsParam(first.Moves[0].ID, second.Moves[0].ID),
	})

	require.NoError(t, (&UnassignMovesTool{assignments: assignments}).Execute(t.Context(), params))

	require.Len(t, assignments.unassigns, 2)
	assert.Equal(t, first.Moves[0].ID, assignments.unassigns[0].ShipmentMoveID)
	assert.Equal(t, second.Moves[0].ID, assignments.unassigns[1].ShipmentMoveID)
	assert.Equal(t, params.OrganizationID, assignments.unassigns[0].TenantInfo.OrgID)
	assert.Equal(t, params.BusinessUnitID, assignments.unassigns[0].TenantInfo.BuID)
	assert.Equal(t, params.Actor.UserID, assignments.unassigns[0].TenantInfo.UserID)
}

// The console unassigns move by move and keeps going past a refusal; the
// tool does the same and says which moves were left, so a model is never
// told a partial unassign failed outright or succeeded.
func TestUnassignMoves_KeepsGoingPastARefusalAndNamesIt(t *testing.T) {
	t.Parallel()

	covered := driverCoveredShipment("S-1001")
	underway := pulid.MustNew("smv_")
	assignments := newFakeUnassigner(covered)

	err := (&UnassignMovesTool{assignments: assignments}).Execute(
		t.Context(),
		agenttoolstest.ExecuteParams(
			map[string]any{fieldMoveIDs: moveIDsParam(underway, covered.Moves[0].ID)},
		),
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unassigned 1 of 2 moves")
	assert.Contains(t, err.Error(), underway.String())
	assert.True(t, errortypes.IsBusinessError(err), "the refusal keeps its type")
	assert.Len(t, assignments.unassigns, 2)
	assert.NotContains(t, assignments.assigned, covered.Moves[0].ID)
}

func TestUnassignMoves_RefusesAMoveNamedTwiceAndMoreThanItsCap(t *testing.T) {
	t.Parallel()

	tool := &UnassignMovesTool{assignments: newFakeUnassigner()}
	moveID := pulid.MustNew("smv_")

	err := tool.Validate(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		fieldMoveIDs: moveIDsParam(moveID, moveID),
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "more than once")

	tooMany := make([]pulid.ID, 0, maxMovesPerDispatchChange+1)
	for range maxMovesPerDispatchChange + 1 {
		tooMany = append(tooMany, pulid.MustNew("smv_"))
	}
	require.Error(t, tool.Validate(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		fieldMoveIDs: moveIDsParam(tooMany...),
	})))
}

func TestUnassignMoves_PolicyIsTheConsolesUnassignPermission(t *testing.T) {
	t.Parallel()

	policy := (&UnassignMovesTool{}).Policy()

	assert.Equal(t, permission.ResourceShipmentMove, policy.Resource)
	assert.Equal(t, permission.OpUnassign, policy.Operation)
	assert.Equal(t, agent.TierPropose, policy.DefaultTier)
	assert.Equal(t, agent.TierAutoExecute, policy.MaxTier)
	assert.Equal(t, []agent.EgressClass{agent.EgressInternal}, policy.Egress)
	assert.True(t, policy.Reversible)
	require.NotNil(t, policy.Condition)
}

// One fresh move may come off its driver on the agent's own say; several at
// once, or one the service would refuse, wait for a person.
func TestUnassignMoves_OnlyOneFreshMoveRunsWithoutADecision(t *testing.T) {
	t.Parallel()

	first, second := driverCoveredShipment("S-1001"), driverCoveredShipment("S-1002")
	tool := &UnassignMovesTool{assignments: newFakeUnassigner(first, second)}
	limit := func(ids ...pulid.ID) agent.AutonomyTier {
		return tool.Policy().Condition.Limit(
			t.Context(),
			agenttoolstest.ExecuteParams(map[string]any{
				fieldMoveIDs: moveIDsParam(ids...),
			}),
		)
	}

	assert.Equal(t, agent.TierAutoExecute, limit(first.Moves[0].ID))
	assert.Equal(t, agent.TierPropose, limit(first.Moves[0].ID, second.Moves[0].ID))
	assert.Equal(t, agent.TierPropose, limit(pulid.MustNew("smv_")))
}

func TestUnassignMoves_PreviewShowsTheAssignmentComingOffAndWhatIsRefused(t *testing.T) {
	t.Parallel()

	covered := driverCoveredShipment("S-1001")
	underway := pulid.MustNew("smv_")
	assignments := newFakeUnassigner(covered)
	tool := &UnassignMovesTool{assignments: assignments}
	params := agenttoolstest.ExecuteParams(map[string]any{
		fieldMoveIDs: moveIDsParam(covered.Moves[0].ID, underway),
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&assignments.guard,
		func() (*agent.ToolPreview, error) {
			return tool.Preview(t.Context(), params)
		},
	)

	removed := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, agent.PreviewOperationDelete, removed.Operation)
	assert.Equal(t, "Assignment on S-1001", removed.Label)
	move := agenttoolstest.PreviewChange(t, preview, 1)
	assert.Equal(t, covered.Moves[0].ID, move.EntityID)
	assert.Equal(t, "New", agenttoolstest.FieldByPath(t, move, common.FieldStatus).After)
	require.NotNil(t, move.Version)
	assert.Equal(t, int64(3), *move.Version)
	shipmentChange := agenttoolstest.PreviewChange(t, preview, 2)
	assert.Equal(t, permission.ResourceShipment, shipmentChange.Resource)
	assert.Equal(t, "New", agenttoolstest.FieldByPath(t, shipmentChange, common.FieldStatus).After)
	assert.Contains(t, preview.Summary, "S-1001")
	agenttoolstest.RequireWarning(t, preview, agent.PreviewWarningWouldFail)
	assert.Contains(t, preview.Warnings[0].Message, underway.String())
}

// fakeMoveStatuses holds moves and refuses what the transition table
// refuses, as the service does.
type fakeMoveStatuses struct {
	guard    agenttoolstest.WriteGuard
	moves    map[pulid.ID]*shipment.ShipmentMove
	shipment *shipment.Shipment
	single   *shipmentmoveports.UpdateMoveStatusRequest
	bulk     *shipmentmoveports.BulkUpdateMoveStatusRequest
}

func newFakeMoveStatuses(status shipment.MoveStatus, n int) *fakeMoveStatuses {
	f := &fakeMoveStatuses{
		moves: map[pulid.ID]*shipment.ShipmentMove{},
		shipment: &shipment.Shipment{
			ID:        pulid.MustNew("shp_"),
			ProNumber: "S-2001",
			Status:    shipment.StatusInTransit,
		},
	}
	for range n {
		move := &shipment.ShipmentMove{
			ID:         pulid.MustNew("smv_"),
			ShipmentID: f.shipment.ID,
			Status:     status,
			Version:    2,
		}
		f.moves[move.ID] = move
		f.shipment.Moves = append(f.shipment.Moves, move)
	}

	return f
}

func (f *fakeMoveStatuses) ids() []pulid.ID {
	out := make([]pulid.ID, 0, len(f.shipment.Moves))
	for _, move := range f.shipment.Moves {
		out = append(out, move.ID)
	}

	return out
}

func (f *fakeMoveStatuses) PreviewUpdateStatus(
	_ context.Context,
	req *shipmentmoveports.BulkUpdateMoveStatusRequest,
) (*shipmentmoveports.MoveStatusPlan, error) {
	plan := &shipmentmoveports.MoveStatusPlan{ShipmentsBefore: []*shipment.Shipment{f.shipment}}
	for _, id := range req.MoveIDs {
		move, ok := f.moves[id]
		if !ok {
			return nil, errortypes.NewNotFoundError("Shipment move not found")
		}
		if move.Status == shipment.MoveStatusCompleted {
			return nil, errortypes.NewBusinessError(
				"Move status transition from Completed to " + string(
					req.Status,
				) + " is not allowed",
			)
		}
		after := *move
		after.Status = req.Status
		plan.MovesBefore = append(plan.MovesBefore, move)
		plan.MovesAfter = append(plan.MovesAfter, &after)
	}
	derived := shipmentcoverage.CloneForUpdate(f.shipment)
	derived.Status = shipment.StatusCompleted
	plan.ShipmentsAfter = []*shipment.Shipment{derived}

	return plan, nil
}

func (f *fakeMoveStatuses) UpdateStatus(
	_ context.Context,
	req *shipmentmoveports.UpdateMoveStatusRequest,
) (*shipment.ShipmentMove, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.single = req

	return f.moves[req.MoveID], nil
}

func (f *fakeMoveStatuses) BulkUpdateStatus(
	_ context.Context,
	req *shipmentmoveports.BulkUpdateMoveStatusRequest,
) ([]*shipment.ShipmentMove, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.bulk = req

	return nil, nil
}

// One move goes through the console's single-move endpoint and several
// through its bulk endpoint, which changes them together or not at all.
func TestUpdateMoveStatus_UsesTheSingleOrTheBulkPathAsTheConsoleDoes(t *testing.T) {
	t.Parallel()

	one := newFakeMoveStatuses(shipment.MoveStatusInTransit, 1)
	params := agenttoolstest.ExecuteParams(map[string]any{
		fieldMoveIDs:       moveIDsParam(one.ids()...),
		common.FieldStatus: "Completed",
	})
	require.NoError(t, (&UpdateMoveStatusTool{moves: one}).Execute(t.Context(), params))
	require.NotNil(t, one.single)
	assert.Nil(t, one.bulk)
	assert.Equal(t, shipment.MoveStatusCompleted, one.single.Status)
	assert.Equal(t, params.OrganizationID, one.single.TenantInfo.OrgID)
	assert.Equal(t, params.Actor.UserID, one.single.TenantInfo.UserID)

	two := newFakeMoveStatuses(shipment.MoveStatusInTransit, 2)
	require.NoError(
		t,
		(&UpdateMoveStatusTool{moves: two}).Execute(t.Context(), agenttoolstest.ExecuteParams(
			map[string]any{
				fieldMoveIDs:       moveIDsParam(two.ids()...),
				common.FieldStatus: "Completed",
			},
		)),
	)
	assert.Nil(t, two.single)
	require.NotNil(t, two.bulk)
	assert.Equal(t, two.ids(), two.bulk.MoveIDs)
}

// Assigned and New follow from covering and uncovering a move; setting them
// by hand would leave a move marked assigned with nobody on it.
func TestUpdateMoveStatus_RefusesTheStatusesCoverageDecides(t *testing.T) {
	t.Parallel()

	moves := newFakeMoveStatuses(shipment.MoveStatusNew, 1)
	for _, status := range []string{"Assigned", "New", "in transit", ""} {
		err := (&UpdateMoveStatusTool{moves: moves}).Execute(
			t.Context(),
			agenttoolstest.ExecuteParams(
				map[string]any{
					fieldMoveIDs:       moveIDsParam(moves.ids()...),
					common.FieldStatus: status,
				},
			),
		)
		require.Error(t, err, status)
	}
	assert.Nil(t, moves.single)
}

func TestUpdateMoveStatus_ACancellationIsAlwaysAProposal(t *testing.T) {
	t.Parallel()

	policy := (&UpdateMoveStatusTool{}).Policy()
	assert.Equal(t, permission.ResourceShipmentMove, policy.Resource)
	assert.Equal(t, permission.OpUpdate, policy.Operation)
	assert.Equal(t, agent.TierActWithApproval, policy.MaxTier)
	assert.False(t, policy.Reversible, "a move never moves back")

	limit := func(status string) agent.AutonomyTier {
		return policy.Condition.Limit(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
			fieldMoveIDs:       moveIDsParam(pulid.MustNew("smv_")),
			common.FieldStatus: status,
		}))
	}
	assert.Equal(t, agent.TierPropose, limit("Canceled"))
	assert.Equal(t, agent.TierActWithApproval, limit("Completed"))
	assert.Equal(t, agent.TierActWithApproval, limit("InTransit"))
}

func TestUpdateMoveStatus_PreviewShowsEachMoveAndTheShipmentItMoves(t *testing.T) {
	t.Parallel()

	moves := newFakeMoveStatuses(shipment.MoveStatusInTransit, 2)
	tool := &UpdateMoveStatusTool{moves: moves}
	params := agenttoolstest.ExecuteParams(map[string]any{
		fieldMoveIDs:       moveIDsParam(moves.ids()...),
		common.FieldStatus: "Completed",
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&moves.guard,
		func() (*agent.ToolPreview, error) {
			return tool.Preview(t.Context(), params)
		},
	)

	require.Len(t, preview.Changes, 3)
	first := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, "Move on S-2001", first.Label)
	assert.Equal(t, "InTransit", agenttoolstest.FieldByPath(t, first, common.FieldStatus).Before)
	assert.Equal(t, "Completed", agenttoolstest.FieldByPath(t, first, common.FieldStatus).After)
	shipmentChange := agenttoolstest.PreviewChange(t, preview, 2)
	assert.Equal(t, permission.ResourceShipment, shipmentChange.Resource)
	assert.Equal(
		t,
		"Completed",
		agenttoolstest.FieldByPath(t, shipmentChange, common.FieldStatus).After,
	)
	assert.Contains(t, preview.Summary, "shipment S-2001 from InTransit to Completed")
	assert.Contains(t, preview.Summary, "releases its tractor and trailer")
	assert.Contains(t, preview.Summary, "together or not at all")
}

func TestUpdateMoveStatus_PreviewWarnsOfATransitionTheServiceRefuses(t *testing.T) {
	t.Parallel()

	moves := newFakeMoveStatuses(shipment.MoveStatusCompleted, 1)
	tool := &UpdateMoveStatusTool{moves: moves}

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&moves.guard,
		func() (*agent.ToolPreview, error) {
			return tool.Preview(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
				fieldMoveIDs:       moveIDsParam(moves.ids()...),
				common.FieldStatus: "InTransit",
			}))
		},
	)

	assert.Empty(t, preview.Changes)
	agenttoolstest.RequireWarning(t, preview, agent.PreviewWarningWouldFail)
}
