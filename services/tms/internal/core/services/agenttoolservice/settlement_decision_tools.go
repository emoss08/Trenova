package agenttoolservice

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/settlementshared"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	paramSettlementID       = "settlementId"
	paramSettlementReason   = "reason"
	paramPaymentMethod      = "paymentMethod"
	paramPaymentReference   = "paymentReference"
	maxSettlementReason     = 500
	maxPaymentMethodChars   = 50
	maxPaymentReferenceChar = 100
)

var ErrSettlementNeedsAPerson = errors.New(
	"a settlement is approved, posted, voided or marked paid only once a person approves " +
		"the proposal",
)

type settlementBook[E any] interface {
	PlanAction(
		ctx context.Context,
		req *settlementshared.ActionRequest,
	) (*settlementshared.ActionPlan[*E], error)
	Perform(
		ctx context.Context,
		req *settlementshared.ActionRequest,
		actor *serviceports.RequestActor,
	) (*E, error)
}

type settlementTotal struct {
	label string
	minor int64
}

type settlementFacts struct {
	id       pulid.ID
	number   string
	version  int64
	status   string
	currency string
	payee    string
	totals   []settlementTotal
}

type settlementLedger[E any] struct {
	resource  permission.Resource
	noun      string
	sources   string
	sensitive []string
	facts     func(*E) *settlementFacts
	effects   func(action settlementshared.Action, plan *settlementshared.ActionPlan[*E]) []string
}

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
	rationale        string
	properties       map[string]any
	required         []string
	fill             func(params map[string]any, req *settlementshared.ActionRequest) error
	fields           []string
	volatile         []string
	refs             map[string]permission.Resource
}

type settlementDecisionTool[E any] struct {
	decision settlementDecision
	ledger   settlementLedger[E]
	book     settlementBook[E]
}

func (t *settlementDecisionTool[E]) Name() string { return t.decision.name }

func (t *settlementDecisionTool[E]) Description() string { return t.decision.description }

func (t *settlementDecisionTool[E]) ParamSchema() map[string]any {
	properties := map[string]any{
		paramSettlementID: map[string]any{
			toolschema.KeyType: toolschema.TypeString,
			toolschema.KeyDescription: "The " + t.ledger.noun + ", from " + t.ledger.sources +
				" or this run's subject. Never guess one.",
		},
	}
	for name, property := range t.decision.properties {
		properties[name] = property
	}

	return map[string]any{
		toolschema.KeyType:       toolschema.TypeObject,
		toolschema.KeyProperties: properties,
		toolschema.KeyRequired: append(
			[]string{paramSettlementID},
			t.decision.required...),
		toolschema.KeyAdditionalProperties: false,
	}
}

func (t *settlementDecisionTool[E]) Policy() serviceports.ToolPolicy {
	policy := serviceports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      t.ledger.resource,
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
		policy.Classify = classifyAsMoney
	}
	if t.decision.holdsWhenTainted {
		policy.TaintHold = &serviceports.TaintHold{
			Description: "Its text is what payroll or the payee reads next, so a run that has " +
				"read outside text proposes it rather than writing it.",
			Applies: func(serviceports.ToolExecuteParams) bool { return true },
		}
	}

	return policy
}

func classifyAsMoney(
	serviceports.ToolExecuteParams,
) serviceports.CallPolicy {
	return serviceports.CallPolicy{Egress: agent.EgressMoney}
}

func (t *settlementDecisionTool[E]) Target(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, paramSettlementID, t.ledger.resource)
}

func (t *settlementDecisionTool[E]) request(
	params *serviceports.ToolExecuteParams,
) (*settlementshared.ActionRequest, error) {
	if err := guardPreview(t, params); err != nil {
		return nil, err
	}

	settlementID, err := requirePulid(params.Params, paramSettlementID)
	if err != nil {
		return nil, err
	}

	req := &settlementshared.ActionRequest{
		TenantInfo:   tenantFrom(*params),
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

func (t *settlementDecisionTool[E]) plan(
	ctx context.Context,
	params *serviceports.ToolExecuteParams,
) (*settlementshared.ActionPlan[*E], error) {
	req, err := t.request(params)
	if err != nil {
		return nil, err
	}

	return t.book.PlanAction(ctx, req)
}

func (t *settlementDecisionTool[E]) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	plan, err := t.plan(ctx, &params)
	if err != nil {
		return err
	}

	return plan.Refusal
}

func (t *settlementDecisionTool[E]) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := guardExecute(t, params); err != nil {
		return err
	}
	if t.decision.personOnly && !params.ApprovedFromProposal() {
		return ErrSettlementNeedsAPerson
	}

	req, err := t.request(&params)
	if err != nil {
		return err
	}

	_, err = t.book.Perform(ctx, req, params.Actor)

	return err
}

func boundedString(
	params map[string]any,
	key string,
	maxLength int,
	required bool,
) (string, error) {
	value := strings.TrimSpace(optionalString(params, key))
	if value == "" && required {
		if _, err := requireString(params, key); err != nil {
			return "", err
		}
	}
	if len(value) > maxLength {
		return "", fmt.Errorf(
			"%s is %d characters; it keeps at most %d",
			key,
			len(value),
			maxLength,
		)
	}

	return value, nil
}

func fillSettlementReason(params map[string]any, req *settlementshared.ActionRequest) error {
	reason, err := boundedString(params, paramSettlementReason, maxSettlementReason, true)
	if err != nil {
		return err
	}
	req.Reason = reason

	return nil
}

func fillSettlementPayment(
	methods []string,
) func(map[string]any, *settlementshared.ActionRequest) error {
	return func(params map[string]any, req *settlementshared.ActionRequest) error {
		method, err := boundedString(params, paramPaymentMethod, maxPaymentMethodChars, true)
		if err != nil {
			return err
		}
		if !slices.Contains(methods, method) {
			return errUnknownValue(paramPaymentMethod, method, methods)
		}
		reference, err := boundedString(
			params,
			paramPaymentReference,
			maxPaymentReferenceChar,
			false,
		)
		if err != nil {
			return err
		}
		req.PaymentMethod = method
		req.PaymentReference = reference

		return nil
	}
}

func settlementReasonProperty(description string) map[string]any {
	return stringProperty(description, maxSettlementReason)
}

func settlementPaymentProperties(methods []string) map[string]any {
	return map[string]any{
		paramPaymentMethod: map[string]any{
			toolschema.KeyType: toolschema.TypeString,
			toolschema.KeyEnum: methods,
			toolschema.KeyDescription: "How it was paid. Take it from the person; never " +
				"assume one.",
		},
		paramPaymentReference: stringProperty("The check number, ACH trace or batch "+
			"reference the payment went out under, when the person gave one.",
			maxPaymentReferenceChar),
	}
}
