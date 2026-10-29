package settlementtools

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/settlementpreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/settlement/app/settlementshared"
	"github.com/emoss08/trenova/pkg/toolschema"
)

const paramSettlementReason = "reason"

const maxSettlementReason = 500

type SettlementBook[E any] interface {
	PlanAction(
		ctx context.Context,
		req *settlementshared.ActionRequest,
	) (*settlementshared.ActionPlan[*E], error)
	Perform(
		ctx context.Context,
		req *settlementshared.ActionRequest,
		actor *permissionports.RequestActor,
	) (*E, error)
}

type adjustmentBound int

const (
	unbounded adjustmentBound = iota
	addedLineBound
	removedLineBound
)

const addedLineBoundText = "Runs on its own only while the settlement is not yet approved and the " +
	"line is within the organization's money limits for agents; any other call waits for a person."

const removedLineBoundText = "Runs on its own only when an agent added the line, the settlement " +
	"is not yet approved and the line is within the organization's money limits for agents; any " +
	"other call waits for a person."

type settlementDecision struct {
	name             string
	description      string
	action           settlementshared.Action
	operation        permission.Operation
	personOnly       bool
	egress           []agent.EgressClass
	defaultTo        agent.AutonomyTier
	maxTier          agent.AutonomyTier
	reversible       bool
	holdsWhenTainted bool
	bound            adjustmentBound
	rationale        string
	properties       map[string]any
	required         []string
	fill             func(params map[string]any, req *settlementshared.ActionRequest) error
	fields           []string
	volatile         []string
	refs             map[string]permission.Resource
	searchTerms      []string
	prerequisites    []string
}

type SettlementDecisionTool[E any] struct {
	decision settlementDecision
	ledger   settlementpreview.SettlementLedger[E]
	book     SettlementBook[E]
}

func (t *SettlementDecisionTool[E]) Name() string { return t.decision.name }

func (t *SettlementDecisionTool[E]) Description() string { return t.decision.description }

func (t *SettlementDecisionTool[E]) SearchTerms() []string { return t.decision.searchTerms }

func (t *SettlementDecisionTool[E]) Prerequisites() []string { return t.decision.prerequisites }

func (t *SettlementDecisionTool[E]) ParamSchema() map[string]any {
	properties := map[string]any{
		settlementpreview.ParamSettlementID: agenttoolschema.KindID(
			"The "+t.ledger.Noun+", from "+t.ledger.Sources+" or this run's subject. "+
				"Never guess one."+t.ledger.ByName,
			t.ledger.Kind,
		),
	}
	for name, property := range t.decision.properties {
		properties[name] = property
	}

	return map[string]any{
		toolschema.KeyType:       toolschema.TypeObject,
		toolschema.KeyProperties: properties,
		toolschema.KeyRequired: append(
			[]string{settlementpreview.ParamSettlementID},
			t.decision.required...),
		toolschema.KeyAdditionalProperties: false,
	}
}

func (t *SettlementDecisionTool[E]) Policy() agenttoolports.ToolPolicy {
	policy := agenttoolports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agenttooltrust.ToolKindAction,
		Resource:      t.ledger.Resource,
		Operation:     t.decision.operation,
		Scope:         agenttooltrust.ToolScopeTenant,
		DefaultTier:   t.decision.defaultTo,
		MaxTier:       t.decision.maxTier,
		Egress:        t.decision.egress,
		Effect:        agenttooltrust.ToolEffectChange,
		Reversible:    t.decision.reversible,
		ReadsExternal: agenttooltrust.ExternalReadNever,
		Rationale:     t.decision.rationale,
	}
	if len(t.decision.egress) > 1 {
		policy.Classify = settlementpreview.ClassifyAsMoney
	}
	switch t.decision.bound {
	case addedLineBound:
		policy.Condition = &agenttoolports.TierCondition{
			Description: addedLineBoundText,
			Bounds:      t.withinBounds,
		}
	case removedLineBound:
		policy.Condition = &agenttoolports.TierCondition{
			Description: removedLineBoundText,
			Bounds:      t.withinBounds,
		}
	case unbounded:
	}
	if t.decision.holdsWhenTainted {
		policy.TaintHold = &agenttoolports.TaintHold{
			Description: "Its text is what payroll or the payee reads next, so a run that has " +
				"read outside text proposes it rather than writing it.",
			Applies: func(agenttoolports.ToolExecuteParams) bool { return true },
		}
	}

	return policy
}

func (t *SettlementDecisionTool[E]) Target(
	params map[string]any,
) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(params, settlementpreview.ParamSettlementID, t.ledger.Resource)
}

func (t *SettlementDecisionTool[E]) request(
	params *agenttoolports.ToolExecuteParams,
) (*settlementshared.ActionRequest, error) {
	if err := toolguard.GuardPreview(t, params); err != nil {
		return nil, err
	}

	settlementID, err := toolparams.RequirePulid(params.Params, settlementpreview.ParamSettlementID)
	if err != nil {
		return nil, err
	}

	req := &settlementshared.ActionRequest{
		TenantInfo:   toolparams.TenantFrom(*params),
		SettlementID: settlementID,
		Action:       t.decision.action,
	}
	if t.decision.fill != nil {
		if err = t.decision.fill(params.Params, req); err != nil {
			return nil, err
		}
	}

	return req, nil
}

func (t *SettlementDecisionTool[E]) plan(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*settlementshared.ActionPlan[*E], error) {
	req, err := t.request(params)
	if err != nil {
		return nil, err
	}

	return t.book.PlanAction(ctx, req)
}

func (t *SettlementDecisionTool[E]) withinBounds(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the TierCondition bounds pass params by value
) agenttoolports.MoneyBound {
	if t.book == nil {
		return agenttoolports.OutsideMoneyBound(agenttooltrust.HeldByMoneyUnproven)
	}
	req, err := t.request(&params)
	if err != nil {
		return agenttoolports.OutsideMoneyBound(agenttooltrust.HeldByMoneyUnproven)
	}
	plan, err := t.book.PlanAction(ctx, req)
	if err != nil || plan == nil || plan.Before == nil || t.ledger.Editable == nil {
		return agenttoolports.OutsideMoneyBound(agenttooltrust.HeldByMoneyUnproven)
	}
	if !t.ledger.Editable(plan.Before) {
		return agenttoolports.OutsideMoneyBound(agenttooltrust.HeldByMoneyRecordState)
	}
	if plan.Refused() {
		return agenttoolports.OutsideMoneyBound(agenttooltrust.HeldByMoneyUnproven)
	}
	if t.ledger.Facts == nil {
		return agenttoolports.OutsideMoneyBound(agenttooltrust.HeldByMoneyUnproven)
	}
	facts := t.ledger.Facts(plan.Before)
	if facts == nil {
		return agenttoolports.OutsideMoneyBound(agenttooltrust.HeldByMoneyUnproven)
	}

	switch t.decision.bound {
	case addedLineBound:
		if req.Adjustment == nil {
			return agenttoolports.OutsideMoneyBound(agenttooltrust.HeldByMoneyUnproven)
		}
		return agenttoolports.WithinMoneyBound(req.Adjustment.AmountMinor, facts.Currency)
	case removedLineBound:
		if t.ledger.Line == nil {
			return agenttoolports.OutsideMoneyBound(agenttooltrust.HeldByMoneyUnproven)
		}
		line, found := t.ledger.Line(plan.Before, req.LineID)
		if !found || !line.Manual {
			return agenttoolports.OutsideMoneyBound(agenttooltrust.HeldByMoneyUnproven)
		}
		if !line.AddedByAgent {
			return agenttoolports.OutsideMoneyBound(agenttooltrust.HeldByMoneyAgentLine)
		}
		return agenttoolports.WithinMoneyBound(line.AmountMinor, facts.Currency)
	case unbounded:
	}

	return agenttoolports.OutsideMoneyBound(agenttooltrust.HeldByMoneyUnproven)
}

func (t *SettlementDecisionTool[E]) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	plan, err := t.plan(ctx, &params)
	if err != nil {
		return err
	}

	return plan.Refusal
}

func (t *SettlementDecisionTool[E]) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := toolguard.GuardExecute(t, params); err != nil {
		return err
	}
	if t.decision.personOnly && !params.ApprovedFromProposal() {
		return settlementpreview.ErrSettlementNeedsAPerson
	}

	req, err := t.request(&params)
	if err != nil {
		return err
	}

	_, err = t.book.Perform(ctx, req, params.Actor)

	return err
}

func fillSettlementReason(params map[string]any, req *settlementshared.ActionRequest) error {
	reason, err := toolparams.BoundedString(
		params,
		paramSettlementReason,
		maxSettlementReason,
		true,
	)
	if err != nil {
		return err
	}
	req.Reason = reason

	return nil
}

func settlementReasonProperty(description string) map[string]any {
	return toolparams.StringProperty(description, maxSettlementReason)
}
