package billingtools

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/billing/app/billingtransferservice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingtransfer"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const paramChargeKind = "chargeKind"

const paramAdditionalChargeID = "additionalChargeId"

const paramRunAction = "action"

var reassignableKinds = agenttoolschema.Source(
	"shipment.reassignableChargeKind",
	[]shipment.ChargeAllocationKind{
		shipment.ChargeAllocationKindFreight,
		shipment.ChargeAllocationKindAccessorial,
	},
)

type chargeReassigner interface {
	ReassignCharge(
		ctx context.Context,
		req *billingqueueports.ReassignChargeRequest,
		actor *permissionports.RequestActor,
	) (*billingqueueports.ReassignChargeResult, error)
	PreviewReassignCharge(
		ctx context.Context,
		req *billingqueueports.ReassignChargeRequest,
		actor *permissionports.RequestActor,
	) (*billingqueueports.ReassignChargePreview, error)
}

func NewReassignBillingChargeTool(billing chargeReassigner) agentports.AgentTool {
	return common.NewReportingReceivableTool(&common.ReceivableSpec{
		Name:     "reassign_billing_charge",
		Artifact: billingQueueRecordEntity,
		Description: "Move who pays one charge on a shipment in billing review: its freight " +
			"or one accessorial, whole to another payer or split among several. A payer " +
			"who gains a share gets a queue item and one left with nothing has theirs " +
			"canceled. An empty allocations list gives the charge back to the shipment's " +
			"own payer. Only while every queue item for the shipment is in review or on hold.",
		Resource:    permission.ResourceBillingQueue,
		Operation:   permission.OpUpdate,
		Egress:      agent.EgressMoney,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierPropose,
		PersonOnly:  true,
		Rationale: "Moves what each customer is billed for a shipment and opens or cancels " +
			"their queue items; only a person reassigns a charge.",
		Properties: map[string]any{
			common.ParamItemID: agenttoolschema.KindID(
				"The billing queue item, from list_billing_queue_items "+
					"or get_billing_queue_item. Never guess one.",
				permission.KindBillingQueueItem,
			),
			paramChargeKind: agenttoolschema.Enum(
				"Which charge: the freight, or one accessorial.",
				reassignableKinds,
			),
			paramAdditionalChargeID: agenttoolschema.KindID("The accessorial charge, from the "+
				"shipment's charges get_billing_queue_item lists. Needed for Accessorial.",
				permission.KindAdditionalCharge),
			common.ParamAllocations: common.AllocationsProperty("The payers and their shares."),
		},
		Required: []string{common.ParamItemID, paramChargeKind, common.ParamAllocations},
		Target: func(params map[string]any) (agentports.ToolTarget, bool) {
			return common.TargetOf(params, common.ParamItemID, permission.ResourceBillingQueue)
		},
	}, common.ReceivablePlan[*billingqueueports.ReassignChargeRequest, *billingqueueports.ReassignChargePreview]{
		Request: reassignRequest,
		Plan: func(
			ctx context.Context,
			req *billingqueueports.ReassignChargeRequest,
			params *agentports.ToolExecuteParams,
		) (*billingqueueports.ReassignChargePreview, error) {
			return billing.PreviewReassignCharge(ctx, req, params.Actor)
		},
		Refused: func(*billingqueueports.ReassignChargeRequest) string {
			return "Would reassign who pays a charge."
		},
		Render: renderReassignment,
		Run: func(
			ctx context.Context,
			req *billingqueueports.ReassignChargeRequest,
			params *agentports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			result, err := billing.ReassignCharge(ctx, req, params.Actor)
			if err != nil {
				return nil, err
			}
			out := &agent.ToolExecutionResult{
				Action: "charge reassigned",
				Kind:   "billing queue item",
				IDs:    map[string]string{common.ParamItemID: req.ItemID.String()},
				Record: common.RecordOf(billingQueueRecordEntity, req.ItemID),
			}
			if result != nil {
				out.Name = fmt.Sprintf(
					"%s, %s",
					common.CountOf(len(result.CreatedItemIDs), "item created"),
					common.CountOf(len(result.CanceledItemIDs), "item canceled"),
				)
			}

			return out, nil
		},
	})
}

func reassignRequest(
	params *agentports.ToolExecuteParams,
) (*billingqueueports.ReassignChargeRequest, error) {
	itemID, err := common.RequirePulid(params.Params, common.ParamItemID)
	if err != nil {
		return nil, err
	}
	kind, err := common.RequireEnum(params.Params, paramChargeKind, reassignableKinds.Values)
	if err != nil {
		return nil, err
	}
	chargeID, _, err := common.OptionalPulid(params.Params, paramAdditionalChargeID)
	if err != nil {
		return nil, fmt.Errorf("parameter %q is not a valid id: %w", paramAdditionalChargeID, err)
	}
	if kind == shipment.ChargeAllocationKindAccessorial && chargeID.IsNil() {
		return nil, fmt.Errorf("parameter %q is required for an accessorial charge",
			paramAdditionalChargeID)
	}
	tenant := common.TenantFrom(*params)
	allocations, err := common.ReadAllocations(params.Params, tenant, kind, nil)
	if err != nil {
		return nil, err
	}
	if allocations == nil {
		return nil, fmt.Errorf("missing required parameter %q", common.ParamAllocations)
	}

	return &billingqueueports.ReassignChargeRequest{
		ItemID:             itemID,
		TenantInfo:         tenant,
		ChargeKind:         kind,
		AdditionalChargeID: chargeID,
		Allocations:        allocations,
	}, nil
}

type payerShareView struct {
	CustomerID pulid.ID        `json:"customerId"`
	Freight    decimal.Decimal `json:"freightAmount"`
	Other      decimal.Decimal `json:"accessorialAmount"`
	Total      decimal.Decimal `json:"totalAmount"`
}

type queueItemStatusView struct {
	Status billingqueue.Status `json:"status"`
}

func renderReassignment(
	_ *billingqueueports.ReassignChargeRequest,
	plan *billingqueueports.ReassignChargePreview,
) (*agent.ToolPreview, error) {
	changes := make([]*agent.RecordChange, 0, len(plan.ToCreate)+len(plan.ToCancel))
	refs := toolpreview.WithRefs(map[string]permission.Resource{
		common.FieldCustomerID: permission.ResourceCustomer,
	})
	for _, share := range plan.ToCreate {
		if share == nil {
			continue
		}
		change, err := toolpreview.Create(
			toolpreview.Record{
				Resource: permission.ResourceBillingQueue,
				Label:    "New queue item for " + plan.Shipment.ProNumber,
			},
			&payerShareView{
				CustomerID: share.PayerID,
				Freight:    share.FreightAmount,
				Other:      share.AccessorialAmount,
				Total:      share.TotalAmount,
			},
			refs,
			toolpreview.SensitiveAs("totalAmount"),
		)
		if err != nil {
			return nil, err
		}
		changes = append(changes, change)
	}
	for _, item := range plan.ToCancel {
		if item == nil {
			continue
		}
		change, err := toolpreview.Changed(
			toolpreview.Record{
				Resource: permission.ResourceBillingQueue,
				ID:       item.ID,
				Label:    "Queue item " + item.Number,
				Version:  common.PinnedVersion(item.Version),
			},
			&queueItemStatusView{Status: item.Status},
			&queueItemStatusView{Status: billingqueue.StatusCanceled},
		)
		if err != nil {
			return nil, err
		}
		changes = append(changes, change)
	}

	return toolpreview.Build(fmt.Sprintf(
		"Would divide %s on shipment %s among %s, opening %s and canceling %s.",
		plan.Description,
		plan.Shipment.ProNumber,
		common.CountOf(len(plan.Shares), "payer"),
		common.CountOf(len(plan.ToCreate), "queue item"),
		common.CountOf(len(plan.ToCancel), "queue item"),
	), changes...), nil
}

// transferRunAction is what the agent may do to a background billing transfer
// the person started: stop one still running, or run what one left behind.
type transferRunAction string

const transferRunStop = transferRunAction("Stop")

const transferRunRetry = transferRunAction("Retry")

var transferRunActions = agenttoolschema.Source(
	"billingTransfer.runAction",
	[]transferRunAction{transferRunStop, transferRunRetry},
)

type transferRunKeeper interface {
	Cancel(
		ctx context.Context,
		req *billingtransferservice.RunRequest,
	) (*billingtransfer.BillingTransferRun, error)
	PreviewCancel(
		ctx context.Context,
		req *billingtransferservice.RunRequest,
	) (*billingtransferservice.RunPreview, error)
	Retry(
		ctx context.Context,
		req *billingtransferservice.RunRequest,
	) (*billingtransfer.BillingTransferRun, error)
	PreviewRetry(
		ctx context.Context,
		req *billingtransferservice.RunRequest,
	) (*billingtransferservice.RunPreview, error)
}

type transferRunCall struct {
	action transferRunAction
	req    *billingtransferservice.RunRequest
}

func NewManageBillingTransferRunTool(runs transferRunKeeper) agentports.AgentTool {
	return common.NewReportingReceivableTool(&common.ReceivableSpec{
		Name:     "manage_billing_transfer_run",
		Artifact: billingQueueRecordEntity,
		Description: "Stop a background transfer to billing that is still running, or " +
			"Retry the shipments a finished one could not transfer. Take the run id from " +
			"transfer_to_billing's result. Only the person who started a transfer can stop " +
			"it; shipments already transferred stay in the queue.",
		Resource:    permission.ResourceShipment,
		Operation:   permission.OpUpdate,
		Egress:      agent.EgressInternal,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierActWithApproval,
		Reversible:  true,
		Rationale: "Stops or reruns a transfer inside Trenova; nothing is invoiced or sent, " +
			"and a stopped run is retried the same way.",
		Properties: map[string]any{
			resultRunID: agenttoolschema.KindID(
				"The billing transfer run, from transfer_to_billing's "+
					"result. Never guess one.",
				permission.KindBillingTransferRun,
			),
			paramRunAction: agenttoolschema.Enum(
				"Stop a running transfer, or Retry a finished one.",
				transferRunActions,
			),
		},
		Required: []string{resultRunID, paramRunAction},
	}, common.ReceivablePlan[*transferRunCall, *billingtransferservice.RunPreview]{
		Request: func(params *agentports.ToolExecuteParams) (*transferRunCall, error) {
			runID, err := common.RequirePulid(params.Params, resultRunID)
			if err != nil {
				return nil, err
			}
			action, err := common.RequireEnum(params.Params, paramRunAction, transferRunActions.Values)
			if err != nil {
				return nil, err
			}

			return &transferRunCall{
				action: action,
				req: &billingtransferservice.RunRequest{
					TenantInfo: pagination.TenantInfo{
						OrgID:  params.OrganizationID,
						BuID:   params.BusinessUnitID,
						UserID: params.Actor.UserID,
					},
					RunID: runID,
				},
			}, nil
		},
		Plan: func(
			ctx context.Context,
			call *transferRunCall,
			_ *agentports.ToolExecuteParams,
		) (*billingtransferservice.RunPreview, error) {
			if call.action == transferRunStop {
				return runs.PreviewCancel(ctx, call.req)
			}

			return runs.PreviewRetry(ctx, call.req)
		},
		Refused: func(call *transferRunCall) string {
			return fmt.Sprintf("Would %s the billing transfer.", call.action)
		},
		Render: renderTransferRun,
		Run: func(
			ctx context.Context,
			call *transferRunCall,
			_ *agentports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			var (
				run *billingtransfer.BillingTransferRun
				err error
			)
			if call.action == transferRunStop {
				run, err = runs.Cancel(ctx, call.req)
			} else {
				run, err = runs.Retry(ctx, call.req)
			}
			if err != nil {
				return nil, err
			}

			return &agent.ToolExecutionResult{
				Action: map[transferRunAction]string{
					transferRunStop:  "stop requested",
					transferRunRetry: "retry started",
				}[call.action],
				Kind: "billing transfer run",
				IDs:  map[string]string{resultRunID: run.ID.String()},
			}, nil
		},
	})
}

type transferRunView struct {
	Status              billingtransfer.RunStatus `json:"status"`
	Scope               billingtransfer.RunScope  `json:"scope"`
	TotalCount          int                       `json:"totalCount"`
	CancelRequestedByID pulid.ID                  `json:"cancelRequestedById"`
	SourceRunID         pulid.ID                  `json:"sourceRunId"`
}

func transferRunViewOf(run *billingtransfer.BillingTransferRun) *transferRunView {
	return &transferRunView{
		Status:              run.Status,
		Scope:               run.Scope,
		TotalCount:          run.TotalCount,
		CancelRequestedByID: run.CancelRequestedByID,
		SourceRunID:         run.SourceRunID,
	}
}

func renderTransferRun(
	call *transferRunCall,
	plan *billingtransferservice.RunPreview,
) (*agent.ToolPreview, error) {
	refs := toolpreview.WithRefs(map[string]permission.Resource{
		"cancelRequestedById": permission.ResourceUser,
	})
	if call.action == transferRunStop {
		change, err := toolpreview.Changed(
			toolpreview.Record{Resource: permission.ResourceShipment, Label: "Billing transfer"},
			transferRunViewOf(plan.Before),
			transferRunViewOf(plan.After),
			refs,
		)
		if err != nil {
			return nil, err
		}

		return toolpreview.Build(fmt.Sprintf(
			"Would stop the billing transfer after the shipment it is on; %d of %d are done "+
				"and stay in the queue.",
			plan.Before.ProcessedCount, plan.Before.TotalCount,
		), change), nil
	}

	change, err := toolpreview.Create(
		toolpreview.Record{Resource: permission.ResourceShipment, Label: "Billing transfer retry"},
		transferRunViewOf(plan.Created),
		refs,
	)
	if err != nil {
		return nil, err
	}

	return toolpreview.Build(fmt.Sprintf(
		"Would transfer again the %s the earlier run could not.",
		common.CountOf(plan.Before.RetryableCount+plan.Before.SkippedCount, "shipment"),
	), change), nil
}
