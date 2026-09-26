package agenttoolservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/dispatchconsoleservice"
	"github.com/emoss08/trenova/internal/core/services/dispatcheligibility"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

var _ serviceports.ToolPreviewer = (*assignMoveTool)(nil)

type assignmentChecker interface {
	PreviewAssignment(
		ctx context.Context,
		req *dispatchconsoleservice.PreviewAssignmentRequest,
	) (*dispatchconsoleservice.AssignmentPreview, error)
}

var assignmentFields = []string{
	fieldPrimaryWorkerID,
	fieldSecondaryWorkerID,
	fieldTractorID,
	fieldTrailerID,
	fieldStatus,
}

var assignmentRefs = map[string]permission.Resource{
	fieldPrimaryWorkerID:   permission.ResourceWorker,
	fieldSecondaryWorkerID: permission.ResourceWorker,
	fieldTractorID:         permission.ResourceTractor,
	fieldTrailerID:         permission.ResourceTrailer,
}

var assignmentLabels = map[string]string{
	fieldPrimaryWorkerID:   "Driver",
	fieldSecondaryWorkerID: "Second driver",
	fieldTractorID:         "Tractor",
	fieldTrailerID:         "Trailer",
}

var assignedStatusLabels = map[string]string{"coverageType": "Covered by"}

func assignmentOptions() []toolpreview.Option {
	return []toolpreview.Option{
		toolpreview.Only(assignmentFields...),
		toolpreview.WithRefs(assignmentRefs),
		toolpreview.Labels(assignmentLabels),
	}
}

type assignedMoveView struct {
	Status       string `json:"status"`
	CoverageType string `json:"coverageType"`
}

type shipmentStatusView struct {
	Status string `json:"status"`
}

func assignedMoveOf(entity *shipment.Shipment, moveID pulid.ID) *assignedMoveView {
	move := entity.FindMove(moveID)
	if move == nil {
		return &assignedMoveView{}
	}

	return &assignedMoveView{Status: string(move.Status), CoverageType: string(move.CoverageType)}
}

func (t *assignMoveTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	if err := guardPreview(t, &params); err != nil {
		return nil, err
	}

	request, err := t.request(&params)
	if err != nil {
		return nil, err
	}

	summary := "Would put a driver and a tractor on this move."
	plan, err := t.assignments.PreviewAssignToMove(ctx, request)
	if err != nil {
		if isRefusal(err) {
			return warnWouldFail(toolpreview.Build(summary), err), nil
		}

		return nil, err
	}

	changes, err := t.assignmentChanges(request, plan)
	if err != nil {
		return nil, err
	}

	check, note, err := t.check(ctx, request)
	if err != nil {
		return nil, err
	}

	shipmentLabel := plan.ShipmentBefore.ProNumber
	summary = fmt.Sprintf("Would put a driver and a tractor on shipment %s", shipmentLabel)
	if check.Score != nil && check.Score.WorkerName != "" {
		summary = fmt.Sprintf(
			"Would put %s on shipment %s with the tractor shown",
			check.Score.WorkerName,
			shipmentLabel,
		)
		labelRefs(changes[0], map[string]string{fieldPrimaryWorkerID: check.Score.WorkerName})
	}
	summary += ". The driver sees the assignment." + findingsSentence(check) + note

	preview := toolpreview.Build(summary, changes...)
	if check.RequiresOverride {
		toolpreview.Warn(
			preview,
			agent.PreviewWarningWouldFail,
			"The organization's dispatch rules block this driver for this move: "+
				blockingFindings(check),
		)
	}

	return preview, nil
}

func (t *assignMoveTool) assignmentChanges(
	request *repositories.AssignShipmentMoveRequest,
	plan *serviceports.AssignmentPlan,
) ([]*agent.RecordChange, error) {
	shipmentLabel := plan.ShipmentBefore.ProNumber
	assignment, err := toolpreview.Create(toolpreview.Record{
		Resource: permission.ResourceShipmentMove,
		Label:    "Assignment on " + shipmentLabel,
	}, plan.Assignment, assignmentOptions()...)
	if err != nil {
		return nil, err
	}

	changes := []*agent.RecordChange{assignment}
	before := plan.ShipmentBefore.FindMove(request.ShipmentMoveID)
	move, err := toolpreview.Changed(toolpreview.Record{
		Resource: permission.ResourceShipmentMove,
		ID:       request.ShipmentMoveID,
		Label:    "Move on " + shipmentLabel,
		Version:  moveVersion(before),
	}, assignedMoveOf(plan.ShipmentBefore, request.ShipmentMoveID),
		assignedMoveOf(plan.ShipmentAfter, request.ShipmentMoveID),
		toolpreview.Labels(assignedStatusLabels),
	)
	if err != nil {
		return nil, err
	}
	if len(move.Fields) > 0 {
		changes = append(changes, move)
	}

	if plan.ShipmentBefore.Status != plan.ShipmentAfter.Status {
		shipmentChange, sErr := toolpreview.Changed(toolpreview.Record{
			Resource: permission.ResourceShipment,
			ID:       plan.ShipmentBefore.ID,
			Label:    shipmentLabel,
			Version:  previewVersion(plan.ShipmentBefore.Version),
		}, &shipmentStatusView{Status: string(plan.ShipmentBefore.Status)},
			&shipmentStatusView{Status: string(plan.ShipmentAfter.Status)},
		)
		if sErr != nil {
			return nil, sErr
		}
		changes = append(changes, shipmentChange)
	}

	return changes, nil
}

func moveVersion(move *shipment.ShipmentMove) *int64 {
	if move == nil {
		return nil
	}

	return previewVersion(move.Version)
}

func (t *assignMoveTool) check(
	ctx context.Context,
	request *repositories.AssignShipmentMoveRequest,
) (*dispatchconsoleservice.AssignmentPreview, string, error) {
	if t.console == nil {
		return &dispatchconsoleservice.AssignmentPreview{}, "", nil
	}

	checkRequest := &dispatchconsoleservice.PreviewAssignmentRequest{
		TenantInfo: request.TenantInfo,
		MoveID:     request.ShipmentMoveID,
		WorkerID:   request.PrimaryWorkerID,
		TractorID:  request.TractorID,
	}
	if request.TrailerID != nil {
		checkRequest.TrailerID = *request.TrailerID
	}

	check, err := t.console.PreviewAssignment(ctx, checkRequest)
	if err != nil {
		if isRefusal(err) || errortypes.IsNotFoundError(err) {
			return &dispatchconsoleservice.AssignmentPreview{},
				" The dispatch check could not rate this driver: " + err.Error() + ".", nil
		}

		return nil, "", err
	}

	return check, "", nil
}

func findingsSentence(check *dispatchconsoleservice.AssignmentPreview) string {
	if check.Score == nil || len(check.Score.Findings) == 0 {
		return ""
	}

	messages := make([]string, 0, len(check.Score.Findings))
	for idx := range check.Score.Findings {
		messages = append(messages, check.Score.Findings[idx].Message)
	}

	return " The dispatch check notes: " + strings.Join(messages, "; ") + "."
}

func blockingFindings(check *dispatchconsoleservice.AssignmentPreview) string {
	if check.Score == nil {
		return "the driver is not eligible"
	}

	messages := make([]string, 0, len(check.Score.Findings))
	for idx := range check.Score.Findings {
		finding := &check.Score.Findings[idx]
		if finding.Severity == dispatcheligibility.SeverityBlock {
			messages = append(messages, finding.Message)
		}
	}
	if len(messages) == 0 {
		return "the driver is not eligible"
	}

	return strings.Join(messages, "; ")
}
