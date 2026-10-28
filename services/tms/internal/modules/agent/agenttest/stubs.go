package agenttest

import (
	"context"
	"errors"
	"sync"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
)

type ScriptedCompletion struct {
	Turns []*llmports.ChatCompletionResult
	// Errors fails the completion at that call index instead of answering, so
	// a test can make the model die after a tool has already run.
	Errors    map[int]error
	CallCount int
	LastReq   *llmports.ChatCompletionRequest
	// Requests is every request, oldest first, so a test can see what a
	// turn sent before the one that answered it.
	Requests       []*llmports.ChatCompletionRequest
	Classification string
	StructuredText string
}

func (s *ScriptedCompletion) CompleteChat(
	_ context.Context,
	req *llmports.ChatCompletionRequest,
) (*llmports.ChatCompletionResult, error) {
	s.LastReq = req
	s.Requests = append(s.Requests, req)
	idx := s.CallCount
	s.CallCount++
	if err, failed := s.Errors[idx]; failed {
		return nil, err
	}
	if len(s.Turns) == 0 {
		return nil, errors.New("no scripted turns")
	}
	if idx >= len(s.Turns) {
		return s.Turns[len(s.Turns)-1], nil
	}

	return s.Turns[idx], nil
}

func (s *ScriptedCompletion) StreamChat(
	ctx context.Context,
	req *llmports.ChatCompletionRequest,
	sink llmports.ChatStreamSink,
) (*llmports.ChatCompletionResult, error) {
	result, err := s.CompleteChat(ctx, req)
	if err != nil {
		return nil, err
	}
	if result.Reasoning != nil && result.Reasoning.Text != "" && req.ReasoningSink != nil {
		req.ReasoningSink(result.Reasoning.Text)
	}
	if result.Text != "" && sink != nil {
		half := len(result.Text) / 2
		sink(result.Text[:half])
		sink(result.Text[half:])
	}

	return result, nil
}

func (s *ScriptedCompletion) CompleteStructured(
	_ context.Context,
	_ *llmports.StructuredCompletionRequest,
) (*llmports.StructuredCompletionResult, error) {
	if s.StructuredText != "" {
		return &llmports.StructuredCompletionResult{Text: s.StructuredText}, nil
	}
	category := s.Classification
	if category == "" {
		category = "TransportationOperations"
	}

	return &llmports.StructuredCompletionResult{
		Text: `{"category":"` + category + `"}`,
	}, nil
}

// SubmitBackground runs the call inline, which is the same answer the router
// gives for a provider whose protocol cannot defer — so a test that exercises
// the deferred path against this stub sees the shape a self-hosted model would
// really produce rather than a handle that never resolves.
func (s *ScriptedCompletion) SubmitBackground(
	ctx context.Context,
	req *llmports.StructuredCompletionRequest,
) (*llmports.BackgroundSubmission, error) {
	result, err := s.CompleteStructured(ctx, req)
	if err != nil {
		return nil, err
	}

	return &llmports.BackgroundSubmission{Result: result}, nil
}

func (s *ScriptedCompletion) PollBackground(
	_ context.Context,
	_ *llmports.BackgroundPollRequest,
) (*llmports.BackgroundOutcome, error) {
	return nil, errors.New("scripted completion runs inline and issues no handle to poll")
}

type StubQueryTool struct {
	ToolName string
	// Desc lets a test give a tool a description worth ranking against;
	// everything that does not care keeps the generic one.
	Desc       string
	Result     any
	Err        error
	Calls      int
	LastParams agenttoolports.QueryToolParams
	LastTenant dbscope.Tenant
	Resource   permission.Resource
	// Reads and Source declare the tool as one that reads outside content.
	Reads   agent.ExternalRead
	Source  agent.TaintSource
	Sources []agent.TaintSource
	// Schema stands in for the tool's declared parameters when set.
	Schema map[string]any
}

func (t *StubQueryTool) Name() string { return t.ToolName }
func (t *StubQueryTool) Description() string {
	if t.Desc != "" {
		return t.Desc
	}

	return "stub query tool"
}

func (t *StubQueryTool) ParamSchema() map[string]any {
	if t.Schema != nil {
		return t.Schema
	}

	return map[string]any{"type": "object"}
}
func (t *StubQueryTool) Policy() agenttoolports.ToolPolicy {
	resource := t.Resource
	if resource == "" {
		resource = permission.ResourceShipment
	}

	reads := t.Reads
	if reads == "" {
		reads = agent.ExternalReadNever
	}

	return agenttoolports.ToolPolicy{
		Name:          t.ToolName,
		Kind:          agent.ToolKindQuery,
		Resource:      resource,
		Operation:     permission.OpRead,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierAutoExecute,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressNone},
		Effect:        agent.ToolEffectLookup,
		Reversible:    true,
		Idempotent:    true,
		ReadsExternal: reads,
		Source:        t.Source,
		Sources:       t.Sources,
		Rationale:     "A stub read.",
	}
}

func (t *StubQueryTool) Query(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
) (any, error) {
	t.Calls++
	t.LastParams = *params
	t.LastTenant, _ = dbscope.TenantFrom(ctx)

	return t.Result, t.Err
}

type StubQueryRegistry struct {
	Tools []agenttoolports.AgentQueryTool
}

func (r *StubQueryRegistry) Get(name string) (agenttoolports.AgentQueryTool, bool) {
	for _, tool := range r.Tools {
		if tool.Name() == name {
			return tool, true
		}
	}

	return nil, false
}

func (r *StubQueryRegistry) All() []agenttoolports.AgentQueryTool { return r.Tools }

func (r *StubQueryRegistry) Descriptors() []agenttoolports.AgentToolDescriptor {
	out := make([]agenttoolports.AgentToolDescriptor, 0, len(r.Tools))
	for _, tool := range r.Tools {
		out = append(out, agenttoolports.DescribeTool(tool, "", true))
	}

	return out
}

const StubFunctionalCurrency = "USD"

type StubActionTool struct {
	ToolName   string
	Tier       agent.AutonomyTier
	Err        error
	Calls      int
	LastParams agenttoolports.ToolExecuteParams
	Resource   permission.Resource
	// Operation is the permission the tool is gated on; update when unset.
	Operation permission.Operation
	// Schema stands in for the tool's declared parameters when set.
	Schema map[string]any
	// Egress is where the write reaches; internal when unset.
	Egress agent.EgressClass
	// CarriesTaint hands the tool the run's taint, as remember is.
	CarriesTaint  bool
	MoneyMinor    *int64
	MoneyCurrency string
	MoneyVouches  []agent.TaintSource
	// MoneyRereads are the records the stub's bound says it read again.
	MoneyRereads []agent.SourcedRef
	Escalates    bool
}

func (t *StubActionTool) Name() string        { return t.ToolName }
func (t *StubActionTool) Description() string { return "stub action tool" }
func (t *StubActionTool) ParamSchema() map[string]any {
	if t.Schema != nil {
		return t.Schema
	}

	return map[string]any{"type": "object"}
}
func (t *StubActionTool) Policy() agenttoolports.ToolPolicy {
	resource := t.Resource
	if resource == "" {
		resource = permission.ResourceShipmentMove
	}
	tier := t.Tier
	if tier == "" {
		tier = agent.TierPropose
	}
	egress := t.Egress
	if egress == "" {
		egress = agent.EgressInternal
	}
	operation := t.Operation
	if operation == "" {
		operation = permission.OpUpdate
	}

	policy := agenttoolports.ToolPolicy{
		Name:          t.ToolName,
		Kind:          agent.ToolKindAction,
		Resource:      resource,
		Operation:     operation,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   tier,
		MaxTier:       agent.TierAutoExecute,
		CarriesTaint:  t.CarriesTaint,
		Escalates:     t.Escalates,
		Egress:        []agent.EgressClass{egress},
		Effect:        agent.ToolEffectChange,
		Reversible:    true,
		Idempotent:    true,
		ReadsExternal: agent.ExternalReadNever,
		Rationale:     "A stub write.",
	}
	if egress == agent.EgressMoney && t.MoneyMinor != nil {
		amount := *t.MoneyMinor
		currency := t.MoneyCurrency
		if currency == "" {
			currency = StubFunctionalCurrency
		}
		vouches, rereads := t.MoneyVouches, t.MoneyRereads
		policy.Condition = &agenttoolports.TierCondition{
			Description: "A stub money write within its bounds.",
			Bounds: func(context.Context, agenttoolports.ToolExecuteParams) agenttoolports.MoneyBound {
				bound := agenttoolports.WithinMoneyBound(amount, currency)
				for _, reread := range rereads {
					bound = bound.Rereading(reread.Source, reread.Ref)
				}

				return bound
			},
			Vouches: vouches,
		}
	}

	return policy
}

func (t *StubActionTool) Execute(
	_ context.Context,
	params agenttoolports.ToolExecuteParams,
) error {
	t.Calls++
	t.LastParams = params

	return t.Err
}

type StubActionRegistry struct{ Tools []agenttoolports.AgentTool }

func (r *StubActionRegistry) Get(name string) (agenttoolports.AgentTool, bool) {
	for _, tool := range r.Tools {
		if tool.Name() == name {
			return tool, true
		}
	}

	return nil, false
}

func (r *StubActionRegistry) All() []agenttoolports.AgentTool { return r.Tools }

func (r *StubActionRegistry) Descriptors() []agenttoolports.AgentToolDescriptor {
	out := make([]agenttoolports.AgentToolDescriptor, 0, len(r.Tools))
	for _, tool := range r.Tools {
		out = append(out, agenttoolports.DescribeTool(tool, tool.Policy().DefaultTier, false))
	}

	return out
}

type StubPermissions struct {
	permissionports.PermissionEngine

	// mu guards Requests: a turn asks its tool checks side by side.
	mu sync.Mutex

	Denied   map[string]bool
	Requests []*permissionports.PermissionCheckRequest
	// GrantedAgents are the agents restricted to roles that the person's
	// roles grant.
	GrantedAgents []pulid.ID
}

func (p *StubPermissions) AgentsUsable(
	_ context.Context,
	_ *permissionports.RequestActor,
	operation permission.Operation,
) (*permissionports.UsableAgents, error) {
	return &permissionports.UsableAgents{
		Assistant:  !p.Denied[permission.ResourceAssistant.String()+":"+string(operation)],
		GrantedIDs: p.GrantedAgents,
	}, nil
}

func (p *StubPermissions) MayUseAgent(
	ctx context.Context,
	actor *permissionports.RequestActor,
	definition *agentdefinition.Definition,
) (bool, error) {
	usable, err := p.AgentsUsable(ctx, actor, permission.OpCreate)
	if err != nil {
		return false, err
	}

	return usable.Allows(definition), nil
}

func (p *StubPermissions) Check(
	_ context.Context,
	req *permissionports.PermissionCheckRequest,
) (*permissionports.PermissionCheckResult, error) {
	p.mu.Lock()
	p.Requests = append(p.Requests, req)
	p.mu.Unlock()
	if p.Denied[req.Resource+":"+string(req.Operation)] {
		return &permissionports.PermissionCheckResult{Allowed: false, Reason: "denied"}, nil
	}

	return &permissionports.PermissionCheckResult{Allowed: true}, nil
}
