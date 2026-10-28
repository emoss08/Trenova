package masterdatatools

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/app/dispatchconsoleservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/dispatcheligibility"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/errortypes"
)

type assignmentChecker interface {
	PreviewAssignment(
		ctx context.Context,
		req *dispatchconsoleservice.PreviewAssignmentRequest,
	) (*dispatchconsoleservice.AssignmentPreview, error)
}

var assignmentFields = []string{
	common.FieldPrimaryWorkerID,
	common.FieldSecondaryWorkerID,
	common.FieldTractorID,
	common.FieldTrailerID,
	common.FieldStatus,
}

var assignmentRefs = map[string]permission.Resource{
	common.FieldPrimaryWorkerID:   permission.ResourceWorker,
	common.FieldSecondaryWorkerID: permission.ResourceWorker,
	common.FieldTractorID:         permission.ResourceTractor,
	common.FieldTrailerID:         permission.ResourceTrailer,
}

var assignmentLabels = map[string]string{
	common.FieldPrimaryWorkerID:   "Driver",
	common.FieldSecondaryWorkerID: "Second driver",
	common.FieldTractorID:         "Tractor",
	common.FieldTrailerID:         "Trailer",
}

func AssignmentOptions() []toolpreview.Option {
	return []toolpreview.Option{
		toolpreview.Only(assignmentFields...),
		toolpreview.WithRefs(assignmentRefs),
		toolpreview.Labels(assignmentLabels),
	}
}

func (t *AssignMoveTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	if err := common.GuardPreview(t, &params); err != nil {
		return nil, err
	}

	request, err := t.request(&params)
	if err != nil {
		return nil, err
	}

	summary := "Would put a driver and a tractor on this move."
	plan, err := t.assignments.PreviewAssignToMove(ctx, request)
	if err != nil {
		if common.IsRefusal(err) {
			return common.WarnWouldFail(toolpreview.Build(summary), err), nil
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
		common.LabelRefs(
			changes[0],
			map[string]string{common.FieldPrimaryWorkerID: check.Score.WorkerName},
		)
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
	if len(plan.EquipmentInUse) > 0 {
		toolpreview.Warn(
			preview,
			agent.PreviewWarningEquipmentInUse,
			equipmentInUseWarning(plan.EquipmentInUse),
			equipmentInUseLoads(plan.EquipmentInUse)...,
		)
	}

	return preview, nil
}

func (t *AssignMoveTool) assignmentChanges(
	request *shipmentports.AssignShipmentMoveRequest,
	plan *shipmentports.AssignmentPlan,
) ([]*agent.RecordChange, error) {
	shipmentLabel := plan.ShipmentBefore.ProNumber
	assignment, err := toolpreview.Create(toolpreview.Record{
		Resource: permission.ResourceShipmentMove,
		Label:    "Assignment on " + shipmentLabel,
	}, plan.Assignment, AssignmentOptions()...)
	if err != nil {
		return nil, err
	}

	coverage, err := common.CoverageChanges(
		plan.ShipmentBefore,
		plan.ShipmentAfter,
		request.ShipmentMoveID,
	)
	if err != nil {
		return nil, err
	}

	return append([]*agent.RecordChange{assignment}, coverage...), nil
}

func (t *AssignMoveTool) check(
	ctx context.Context,
	request *shipmentports.AssignShipmentMoveRequest,
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
		if common.IsRefusal(err) || errortypes.IsNotFoundError(err) {
			return &dispatchconsoleservice.AssignmentPreview{},
				" The dispatch check could not rate this driver: " + err.Error() + ".", nil
		}

		return nil, "", err
	}

	return check, "", nil
}

func equipmentInUseWarning(inUse []shipmentports.EquipmentInUse) string {
	parts := make([]string, 0, len(inUse))
	for _, item := range inUse {
		where := "another move still in progress"
		if item.ProNumber != "" {
			where = item.ProNumber + ", a load still in progress"
		}
		parts = append(parts, "the "+item.Kind+" is on "+where)
	}

	return "Equipment in use: " + strings.Join(parts, "; ") + ". The assignment can be " +
		"saved, but no stop on this move can be started with that equipment until the " +
		"other load is done. Pick free equipment unless this load is booked for after it, " +
		"and say which."
}

func equipmentInUseLoads(inUse []shipmentports.EquipmentInUse) []string {
	loads := make([]string, 0, len(inUse))
	for _, item := range inUse {
		if item.ProNumber != "" {
			loads = append(loads, item.ProNumber)
		}
	}

	return loads
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
