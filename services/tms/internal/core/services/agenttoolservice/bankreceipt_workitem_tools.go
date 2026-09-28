package agenttoolservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/bankreceiptworkitem"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolschema"
	"github.com/emoss08/trenova/internal/core/services/bankreceiptworkitemservice"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/shared/pulid"
)

type workItemMove string

const (
	workItemAssign      = workItemMove("Assign")
	workItemStartReview = workItemMove("StartReview")
	paramWorkItemID     = "workItemId"
	paramWorkItemMove   = "action"
	paramAssigneeID     = "assigneeUserId"
)

var (
	workItemMoves = agenttoolschema.Source(
		"bankReceiptWorkItem.agentMove",
		[]workItemMove{workItemAssign, workItemStartReview},
	)
	errAssigneeForAssign = errors.New("assigneeUserId is required to assign a work item")
	errAssigneeForReview = errors.New(
		"assigneeUserId is only for Assign; StartReview marks the item under review as it is",
	)
)

type workItemRouter interface {
	PlanAssign(
		ctx context.Context,
		req *serviceports.AssignBankReceiptWorkItemRequest,
		actor *serviceports.RequestActor,
	) (*serviceports.BankReceiptWorkItemChange, error)
	Assign(
		ctx context.Context,
		req *serviceports.AssignBankReceiptWorkItemRequest,
		actor *serviceports.RequestActor,
	) (*bankreceiptworkitem.WorkItem, error)
	PlanStartReview(
		ctx context.Context,
		req *serviceports.GetBankReceiptWorkItemRequest,
		actor *serviceports.RequestActor,
	) (*serviceports.BankReceiptWorkItemChange, error)
	StartReview(
		ctx context.Context,
		req *serviceports.GetBankReceiptWorkItemRequest,
		actor *serviceports.RequestActor,
	) (*bankreceiptworkitem.WorkItem, error)
}

var _ workItemRouter = (*bankreceiptworkitemservice.Service)(nil)

type workItemRoute struct {
	move     workItemMove
	id       pulid.ID
	assignee pulid.ID
	params   *serviceports.ToolExecuteParams
}

func workItemRouteFrom(params *serviceports.ToolExecuteParams) (*workItemRoute, error) {
	id, err := requirePulid(params.Params, paramWorkItemID)
	if err != nil {
		return nil, err
	}
	move, err := requireEnum(params.Params, paramWorkItemMove, workItemMoves.Values)
	if err != nil {
		return nil, err
	}
	assignee, err := optionalPulidParam(params.Params, paramAssigneeID)
	if err != nil {
		return nil, err
	}

	route := &workItemRoute{move: move, id: id, params: params}
	switch {
	case move == workItemAssign && assignee == nil:
		return nil, errAssigneeForAssign
	case move == workItemStartReview && assignee != nil:
		return nil, errAssigneeForReview
	case assignee != nil:
		route.assignee = *assignee
	}

	return route, nil
}

func (r *workItemRoute) plan(
	ctx context.Context,
	items workItemRouter,
) (*serviceports.BankReceiptWorkItemChange, error) {
	tenant := tenantFrom(*r.params)
	if r.move == workItemAssign {
		return items.PlanAssign(ctx, &serviceports.AssignBankReceiptWorkItemRequest{
			WorkItemID:       r.id,
			AssignedToUserID: r.assignee,
			TenantInfo:       tenant,
		}, r.params.Actor)
	}

	return items.PlanStartReview(ctx, &serviceports.GetBankReceiptWorkItemRequest{
		WorkItemID: r.id,
		TenantInfo: tenant,
	}, r.params.Actor)
}

func (r *workItemRoute) run(ctx context.Context, items workItemRouter) error {
	tenant := tenantFrom(*r.params)
	if r.move == workItemAssign {
		_, err := items.Assign(ctx, &serviceports.AssignBankReceiptWorkItemRequest{
			WorkItemID:       r.id,
			AssignedToUserID: r.assignee,
			TenantInfo:       tenant,
		}, r.params.Actor)

		return err
	}

	_, err := items.StartReview(ctx, &serviceports.GetBankReceiptWorkItemRequest{
		WorkItemID: r.id,
		TenantInfo: tenant,
	}, r.params.Actor)

	return err
}

func newTriageBankReceiptWorkItemTool(items workItemRouter) serviceports.AgentTool {
	return newReceivableTool(&receivableSpec{
		name: "triage_bank_receipt_work_item",
		description: "Assign a bank receipt's reconciliation work item to a person who will " +
			"work it, or mark it under review while it is being looked into. Nothing is " +
			"matched or closed; match_bank_receipt and resolve_bank_receipt_work_item do that.",
		resource:    permission.ResourceBankReceiptWorkItem,
		operation:   permission.OpUpdate,
		egress:      agent.EgressInternal,
		defaultTier: agent.TierActWithApproval,
		maxTier:     agent.TierAutoExecute,
		reversible:  true,
		rationale: "Changes who works a reconciliation item and says it is being looked " +
			"into; it moves no money and the item is reassigned the same way.",
		properties: map[string]any{
			paramWorkItemID: stringProperty("The work item's id, from get_bank_receipt or "+
				"this run's subject. Never guess one.", 0),
			paramWorkItemMove: agenttoolschema.Enum("Assign hands it to a person; "+
				"StartReview marks it under review.", workItemMoves),
			paramAssigneeID: stringProperty("For Assign: the person's user id, the "+
				"assignedToUserId get_bank_receipt shows on a work item, or the person who "+
				"asked. Never guess one.", 0),
		},
		required: []string{paramWorkItemID, paramWorkItemMove},
		target: func(params map[string]any) (serviceports.ToolTarget, bool) {
			return targetOf(params, paramWorkItemID, permission.ResourceBankReceiptWorkItem)
		},
	}, receivablePlan[*workItemRoute, *serviceports.BankReceiptWorkItemChange]{
		request: workItemRouteFrom,
		plan: func(
			ctx context.Context,
			route *workItemRoute,
			_ *serviceports.ToolExecuteParams,
		) (*serviceports.BankReceiptWorkItemChange, error) {
			return route.plan(ctx, items)
		},
		refused: func(route *workItemRoute) string {
			if route.move == workItemAssign {
				return "Would assign a bank receipt work item."
			}

			return "Would mark a bank receipt work item under review."
		},
		render: renderWorkItemRoute,
		run: func(
			ctx context.Context,
			route *workItemRoute,
			_ *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			return nil, route.run(ctx, items)
		},
	})
}

func renderWorkItemRoute(
	route *workItemRoute,
	change *serviceports.BankReceiptWorkItemChange,
) (*agent.ToolPreview, error) {
	recorded, err := toolpreview.Changed(
		toolpreview.Record{
			Resource: permission.ResourceBankReceiptWorkItem,
			ID:       change.Before.ID,
			Label:    "Reconciliation item",
			Version:  pinnedVersion(change.Before.Version),
		},
		change.Before,
		change.After,
		toolpreview.Only(fieldStatus, "assignedToUserId", "assignedAt"),
		toolpreview.Volatile("assignedAt"),
		toolpreview.WithRefs(map[string]permission.Resource{
			"assignedToUserId": permission.ResourceUser,
		}),
	)
	if err != nil {
		return nil, err
	}

	summary := fmt.Sprintf(
		"Would mark the reconciliation item (now %s) under review; nothing is matched or closed.",
		change.Before.Status,
	)
	if route.move == workItemAssign {
		summary = fmt.Sprintf(
			"Would assign the reconciliation item (now %s) to the person named; nothing is "+
				"matched or closed.",
			change.Before.Status,
		)
	}

	return toolpreview.Build(summary, recorded), nil
}

func provideTriageBankReceiptWorkItemTool(
	items *bankreceiptworkitemservice.Service,
) serviceports.AgentTool {
	return newTriageBankReceiptWorkItemTool(items)
}
