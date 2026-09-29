package agenttoolservice

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/billingtransfer"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolschema"
	"github.com/emoss08/trenova/internal/core/services/billingtransfercriteria"
	"github.com/emoss08/trenova/internal/core/services/billingtransferservice"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
)

const (
	paramAllTransferable    = "allTransferable"
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
	ListBillingTransferCandidateIDs(
		ctx context.Context,
		req *serviceports.ListBillingTransferCandidateIDsRequest,
	) (*serviceports.BillingTransferCandidateIDsResponse, error)
}

var errUnapprovedTransferSelection = errors.New(
	"transfer_to_billing runs only on the shipments a person approved, and this call names " +
		"none; propose it again so the shipments it resolves to are listed",
)

type billingTransferRunStarter interface {
	Start(
		ctx context.Context,
		req *billingtransferservice.StartRunRequest,
	) (*billingtransfer.BillingTransferRun, error)
}

var (
	_ serviceports.ToolPreviewer         = (*transferToBillingTool)(nil)
	_ serviceports.ToolValidator         = (*transferToBillingTool)(nil)
	_ serviceports.ToolResultReporter    = (*transferToBillingTool)(nil)
	_ serviceports.ToolSelectionResolver = (*transferToBillingTool)(nil)
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
		"dialog does: each becomes a queue item a biller reviews. When the person means every " +
		"shipment that can go (all of them, everything ready), set allTransferable to true, " +
		"with the same filters list_billing_transfer_candidates takes, instead of copying ids: " +
		"the proposal lists each shipment it resolves to. For a hand-picked set, give " +
		"shipmentIds from list_billing_transfer_candidates. Cover every shipment in one call " +
		"rather than one call per shipment; a shipment the checks refuse is reported, not " +
		"transferred, and the rest still go. The person approving may untick shipments. " +
		"Up to 100 transfer at once; more run as a background transfer."
}

func (t *transferToBillingTool) ParamSchema() map[string]any {
	properties := billingtransfercriteria.Properties()
	properties[paramAllTransferable] = map[string]any{
		toolschema.KeyType: toolschema.TypeBoolean,
		toolschema.KeyDescription: "True to transfer every shipment " +
			"list_billing_transfer_candidates says would transfer, narrowed by query, status, " +
			"customerId, deliveredFrom and deliveredTo. Use it instead of shipmentIds, never " +
			"with them.",
	}
	properties[paramShipmentIDs] = toolschema.RecordSubset(permission.ResourceShipment.String(),
		map[string]any{
			toolschema.KeyType: toolschema.TypeArray,
			toolschema.KeyDescription: "The shipments to transfer, by id from " +
				"list_billing_transfer_candidates, when not allTransferable. Never guess one.",
			toolschema.KeyMinItems: 1,
			toolschema.KeyMaxItems: serviceports.MaxBillingTransferCandidateIDs,
			toolschema.KeyItems: map[string]any{
				toolschema.KeyType: toolschema.TypeString,
			},
		})
	properties[paramBillType] = agenttoolschema.Enum(
		"What the queue items bill. Defaults to Invoice.", transferBillTypes,
	)
	properties[paramMarkCompletedReady] = map[string]any{
		toolschema.KeyType: toolschema.TypeBoolean,
		toolschema.KeyDescription: "Mark Completed shipments ready to invoice first, as " +
			"the dialog can. Defaults to false, which leaves a Completed shipment where " +
			"it is.",
	}

	return map[string]any{
		toolschema.KeyType:                 toolschema.TypeObject,
		toolschema.KeyProperties:           properties,
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

var transferBillTypes = agenttoolschema.Source("billingQueue.billType", []billingqueue.BillType{
	billingqueue.BillTypeInvoice,
	billingqueue.BillTypeCreditMemo,
	billingqueue.BillTypeDebitMemo,
})

type transferRequest struct {
	shipmentIDs []pulid.ID
	billType    billingqueue.BillType
	markReady   bool
	all         bool
	criteria    billingtransfercriteria.Criteria
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

	request := transferRequest{
		billType:  billingqueue.BillTypeInvoice,
		markReady: optionalBool(params.Params, paramMarkCompletedReady),
		all:       optionalBool(params.Params, paramAllTransferable),
	}
	if raw := optionalString(params.Params, paramBillType); raw != "" {
		request.billType = billingqueue.BillType(raw)
		if !isBillType(request.billType) {
			return transferRequest{}, fmt.Errorf(
				"billType %q is not one of %s", raw, strings.Join(transferBillTypes.Names(), ", "),
			)
		}
	}

	named := params.Params[paramShipmentIDs] != nil
	filters := billingtransfercriteria.Given(params.Params)
	switch {
	case named && request.all:
		return transferRequest{}, errors.New(
			"give shipmentIds or allTransferable, not both",
		)
	case !named && !request.all:
		return transferRequest{}, errors.New(
			"name the shipments in shipmentIds, or set allTransferable to true to transfer " +
				"every shipment list_billing_transfer_candidates says would transfer",
		)
	case named && len(filters) > 0:
		return transferRequest{}, fmt.Errorf(
			"%s only narrow allTransferable; leave them out when naming shipmentIds",
			strings.Join(filters, ", "),
		)
	}

	if request.all {
		criteria, err := billingtransfercriteria.Read(
			params.Params, filtercatalog.NewClock(params.Timezone),
		)
		if err != nil {
			return transferRequest{}, err
		}
		request.criteria = criteria

		return request, nil
	}

	ids, err := requirePulidSlice(
		params.Params,
		paramShipmentIDs,
		serviceports.MaxBillingTransferCandidateIDs,
	)
	if err != nil {
		return transferRequest{}, err
	}
	request.shipmentIDs = sliceutils.Dedupe(ids)

	return request, nil
}

func (t *transferToBillingTool) ResolveSelection(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolSelectionResolver interface passes params by value
) (map[string]any, error) {
	request, err := t.request(&params)
	if err != nil {
		return nil, err
	}
	if !request.all {
		return params.Params, nil
	}

	ids, _, err := t.transferable(ctx, &params, &request)
	if err != nil {
		return nil, err
	}

	resolved := make(map[string]any, len(params.Params)+1)
	for key, value := range params.Params {
		if key == paramAllTransferable || billingtransfercriteria.IsParam(key) {
			continue
		}
		resolved[key] = value
	}
	named := make([]any, 0, len(ids))
	for _, id := range ids {
		named = append(named, id.String())
	}
	resolved[paramShipmentIDs] = named

	return resolved, nil
}

func (t *transferToBillingTool) transferable(
	ctx context.Context,
	params *serviceports.ToolExecuteParams,
	request *transferRequest,
) ([]pulid.ID, *serviceports.BillingTransferPlan, error) {
	tenant := tenantFrom(*params)
	candidates, err := t.shipments.ListBillingTransferCandidateIDs(
		ctx,
		&serviceports.ListBillingTransferCandidateIDsRequest{
			Filter: request.criteria.QueryOptions(tenant, pagination.Info{}),
			Status: request.criteria.Status,
		},
	)
	if err != nil {
		return nil, nil, err
	}
	if candidates.Truncated {
		return nil, nil, fmt.Errorf(
			"%d shipments match, more than one transfer takes (%d); narrow the selection "+
				"with status, customerId, query or a delivery window",
			candidates.TotalCount, serviceports.MaxBillingTransferCandidateIDs,
		)
	}
	if len(candidates.IDs) == 0 {
		return nil, nil, errors.New(
			"no shipment outside the billing queue matches; list_billing_transfer_candidates " +
				"with the same filters shows what is waiting",
		)
	}

	plan, err := t.shipments.PlanBillingTransfers(ctx, &serviceports.PlanBillingTransfersRequest{
		TenantInfo:                  tenant,
		ShipmentIDs:                 candidates.IDs,
		MarkCompletedReadyToInvoice: request.markReady,
	})
	if err != nil {
		return nil, nil, err
	}

	going := &serviceports.BillingTransferPlan{
		Decisions: make([]serviceports.BillingTransferDecision, 0, plan.Transfer),
	}
	ids := make([]pulid.ID, 0, plan.Transfer)
	for idx := range plan.Decisions {
		decision := &plan.Decisions[idx]
		if !decision.Outcome.Transfers() {
			continue
		}
		ids = append(ids, decision.ShipmentID)
		going.Decisions = append(going.Decisions, *decision)
	}
	going.Transfer = len(ids)
	if len(ids) == 0 {
		return nil, nil, fmt.Errorf(
			"none of the %d shipments these filters select would transfer as it stands "+
				"(%d refused, %d left with operations); list_billing_transfer_candidates says "+
				"why each one is held",
			len(plan.Decisions), plan.Refused, plan.Returned,
		)
	}

	return ids, going, nil
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
	if request.all {
		return nil, errUnapprovedTransferSelection
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
