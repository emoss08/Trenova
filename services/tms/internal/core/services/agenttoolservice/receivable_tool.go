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
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/money"
	"github.com/shopspring/decimal"
)

var ErrNeedsAPersonsApproval = errors.New("this runs only once a person approves the proposal")

type receivableSpec struct {
	name        string
	description string
	resource    permission.Resource
	operation   permission.Operation
	egress      agent.EgressClass
	defaultTier agent.AutonomyTier
	maxTier     agent.AutonomyTier
	personOnly  bool
	reversible  bool
	idempotent  bool
	taintHold   string
	condition   *serviceports.TierCondition
	artifact    string
	rationale   string
	properties  map[string]any
	required    []string
	target      func(params map[string]any) (serviceports.ToolTarget, bool)
}

type receivablePlan[R, P any] struct {
	request func(params *serviceports.ToolExecuteParams) (R, error)
	plan    func(ctx context.Context, req R, params *serviceports.ToolExecuteParams) (P, error)
	refused func(req R) string
	render  func(req R, plan P) (*agent.ToolPreview, error)
	run     func(
		ctx context.Context,
		req R,
		params *serviceports.ToolExecuteParams,
	) (*agent.ToolExecutionResult, error)
}

type receivableTool[R, P any] struct {
	spec  receivableSpec
	steps receivablePlan[R, P]
}

type reportingReceivableTool[R, P any] struct {
	*receivableTool[R, P]
}

var (
	_ serviceports.ToolPreviewer      = (*receivableTool[struct{}, struct{}])(nil)
	_ serviceports.ToolValidator      = (*receivableTool[struct{}, struct{}])(nil)
	_ serviceports.TargetedTool       = (*receivableTool[struct{}, struct{}])(nil)
	_ serviceports.ToolResultReporter = reportingReceivableTool[struct{}, struct{}]{}
)

func newReceivableTool[R, P any](
	spec receivableSpec,
	steps receivablePlan[R, P],
) *receivableTool[R, P] {
	return &receivableTool[R, P]{spec: spec, steps: steps}
}

func newReportingReceivableTool[R, P any](
	spec receivableSpec,
	steps receivablePlan[R, P],
) reportingReceivableTool[R, P] {
	return reportingReceivableTool[R, P]{receivableTool: newReceivableTool(spec, steps)}
}

func (t *receivableTool[R, P]) Name() string { return t.spec.name }

func (t *receivableTool[R, P]) Description() string { return t.spec.description }

func (t *receivableTool[R, P]) ParamSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType:                 toolschema.TypeObject,
		toolschema.KeyProperties:           t.spec.properties,
		toolschema.KeyRequired:             t.spec.required,
		toolschema.KeyAdditionalProperties: false,
	}
}

func (t *receivableTool[R, P]) Policy() serviceports.ToolPolicy {
	policy := serviceports.ToolPolicy{
		Name:          t.spec.name,
		Kind:          agent.ToolKindAction,
		Resource:      t.spec.resource,
		Operation:     t.spec.operation,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   t.spec.defaultTier,
		MaxTier:       t.spec.maxTier,
		Egress:        []agent.EgressClass{t.spec.egress},
		Effect:        agent.ToolEffectChange,
		Reversible:    t.spec.reversible,
		Idempotent:    t.spec.idempotent,
		ReadsExternal: agent.ExternalReadNever,
		Artifact:      t.spec.artifact,
		Rationale:     t.spec.rationale,
		Condition:     t.spec.condition,
	}
	if t.spec.taintHold != "" {
		policy.TaintHold = &serviceports.TaintHold{
			Description: t.spec.taintHold,
			Applies:     func(serviceports.ToolExecuteParams) bool { return true },
		}
	}

	return policy
}

func (t *receivableTool[R, P]) Target(params map[string]any) (serviceports.ToolTarget, bool) {
	if t.spec.target == nil {
		return serviceports.ToolTarget{}, false
	}

	return t.spec.target(params)
}

func (t *receivableTool[R, P]) request(params *serviceports.ToolExecuteParams) (R, error) {
	if err := guardPreview(t, params); err != nil {
		var zero R
		return zero, err
	}

	return t.steps.request(params)
}

func (t *receivableTool[R, P]) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	req, err := t.request(&params)
	if err != nil {
		return err
	}
	_, err = t.steps.plan(ctx, req, &params)

	return err
}

func (t *receivableTool[R, P]) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	req, err := t.request(&params)
	if err != nil {
		return nil, err
	}

	plan, err := t.steps.plan(ctx, req, &params)
	if err != nil {
		return warnRefusal(toolpreview.Build(t.steps.refused(req)), err)
	}

	return t.steps.render(req, plan)
}

func (t *receivableTool[R, P]) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	_, err := t.execute(ctx, &params)

	return err
}

func (t *receivableTool[R, P]) execute(
	ctx context.Context,
	params *serviceports.ToolExecuteParams,
) (*agent.ToolExecutionResult, error) {
	if t.spec.personOnly && !params.ApprovedFromProposal() {
		return nil, fmt.Errorf("%s: %w", t.spec.name, ErrNeedsAPersonsApproval)
	}
	if err := guardExecute(t, *params); err != nil {
		return nil, err
	}

	req, err := t.steps.request(params)
	if err != nil {
		return nil, err
	}

	return t.steps.run(ctx, req, params)
}

func (t reportingReceivableTool[R, P]) ExecuteWithResult(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolResultReporter interface passes params by value
) (*agent.ToolExecutionResult, error) {
	return t.execute(ctx, &params)
}

func targetInvoice(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, paramInvoiceID, permission.ResourceInvoice)
}

func stringProperty(description string, maxLength int) map[string]any {
	property := map[string]any{
		toolschema.KeyType:        toolschema.TypeString,
		toolschema.KeyDescription: description,
	}
	if maxLength > 0 {
		property[toolschema.KeyMaxLength] = maxLength
	}

	return property
}

func enumProperty(description string, values []string) map[string]any {
	return map[string]any{
		toolschema.KeyType:        toolschema.TypeString,
		toolschema.KeyEnum:        values,
		toolschema.KeyDescription: description,
	}
}

func idListProperty(description string, maxItems int) map[string]any {
	return map[string]any{
		toolschema.KeyType:        toolschema.TypeArray,
		toolschema.KeyDescription: description,
		toolschema.KeyItems:       map[string]any{toolschema.KeyType: toolschema.TypeString},
		toolschema.KeyMinItems:    1,
		toolschema.KeyMaxItems:    maxItems,
	}
}

func requireEnum[T ~string](params map[string]any, key string, values []T) (T, error) {
	raw, err := requireString(params, key)
	if err != nil {
		return "", err
	}

	value := T(strings.TrimSpace(raw))
	if !slices.Contains(values, value) {
		return "", fmt.Errorf("%s %q is not one of %s", key, raw, enumList(values))
	}

	return value, nil
}

func optionalEnum[T ~string](params map[string]any, key string, values []T) (T, error) {
	if strings.TrimSpace(optionalString(params, key)) == "" {
		return "", nil
	}

	return requireEnum(params, key, values)
}

func enumNames[T ~string](values []T) []string {
	names := make([]string, 0, len(values))
	for _, value := range values {
		names = append(names, string(value))
	}

	return names
}

func enumList[T ~string](values []T) string {
	return strings.Join(enumNames(values), ", ")
}

func requireAmount(params map[string]any, key string) (decimal.Decimal, error) {
	minor, err := requireMoney(params, key)
	if err != nil {
		return decimal.Zero, err
	}

	return money.DecimalFromMinor(minor), nil
}

func boundedText(params map[string]any, key string, limit int) (string, error) {
	text := strings.TrimSpace(optionalString(params, key))
	if len(text) > limit {
		return "", fmt.Errorf("%s is %d characters; at most %d are kept", key, len(text), limit)
	}

	return text, nil
}

func requireBoundedText(params map[string]any, key string, limit int) (string, error) {
	if _, err := requireString(params, key); err != nil {
		return "", err
	}

	return boundedText(params, key, limit)
}
