package agenttoolservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/billingtransfer"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolschema"
	"github.com/emoss08/trenova/internal/core/services/billingtransferservice"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const (
	paramChargeKind         = "chargeKind"
	paramAdditionalChargeID = "additionalChargeId"
	paramRunAction          = "action"
)

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
		req *serviceports.ReassignChargeRequest,
		actor *serviceports.RequestActor,
	) (*serviceports.ReassignChargeResult, error)
	PreviewReassignCharge(
		ctx context.Context,
		req *serviceports.ReassignChargeRequest,
		actor *serviceports.RequestActor,
	) (*serviceports.ReassignChargePreview, error)
}

func newReassignBillingChargeTool(billing chargeReassigner) serviceports.AgentTool {
	return newReportingReceivableTool(&receivableSpec{
		name:     "reassign_billing_charge",
		artifact: billingQueueRecordEntity,
		description: "Move who pays one charge on a shipment in billing review: its freight " +
			"or one accessorial, whole to another payer or split among several. A payer " +
			"who gains a share gets a queue item and one left with nothing has theirs " +
			"canceled. An empty allocations list gives the charge back to the shipment's " +
			"own payer. Only while every queue item for the shipment is in review or on hold.",
		resource:    permission.ResourceBillingQueue,
		operation:   permission.OpUpdate,
		egress:      agent.EgressMoney,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierPropose,
		personOnly:  true,
		rationale: "Moves what each customer is billed for a shipment and opens or cancels " +
			"their queue items; only a person reassigns a charge.",
		properties: map[string]any{
			paramItemID: agenttoolschema.KindID(
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
			paramAllocations: allocationsProperty("The payers and their shares."),
		},
		required: []string{paramItemID, paramChargeKind, paramAllocations},
		target: func(params map[string]any) (serviceports.ToolTarget, bool) {
			return targetOf(params, paramItemID, permission.ResourceBillingQueue)
		},
	}, receivablePlan[*serviceports.ReassignChargeRequest, *serviceports.ReassignChargePreview]{
		request: reassignRequest,
		plan: func(
			ctx context.Context,
			req *serviceports.ReassignChargeRequest,
			params *serviceports.ToolExecuteParams,
		) (*serviceports.ReassignChargePreview, error) {
			return billing.PreviewReassignCharge(ctx, req, params.Actor)
		},
		refused: func(*serviceports.ReassignChargeRequest) string {
			return "Would reassign who pays a charge."
		},
		render: renderReassignment,
		run: func(
			ctx context.Context,
			req *serviceports.ReassignChargeRequest,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			result, err := billing.ReassignCharge(ctx, req, params.Actor)
			if err != nil {
				return nil, err
			}
			out := &agent.ToolExecutionResult{
				Action: "charge reassigned",
				Kind:   "billing queue item",
				IDs:    map[string]string{paramItemID: req.ItemID.String()},
				Record: recordOf(billingQueueRecordEntity, req.ItemID),
			}
			if result != nil {
				out.Name = fmt.Sprintf(
					"%s, %s",
					countOf(len(result.CreatedItemIDs), "item created"),
					countOf(len(result.CanceledItemIDs), "item canceled"),
				)
			}

			return out, nil
		},
	})
}

func reassignRequest(
	params *serviceports.ToolExecuteParams,
) (*serviceports.ReassignChargeRequest, error) {
	itemID, err := requirePulid(params.Params, paramItemID)
	if err != nil {
		return nil, err
	}
	kind, err := requireEnum(params.Params, paramChargeKind, reassignableKinds.Values)
	if err != nil {
		return nil, err
	}
	chargeID, _, err := optionalPulid(params.Params, paramAdditionalChargeID)
	if err != nil {
		return nil, fmt.Errorf("parameter %q is not a valid id: %w", paramAdditionalChargeID, err)
	}
	if kind == shipment.ChargeAllocationKindAccessorial && chargeID.IsNil() {
		return nil, fmt.Errorf("parameter %q is required for an accessorial charge",
			paramAdditionalChargeID)
	}
	tenant := tenantFrom(*params)
	allocations, err := readAllocations(params.Params, tenant, kind, nil)
	if err != nil {
		return nil, err
	}
	if allocations == nil {
		return nil, fmt.Errorf("missing required parameter %q", paramAllocations)
	}

	return &serviceports.ReassignChargeRequest{
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
	_ *serviceports.ReassignChargeRequest,
	plan *serviceports.ReassignChargePreview,
) (*agent.ToolPreview, error) {
	changes := make([]*agent.RecordChange, 0, len(plan.ToCreate)+len(plan.ToCancel))
	refs := toolpreview.WithRefs(map[string]permission.Resource{
		fieldCustomerID: permission.ResourceCustomer,
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
				Version:  pinnedVersion(item.Version),
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
		countOf(len(plan.Shares), "payer"),
		countOf(len(plan.ToCreate), "queue item"),
		countOf(len(plan.ToCancel), "queue item"),
	), changes...), nil
}

// transferRunAction is what the agent may do to a background billing transfer
// the person started: stop one still running, or run what one left behind.
type transferRunAction string

const (
	transferRunStop  = transferRunAction("Stop")
	transferRunRetry = transferRunAction("Retry")
)

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

func newManageBillingTransferRunTool(runs transferRunKeeper) serviceports.AgentTool {
	return newReportingReceivableTool(&receivableSpec{
		name:     "manage_billing_transfer_run",
		artifact: billingQueueRecordEntity,
		description: "Stop a background transfer to billing that is still running, or " +
			"Retry the shipments a finished one could not transfer. Take the run id from " +
			"transfer_to_billing's result. Only the person who started a transfer can stop " +
			"it; shipments already transferred stay in the queue.",
		resource:    permission.ResourceShipment,
		operation:   permission.OpUpdate,
		egress:      agent.EgressInternal,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierActWithApproval,
		reversible:  true,
		rationale: "Stops or reruns a transfer inside Trenova; nothing is invoiced or sent, " +
			"and a stopped run is retried the same way.",
		properties: map[string]any{
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
		required: []string{resultRunID, paramRunAction},
	}, receivablePlan[*transferRunCall, *billingtransferservice.RunPreview]{
		request: func(params *serviceports.ToolExecuteParams) (*transferRunCall, error) {
			runID, err := requirePulid(params.Params, resultRunID)
			if err != nil {
				return nil, err
			}
			action, err := requireEnum(params.Params, paramRunAction, transferRunActions.Values)
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
		plan: func(
			ctx context.Context,
			call *transferRunCall,
			_ *serviceports.ToolExecuteParams,
		) (*billingtransferservice.RunPreview, error) {
			if call.action == transferRunStop {
				return runs.PreviewCancel(ctx, call.req)
			}

			return runs.PreviewRetry(ctx, call.req)
		},
		refused: func(call *transferRunCall) string {
			return fmt.Sprintf("Would %s the billing transfer.", call.action)
		},
		render: renderTransferRun,
		run: func(
			ctx context.Context,
			call *transferRunCall,
			_ *serviceports.ToolExecuteParams,
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
		countOf(plan.Before.RetryableCount+plan.Before.SkippedCount, "shipment"),
	), change), nil
}
