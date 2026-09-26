package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/assignmentservice"
	"github.com/emoss08/trenova/internal/core/services/dispatchcandidateservice"
	"github.com/emoss08/trenova/internal/core/services/dispatchconsoleservice"
	"github.com/emoss08/trenova/internal/core/services/dispatcheligibility"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type savingAssignments struct {
	serviceports.AssignmentService

	shipment *shipment.Shipment
	existing *shipment.Assignment
	saved    *shipment.Assignment
	guard    writeGuard
}

func (f *savingAssignments) plan(
	req *repositories.AssignShipmentMoveRequest,
) (*serviceports.AssignmentPlan, error) {
	assignment, err := assignmentservice.NewMoveAssignment(req, f.existing)
	if err != nil {
		return nil, err
	}
	after := shipment.CloneForUpdate(f.shipment)
	move := after.FindMove(req.ShipmentMoveID)
	move.Assignment = assignment
	move.CoverageType = shipment.MoveCoverageTypeDriver
	move.Status = shipment.MoveStatusAssigned
	after.Status = shipment.StatusAssigned

	return &serviceports.AssignmentPlan{
		Assignment:     assignment,
		ShipmentBefore: f.shipment,
		ShipmentAfter:  after,
	}, nil
}

func (f *savingAssignments) PreviewAssignToMove(
	_ context.Context,
	req *repositories.AssignShipmentMoveRequest,
) (*serviceports.AssignmentPlan, error) {
	return f.plan(req)
}

func (f *savingAssignments) AssignToMove(
	_ context.Context,
	req *repositories.AssignShipmentMoveRequest,
) (*shipment.Assignment, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	plan, err := f.plan(req)
	if err != nil {
		return nil, err
	}
	f.saved = plan.Assignment
	f.saved.ID = pulid.MustNew("asg_")

	return f.saved, nil
}

type checkingConsole struct {
	serviceports.DispatchConsoleService

	preview *dispatchconsoleservice.AssignmentPreview
}

func (f *checkingConsole) PreviewAssignment(
	context.Context,
	*dispatchconsoleservice.PreviewAssignmentRequest,
) (*dispatchconsoleservice.AssignmentPreview, error) {
	return f.preview, nil
}

func uncoveredShipment() *shipment.Shipment {
	moveID := pulid.MustNew("smv_")

	return &shipment.Shipment{
		ID:        pulid.MustNew("shp_"),
		ProNumber: "S-88213",
		Status:    shipment.StatusNew,
		Moves: []*shipment.ShipmentMove{{
			ID:           moveID,
			Status:       shipment.MoveStatusNew,
			CoverageType: shipment.MoveCoverageTypeUnassigned,
			Version:      2,
		}},
	}
}

func assignParams(entity *shipment.Shipment) (serviceports.ToolExecuteParams, pulid.ID, pulid.ID) {
	workerID := pulid.MustNew("wrk_")
	tractorID := pulid.MustNew("trc_")

	return executeParams(map[string]any{
		"shipmentMoveId":  entity.Moves[0].ID.String(),
		"primaryWorkerId": workerID.String(),
		"tractorId":       tractorID.String(),
	}), workerID, tractorID
}

func TestAssignMove_PreviewMatchesWhatIsSaved(t *testing.T) {
	t.Parallel()

	entity := uncoveredShipment()
	assignments := &savingAssignments{shipment: entity}
	console := &checkingConsole{preview: &dispatchconsoleservice.AssignmentPreview{
		Score: &dispatchcandidateservice.CandidateScore{
			WorkerName: "Ana Reyes",
			Findings: []dispatcheligibility.Finding{{
				Severity: dispatcheligibility.SeverityWarn,
				Message:  "Medical card expires in 12 days",
			}},
		},
	}}
	tool := newAssignMoveTool(assignments, console).(*assignMoveTool)
	params, workerID, tractorID := assignParams(entity)

	preview := previewWithoutWrites(t, &assignments.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), params)
	})

	assert.Empty(t, preview.Warnings)
	assert.Contains(t, preview.Summary, "Ana Reyes")
	assert.Contains(t, preview.Summary, "Medical card expires in 12 days")
	require.Len(t, preview.Changes, 3)

	created := previewChange(t, preview, 0)
	assert.Equal(t, agent.PreviewOperationCreate, created.Operation)
	driver := fieldByPath(t, created, "primaryWorkerId")
	assert.Equal(t, "Driver", driver.Label)
	require.NotNil(t, driver.AfterRef)
	assert.Equal(t, workerID, driver.AfterRef.ID)
	assert.Equal(t, "Ana Reyes", driver.AfterRef.Label)
	tractor := fieldByPath(t, created, "tractorId")
	require.NotNil(t, tractor.AfterRef)
	assert.Equal(t, permission.ResourceTractor, tractor.AfterRef.Resource)
	assert.Equal(t, tractorID, tractor.AfterRef.ID)

	move := previewChange(t, preview, 1)
	assert.Equal(t, entity.Moves[0].ID, move.EntityID)
	assert.Equal(t, "Assigned", fieldByPath(t, move, "status").After)
	shipmentChange := previewChange(t, preview, 2)
	assert.Equal(t, permission.ResourceShipment, shipmentChange.Resource)

	require.NoError(t, tool.Execute(t.Context(), params))
	requireCreateParity(t, created, assignments.saved, assignmentOptions()...)
}

func TestAssignMove_PreviewWarnsWhenDispatchRulesBlockTheDriver(t *testing.T) {
	t.Parallel()

	entity := uncoveredShipment()
	assignments := &savingAssignments{shipment: entity}
	console := &checkingConsole{preview: &dispatchconsoleservice.AssignmentPreview{
		Blocked:          true,
		RequiresOverride: true,
		Score: &dispatchcandidateservice.CandidateScore{
			Findings: []dispatcheligibility.Finding{{
				Severity: dispatcheligibility.SeverityBlock,
				Message:  "Driver is out of drive time",
			}},
		},
	}}
	tool := newAssignMoveTool(assignments, console).(*assignMoveTool)
	params, _, _ := assignParams(entity)

	preview := previewWithoutWrites(t, &assignments.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), params)
	})

	requireWarning(t, preview, agent.PreviewWarningWouldFail)
	assert.Contains(t, preview.Warnings[0].Message, "Driver is out of drive time")
}

func TestAssignMove_PreviewWarnsForAMoveAlreadyCovered(t *testing.T) {
	t.Parallel()

	entity := uncoveredShipment()
	assignments := &savingAssignments{
		shipment: entity,
		existing: &shipment.Assignment{ID: pulid.MustNew("asg_")},
	}
	tool := newAssignMoveTool(assignments, nil).(*assignMoveTool)
	params, _, _ := assignParams(entity)

	preview := previewWithoutWrites(t, &assignments.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), params)
	})

	assert.Empty(t, preview.Changes)
	requireWarning(t, preview, agent.PreviewWarningWouldFail)
}
