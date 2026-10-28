package billingtools

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/billing/app/billingqueueservice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/timeutils"
)

//nolint:exhaustive // the statuses a decision tool moves an item to
var decisionFields = map[billingqueue.Status][]string{
	billingqueue.StatusOnHold:        {common.FieldStatus, "reviewNotes"},
	billingqueue.StatusException:     {common.FieldStatus, "exceptionReasonCode", "exceptionNotes"},
	billingqueue.StatusSentBackToOps: {common.FieldStatus, "exceptionReasonCode", "exceptionNotes"},
	billingqueue.StatusApproved:      {common.FieldStatus, "reviewNotes", fieldReviewCompletedAt},
	billingqueue.StatusCanceled: {
		common.FieldStatus, common.FieldCancelReason, common.FieldCanceledByID, "canceledAt",
	},
}

var decisionVolatile = []string{fieldReviewCompletedAt, "canceledAt", fieldReviewStartedAt}

var decisionRefs = map[string]permission.Resource{
	common.FieldCanceledByID: permission.ResourceUser,
	"assignedBillerId":       permission.ResourceUser,
}

// planned is the item as the decision leaves it, by the plan Validate and
// the billing queue's UpdateStatus apply.
func (t *BillingQueueDecisionTool) planned(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (common.PlannedChange, *billingqueue.BillingQueueItem, error) {
	req, err := t.request(params)
	if err != nil {
		return common.PlannedChange{}, nil, err
	}

	item, err := t.load(ctx, req)
	if err != nil {
		return common.PlannedChange{}, nil, err
	}

	now := timeutils.NowUnix()
	plan, err := common.PlanUpdate(
		common.QueueRecord(item),
		item,
		func(after *billingqueue.BillingQueueItem) error {
			if planErr := t.plan(after, req, params.Actor, now); planErr != nil {
				return planErr
			}

			return approvalBlock(after, req.NewStatus)
		},
		toolpreview.Only(decisionFields[t.decision.status]...),
		toolpreview.Volatile(decisionVolatile...),
		toolpreview.WithRefs(decisionRefs),
	)
	if err != nil {
		return common.PlannedChange{}, nil, err
	}

	return plan, item, nil
}

func (t *BillingQueueDecisionTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
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
	if t.decision.status != billingqueue.StatusSentBackToOps || !plan.Accepted() {
		return plan.PlanPreview(summary), nil
	}

	after := *item
	after.ExceptionReasonCode = t.reasonOf(&params)
	after.ExceptionNotes = common.OptionalString(params.Params, common.ParamNote)
	comment := billingqueueservice.SendBackComment(&after, params.Actor.UserIDOrNil())
	note := toolpreview.Send(toolpreview.Record{
		Resource: permission.ResourceShipment,
		ID:       item.ShipmentID,
		Label:    common.ShipmentLabel(item),
	}, &agent.MessagePreview{
		Channel:    agent.MessageChannelComment,
		Body:       comment.Comment,
		Visibility: string(comment.Visibility),
	})

	return plan.PlanPreview(summary+" Operations would see a high-priority note on the shipment.",
		note), nil
}

func (t *BillingQueueDecisionTool) reasonOf(
	params *agenttoolports.ToolExecuteParams,
) *billingqueue.ExceptionReasonCode {
	code := billingqueue.ExceptionReasonCode(
		common.OptionalString(params.Params, paramExceptionReasonCode),
	)

	return &code
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
		return common.VerbApprove
	case billingqueue.StatusCanceled:
		return "cancel, dropping its charge from billing,"
	default:
		return "move to " + string(status)
	}
}

func (t *AssignBillerTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	req, asker, err := t.request(ctx, &params)
	if err != nil {
		return nil, err
	}

	item, err := t.load(ctx, req)
	if err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	plan, err := common.PlanUpdate(
		common.QueueRecord(item),
		item,
		func(after *billingqueue.BillingQueueItem) error {
			return billingqueueservice.PlanAssignBiller(after, req.BillerID, now)
		},
		toolpreview.Only("assignedBillerId", common.FieldStatus, fieldReviewStartedAt),
		toolpreview.Volatile(decisionVolatile...),
		toolpreview.WithRefs(decisionRefs),
	)
	if err != nil {
		return nil, err
	}

	return plan.PlanPreview(fmt.Sprintf(
		"Would assign %s to billing queue item %s (now %s).",
		billerClause(asker),
		item.Number,
		item.Status,
	)), nil
}
