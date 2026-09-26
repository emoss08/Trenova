package agenttoolservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/billingtransfer"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/billingtransferservice"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
)

const (
	paramShipmentIDs        = "shipmentIds"
	paramBillType           = "billType"
	paramMarkCompletedReady = "markCompletedReadyToInvoice"
	resultRunID             = "billingTransferRunId"
	// billingQueueRecordEntity is the record-link registry's name for a
	// billing queue item, which a transfer makes one of per payer.
	billingQueueRecordEntity = "billing_queue_item"
	maxResultFailures        = 3
)

type billingTransferPlanner interface {
	PlanBillingTransfers(
		ctx context.Context,
		req *serviceports.PlanBillingTransfersRequest,
	) (*serviceports.BillingTransferPlan, error)
	BulkTransferToBilling(
		ctx context.Context,
		req *serviceports.BulkTransferShipmentToBillingRequest,
		actor *serviceports.RequestActor,
	) (*serviceports.BulkTransferToBillingResponse, error)
}

type billingTransferRunStarter interface {
	Start(
		ctx context.Context,
		req *billingtransferservice.StartRunRequest,
	) (*billingtransfer.BillingTransferRun, error)
}

var (
	_ serviceports.ToolPreviewer      = (*transferToBillingTool)(nil)
	_ serviceports.ToolValidator      = (*transferToBillingTool)(nil)
	_ serviceports.ToolResultReporter = (*transferToBillingTool)(nil)
)

// transferToBillingTool queues delivered shipments for billing. A set the
// synchronous transfer takes is transferred in the call; a larger one is
// handed to the background transfer run the dialog uses for the same size.
type transferToBillingTool struct {
	shipments billingTransferPlanner
	runs      billingTransferRunStarter
}

func newTransferToBillingTool(
	shipments billingTransferPlanner,
	runs billingTransferRunStarter,
) serviceports.AgentTool {
	return &transferToBillingTool{shipments: shipments, runs: runs}
}

func (t *transferToBillingTool) Name() string { return "transfer_to_billing" }

func (t *transferToBillingTool) Description() string {
	return "Transfer delivered shipments to the billing queue, as the transfer-to-billing " +
		"dialog does: each becomes a queue item a biller reviews. Take the ids from " +
		"list_billing_transfer_candidates and cover every shipment that can go in one call " +
		"rather than one call per shipment; a shipment the checks refuse is reported, not " +
		"transferred, and the rest still go. The person approving may untick shipments. " +
		"Up to 100 transfer at once; more run as a background transfer."
}

func (t *transferToBillingTool) ParamSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: map[string]any{
			paramShipmentIDs: toolschema.RecordSubset(permission.ResourceShipment.String(),
				map[string]any{
					toolschema.KeyType: toolschema.TypeArray,
					toolschema.KeyDescription: "The shipments to transfer, by id from " +
						"list_billing_transfer_candidates. Never guess one.",
					toolschema.KeyMinItems: 1,
					toolschema.KeyMaxItems: serviceports.MaxBillingTransferCandidateIDs,
					toolschema.KeyItems: map[string]any{
						toolschema.KeyType: toolschema.TypeString,
					},
				}),
			paramBillType: map[string]any{
				toolschema.KeyType:        toolschema.TypeString,
				toolschema.KeyEnum:        billTypeNames(),
				toolschema.KeyDescription: "What the queue items bill. Defaults to Invoice.",
			},
			paramMarkCompletedReady: map[string]any{
				toolschema.KeyType: toolschema.TypeBoolean,
				toolschema.KeyDescription: "Mark Completed shipments ready to invoice first, as " +
					"the dialog can. Defaults to false, which leaves a Completed shipment where " +
					"it is.",
			},
		},
		toolschema.KeyRequired:             []string{paramShipmentIDs},
		toolschema.KeyAdditionalProperties: false,
	}
}

func (t *transferToBillingTool) Policy() serviceports.ToolPolicy {
	return serviceports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceShipment,
		Operation:     permission.OpUpdate,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agent.ToolEffectChange,
		Artifact:      billingQueueRecordEntity,
		Reversible:    true,
		ReadsExternal: agent.ExternalReadNever,
		Rationale: "Hands delivered shipments to the billing queue inside Trenova, by the " +
			"checks the transfer dialog makes; a biller, or the organization's own " +
			"auto-approve rule, still decides every item.",
	}
}

func billTypeNames() []string {
	return []string{
		string(billingqueue.BillTypeInvoice),
		string(billingqueue.BillTypeCreditMemo),
		string(billingqueue.BillTypeDebitMemo),
	}
}

// transferRequest is what both the preview and the write act on: the
// shipments, once each, and how to bill them.
type transferRequest struct {
	shipmentIDs []pulid.ID
	billType    billingqueue.BillType
	markReady   bool
}

func (r transferRequest) background() bool {
	return len(r.shipmentIDs) > serviceports.MaxBulkTransferToBillingShipments
}

func (t *transferToBillingTool) request(
	params *serviceports.ToolExecuteParams,
) (transferRequest, error) {
	if err := guardPreview(t, params); err != nil {
		return transferRequest{}, err
	}

	ids, err := requirePulidSlice(
		params.Params,
		paramShipmentIDs,
		serviceports.MaxBillingTransferCandidateIDs,
	)
	if err != nil {
		return transferRequest{}, err
	}

	billType := billingqueue.BillTypeInvoice
	if raw := optionalString(params.Params, paramBillType); raw != "" {
		billType = billingqueue.BillType(raw)
		if !isBillType(billType) {
			return transferRequest{}, fmt.Errorf(
				"billType %q is not one of %s", raw, strings.Join(billTypeNames(), ", "),
			)
		}
	}

	return transferRequest{
		shipmentIDs: sliceutils.Dedupe(ids),
		billType:    billType,
		markReady:   optionalBool(params.Params, paramMarkCompletedReady),
	}, nil
}

func isBillType(billType billingqueue.BillType) bool {
	switch billType {
	case billingqueue.BillTypeInvoice, billingqueue.BillTypeCreditMemo,
		billingqueue.BillTypeDebitMemo:
		return true
	default:
		return false
	}
}

func (t *transferToBillingTool) Validate(
	_ context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	_, err := t.request(&params)

	return err
}

func (t *transferToBillingTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	_, err := t.ExecuteWithResult(ctx, params)

	return err
}

// ExecuteWithResult transfers the shipments and says what became of them: the
// count of each outcome and the first refusals, or the background run a large
// set was handed to.
func (t *transferToBillingTool) ExecuteWithResult(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolResultReporter interface passes params by value
) (*agent.ToolExecutionResult, error) {
	if err := guardExecute(t, params); err != nil {
		return nil, err
	}
	request, err := t.request(&params)
	if err != nil {
		return nil, err
	}

	if request.background() {
		return t.startRun(ctx, &params, request)
	}

	response, err := t.shipments.BulkTransferToBilling(
		ctx,
		&serviceports.BulkTransferShipmentToBillingRequest{
			ShipmentIDs:                 request.shipmentIDs,
			BillType:                    request.billType,
			MarkCompletedReadyToInvoice: request.markReady,
		},
		params.Actor,
	)
	if err != nil {
		return nil, err
	}

	return &agent.ToolExecutionResult{
		Action: "transferred",
		Kind:   "shipments",
		Name:   transferOutcome(response),
	}, nil
}

func (t *transferToBillingTool) startRun(
	ctx context.Context,
	params *serviceports.ToolExecuteParams,
	request transferRequest,
) (*agent.ToolExecutionResult, error) {
	run, err := t.runs.Start(ctx, &billingtransferservice.StartRunRequest{
		TenantInfo:                  tenantFrom(*params),
		Scope:                       billingtransfer.RunScopeSelected,
		ShipmentIDs:                 request.shipmentIDs,
		BillType:                    request.billType,
		MarkCompletedReadyToInvoice: request.markReady,
	})
	if err != nil {
		return nil, err
	}

	return &agent.ToolExecutionResult{
		Action: "started",
		Kind:   "billing transfer",
		Name: fmt.Sprintf(
			"A background transfer of %d shipments; the billing queue's transfer panel "+
				"reports each one as it goes",
			len(request.shipmentIDs),
		),
		IDs: map[string]string{resultRunID: run.ID.String()},
	}, nil
}

func transferOutcome(response *serviceports.BulkTransferToBillingResponse) string {
	outcome := fmt.Sprintf(
		"%d of %d transferred",
		response.SuccessCount,
		response.TotalCount,
	)
	failures := make([]string, 0, maxResultFailures)
	for idx := range response.Results {
		result := &response.Results[idx]
		if result.Success {
			continue
		}
		if len(failures) == maxResultFailures {
			break
		}
		label := result.ProNumber
		if label == "" {
			label = result.ShipmentID.String()
		}
		failures = append(failures, label+" "+string(result.FailureCode))
	}
	if len(failures) > 0 {
		outcome += "; refused: " + strings.Join(failures, ", ")
	}

	return outcome
}
