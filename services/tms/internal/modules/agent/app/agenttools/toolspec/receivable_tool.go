package toolspec

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/toolschema"
)

var ErrNeedsAPersonsApproval = errors.New("this runs only once a person approves the proposal")

type ReceivableSpec struct {
	Name        string                                                           `json:"-"`
	Description string                                                           `json:"-"`
	Resource    permission.Resource                                              `json:"-"`
	Operation   permission.Operation                                             `json:"-"`
	Egress      agent.EgressClass                                                `json:"-"`
	AlsoEgress  []agent.EgressClass                                              `json:"-"`
	Classify    func(agenttoolports.ToolExecuteParams) agenttoolports.CallPolicy `json:"-"`
	DefaultTier agent.AutonomyTier                                               `json:"-"`
	MaxTier     agent.AutonomyTier                                               `json:"-"`
	PersonOnly  bool                                                             `json:"-"`
	Reversible  bool                                                             `json:"-"`
	Idempotent  bool                                                             `json:"-"`
	TaintHold   string                                                           `json:"-"`
	Condition   *agenttoolports.TierCondition                                    `json:"-"`
	Artifact    string                                                           `json:"-"`
	Rationale   string                                                           `json:"-"`
	Properties  map[string]any                                                   `json:"-"`
	Required    []string                                                         `json:"-"`
	SearchTerms []string                                                         `json:"-"`
	// prerequisites and recipe are the lookups the tool's ids come from and
	// the order a task with it is done in; see serviceports.RecipeTool.
	Prerequisites []string                                                      `json:"-"`
	Recipe        []string                                                      `json:"-"`
	Target        func(params map[string]any) (agenttoolports.ToolTarget, bool) `json:"-"`
}

type ReceivablePlan[R, P any] struct {
	Request func(params *agenttoolports.ToolExecuteParams) (R, error)                             `json:"-"`
	Settle  func(ctx context.Context, req R, params *agenttoolports.ToolExecuteParams) error      `json:"-"`
	Plan    func(ctx context.Context, req R, params *agenttoolports.ToolExecuteParams) (P, error) `json:"-"`
	Refused func(req R) string                                                                    `json:"-"`
	Render  func(req R, plan P) (*agent.ToolPreview, error)                                       `json:"-"`
	Run     func(
		ctx context.Context,
		req R,
		params *agenttoolports.ToolExecuteParams,
	) (*agent.ToolExecutionResult, error) `json:"-"`
}

type ReceivableTool[R, P any] struct {
	spec  *ReceivableSpec
	steps ReceivablePlan[R, P]
}

type ReportingReceivableTool[R, P any] struct {
	*ReceivableTool[R, P]
}

func NewReceivableTool[R, P any](
	spec *ReceivableSpec,
	steps ReceivablePlan[R, P],
) *ReceivableTool[R, P] {
	return &ReceivableTool[R, P]{spec: spec, steps: steps}
}

func NewReportingReceivableTool[R, P any](
	spec *ReceivableSpec,
	steps ReceivablePlan[R, P],
) ReportingReceivableTool[R, P] {
	return ReportingReceivableTool[R, P]{ReceivableTool: NewReceivableTool(spec, steps)}
}

func (t *ReceivableTool[R, P]) Name() string { return t.spec.Name }

func (t *ReceivableTool[R, P]) Description() string { return t.spec.Description }

func (t *ReceivableTool[R, P]) SearchTerms() []string { return t.spec.SearchTerms }

func (t *ReceivableTool[R, P]) Prerequisites() []string { return t.spec.Prerequisites }

func (t *ReceivableTool[R, P]) Recipe() []string { return t.spec.Recipe }

func (t *ReceivableTool[R, P]) ParamSchema() map[string]any {
	schema := map[string]any{
		toolschema.KeyType:                 toolschema.TypeObject,
		toolschema.KeyProperties:           t.spec.Properties,
		toolschema.KeyAdditionalProperties: false,
	}
	if len(t.spec.Required) > 0 {
		schema[toolschema.KeyRequired] = t.spec.Required
	}

	return schema
}

func (t *ReceivableTool[R, P]) Policy() agenttoolports.ToolPolicy {
	policy := agenttoolports.ToolPolicy{
		Name:          t.spec.Name,
		Kind:          agent.ToolKindAction,
		Resource:      t.spec.Resource,
		Operation:     t.spec.Operation,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   t.spec.DefaultTier,
		MaxTier:       t.spec.MaxTier,
		Egress:        append([]agent.EgressClass{t.spec.Egress}, t.spec.AlsoEgress...),
		Effect:        agent.ToolEffectChange,
		Reversible:    t.spec.Reversible,
		Idempotent:    t.spec.Idempotent,
		ReadsExternal: agent.ExternalReadNever,
		Artifact:      t.spec.Artifact,
		Rationale:     t.spec.Rationale,
		Condition:     t.spec.Condition,
		Classify:      t.spec.Classify,
	}
	if t.spec.TaintHold != "" {
		policy.TaintHold = &agenttoolports.TaintHold{
			Description: t.spec.TaintHold,
			Applies:     func(agenttoolports.ToolExecuteParams) bool { return true },
		}
	}

	return policy
}

func (t *ReceivableTool[R, P]) Target(params map[string]any) (agenttoolports.ToolTarget, bool) {
	if t.spec.Target == nil {
		return agenttoolports.ToolTarget{}, false
	}

	return t.spec.Target(params)
}

func (t *ReceivableTool[R, P]) request(params *agenttoolports.ToolExecuteParams) (R, error) {
	if err := toolguard.GuardPreview(t, params); err != nil {
		var zero R
		return zero, err
	}

	return t.steps.Request(params)
}

func (t *ReceivableTool[R, P]) settled(
	ctx context.Context,
	req R,
	params *agenttoolports.ToolExecuteParams,
) error {
	if t.steps.Settle == nil {
		return nil
	}

	return t.steps.Settle(ctx, req, params)
}

func (t *ReceivableTool[R, P]) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	req, err := t.request(&params)
	if err != nil {
		return err
	}
	if err = t.settled(ctx, req, &params); err != nil {
		return err
	}
	_, err = t.steps.Plan(ctx, req, &params)

	return err
}

func (t *ReceivableTool[R, P]) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	req, err := t.request(&params)
	if err != nil {
		return nil, err
	}
	if err = t.settled(ctx, req, &params); err != nil {
		return changepreview.WarnRefusal(toolpreview.Build(t.steps.Refused(req)), err)
	}

	plan, err := t.steps.Plan(ctx, req, &params)
	if err != nil {
		return changepreview.WarnRefusal(toolpreview.Build(t.steps.Refused(req)), err)
	}

	return t.steps.Render(req, plan)
}

func (t *ReceivableTool[R, P]) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	_, err := t.execute(ctx, &params)

	return err
}

func (t *ReceivableTool[R, P]) execute(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*agent.ToolExecutionResult, error) {
	if t.spec.PersonOnly && !params.ApprovedFromProposal() {
		return nil, fmt.Errorf("%s: %w", t.spec.Name, ErrNeedsAPersonsApproval)
	}
	if err := toolguard.GuardExecute(t, *params); err != nil {
		return nil, err
	}

	req, err := t.steps.Request(params)
	if err != nil {
		return nil, err
	}
	if err = t.settled(ctx, req, params); err != nil {
		return nil, err
	}

	return t.steps.Run(ctx, req, params)
}

func (t ReportingReceivableTool[R, P]) ExecuteWithResult(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolResultReporter interface passes params by value
) (*agent.ToolExecutionResult, error) {
	return t.execute(ctx, &params)
}
