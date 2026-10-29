package editools

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/edi/app/edidecision"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edisync"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editenderchange"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/errortypes"
)

const changeFieldReason = "failureReason"

var changeReviewLabels = map[string]string{
	changepreview.FieldStatus: "Change status",
	changeFieldReason:         "Reason recorded",
}

var tenderPayloadIgnored = []string{
	"purposeCode",
	"ratingDetail",
	"requiredMappingEntityIds",
}

func (t *EdiChangeReviewTool) Preview(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*agentpreview.ToolPreview, error) {
	plan, err := t.plan(ctx, params)
	if err != nil {
		if changepreview.IsRefusal(err) || errortypes.IsNotFoundError(err) {
			return changepreview.WarnWouldFail(toolpreview.Build(t.kind.description), err), nil
		}

		return nil, err
	}
	preview := toolpreview.Build(plan.summary)
	if plan.refusal == nil {
		preview = toolpreview.Build(plan.summary, plan.changes...)
		preview.Partial = preview.Partial || plan.partial
	}

	return changepreview.WarnWouldFail(preview, plan.refusal), nil
}

func planTenderChangeReview(
	change *editenderchange.TenderChange,
	call *changeReviewCall,
) (*changeReviewPlan, error) {
	plan := &changeReviewPlan{summary: fmt.Sprintf(
		"Would %s the change another organization made to load tender BOL %s.",
		call.verb(), change.NewTenderPayload.BOL,
	)}
	plan.refusal = edidecision.CheckTenderChangeReview(change, call.tenant)
	if plan.refusal == nil {
		if err := describeTenderChangeReview(plan, change, call); err != nil {
			return nil, err
		}
	}

	return plan, nil
}

func describeTenderChangeReview(
	plan *changeReviewPlan,
	change *editenderchange.TenderChange,
	call *changeReviewCall,
) error {
	after := *change
	edidecision.MarkTenderChangeReviewed(&after, call.mark())
	status, err := changeStatusRecord(toolpreview.Record{
		Resource: permission.ResourceEDI,
		ID:       change.ID,
		Label:    "Tender change to BOL " + change.NewTenderPayload.BOL,
		Version:  changepreview.PinnedVersion(change.Version),
	}, change, &after)
	if err != nil {
		return err
	}
	plan.changes = []*agentpreview.RecordChange{status}
	if !call.apply {
		plan.summary += " The load stays as it is, and the other organization sees the " +
			"change rejected."
		return nil
	}

	tender, err := toolpreview.Changed(toolpreview.Record{
		Resource: permission.ResourceEDI,
		ID:       change.InternalTransferID,
		Label:    "Load tender BOL " + change.NewTenderPayload.BOL + " as changed",
	}, &change.PreviousBaselinePayload, &change.NewTenderPayload,
		toolpreview.Ignore(tenderPayloadIgnored...))
	if err != nil {
		return err
	}
	plan.changes = append(plan.changes, tender)
	target := "the tender, which waits for approval again"
	if change.ShipmentLinkID.IsNotNil() {
		target = "the shipment made from the tender"
	}
	plan.summary += " What it changes is written onto " + target + "."

	return nil
}

func planTransferChangeReview(
	ctx context.Context,
	reviewer transferChangeReviewer,
	change *edisync.TransferChange,
	call *changeReviewCall,
) (*changeReviewPlan, error) {
	plan := &changeReviewPlan{summary: fmt.Sprintf(
		"Would %s the %s the other organization on this linked load reported.",
		call.verb(), transferChangeWords(change.ChangeType),
	)}
	plan.refusal = reviewer.CheckTransferChangeReview(ctx, change, call.tenant)
	if plan.refusal == nil {
		review := transferChangeReview{reviewer: reviewer, change: change, call: call}
		if err := review.describe(ctx, plan); err != nil {
			return nil, err
		}
	}

	return plan, nil
}

type transferChangeReview struct {
	reviewer transferChangeReviewer
	change   *edisync.TransferChange
	call     *changeReviewCall
}

func (r transferChangeReview) describe(ctx context.Context, plan *changeReviewPlan) error {
	change, call := r.change, r.call
	after := *change
	edidecision.MarkTransferChangeReviewed(&after, call.mark())
	status, err := changeStatusRecord(toolpreview.Record{
		Resource: permission.ResourceEDI,
		ID:       change.ID,
		Label:    "Linked load " + transferChangeWords(change.ChangeType),
		Version:  changepreview.PinnedVersion(change.Version),
	}, change, &after)
	if err != nil {
		return err
	}
	plan.changes = []*agentpreview.RecordChange{status}
	if !call.apply {
		plan.summary += " The shipment stays as it is."
		return nil
	}

	effect, err := r.reviewer.PlanTransferChangeEffect(ctx, call.tenant, change)
	if err != nil {
		if changepreview.IsRefusal(err) || errortypes.IsNotFoundError(err) {
			plan.refusal = err
			return nil
		}
		return err
	}
	if effect.Shipment == nil {
		plan.partial = true
		plan.summary += " The shipment's stops are brought in line by the lifecycle " +
			"rules when it is applied; that effect is not shown here."
		return nil
	}
	if effect.Shipment.Status == effect.NextStatus {
		plan.summary += " The shipment already has that status, so only the change is closed."
		return nil
	}

	moved := *effect.Shipment
	moved.Status = effect.NextStatus
	shipmentChange, err := toolpreview.Changed(toolpreview.Record{
		Resource: permission.ResourceShipment,
		ID:       effect.Shipment.ID,
		Label:    effect.Shipment.ProNumber,
		Version:  changepreview.PinnedVersion(effect.Shipment.Version),
	}, effect.Shipment, &moved, toolpreview.Only(changepreview.FieldStatus))
	if err != nil {
		return err
	}
	plan.changes = append(plan.changes, shipmentChange)
	plan.summary += fmt.Sprintf(" Shipment %s would move from %s to %s.",
		effect.Shipment.ProNumber, effect.Shipment.Status, effect.NextStatus)

	return nil
}

func changeStatusRecord[T any](
	record toolpreview.Record,
	before, after *T,
) (*agentpreview.RecordChange, error) {
	return toolpreview.Changed(record, before, after,
		toolpreview.Only(changepreview.FieldStatus, changeFieldReason),
		toolpreview.Labels(changeReviewLabels),
	)
}

func transferChangeWords(changeType string) string {
	switch changeType {
	case edisync.TransferChangeTypeShipmentStatus214:
		return "status update"
	case edisync.TransferChangeTypeShipmentCancel214:
		return "cancellation"
	case edisync.TransferChangeTypeShipmentLifecycle214:
		return "stop progress"
	default:
		return "change"
	}
}
