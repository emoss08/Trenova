package billingtools

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/billing/app/billingtransfercriteria"
	"github.com/emoss08/trenova/internal/modules/billing/app/billingtransferservice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingtransfer"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
)

const paramAllTransferable = "allTransferable"

const paramMarkCompletedReady = "markCompletedReadyToInvoice"

const resultRunID = "billingTransferRunId"

// billingQueueRecordEntity is the record-link registry's name for a
// billing queue item, which a transfer makes one of per payer.
const billingQueueRecordEntity = "billing_queue_item"

const maxResultFailures = 3

type billingTransferPlanner interface {
	PlanBillingTransfers(
		ctx context.Context,
		req *shipmentports.PlanBillingTransfersRequest,
	) (*shipmentports.BillingTransferPlan, error)
	BulkTransferToBilling(
		ctx context.Context,
		req *shipmentports.BulkTransferShipmentToBillingRequest,
		actor *permissionports.RequestActor,
	) (*shipmentports.BulkTransferToBillingResponse, error)
	ListBillingTransferCandidateIDs(
		ctx context.Context,
		req *shipmentports.ListBillingTransferCandidateIDsRequest,
	) (*shipmentports.BillingTransferCandidateIDsResponse, error)
}

// errBothSelections answers a call that names shipments and also asks for
// every transferable one. The two say different things when the filters match
// more than the named shipments, so the tool cannot pick; the refusal spells
// out both calls so the next one is the one the person meant.
var errBothSelections = errors.New(
	"give shipmentIds or allTransferable, not both. To transfer exactly the shipments " +
		"you named, send shipmentIds without allTransferable, for example " +
		`{"shipmentIds": ["shp_…"], "markCompletedReadyToInvoice": true}. To transfer ` +
		"every shipment the filters match, send allTransferable true with the filters " +
		"and no shipmentIds",
)

var errUnapprovedTransferSelection = errors.New(
	"billing_transfer runs only on the shipments a person approved, and this call names " +
		"none; propose it again so the shipments it resolves to are listed",
)

type billingTransferRunStarter interface {
	Start(
		ctx context.Context,
		req *billingtransferservice.StartRunRequest,
	) (*billingtransfer.BillingTransferRun, error)
}

// transferToBillingTool queues delivered shipments for billing. A set the
// synchronous transfer takes is transferred in the call; a larger one is
// handed to the background transfer run the dialog uses for the same size.
type TransferToBillingTool struct {
	shipments billingTransferPlanner
	runs      billingTransferRunStarter
}

func NewTransferToBillingTool(
	shipments billingTransferPlanner,
	runs billingTransferRunStarter,
) agenttoolports.AgentTool {
	return &TransferToBillingTool{shipments: shipments, runs: runs}
}

func (t *TransferToBillingTool) Name() string { return "billing_transfer" }

func (t *TransferToBillingTool) Recipe() []string {
	return []string{"billing_list_transfer_candidates", "billing_transfer"}
}

func (t *TransferToBillingTool) SearchTerms() []string {
	return []string{"send to billing"}
}

func (t *TransferToBillingTool) Description() string {
	return "Transfer delivered shipments to the billing queue, as the transfer-to-billing " +
		"dialog does: each becomes a queue item a biller reviews. When the person means every " +
		"shipment that can go (all of them, everything ready), set allTransferable to true, " +
		"with the same filters billing_list_transfer_candidates takes, instead of copying ids: " +
		"the proposal lists each shipment it resolves to. For a hand-picked set, give " +
		"shipmentIds from billing_list_transfer_candidates. Cover every shipment in one call " +
		"rather than one call per shipment; a shipment the checks refuse is reported, not " +
		"transferred, and the rest still go. The person approving may untick shipments. " +
		"Up to 100 transfer at once; more run as a background transfer."
}

func (t *TransferToBillingTool) ParamSchema() map[string]any {
	properties := billingtransfercriteria.Properties()
	properties[paramAllTransferable] = map[string]any{
		toolschema.KeyType: toolschema.TypeBoolean,
		toolschema.KeyDescription: "True to transfer every shipment " +
			"billing_list_transfer_candidates says would transfer, narrowed by query, status, " +
			"customerId, deliveredFrom and deliveredTo. Use it instead of shipmentIds, never " +
			"with them; leave it out when naming shipmentIds, whose filters are then ignored.",
	}
	properties[common.ParamShipmentIDs] = toolschema.RecordSubset(
		permission.ResourceShipment.String(),
		agenttoolschema.OfResource(map[string]any{
			toolschema.KeyType: toolschema.TypeArray,
			toolschema.KeyDescription: "The shipments to transfer, by id from " +
				"billing_list_transfer_candidates, when not allTransferable. Never guess one.",
			toolschema.KeyMinItems: 1,
			toolschema.KeyMaxItems: shipmentports.MaxBillingTransferCandidateIDs,
			toolschema.KeyItems: map[string]any{
				toolschema.KeyType: toolschema.TypeString,
			},
		}, permission.ResourceShipment),
	)
	properties[common.ParamBillType] = agenttoolschema.Enum(
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

func (t *TransferToBillingTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
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
	return len(r.shipmentIDs) > shipmentports.MaxBulkTransferToBillingShipments
}

func (t *TransferToBillingTool) request(
	params *agenttoolports.ToolExecuteParams,
) (transferRequest, error) {
	if err := common.GuardPreview(t, params); err != nil {
		return transferRequest{}, err
	}

	request := transferRequest{
		billType:  billingqueue.BillTypeInvoice,
		markReady: common.OptionalBool(params.Params, paramMarkCompletedReady),
		all:       common.OptionalBool(params.Params, paramAllTransferable),
	}
	if raw := common.OptionalString(params.Params, common.ParamBillType); raw != "" {
		request.billType = billingqueue.BillType(raw)
		if !isBillType(request.billType) {
			return transferRequest{}, fmt.Errorf(
				"billType %q is not one of %s", raw, strings.Join(transferBillTypes.Names(), ", "),
			)
		}
	}

	named := params.Params[common.ParamShipmentIDs] != nil
	switch {
	case named && request.all:
		return transferRequest{}, errBothSelections
	case !named && !request.all:
		return transferRequest{}, errors.New(
			"name the shipments in shipmentIds, or set allTransferable to true to transfer " +
				"every shipment billing_list_transfer_candidates says would transfer",
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

	ids, err := common.RequirePulidSlice(
		params.Params,
		common.ParamShipmentIDs,
		shipmentports.MaxBillingTransferCandidateIDs,
	)
	if err != nil {
		return transferRequest{}, err
	}
	request.shipmentIDs = sliceutils.Dedupe(ids)

	return request, nil
}

func (t *TransferToBillingTool) ResolveSelection(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolSelectionResolver interface passes params by value
) (map[string]any, error) {
	request, err := t.request(&params)
	if err != nil {
		return nil, err
	}
	if !request.all {
		return withoutCriteria(params.Params), nil
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
	resolved[common.ParamShipmentIDs] = named

	return resolved, nil
}

// withoutCriteria is a call that names its shipments, with the filters that
// only narrow allTransferable taken off. A model that looked the shipment up
// with billing_list_transfer_candidates sends the same query and status back
// beside the id; they cannot widen what the ids name, so they are set aside
// rather than refused, and the proposal a person reads holds only the ids.
func withoutCriteria(params map[string]any) map[string]any {
	if len(billingtransfercriteria.Given(params)) == 0 {
		return params
	}
	kept := make(map[string]any, len(params))
	for key, value := range params {
		if billingtransfercriteria.IsParam(key) {
			continue
		}
		kept[key] = value
	}

	return kept
}

func (t *TransferToBillingTool) transferable(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
	request *transferRequest,
) ([]pulid.ID, *shipmentports.BillingTransferPlan, error) {
	tenant := common.TenantFrom(*params)
	candidates, err := t.shipments.ListBillingTransferCandidateIDs(
		ctx,
		&shipmentports.ListBillingTransferCandidateIDsRequest{
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
			candidates.TotalCount, shipmentports.MaxBillingTransferCandidateIDs,
		)
	}
	if len(candidates.IDs) == 0 {
		return nil, nil, errors.New(
			"no shipment outside the billing queue matches; billing_list_transfer_candidates " +
				"with the same filters shows what is waiting",
		)
	}

	plan, err := t.shipments.PlanBillingTransfers(ctx, &shipmentports.PlanBillingTransfersRequest{
		TenantInfo:                  tenant,
		ShipmentIDs:                 candidates.IDs,
		MarkCompletedReadyToInvoice: request.markReady,
	})
	if err != nil {
		return nil, nil, err
	}

	going := &shipmentports.BillingTransferPlan{
		Decisions: make([]shipmentports.BillingTransferDecision, 0, plan.Transfer),
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
				"(%d refused, %d left with operations); billing_list_transfer_candidates says "+
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

func (t *TransferToBillingTool) Validate(
	_ context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	_, err := t.request(&params)

	return err
}

func (t *TransferToBillingTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	_, err := t.ExecuteWithResult(ctx, params)

	return err
}

// ExecuteWithResult transfers the shipments and says what became of them: the
// count of each outcome and the first refusals, or the background run a large
// set was handed to.
func (t *TransferToBillingTool) ExecuteWithResult(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolResultReporter interface passes params by value
) (*agent.ToolExecutionResult, error) {
	if err := common.GuardExecute(t, params); err != nil {
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
		&shipmentports.BulkTransferShipmentToBillingRequest{
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

func (t *TransferToBillingTool) startRun(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
	request transferRequest,
) (*agent.ToolExecutionResult, error) {
	run, err := t.runs.Start(ctx, &billingtransferservice.StartRunRequest{
		TenantInfo:                  common.TenantFrom(*params),
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

func transferOutcome(response *shipmentports.BulkTransferToBillingResponse) string {
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
