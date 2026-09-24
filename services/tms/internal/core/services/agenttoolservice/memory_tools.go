package agenttoolservice

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
)

// rememberTool records something the agent was told or worked out, for
// every later run to read.
//
// It is a write like any other: it goes through the proposal tiers, so an
// organization decides whether an agent may add to its memory on its own or
// only with a person's approval. What it records is bounded and never a
// secret, and a person can retire it from AI Control.
type rememberTool struct {
	memories serviceports.AgentMemoryService
}

func newRememberTool(memories serviceports.AgentMemoryService) serviceports.AgentTool {
	return &rememberTool{memories: memories}
}

func (t *rememberTool) Name() string { return "remember" }

func (t *rememberTool) Description() string {
	return "Record a standing instruction or a fact for every later run of every agent in " +
		"this organization to know. Use kind Instruction for a rule a person gave you, Fact " +
		"for something you were told or confirmed that is not in any record. Scope it to one " +
		"customer, location, driver or carrier with subjectType and subjectId when it is " +
		"about that record; leave both out for something organization-wide. Do not record " +
		"what a record already says, a guess, or anything a person asked you to keep " +
		"private. Use recall_memory first to see whether it is already known."
}

func (t *rememberTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"content": map[string]any{
				"type": "string",
				"description": "One or two plain sentences, at most 2000 characters, " +
					"written so a reader with no other context understands them.",
			},
			"kind": map[string]any{
				"type": "string",
				"enum": []string{
					string(agent.MemoryKindInstruction),
					string(agent.MemoryKindFact),
				},
				"description": "Instruction for a rule a person gave; Fact for something learned. Defaults to Fact.",
			},
			"subjectType": map[string]any{
				"type":        "string",
				"enum":        memorySubjectTypeNames(),
				"description": "The kind of record the memory is about, with subjectId. Omit for organization-wide.",
			},
			"subjectId": map[string]any{
				"type": "string",
				"description": "The record the memory is about: this run's subject, the page, or " +
					"an id from list_customers, list_locations, list_workers or list_carriers. " +
					"Never guessed.",
			},
			"expiresOn": map[string]any{
				"type":        "string",
				"description": "Optional YYYY-MM-DD after which the memory no longer applies, such as a temporary arrangement.",
			},
		},
		"required":             []string{"content"},
		"additionalProperties": false,
	}
}

func (t *rememberTool) Policy() serviceports.ToolPolicy {
	return serviceports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceAgentMemory,
		Operation:     permission.OpCreate,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierActWithApproval,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agent.ToolEffectChange,
		Reversible:    true,
		ReadsExternal: agent.ExternalReadNever,
		CarriesTaint:  true,
		Rationale: "Saves a memory later runs read, so it keeps the taint of the run that " +
			"wrote it.",
	}
}

func (t *rememberTool) Execute(ctx context.Context, params serviceports.ToolExecuteParams) error {
	if err := guardExecute(t, params); err != nil {
		return err
	}

	content, err := requireString(params.Params, "content")
	if err != nil {
		return err
	}

	kind := agent.MemoryKind(optionalString(params.Params, "kind"))
	if kind == "" {
		kind = agent.MemoryKindFact
	}
	if kind == agent.MemoryKindCorrection || !kind.IsValid() {
		return fmt.Errorf("kind must be Instruction or Fact, not %q", kind)
	}

	subjectType, subjectID, err := memorySubject(params.Params)
	if err != nil {
		return err
	}

	expiresAt, err := memoryExpiry(optionalString(params.Params, "expiresOn"))
	if err != nil {
		return err
	}

	_, err = t.memories.Remember(ctx, &serviceports.RememberRequest{
		TenantInfo:  tenantFrom(params),
		Kind:        kind,
		Content:     content,
		SubjectType: subjectType,
		SubjectID:   subjectID,
		ExpiresAt:   expiresAt,
		RunID:       params.RunID,
		Taint:       params.Taint,
	}, params.Actor)

	return err
}

// forgetMemoryTool retires a memory that no longer holds. It is a status
// change, not a delete, so a person can see what was forgotten and restore
// it.
type forgetMemoryTool struct {
	memories serviceports.AgentMemoryService
}

func newForgetMemoryTool(memories serviceports.AgentMemoryService) serviceports.AgentTool {
	return &forgetMemoryTool{memories: memories}
}

func (t *forgetMemoryTool) Name() string { return "forget_memory" }

func (t *forgetMemoryTool) Description() string {
	return "Retire a recorded memory that no longer holds, by its id from recall_memory: " +
		"an arrangement that ended, a fact a person says is wrong. It stays readable in " +
		"AI Control and can be restored. Use this only when told the memory is wrong or " +
		"over, never to make room."
}

func (t *forgetMemoryTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"memoryId": map[string]any{
				"type":        "string",
				"description": "The memory to retire, by the id recall_memory returned.",
			},
		},
		"required":             []string{"memoryId"},
		"additionalProperties": false,
	}
}

func (t *forgetMemoryTool) Policy() serviceports.ToolPolicy {
	return serviceports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceAgentMemory,
		Operation:     permission.OpUpdate,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierActWithApproval,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agent.ToolEffectChange,
		Reversible:    true,
		ReadsExternal: agent.ExternalReadNever,
		Rationale:     "Retires an agent memory inside Trenova.",
	}
}

func (t *forgetMemoryTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) error {
	if err := guardExecute(t, params); err != nil {
		return err
	}

	memoryID, err := requirePulid(params.Params, "memoryId")
	if err != nil {
		return err
	}

	_, err = t.memories.SetStatus(ctx, serviceports.SetAgentMemoryStatusRequest{
		ID:         memoryID,
		TenantInfo: tenantFrom(params),
		Status:     agent.MemoryStatusRetired,
	}, params.Actor)

	return err
}

func memorySubjectTypeNames() []string {
	types := agent.AllMemorySubjectTypes()
	names := make([]string, 0, len(types))
	for _, subjectType := range types {
		names = append(names, string(subjectType))
	}

	return names
}

// memorySubject reads the optional subject pair, refusing half of one: a
// type without an id names nothing, and an id without a type cannot be
// looked up.
func memorySubject(params map[string]any) (agent.MemorySubjectType, pulid.ID, error) {
	subjectType := agent.MemorySubjectType(optionalString(params, "subjectType"))
	rawID := optionalString(params, "subjectId")

	if subjectType == "" && rawID == "" {
		return "", pulid.Nil, nil
	}
	if subjectType == "" || rawID == "" {
		return "", pulid.Nil, fmt.Errorf(
			"subjectType and subjectId go together; give both or neither",
		)
	}
	if !subjectType.IsValid() {
		return "", pulid.Nil, fmt.Errorf("subjectType %q is not one of %s",
			subjectType, strings.Join(memorySubjectTypeNames(), ", "))
	}

	subjectID, err := pulid.Parse(rawID)
	if err != nil {
		return "", pulid.Nil, fmt.Errorf("subjectId %q is not a record id", rawID)
	}

	return subjectType, subjectID, nil
}

// memoryExpiry reads a YYYY-MM-DD as the end of that day, so "expires on the
// 30th" still applies on the 30th.
func memoryExpiry(raw string) (*int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	day, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return nil, fmt.Errorf("expiresOn must be YYYY-MM-DD, got %q", raw)
	}

	end := day.Add(24 * time.Hour).Unix()

	return &end, nil
}
