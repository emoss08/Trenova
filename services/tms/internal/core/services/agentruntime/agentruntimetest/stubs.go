package agentruntimetest

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
)

type ScriptedCompletion struct {
	Turns          []*serviceports.ChatCompletionResult
	CallCount      int
	LastReq        *serviceports.ChatCompletionRequest
	Classification string
	StructuredText string
}

func (s *ScriptedCompletion) CompleteChat(
	_ context.Context,
	req *serviceports.ChatCompletionRequest,
) (*serviceports.ChatCompletionResult, error) {
	s.LastReq = req
	idx := s.CallCount
	s.CallCount++
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
	req *serviceports.ChatCompletionRequest,
	sink serviceports.ChatStreamSink,
) (*serviceports.ChatCompletionResult, error) {
	result, err := s.CompleteChat(ctx, req)
	if err != nil {
		return nil, err
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
	_ *serviceports.StructuredCompletionRequest,
) (*serviceports.StructuredCompletionResult, error) {
	if s.StructuredText != "" {
		return &serviceports.StructuredCompletionResult{Text: s.StructuredText}, nil
	}
	category := s.Classification
	if category == "" {
		category = "TransportationOperations"
	}

	return &serviceports.StructuredCompletionResult{
		Text: `{"category":"` + category + `"}`,
	}, nil
}

// SubmitBackground runs the call inline, which is the same answer the router
// gives for a provider whose protocol cannot defer — so a test that exercises
// the deferred path against this stub sees the shape a self-hosted model would
// really produce rather than a handle that never resolves.
func (s *ScriptedCompletion) SubmitBackground(
	ctx context.Context,
	req *serviceports.StructuredCompletionRequest,
) (*serviceports.BackgroundSubmission, error) {
	result, err := s.CompleteStructured(ctx, req)
	if err != nil {
		return nil, err
	}

	return &serviceports.BackgroundSubmission{Result: result}, nil
}

func (s *ScriptedCompletion) PollBackground(
	_ context.Context,
	_ *serviceports.BackgroundPollRequest,
) (*serviceports.BackgroundOutcome, error) {
	return nil, errors.New("scripted completion runs inline and issues no handle to poll")
}

type StubQueryTool struct {
	ToolName   string
	Result     any
	Err        error
	Calls      int
	LastParams serviceports.QueryToolParams
	Resource   permission.Resource
}

func (t *StubQueryTool) Name() string                { return t.ToolName }
func (t *StubQueryTool) Description() string         { return "stub query tool" }
func (t *StubQueryTool) ParamSchema() map[string]any { return map[string]any{"type": "object"} }
func (t *StubQueryTool) PermissionResource() permission.Resource {
	if t.Resource == "" {
		return permission.ResourceShipment
	}

	return t.Resource
}

func (t *StubQueryTool) Query(
	_ context.Context,
	params serviceports.QueryToolParams,
) (any, error) {
	t.Calls++
	t.LastParams = params

	return t.Result, t.Err
}

type StubQueryRegistry struct{ Tools []serviceports.AgentQueryTool }

func (r *StubQueryRegistry) Get(name string) (serviceports.AgentQueryTool, bool) {
	for _, tool := range r.Tools {
		if tool.Name() == name {
			return tool, true
		}
	}

	return nil, false
}

func (r *StubQueryRegistry) All() []serviceports.AgentQueryTool { return r.Tools }

func (r *StubQueryRegistry) Descriptors() []serviceports.AgentToolDescriptor {
	out := make([]serviceports.AgentToolDescriptor, 0, len(r.Tools))
	for _, tool := range r.Tools {
		out = append(out, serviceports.AgentToolDescriptor{
			Name:        tool.Name(),
			Description: tool.Description(),
			Parameters:  tool.ParamSchema(),
		})
	}

	return out
}

type StubActionTool struct {
	ToolName   string
	Tier       agent.AutonomyTier
	Err        error
	Calls      int
	LastParams serviceports.ToolExecuteParams
	Resource   permission.Resource
}

func (t *StubActionTool) Name() string                { return t.ToolName }
func (t *StubActionTool) Description() string         { return "stub action tool" }
func (t *StubActionTool) ParamSchema() map[string]any { return map[string]any{"type": "object"} }
func (t *StubActionTool) Reversible() bool            { return true }
func (t *StubActionTool) PermissionResource() permission.Resource {
	if t.Resource == "" {
		return permission.ResourceShipmentMove
	}

	return t.Resource
}
func (t *StubActionTool) PermissionOperation() permission.Operation { return permission.OpUpdate }
func (t *StubActionTool) RequiresIdempotencyKey() bool              { return true }
func (t *StubActionTool) DefaultAutonomyTier() agent.AutonomyTier {
	if t.Tier == "" {
		return agent.TierPropose
	}

	return t.Tier
}

func (t *StubActionTool) Execute(
	_ context.Context,
	params serviceports.ToolExecuteParams,
) error {
	t.Calls++
	t.LastParams = params

	return t.Err
}

type StubActionRegistry struct{ Tools []serviceports.AgentTool }

func (r *StubActionRegistry) Get(name string) (serviceports.AgentTool, bool) {
	for _, tool := range r.Tools {
		if tool.Name() == name {
			return tool, true
		}
	}

	return nil, false
}

func (r *StubActionRegistry) All() []serviceports.AgentTool { return r.Tools }

func (r *StubActionRegistry) Descriptors() []serviceports.AgentToolDescriptor {
	out := make([]serviceports.AgentToolDescriptor, 0, len(r.Tools))
	for _, tool := range r.Tools {
		out = append(out, serviceports.AgentToolDescriptor{
			Name:         tool.Name(),
			Description:  tool.Description(),
			Parameters:   tool.ParamSchema(),
			AutonomyTier: tool.DefaultAutonomyTier(),
		})
	}

	return out
}

type StubPermissions struct {
	serviceports.PermissionEngine

	Denied   map[string]bool
	Requests []*serviceports.PermissionCheckRequest
}

func (p *StubPermissions) Check(
	_ context.Context,
	req *serviceports.PermissionCheckRequest,
) (*serviceports.PermissionCheckResult, error) {
	p.Requests = append(p.Requests, req)
	if p.Denied[req.Resource+":"+string(req.Operation)] {
		return &serviceports.PermissionCheckResult{Allowed: false, Reason: "denied"}, nil
	}

	return &serviceports.PermissionCheckResult{Allowed: true}, nil
}
