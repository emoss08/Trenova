package paytools

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/settlementpreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/settlement/app/driversettlementservice"
	"github.com/emoss08/trenova/internal/modules/settlement/app/settlementshared"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/pkg/toolschema"
)

const ParamApplyRecurring = "applyRecurring"

type instantPayer interface {
	PlanPayWorkerNow(
		ctx context.Context,
		req *driversettlementservice.PayWorkerNowRequest,
	) (*driversettlementservice.InstantPayPlan, error)
	PayWorkerNow(
		ctx context.Context,
		req *driversettlementservice.PayWorkerNowRequest,
		actor *permissionports.RequestActor,
	) (*driversettlement.Settlement, error)
}

type PayWorkerNowTool struct {
	Settlements instantPayer `json:"-"`
}

func ProvidePayWorkerNowTool(s *driversettlementservice.Service) agenttoolports.AgentTool {
	return &PayWorkerNowTool{Settlements: s}
}

func (t *PayWorkerNowTool) Name() string { return "settlement_pay_now" }

func (t *PayWorkerNowTool) Description() string {
	return "Propose paying a driver now, off cycle, for pay they have accrued. It builds a " +
		"settlement from the chosen pay events, or all accrued and unheld, then approves, " +
		"posts and marks it paid in one step. A person always decides. Recurring " +
		"deductions are left to the regular settlement unless applyRecurring is true."
}

func (t *PayWorkerNowTool) ParamSchema() map[string]any {
	properties := settlementpreview.SettlementPaymentProperties(
		settlementpreview.DriverPaymentMethods,
	)
	properties[toolparams.ParamWorkerID] = agenttoolschema.RecordIDText(permission.ResourceWorker,
		"The driver, from worker_list or settlement_list_pay_events.")
	properties[ParamPayEventIDs] = toolschema.KeepEmpty(agenttoolschema.OfKinds(map[string]any{
		toolschema.KeyType:     toolschema.TypeArray,
		toolschema.KeyMaxItems: maxPayEventsPerAttach,
		toolschema.KeyItems:    map[string]any{toolschema.KeyType: toolschema.TypeString},
		toolschema.KeyDescription: "The accrued pay events to pay, from " + payEventSources +
			". Leave it out to pay everything the driver has accrued and not held.",
	}, recordkind.PayEvent))
	properties[ParamApplyRecurring] = map[string]any{
		toolschema.KeyType: toolschema.TypeBoolean,
		toolschema.KeyDescription: "true to also take recurring deductions, escrow and " +
			"advance recovery now instead of on the regular settlement.",
	}

	return map[string]any{
		toolschema.KeyType:       toolschema.TypeObject,
		toolschema.KeyProperties: properties,
		toolschema.KeyRequired: []string{
			toolparams.ParamWorkerID,
			settlementpreview.ParamPaymentMethod,
		},
		toolschema.KeyAdditionalProperties: false,
	}
}

func (t *PayWorkerNowTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agenttooltrust.ToolKindAction,
		Resource:      permission.ResourceDriverSettlement,
		Operation:     permission.OpApprove,
		Scope:         agenttooltrust.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierPropose,
		Egress:        []agent.EgressClass{agent.EgressDriverVisible, agent.EgressMoney},
		Classify:      settlementpreview.ClassifyAsMoney,
		Effect:        agenttooltrust.ToolEffectChange,
		Artifact:      settlementpreview.DriverSettlementEntity,
		ReadsExternal: agenttooltrust.ExternalReadNever,
		Rationale: "Approves, posts and records a payment to a driver in one step; only a " +
			"person pays someone.",
	}
}

func (t *PayWorkerNowTool) request(
	params *agenttoolports.ToolExecuteParams,
) (*driversettlementservice.PayWorkerNowRequest, error) {
	if err := toolguard.GuardPreview(t, params); err != nil {
		return nil, err
	}
	workerID, err := toolparams.RequirePulid(params.Params, toolparams.ParamWorkerID)
	if err != nil {
		return nil, err
	}
	payment := &settlementshared.ActionRequest{}
	if err = settlementpreview.FillSettlementPayment(
		settlementpreview.DriverPaymentMethods,
	)(
		params.Params,
		payment,
	); err != nil {
		return nil, err
	}
	req := &driversettlementservice.PayWorkerNowRequest{
		TenantInfo:       toolparams.TenantFrom(*params),
		WorkerID:         workerID,
		ApplyRecurring:   toolparams.OptionalBool(params.Params, ParamApplyRecurring),
		PaymentMethod:    payment.PaymentMethod,
		PaymentReference: payment.PaymentReference,
	}
	if _, listed := params.Params[ParamPayEventIDs]; listed {
		if req.PayEventIDs, err = toolparams.RequirePulidSlice(
			params.Params,
			ParamPayEventIDs,
			maxPayEventsPerAttach,
		); err != nil {
			return nil, err
		}
	}

	return req, nil
}

func (t *PayWorkerNowTool) plan(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*driversettlementservice.InstantPayPlan, error) {
	req, err := t.request(params)
	if err != nil {
		return nil, err
	}

	return t.Settlements.PlanPayWorkerNow(ctx, req)
}

func (t *PayWorkerNowTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	plan, err := t.plan(ctx, &params)
	if err != nil {
		return err
	}

	return plan.Refusal
}

func (t *PayWorkerNowTool) ExecuteWithResult(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolResultReporter interface passes params by value
) (*agentrecord.ToolExecutionResult, error) {
	if err := toolguard.GuardExecute(t, params); err != nil {
		return nil, err
	}
	if !params.ApprovedFromProposal() {
		return nil, settlementpreview.ErrSettlementNeedsAPerson
	}
	req, err := t.request(&params)
	if err != nil {
		return nil, err
	}
	paid, err := t.Settlements.PayWorkerNow(ctx, req, params.Actor)
	if err != nil {
		return nil, err
	}

	return settlementpreview.SettlementResult(paid), nil
}

func (t *PayWorkerNowTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	_, err := t.ExecuteWithResult(ctx, params)

	return err
}

func (t *PayWorkerNowTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agentpreview.ToolPreview, error) {
	plan, err := t.plan(ctx, &params)
	if err != nil {
		return nil, err
	}

	summary := "Would pay the driver now, off cycle."
	if plan.Refusal != nil {
		return changepreview.WouldFail(toolpreview.Build(summary), plan.Refusal)
	}

	settlement := plan.Settlement
	summary = fmt.Sprintf(
		"Would pay the driver now for %s: a settlement is generated, approved, posted and "+
			"marked paid by %s in one step, and the driver is told in the driver portal.",
		changepreview.CountOf(settlement.ShipmentCount, "load"),
		settlement.PaymentMethod,
	)
	change, err := settlementpreview.NewSettlementChange(settlement, "Off-cycle driver settlement")
	if err != nil {
		return nil, err
	}
	changes := []*agentpreview.RecordChange{change}
	if plan.Journal != nil {
		journal, journalErr := settlementpreview.SettlementJournalChange(
			plan.Journal,
			settlementpreview.DriverSettlementFacts(settlement),
		)
		if journalErr != nil {
			return nil, journalErr
		}
		changes = append(changes, journal)
		summary += " " + settlementpreview.JournalSentence(plan.Journal)
	}

	return toolpreview.Build(summary, changes...), nil
}
