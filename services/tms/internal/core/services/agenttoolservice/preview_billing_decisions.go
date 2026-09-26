package agenttoolservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/billingqueueservice"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/shared/timeutils"
)

// decisionFields are the item's values each decision sets, and the ones among
// them that record when rather than what.
var (
	//nolint:exhaustive // the statuses a decision tool moves an item to
	decisionFields = map[billingqueue.Status][]string{
		billingqueue.StatusOnHold:        {fieldStatus, "reviewNotes"},
		billingqueue.StatusException:     {fieldStatus, "exceptionReasonCode", "exceptionNotes"},
		billingqueue.StatusSentBackToOps: {fieldStatus, "exceptionReasonCode", "exceptionNotes"},
		billingqueue.StatusApproved:      {fieldStatus, "reviewNotes", fieldReviewCompletedAt},
		billingqueue.StatusCanceled: {
			fieldStatus, paramCancelReason, fieldCanceledByID, "canceledAt",
		},
	}
	decisionVolatile = []string{fieldReviewCompletedAt, "canceledAt", fieldReviewStartedAt}
	decisionRefs     = map[string]permission.Resource{
		fieldCanceledByID:  permission.ResourceUser,
		"assignedBillerId": permission.ResourceUser,
	}
)

func queueRecord(item *billingqueue.BillingQueueItem) toolpreview.Record {
	return toolpreview.Record{
		Resource: permission.ResourceBillingQueue,
		ID:       item.ID,
		Label:    item.Number,
		Version:  previewVersion(item.Version),
	}
}

// planned is the item as the decision leaves it, by the plan Validate and
// the billing queue's UpdateStatus apply.
func (t *billingQueueDecisionTool) planned(
	ctx context.Context,
	params *serviceports.ToolExecuteParams,
) (plannedChange, *billingqueue.BillingQueueItem, error) {
	req, err := t.request(params)
	if err != nil {
		return plannedChange{}, nil, err
	}

	item, err := t.load(ctx, req)
	if err != nil {
		return plannedChange{}, nil, err
	}

	now := timeutils.NowUnix()
	plan, err := planUpdate(
		queueRecord(item),
		item,
		func(after *billingqueue.BillingQueueItem) error {
			if planErr := t.plan(after, req, params.Actor, now); planErr != nil {
				return planErr
			}

			return detentionHold(after, req.NewStatus)
		},
		toolpreview.Only(decisionFields[t.decision.status]...),
		toolpreview.Volatile(decisionVolatile...),
		toolpreview.WithRefs(decisionRefs),
	)
	if err != nil {
		return plannedChange{}, nil, err
	}

	return plan, item, nil
}

func (t *billingQueueDecisionTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	plan, item, err := t.planned(ctx, &params)
	if err != nil {
		return nil, err
	}

	summary := fmt.Sprintf(
		"Would %s billing queue item %s (now %s).",
		decisionVerb(t.decision.status),
		item.Number,
		item.Status,
	)
	if t.decision.status != billingqueue.StatusSentBackToOps || !plan.accepted() {
		return plan.preview(summary), nil
	}

	after := *item
	after.ExceptionReasonCode = t.reasonOf(&params)
	after.ExceptionNotes = optionalString(params.Params, paramDecisionNotes)
	comment := billingqueueservice.SendBackComment(&after, params.Actor.UserIDOrNil())
	note := toolpreview.Send(toolpreview.Record{
		Resource: permission.ResourceShipment,
		ID:       item.ShipmentID,
		Label:    shipmentLabel(item),
	}, &agent.MessagePreview{
		Channel:    agent.MessageChannelComment,
		Body:       comment.Comment,
		Visibility: string(comment.Visibility),
	})

	return plan.preview(summary+" Operations would see a high-priority note on the shipment.",
		note), nil
}

func (t *billingQueueDecisionTool) reasonOf(
	params *serviceports.ToolExecuteParams,
) *billingqueue.ExceptionReasonCode {
	code := billingqueue.ExceptionReasonCode(
		optionalString(params.Params, paramExceptionReasonCode),
	)

	return &code
}

func shipmentLabel(item *billingqueue.BillingQueueItem) string {
	if item.Shipment != nil && item.Shipment.ProNumber != "" {
		return item.Shipment.ProNumber
	}

	return labelShipment
}

func decisionVerb(status billingqueue.Status) string {
	switch status { //nolint:exhaustive // the statuses a decision tool moves an item to
	case billingqueue.StatusOnHold:
		return "put on hold"
	case billingqueue.StatusException:
		return "move into exception"
	case billingqueue.StatusSentBackToOps:
		return "send back to operations"
	case billingqueue.StatusApproved:
		return verbApprove
	case billingqueue.StatusCanceled:
		return "cancel, dropping its charge from billing,"
	default:
		return "move to " + string(status)
	}
}

func (t *assignBillerTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	req, err := t.request(&params)
	if err != nil {
		return nil, err
	}

	item, err := t.load(ctx, req)
	if err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	plan, err := planUpdate(
		queueRecord(item),
		item,
		func(after *billingqueue.BillingQueueItem) error {
			return billingqueueservice.PlanAssignBiller(after, req.BillerID, now)
		},
		toolpreview.Only("assignedBillerId", fieldStatus, fieldReviewStartedAt),
		toolpreview.Volatile(decisionVolatile...),
		toolpreview.WithRefs(decisionRefs),
	)
	if err != nil {
		return nil, err
	}

	return plan.preview(fmt.Sprintf(
		"Would assign a biller to billing queue item %s (now %s).",
		item.Number,
		item.Status,
	)), nil
}
