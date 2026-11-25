package movetools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/dispatchconsoleports"
	"github.com/emoss08/trenova/internal/modules/operations/app/assignmentservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/dispatchconsoleservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentcoverage"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/assignmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchboardports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fleetAssignments plans and saves assignments over several shipments, and
// refuses the moves it is told to.
type fleetAssignments struct {
	assignmentports.AssignmentService

	byMove  map[pulid.ID]*shipment.Shipment
	refuse  map[pulid.ID]error
	saved   []pulid.ID
	workers map[pulid.ID]pulid.ID
}

func (f *fleetAssignments) plan(
	req *assignmentports.AssignShipmentMoveRequest,
) (*assignmentports.AssignmentPlan, error) {
	if err := f.refuse[req.ShipmentMoveID]; err != nil {
		return nil, err
	}
	before, ok := f.byMove[req.ShipmentMoveID]
	if !ok {
		return nil, errortypes.NewNotFoundError("Shipment move not found")
	}
	assignment, err := assignmentservice.NewMoveAssignment(req, nil)
	if err != nil {
		return nil, err
	}
	after := shipmentcoverage.CloneForUpdate(before)
	move := after.FindMove(req.ShipmentMoveID)
	move.Assignment = assignment
	move.CoverageType = shipmentvocab.MoveCoverageTypeDriver
	move.Status = shipmentvocab.MoveStatusAssigned
	after.Status = shipmentvocab.StatusAssigned

	return &assignmentports.AssignmentPlan{
		Assignment:     assignment,
		ShipmentBefore: before,
		ShipmentAfter:  after,
	}, nil
}

func (f *fleetAssignments) PreviewAssignToMove(
	_ context.Context,
	req *assignmentports.AssignShipmentMoveRequest,
) (*assignmentports.AssignmentPlan, error) {
	return f.plan(req)
}

func (f *fleetAssignments) AssignToMove(
	_ context.Context,
	req *assignmentports.AssignShipmentMoveRequest,
) (*shipment.Assignment, error) {
	plan, err := f.plan(req)
	if err != nil {
		return nil, err
	}
	f.saved = append(f.saved, req.ShipmentMoveID)
	if f.workers == nil {
		f.workers = map[pulid.ID]pulid.ID{}
	}
	f.workers[req.ShipmentMoveID] = req.PrimaryWorkerID

	return plan.Assignment, nil
}

// namedConsole rates every driver without a finding and names a move by its
// shipment's PRO number.
type namedConsole struct {
	dispatchconsoleports.DispatchConsoleService

	byMove   map[pulid.ID]*shipment.Shipment
	assigned *fleetAssignments
	reads    int
}

func (f *namedConsole) PreviewAssignment(
	context.Context,
	*dispatchconsoleservice.PreviewAssignmentRequest,
) (*dispatchconsoleservice.AssignmentPreview, error) {
	return &dispatchconsoleservice.AssignmentPreview{}, nil
}

func (f *namedConsole) GetMoves(
	ctx context.Context,
	tenant pagination.TenantInfo,
	moveIDs []pulid.ID,
) ([]*dispatchconsoleservice.BoardMove, error) {
	f.reads++
	out := make([]*dispatchconsoleservice.BoardMove, 0, len(moveIDs))
	for _, id := range moveIDs {
		move, err := f.GetMove(ctx, tenant, id)
		if err != nil {
			continue
		}
		if f.assigned != nil {
			if worker, ok := f.assigned.workers[id]; ok {
				move.AssignedWorkerName = "Driver " + worker.String()[len(worker.String())-4:]
				move.AssignedTractorCode = "T-104"
				move.MoveStatus = shipmentvocab.MoveStatusAssigned
			}
		}
		out = append(out, move)
	}

	return out, nil
}

func (f *namedConsole) GetMove(
	_ context.Context,
	_ pagination.TenantInfo,
	moveID pulid.ID,
) (*dispatchconsoleservice.BoardMove, error) {
	entity, ok := f.byMove[moveID]
	if !ok {
		return nil, errortypes.NewNotFoundError("Shipment move not found")
	}

	return &dispatchconsoleservice.BoardMove{
		BoardMove: &dispatchboardports.BoardMove{
			MoveID: moveID, ShipmentID: entity.ID, ProNumber: entity.ProNumber,
			MoveStatus: shipmentvocab.MoveStatusNew,
		},
	}, nil
}

type fleet struct {
	shipments []*shipment.Shipment
	byMove    map[pulid.ID]*shipment.Shipment
}

func newFleet(pros ...string) *fleet {
	out := &fleet{byMove: map[pulid.ID]*shipment.Shipment{}}
	for _, pro := range pros {
		entity := uncoveredShipment()
		entity.ProNumber = pro
		out.shipments = append(out.shipments, entity)
		out.byMove[entity.Moves[0].ID] = entity
	}

	return out
}

func (f *fleet) moveID(idx int) pulid.ID { return f.shipments[idx].Moves[0].ID }

func row(moveID, workerID, tractorID pulid.ID) map[string]any {
	return map[string]any{
		"shipmentMoveId":  moveID.String(),
		"primaryWorkerId": workerID.String(),
		"tractorId":       tractorID.String(),
		"reason":          "Closest driver with hours to spare.",
	}
}

func assignTool(f *fleet, refuse map[pulid.ID]error) (*AssignMovesTool, *fleetAssignments) {
	assignments := &fleetAssignments{byMove: f.byMove, refuse: refuse}
	console := &namedConsole{byMove: f.byMove, assigned: assignments}
	tool := NewAssignMovesTool(assignments, console).(*AssignMovesTool)

	return tool, assignments
}

func TestAssignMoves_NamesTheToolAndTakesRows(t *testing.T) {
	t.Parallel()

	tool, _ := assignTool(newFleet("S-1"), nil)

	assert.Equal(t, "dispatch_assign_move", tool.Name())
	schema := tool.ParamSchema()
	assert.Equal(t, []string{"assignments"}, schema["required"])
	properties := schema["properties"].(map[string]any)
	assert.Contains(t, properties, "shipmentMoveIds")
	rows := properties["assignments"].(map[string]any)
	assert.Equal(t, 50, rows["maxItems"])
	item := rows["items"].(map[string]any)
	assert.Contains(t, item["properties"], "primaryWorkerId")
}

func TestAssignMoves_OneMoveIsTheSingleToolsCall(t *testing.T) {
	t.Parallel()

	f := newFleet("S-1")
	tool, assignments := assignTool(f, nil)
	workerID, tractorID := pulid.MustNew("wrk_"), pulid.MustNew("trc_")
	params := agenttoolstest.PersonParams(map[string]any{
		"assignments": []any{row(f.moveID(0), workerID, tractorID)},
	})

	preview, err := tool.Preview(t.Context(), params)
	require.NoError(t, err)
	assert.Contains(t, preview.Summary, "S-1")

	target, ok := tool.Target(params.Params)
	require.True(t, ok)
	assert.Equal(t, f.moveID(0), target.ID)

	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, []pulid.ID{f.moveID(0)}, assignments.saved)
	assert.Equal(t, workerID, assignments.workers[f.moveID(0)])
}

func TestAssignMoves_SeveralMovesEachTakeTheirOwnDriver(t *testing.T) {
	t.Parallel()

	f := newFleet("S-1", "S-2", "S-3")
	tool, assignments := assignTool(f, nil)
	drivers := []pulid.ID{pulid.MustNew("wrk_"), pulid.MustNew("wrk_"), pulid.MustNew("wrk_")}
	rows := make([]any, 0, 3)
	for idx := range 3 {
		rows = append(rows, row(f.moveID(idx), drivers[idx], pulid.MustNew("trc_")))
	}
	params := agenttoolstest.Approved(agenttoolstest.PersonParams(map[string]any{
		"assignments": rows,
	}))

	preview, err := tool.Preview(t.Context(), params)
	require.NoError(t, err)
	assert.Contains(t, preview.Summary, "3 would be assigned")
	assert.Empty(t, preview.Warnings)
	_, ok := tool.Target(params.Params)
	assert.False(t, ok, "a call over several moves pins none of them")

	result, err := tool.ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)
	assert.Equal(t, 3, result.Total)
	for idx := range 3 {
		assert.Equal(t, drivers[idx], assignments.workers[f.moveID(idx)])
	}
}

func TestAssignMoves_ApproverDroppingAMoveSkipsItsRow(t *testing.T) {
	t.Parallel()

	f := newFleet("S-1", "S-2")
	tool, assignments := assignTool(f, nil)
	call := map[string]any{"assignments": []any{
		row(f.moveID(0), pulid.MustNew("wrk_"), pulid.MustNew("trc_")),
		row(f.moveID(1), pulid.MustNew("wrk_"), pulid.MustNew("trc_")),
	}}

	pinned, err := tool.ResolveSelection(t.Context(), agenttoolstest.PersonParams(call))
	require.NoError(t, err)
	assert.Equal(t, []any{f.moveID(0).String(), f.moveID(1).String()}, pinned["shipmentMoveIds"])

	pinned["shipmentMoveIds"] = []any{f.moveID(1).String()}
	require.NoError(t, tool.Execute(t.Context(), agenttoolstest.Approved(
		agenttoolstest.PersonParams(pinned),
	)))
	assert.Equal(t, []pulid.ID{f.moveID(1)}, assignments.saved)
}

func TestAssignMoves_RefusesAListedMoveWithoutARow(t *testing.T) {
	t.Parallel()

	f := newFleet("S-1", "S-2")
	tool, _ := assignTool(f, nil)
	params := agenttoolstest.PersonParams(map[string]any{
		"shipmentMoveIds": []any{f.moveID(0).String(), f.moveID(1).String()},
		"assignments":     []any{row(f.moveID(0), pulid.MustNew("wrk_"), pulid.MustNew("trc_"))},
	})

	err := tool.Validate(t.Context(), params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), f.moveID(1).String())
}

func TestAssignMoves_RefusesAMoveNamedTwice(t *testing.T) {
	t.Parallel()

	f := newFleet("S-1")
	tool, _ := assignTool(f, nil)
	params := agenttoolstest.PersonParams(map[string]any{"assignments": []any{
		row(f.moveID(0), pulid.MustNew("wrk_"), pulid.MustNew("trc_")),
		row(f.moveID(0), pulid.MustNew("wrk_"), pulid.MustNew("trc_")),
	}})

	err := tool.Validate(t.Context(), params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "twice")
}

func TestAssignMoves_ARefusedMoveIsReportedAndTheRestGo(t *testing.T) {
	t.Parallel()

	f := newFleet("S-1", "S-2")
	refused := errortypes.NewBusinessError("Shipment move is covered by an external carrier")
	tool, assignments := assignTool(f, map[pulid.ID]error{f.moveID(0): refused})
	params := agenttoolstest.Approved(agenttoolstest.PersonParams(map[string]any{
		"assignments": []any{
			row(f.moveID(0), pulid.MustNew("wrk_"), pulid.MustNew("trc_")),
			row(f.moveID(1), pulid.MustNew("wrk_"), pulid.MustNew("trc_")),
		},
	}))

	preview, err := tool.Preview(t.Context(), params)
	require.NoError(t, err)
	assert.Contains(t, preview.Summary, "1 would be refused")
	require.NoError(t, tool.Validate(t.Context(), params))

	result, err := tool.ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)
	assert.Equal(t, []pulid.ID{f.moveID(1)}, assignments.saved)
	require.Len(t, result.Failed, 1)
	assert.Equal(t, "S-1", result.Failed[0].Label, "a failure is named by its load, never its id")
	assert.Equal(t, f.shipments[0].ID.String(), result.Failed[0].ID, "and links to its shipment")
	assert.Equal(t, "shipment", tool.Policy().Artifact)
}

func TestAssignMoves_EveryMoveRefusedFails(t *testing.T) {
	t.Parallel()

	f := newFleet("S-1", "S-2")
	refused := errortypes.NewBusinessError("Shipment move is not assignable")
	tool, _ := assignTool(f, map[pulid.ID]error{f.moveID(0): refused, f.moveID(1): refused})
	params := agenttoolstest.Approved(agenttoolstest.PersonParams(map[string]any{
		"assignments": []any{
			row(f.moveID(0), pulid.MustNew("wrk_"), pulid.MustNew("trc_")),
			row(f.moveID(1), pulid.MustNew("wrk_"), pulid.MustNew("trc_")),
		},
	}))

	require.Error(t, tool.Validate(t.Context(), params))
	_, err := tool.ExecuteWithResult(t.Context(), params)
	require.Error(t, err)
}

func TestAssignMoves_FlagsOneDriverOnSeveralLoadsByLoad(t *testing.T) {
	t.Parallel()

	f := newFleet("S-1", "S-2", "S-3")
	tool, _ := assignTool(f, nil)
	driver := pulid.MustNew("wrk_")
	params := agenttoolstest.PersonParams(map[string]any{"assignments": []any{
		row(f.moveID(0), driver, pulid.MustNew("trc_")),
		row(f.moveID(1), driver, pulid.MustNew("trc_")),
		row(f.moveID(2), pulid.MustNew("wrk_"), pulid.MustNew("trc_")),
	}})

	preview, err := tool.Preview(t.Context(), params)
	require.NoError(t, err)
	require.Len(t, preview.Warnings, 1)
	warning := preview.Warnings[0]
	assert.Equal(t, agentpreview.PreviewWarningDoubleBooked, warning.Code)
	assert.Contains(t, warning.Message, "one driver on S-1 and S-2")
	assert.NotContains(t, warning.Message, driver.String(), "a warning never shows a driver's id")
	assert.Equal(t, []string{"S-1", "S-2"}, warning.Args)
}

func TestAssignMoves_SeveralMovesAlwaysWaitForAPerson(t *testing.T) {
	t.Parallel()

	f := newFleet("S-1", "S-2")
	tool, _ := assignTool(f, nil)
	limit := tool.Policy().Condition.Limit
	one := agenttoolstest.PersonParams(map[string]any{"assignments": []any{
		row(f.moveID(0), pulid.MustNew("wrk_"), pulid.MustNew("trc_")),
	}})
	two := agenttoolstest.PersonParams(map[string]any{"assignments": []any{
		row(f.moveID(0), pulid.MustNew("wrk_"), pulid.MustNew("trc_")),
		row(f.moveID(1), pulid.MustNew("wrk_"), pulid.MustNew("trc_")),
	}})

	assert.Equal(t, agent.TierAutoExecute, limit(t.Context(), one))
	assert.Equal(t, agent.TierActWithApproval, limit(t.Context(), two))
}

func TestAssignMoves_SplitsIntoOneCallPerMove(t *testing.T) {
	t.Parallel()

	f := newFleet("S-1", "S-2")
	tool, _ := assignTool(f, nil)
	rows := []any{
		row(f.moveID(0), pulid.MustNew("wrk_"), pulid.MustNew("trc_")),
		row(f.moveID(1), pulid.MustNew("wrk_"), pulid.MustNew("trc_")),
	}

	calls, err := tool.SplitRecords(map[string]any{"assignments": rows})
	require.NoError(t, err)
	require.Len(t, calls, 2)
	for idx, call := range calls {
		assert.Equal(t, []any{f.moveID(idx).String()}, call["shipmentMoveIds"])
		assert.Equal(t, []any{rows[idx]}, call["assignments"])
	}
}

var _ agenttoolports.ToolRecordSplitter = (*AssignMovesTool)(nil)
var _ agenttoolports.ToolSelectionResolver = (*AssignMovesTool)(nil)

func TestAssignMoves_ARunSaysWhoIsOnEachLoadFromOneRead(t *testing.T) {
	t.Parallel()

	f := newFleet("S-1", "S-2")
	refused := errortypes.NewBusinessError("Shipment move is covered by an external carrier")
	assignments := &fleetAssignments{byMove: f.byMove, refuse: map[pulid.ID]error{f.moveID(1): refused}}
	console := &namedConsole{byMove: f.byMove, assigned: assignments}
	tool := NewAssignMovesTool(assignments, console).(*AssignMovesTool)
	driver := pulid.MustNew("wrk_")
	params := agenttoolstest.Approved(agenttoolstest.PersonParams(map[string]any{
		"assignments": []any{
			row(f.moveID(0), driver, pulid.MustNew("trc_")),
			row(f.moveID(1), pulid.MustNew("wrk_"), pulid.MustNew("trc_")),
		},
	}))

	result, err := tool.ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)

	assert.Contains(t, result.State, "S-1: Driver "+driver.String()[len(driver.String())-4:]+
		" with tractor T-104, move Assigned")
	assert.Contains(t, result.State, "S-2: no driver shows on it, move New",
		"a refused load reads as it stands, beside its reason in Failed")
	assert.Equal(t, 1, console.reads, "the batch reads every move back once, not one by one")
	assert.Contains(t, result.Bounded().Describe(), "Now: S-1")
}

func TestAssignMoves_OneMoveRunSaysWhoIsOnItAndLinksItsShipment(t *testing.T) {
	t.Parallel()

	f := newFleet("S-1")
	tool, _ := assignTool(f, nil)
	params := agenttoolstest.PersonParams(map[string]any{
		"assignments": []any{row(f.moveID(0), pulid.MustNew("wrk_"), pulid.MustNew("trc_"))},
	})

	result, err := tool.ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)

	assert.Equal(t, "S-1", result.Name)
	assert.Contains(t, result.State, "with tractor T-104, move Assigned")
	require.NotNil(t, result.Record)
	assert.Equal(t, "shipment", result.Record.EntityType)
	assert.Equal(t, f.shipments[0].ID.String(), result.Record.ID)
}
