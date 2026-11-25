package memorytools

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemory"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/shared/stringutils"
)

const memoryResultKind = "memory"

// ExecuteWithResult keeps the memory, as an approved proposal runs it, and
// says whether it took effect. An approved team memory said only "it ran",
// and the agent, whose description says a shared memory may be offered for
// an administrator instead, told the person it still waited on one.
func (t *RememberTool) ExecuteWithResult(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*agentrecord.ToolExecutionResult, error) {
	memory, err := t.Record(ctx, params)
	if err != nil {
		return nil, err
	}

	return memoryResult(memory), nil
}

// ExecuteWithResult edits the memory, as an approved proposal runs it, and
// says whether the edit took effect or was offered.
func (t *EditMemoryTool) ExecuteWithResult(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*agentrecord.ToolExecutionResult, error) {
	memory, err := t.Record(ctx, params)
	if err != nil {
		return nil, err
	}

	return memoryResult(memory), nil
}

// memoryResult names the memory a remember kept and where it stands: in
// effect, offered to someone to accept, or one already kept and refreshed.
func memoryResult(memory *agentmemory.Memory) *agentrecord.ToolExecutionResult {
	if memory == nil {
		return nil
	}

	action := "saved"
	switch {
	case memory.Refreshed:
		action = "refreshed"
	case memory.Status == agentmemory.MemoryStatusSuggested:
		action = "offered"
	}

	return &agentrecord.ToolExecutionResult{
		Action: action,
		Kind:   memoryResultKind,
		Name:   stringutils.Ellipsize(memory.Content, memoryLabelRunes),
		IDs:    map[string]string{fieldMemoryID: memory.ID.String()},
		State:  memoryState(memory),
	}
}

// memoryState says in a few words whether the memory is read by later runs,
// and whose.
func memoryState(memory *agentmemory.Memory) string {
	readers := memoryReaders(memory.Scope)
	switch memory.Status {
	case agentmemory.MemoryStatusActive:
		if memory.Refreshed {
			return "already kept and in effect, read in " + readers + "; nothing new was recorded"
		}
		return "kept and in effect, read in " + readers
	case agentmemory.MemoryStatusSuggested:
		if memory.Scope.Shared() {
			return "offered for an administrator to accept; no run reads it until one does, " +
				"then it is read in " + readers
		}
		return "offered for this person to accept; no run reads it until they do"
	case agentmemory.MemoryStatusPaused, agentmemory.MemoryStatusRetired,
		agentmemory.MemoryStatusDismissed:
		return string(memory.Status) + "; no run reads it"
	default:
		return string(memory.Status)
	}
}

func memoryReaders(scope agentmemory.MemoryScope) string {
	switch scope {
	case agentmemory.MemoryScopeUser:
		return "this person's later conversations"
	case agentmemory.MemoryScopeRole:
		return "the later conversations of everyone in this person's role"
	case agentmemory.MemoryScopeAgent:
		return "this agent's later runs"
	case agentmemory.MemoryScopeOrganization:
		return "every later run of every agent"
	default:
		return "later runs"
	}
}
