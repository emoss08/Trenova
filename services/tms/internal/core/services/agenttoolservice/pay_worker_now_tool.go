package agenttoolservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/driversettlement"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/driversettlementservice"
	"github.com/emoss08/trenova/internal/core/services/settlementshared"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/pkg/toolschema"
)

const paramApplyRecurring = "applyRecurring"

type instantPayer interface {
	PlanPayWorkerNow(
		ctx context.Context,
		req *driversettlementservice.PayWorkerNowRequest,
	) (*driversettlementservice.InstantPayPlan, error)
	PayWorkerNow(
		ctx context.Context,
		req *driversettlementservice.PayWorkerNowRequest,
		actor *serviceports.RequestActor,
	) (*driversettlement.Settlement, error)
}

type payWorkerNowTool struct {
	settlements instantPayer
}

var (
	_ serviceports.ToolPreviewer      = (*payWorkerNowTool)(nil)
	_ serviceports.ToolValidator      = (*payWorkerNowTool)(nil)
	_ serviceports.ToolResultReporter = (*payWorkerNowTool)(nil)
)

func providePayWorkerNowTool(s *driversettlementservice.Service) serviceports.AgentTool {
	return &payWorkerNowTool{settlements: s}
}

func (t *payWorkerNowTool) Name() string { return "pay_driver_now" }

func (t *payWorkerNowTool) Description() string {
	return "Propose paying a driver now, off cycle, for pay they have accrued. It builds a " +
		"settlement from the chosen pay events, or all accrued and unheld, then approves, " +
		"posts and marks it paid in one step. A person always decides. Recurring " +
		"deductions are left to the regular settlement unless applyRecurring is true."
}

func (t *payWorkerNowTool) ParamSchema() map[string]any {
	properties := settlementPaymentProperties(driverPaymentMethods)
	properties[paramWorkerID] = map[string]any{
		toolschema.KeyType:        toolschema.TypeString,
		toolschema.KeyDescription: "The driver, from search_worker or list_driver_pay_events.",
	}
	properties[paramPayEventIDs] = map[string]any{
		toolschema.KeyType:     toolschema.TypeArray,
		toolschema.KeyMaxItems: maxPayEventsPerAttach,
		toolschema.KeyItems:    map[string]any{toolschema.KeyType: toolschema.TypeString},
		toolschema.KeyDescription: "The accrued pay events to pay, from " + payEventSources +
			". Leave it out to pay everything the driver has accrued and not held.",
	}
	properties[paramApplyRecurring] = map[string]any{
		toolschema.KeyType: toolschema.TypeBoolean,
		toolschema.KeyDescription: "true to also take recurring deductions, escrow and " +
			"advance recovery now instead of on the regular settlement.",
	}

	return map[string]any{
		toolschema.KeyType:                 toolschema.TypeObject,
		toolschema.KeyProperties:           properties,
		toolschema.KeyRequired:             []string{paramWorkerID, paramPaymentMethod},
		toolschema.KeyAdditionalProperties: false,
	}
}

func (t *payWorkerNowTool) Policy() serviceports.ToolPolicy {
	return serviceports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceDriverSettlement,
		Operation:     permission.OpApprove,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierPropose,
		Egress:        []agent.EgressClass{agent.EgressDriverVisible, agent.EgressMoney},
		Classify:      classifyAsMoney,
		Effect:        agent.ToolEffectChange,
		Artifact:      driverSettlementEntity,
		ReadsExternal: agent.ExternalReadNever,
		Rationale: "Approves, posts and records a payment to a driver in one step; only a " +
			"person pays someone.",
	}
}

func (t *payWorkerNowTool) request(
	params *serviceports.ToolExecuteParams,
) (*driversettlementservice.PayWorkerNowRequest, error) {
	if err := guardPreview(t, params); err != nil {
		return nil, err
	}
	workerID, err := requirePulid(params.Params, paramWorkerID)
	if err != nil {
		return nil, err
	}
	payment := &settlementshared.ActionRequest{}
	if err = fillSettlementPayment(driverPaymentMethods)(params.Params, payment); err != nil {
		return nil, err
	}
	req := &driversettlementservice.PayWorkerNowRequest{
		TenantInfo:       tenantFrom(*params),
		WorkerID:         workerID,
		ApplyRecurring:   optionalBool(params.Params, paramApplyRecurring),
		PaymentMethod:    payment.PaymentMethod,
		PaymentReference: payment.PaymentReference,
	}
	if _, listed := params.Params[paramPayEventIDs]; listed {
		if req.PayEventIDs, err = requirePulidSlice(
			params.Params,
			paramPayEventIDs,
			maxPayEventsPerAttach,
		); err != nil {
			return nil, err
		}
	}

	return req, nil
}

func (t *payWorkerNowTool) plan(
	ctx context.Context,
	params *serviceports.ToolExecuteParams,
) (*driversettlementservice.InstantPayPlan, error) {
	req, err := t.request(params)
	if err != nil {
		return nil, err
	}

	return t.settlements.PlanPayWorkerNow(ctx, req)
}

func (t *payWorkerNowTool) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	plan, err := t.plan(ctx, &params)
	if err != nil {
		return err
	}

	return plan.Refusal
}

func (t *payWorkerNowTool) ExecuteWithResult(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolResultReporter interface passes params by value
) (*agent.ToolExecutionResult, error) {
	if err := guardExecute(t, params); err != nil {
		return nil, err
	}
	if !params.ApprovedFromProposal() {
		return nil, ErrSettlementNeedsAPerson
	}
	req, err := t.request(&params)
	if err != nil {
		return nil, err
	}
	paid, err := t.settlements.PayWorkerNow(ctx, req, params.Actor)
	if err != nil {
		return nil, err
	}

	return settlementResult(paid), nil
}

func (t *payWorkerNowTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	_, err := t.ExecuteWithResult(ctx, params)

	return err
}

func (t *payWorkerNowTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	plan, err := t.plan(ctx, &params)
	if err != nil {
		return nil, err
	}

	summary := "Would pay the driver now, off cycle."
	if plan.Refusal != nil {
		return wouldFail(toolpreview.Build(summary), plan.Refusal)
	}

	settlement := plan.Settlement
	summary = fmt.Sprintf(
		"Would pay the driver now for %s: a settlement is generated, approved, posted and "+
			"marked paid by %s in one step, and the driver is told in the driver portal.",
		countOf(settlement.ShipmentCount, "load"),
		settlement.PaymentMethod,
	)
	change, err := newSettlementChange(settlement, "Off-cycle driver settlement")
	if err != nil {
		return nil, err
	}
	changes := []*agent.RecordChange{change}
	if plan.Journal != nil {
		journal, journalErr := settlementJournalChange(
			plan.Journal,
			driverSettlementFacts(settlement),
		)
		if journalErr != nil {
			return nil, journalErr
		}
		changes = append(changes, journal)
		summary += " " + journalSentence(plan.Journal)
	}

	return toolpreview.Build(summary, changes...), nil
}
