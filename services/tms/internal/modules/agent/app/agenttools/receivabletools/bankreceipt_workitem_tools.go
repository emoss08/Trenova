package receivabletools

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/billing/app/bankreceiptworkitemservice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceiptworkitem"
	"github.com/emoss08/trenova/internal/modules/billing/ports/bankreceiptworkitemports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/shared/pulid"
)

type workItemMove string

const workItemAssign = workItemMove("Assign")

const workItemStartReview = workItemMove("StartReview")

const paramWorkItemID = "workItemId"

const paramWorkItemMove = "action"

const paramAssigneeID = "assigneeUserId"

var workItemMoves = agenttoolschema.Source(
	"bankReceiptWorkItem.agentMove",
	[]workItemMove{workItemAssign, workItemStartReview},
)

var errAssigneeForAssign = errors.New("assigneeUserId is required to assign a work item")

var errAssigneeForReview = errors.New(
	"assigneeUserId is only for Assign; StartReview marks the item under review as it is",
)

type WorkItemRouter interface {
	PlanAssign(
		ctx context.Context,
		req *bankreceiptworkitemports.AssignBankReceiptWorkItemRequest,
		actor *permissionports.RequestActor,
	) (*bankreceiptworkitemports.BankReceiptWorkItemChange, error)
	Assign(
		ctx context.Context,
		req *bankreceiptworkitemports.AssignBankReceiptWorkItemRequest,
		actor *permissionports.RequestActor,
	) (*bankreceiptworkitem.WorkItem, error)
	PlanStartReview(
		ctx context.Context,
		req *bankreceiptworkitemports.GetBankReceiptWorkItemRequest,
		actor *permissionports.RequestActor,
	) (*bankreceiptworkitemports.BankReceiptWorkItemChange, error)
	StartReview(
		ctx context.Context,
		req *bankreceiptworkitemports.GetBankReceiptWorkItemRequest,
		actor *permissionports.RequestActor,
	) (*bankreceiptworkitem.WorkItem, error)
}

type workItemRoute struct {
	move     workItemMove
	id       pulid.ID
	assignee pulid.ID
	params   *agentports.ToolExecuteParams
}

func workItemRouteFrom(params *agentports.ToolExecuteParams) (*workItemRoute, error) {
	id, err := common.RequirePulid(params.Params, paramWorkItemID)
	if err != nil {
		return nil, err
	}
	move, err := common.RequireEnum(params.Params, paramWorkItemMove, workItemMoves.Values)
	if err != nil {
		return nil, err
	}
	assignee, err := common.OptionalPulidParam(params.Params, paramAssigneeID)
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
	items WorkItemRouter,
) (*bankreceiptworkitemports.BankReceiptWorkItemChange, error) {
	tenant := common.TenantFrom(*r.params)
	if r.move == workItemAssign {
		return items.PlanAssign(ctx, &bankreceiptworkitemports.AssignBankReceiptWorkItemRequest{
			WorkItemID:       r.id,
			AssignedToUserID: r.assignee,
			TenantInfo:       tenant,
		}, r.params.Actor)
	}

	return items.PlanStartReview(ctx, &bankreceiptworkitemports.GetBankReceiptWorkItemRequest{
		WorkItemID: r.id,
		TenantInfo: tenant,
	}, r.params.Actor)
}

func (r *workItemRoute) run(ctx context.Context, items WorkItemRouter) error {
	tenant := common.TenantFrom(*r.params)
	if r.move == workItemAssign {
		_, err := items.Assign(ctx, &bankreceiptworkitemports.AssignBankReceiptWorkItemRequest{
			WorkItemID:       r.id,
			AssignedToUserID: r.assignee,
			TenantInfo:       tenant,
		}, r.params.Actor)

		return err
	}

	_, err := items.StartReview(ctx, &bankreceiptworkitemports.GetBankReceiptWorkItemRequest{
		WorkItemID: r.id,
		TenantInfo: tenant,
	}, r.params.Actor)

	return err
}

func newTriageBankReceiptWorkItemTool(items WorkItemRouter) agentports.AgentTool {
	return common.NewReceivableTool(&common.ReceivableSpec{
		Name:   "triage_bank_receipt_work_item",
		Recipe: []string{"get_bank_receipt", "triage_bank_receipt_work_item"},
		Description: "Assign a bank receipt's reconciliation work item to a person who will " +
			"work it, or mark it under review while it is being looked into. Nothing is " +
			"matched or closed; match_bank_receipt and resolve_bank_receipt_work_item do that.",
		Resource:    permission.ResourceBankReceiptWorkItem,
		Operation:   permission.OpUpdate,
		Egress:      agent.EgressInternal,
		DefaultTier: agent.TierActWithApproval,
		MaxTier:     agent.TierAutoExecute,
		Reversible:  true,
		Rationale: "Changes who works a reconciliation item and says it is being looked " +
			"into; it moves no money and the item is reassigned the same way.",
		Properties: map[string]any{
			paramWorkItemID: agenttoolschema.RecordIDText(
				permission.ResourceBankReceiptWorkItem,
				"The work item's id, from get_bank_receipt or "+
					"this run's subject. Never guess one.",
			),
			paramWorkItemMove: agenttoolschema.Enum("Assign hands it to a person; "+
				"StartReview marks it under review.", workItemMoves),
			paramAssigneeID: agenttoolschema.RecordIDText(
				permission.ResourceUser,
				"For Assign: the person's user id, the "+
					"assignedToUserId get_bank_receipt shows on a work item, or the person who "+
					"asked. Never guess one.",
			),
		},
		Required: []string{paramWorkItemID, paramWorkItemMove},
		Target: func(params map[string]any) (agentports.ToolTarget, bool) {
			return common.TargetOf(params, paramWorkItemID, permission.ResourceBankReceiptWorkItem)
		},
	}, common.ReceivablePlan[*workItemRoute, *bankreceiptworkitemports.BankReceiptWorkItemChange]{
		Request: workItemRouteFrom,
		Plan: func(
			ctx context.Context,
			route *workItemRoute,
			_ *agentports.ToolExecuteParams,
		) (*bankreceiptworkitemports.BankReceiptWorkItemChange, error) {
			return route.plan(ctx, items)
		},
		Refused: func(route *workItemRoute) string {
			if route.move == workItemAssign {
				return "Would assign a bank receipt work item."
			}

			return "Would mark a bank receipt work item under review."
		},
		Render: renderWorkItemRoute,
		Run: func(
			ctx context.Context,
			route *workItemRoute,
			_ *agentports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			return nil, route.run(ctx, items)
		},
	})
}

func renderWorkItemRoute(
	route *workItemRoute,
	change *bankreceiptworkitemports.BankReceiptWorkItemChange,
) (*agent.ToolPreview, error) {
	recorded, err := toolpreview.Changed(
		toolpreview.Record{
			Resource: permission.ResourceBankReceiptWorkItem,
			ID:       change.Before.ID,
			Label:    "Reconciliation item",
			Version:  common.PinnedVersion(change.Before.Version),
		},
		change.Before,
		change.After,
		toolpreview.Only(common.FieldStatus, "assignedToUserId", "assignedAt"),
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

func ProvideTriageBankReceiptWorkItemTool(
	items *bankreceiptworkitemservice.Service,
) agentports.AgentTool {
	return newTriageBankReceiptWorkItemTool(items)
}
