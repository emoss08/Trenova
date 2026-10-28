package dispatchtools

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/masterdatatools"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

type moveStatusView struct {
	Status string `json:"status"`
}

func (t *UnassignMovesTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	requests, err := t.requests(&params)
	if err != nil {
		return nil, err
	}

	changes := make([]*agent.RecordChange, 0, len(requests)*3)
	shipments := make([]string, 0, len(requests))
	refusals := make([]string, 0)
	for _, request := range requests {
		plan, pErr := t.assignments.PreviewUnassign(ctx, request)
		if pErr != nil {
			if !common.IsRefusal(pErr) && !errortypes.IsNotFoundError(pErr) {
				return nil, pErr
			}
			refusals = append(refusals,
				fmt.Sprintf("move %s: %s", request.ShipmentMoveID, strings.TrimSpace(pErr.Error())))
			continue
		}

		removed, rErr := toolpreview.Delete(toolpreview.Record{
			Resource: permission.ResourceShipmentMove,
			Label:    "Assignment on " + plan.ShipmentBefore.ProNumber,
		}, plan.Assignment, masterdatatools.AssignmentOptions()...)
		if rErr != nil {
			return nil, rErr
		}
		coverage, cErr := common.CoverageChanges(
			plan.ShipmentBefore,
			plan.ShipmentAfter,
			request.ShipmentMoveID,
		)
		if cErr != nil {
			return nil, cErr
		}
		changes = append(changes, removed)
		changes = append(changes, coverage...)
		shipments = append(shipments, plan.ShipmentBefore.ProNumber)
	}

	summary := "Would take no move off its driver."
	if len(shipments) > 0 {
		summary = fmt.Sprintf(
			"Would take the driver and equipment off %s (%s), leaving each uncovered. "+
				"The drivers are told the load was taken off them.",
			common.CountOf(len(shipments), "move"),
			strings.Join(shipments, ", "),
		)
	}

	preview := toolpreview.Build(summary, changes...)
	if len(refusals) > 0 {
		toolpreview.Warn(preview, agent.PreviewWarningWouldFail,
			"These moves would be refused and left as they are: "+strings.Join(refusals, "; "))
	}

	return preview, nil
}

func (t *UpdateMoveStatusTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	request, err := t.request(&params)
	if err != nil {
		return nil, err
	}

	summary := fmt.Sprintf("Would set %s to %s.",
		common.CountOf(len(request.MoveIDs), "move"), request.Status)
	plan, err := t.moves.PreviewUpdateStatus(ctx, request)
	if err != nil {
		if common.IsRefusal(err) || errortypes.IsNotFoundError(err) {
			return common.WarnWouldFail(toolpreview.Build(summary), err), nil
		}

		return nil, err
	}

	pros := make(map[pulid.ID]string, len(plan.ShipmentsBefore))
	for _, entity := range plan.ShipmentsBefore {
		pros[entity.ID] = entity.ProNumber
	}

	changes := make([]*agent.RecordChange, 0, len(plan.MovesBefore)+len(plan.ShipmentsBefore))
	for idx, before := range plan.MovesBefore {
		change, cErr := toolpreview.Changed(toolpreview.Record{
			Resource: permission.ResourceShipmentMove,
			ID:       before.ID,
			Label:    "Move on " + pros[before.ShipmentID],
			Version:  common.PreviewVersion(before.Version),
		}, &moveStatusView{Status: string(before.Status)},
			&moveStatusView{Status: string(plan.MovesAfter[idx].Status)},
		)
		if cErr != nil {
			return nil, cErr
		}
		changes = append(changes, change)
	}

	moved := make([]string, 0, len(plan.ShipmentsBefore))
	for idx, before := range plan.ShipmentsBefore {
		change, sErr := common.ShipmentStatusChange(before, plan.ShipmentsAfter[idx])
		if sErr != nil {
			return nil, sErr
		}
		if change == nil {
			continue
		}
		changes = append(changes, change)
		moved = append(moved, fmt.Sprintf("shipment %s from %s to %s",
			before.ProNumber, before.Status, plan.ShipmentsAfter[idx].Status))
	}

	summary = strings.TrimSuffix(summary, ".") + moveStatusConsequences(request.Status, moved)
	if len(request.MoveIDs) > 1 {
		summary += " The moves change together or not at all."
	}

	return toolpreview.Build(summary, changes...), nil
}

func moveStatusConsequences(status shipment.MoveStatus, moved []string) string {
	out := ""
	if len(moved) > 0 {
		out = ", which moves " + strings.Join(moved, " and ")
	}
	out += "."
	if status == shipment.MoveStatusCompleted {
		out += " Completing a move releases its tractor and trailer for their next move."
	}

	return out
}
