package settlementtools

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
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
	ledger   common.SettlementLedger[E]
	book     SettlementBook[E]
}

func (t *SettlementDecisionTool[E]) Name() string { return t.decision.name }

func (t *SettlementDecisionTool[E]) Description() string { return t.decision.description }

func (t *SettlementDecisionTool[E]) SearchTerms() []string { return t.decision.searchTerms }

func (t *SettlementDecisionTool[E]) Prerequisites() []string { return t.decision.prerequisites }

func (t *SettlementDecisionTool[E]) ParamSchema() map[string]any {
	properties := map[string]any{
		common.ParamSettlementID: agenttoolschema.KindID(
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
			[]string{common.ParamSettlementID},
			t.decision.required...),
		toolschema.KeyAdditionalProperties: false,
	}
}

func (t *SettlementDecisionTool[E]) Policy() agentports.ToolPolicy {
	policy := agentports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      t.ledger.Resource,
		Operation:     t.decision.operation,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   t.decision.defaultTo,
		MaxTier:       t.decision.maxTier,
		Egress:        t.decision.egress,
		Effect:        agent.ToolEffectChange,
		Reversible:    t.decision.reversible,
		ReadsExternal: agent.ExternalReadNever,
		Rationale:     t.decision.rationale,
	}
	if len(t.decision.egress) > 1 {
		policy.Classify = common.ClassifyAsMoney
	}
	switch t.decision.bound {
	case addedLineBound:
		policy.Condition = &agentports.TierCondition{
			Description: addedLineBoundText,
			Bounds:      t.withinBounds,
		}
	case removedLineBound:
		policy.Condition = &agentports.TierCondition{
			Description: removedLineBoundText,
			Bounds:      t.withinBounds,
		}
	case unbounded:
	}
	if t.decision.holdsWhenTainted {
		policy.TaintHold = &agentports.TaintHold{
			Description: "Its text is what payroll or the payee reads next, so a run that has " +
				"read outside text proposes it rather than writing it.",
			Applies: func(agentports.ToolExecuteParams) bool { return true },
		}
	}

	return policy
}

func (t *SettlementDecisionTool[E]) Target(params map[string]any) (agentports.ToolTarget, bool) {
	return common.TargetOf(params, common.ParamSettlementID, t.ledger.Resource)
}

func (t *SettlementDecisionTool[E]) request(
	params *agentports.ToolExecuteParams,
) (*settlementshared.ActionRequest, error) {
	if err := common.GuardPreview(t, params); err != nil {
		return nil, err
	}

	settlementID, err := common.RequirePulid(params.Params, common.ParamSettlementID)
	if err != nil {
		return nil, err
	}

	req := &settlementshared.ActionRequest{
		TenantInfo:   common.TenantFrom(*params),
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
	params *agentports.ToolExecuteParams,
) (*settlementshared.ActionPlan[*E], error) {
	req, err := t.request(params)
	if err != nil {
		return nil, err
	}

	return t.book.PlanAction(ctx, req)
}

func (t *SettlementDecisionTool[E]) withinBounds(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the TierCondition bounds pass params by value
) agentports.MoneyBound {
	if t.book == nil {
		return agentports.OutsideMoneyBound(agent.HeldByMoneyUnproven)
	}
	req, err := t.request(&params)
	if err != nil {
		return agentports.OutsideMoneyBound(agent.HeldByMoneyUnproven)
	}
	plan, err := t.book.PlanAction(ctx, req)
	if err != nil || plan == nil || plan.Before == nil || t.ledger.Editable == nil {
		return agentports.OutsideMoneyBound(agent.HeldByMoneyUnproven)
	}
	if !t.ledger.Editable(plan.Before) {
		return agentports.OutsideMoneyBound(agent.HeldByMoneyRecordState)
	}
	if plan.Refused() {
		return agentports.OutsideMoneyBound(agent.HeldByMoneyUnproven)
	}
	if t.ledger.Facts == nil {
		return agentports.OutsideMoneyBound(agent.HeldByMoneyUnproven)
	}
	facts := t.ledger.Facts(plan.Before)
	if facts == nil {
		return agentports.OutsideMoneyBound(agent.HeldByMoneyUnproven)
	}

	switch t.decision.bound {
	case addedLineBound:
		if req.Adjustment == nil {
			return agentports.OutsideMoneyBound(agent.HeldByMoneyUnproven)
		}
		return agentports.WithinMoneyBound(req.Adjustment.AmountMinor, facts.Currency)
	case removedLineBound:
		if t.ledger.Line == nil {
			return agentports.OutsideMoneyBound(agent.HeldByMoneyUnproven)
		}
		line, found := t.ledger.Line(plan.Before, req.LineID)
		if !found || !line.Manual {
			return agentports.OutsideMoneyBound(agent.HeldByMoneyUnproven)
		}
		if !line.AddedByAgent {
			return agentports.OutsideMoneyBound(agent.HeldByMoneyAgentLine)
		}
		return agentports.WithinMoneyBound(line.AmountMinor, facts.Currency)
	case unbounded:
	}

	return agentports.OutsideMoneyBound(agent.HeldByMoneyUnproven)
}

func (t *SettlementDecisionTool[E]) Validate(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	plan, err := t.plan(ctx, &params)
	if err != nil {
		return err
	}

	return plan.Refusal
}

func (t *SettlementDecisionTool[E]) Execute(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := common.GuardExecute(t, params); err != nil {
		return err
	}
	if t.decision.personOnly && !params.ApprovedFromProposal() {
		return common.ErrSettlementNeedsAPerson
	}

	req, err := t.request(&params)
	if err != nil {
		return err
	}

	_, err = t.book.Perform(ctx, req, params.Actor)

	return err
}

func fillSettlementReason(params map[string]any, req *settlementshared.ActionRequest) error {
	reason, err := common.BoundedString(params, paramSettlementReason, maxSettlementReason, true)
	if err != nil {
		return err
	}
	req.Reason = reason

	return nil
}

func settlementReasonProperty(description string) map[string]any {
	return common.StringProperty(description, maxSettlementReason)
}
