package agentruntimeports

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// ErrAgentRunAlreadyOpen is a start refused because the agent already has a
// run open for that subject. The open run will see what the new one would
// have; it is a skip, not a failure.
var ErrAgentRunAlreadyOpen = errors.New("this agent already has an open run for that subject")

// StartAgentRunForDefinitionRequest starts a background run of one agent.
// The definition is named by id or by its system key; the subject is optional
// and defaults to the organization itself.
type StartAgentRunForDefinitionRequest struct {
	DefinitionID pulid.ID
	SystemKey    string
	SubjectType  agent.SubjectType
	SubjectID    pulid.ID
	Trigger      agent.RunTrigger
	EventKind    agenttrigger.EventKind
	// Slot is the schedule slot the run fills. It keys the workflow id, so a
	// slot starts one run however often its start is retried.
	Slot       int64
	TenantInfo pagination.TenantInfo
	// WaitID is the wait whose ending starts a run that picks the work up.
	WaitID pulid.ID
	// CaseOptionID is the option a person chose on the record's case, which
	// the run drafts the plan for.
	CaseOptionID pulid.ID
	// Handed is what the run was handed to work, in a sentence or two: the
	// Watchtower item a person passed to the agent. Without it the agent is
	// told only which record the run concerns and follows its routine,
	// whatever was raised.
	Handed string
	// Watched says a person is waiting on the run in front of them (a case
	// handed to an agent, a case option's plan being drafted), so the run
	// hosts a stream of what the agent writes. A run nobody watches hosts
	// none: a stream costs history, and its close waits on a reader.
	Watched bool
}

// StartInlineAgentRunRequest opens a run for an agent whose reasoning happens in-process
// rather than in a Temporal workflow.
type StartInlineAgentRunRequest struct {
	AgentType         agent.Type
	AgentDefinitionID pulid.ID
	SubjectType       agent.SubjectType
	SubjectID         pulid.ID
	PromptVersion     string
	Summary           string
	Trigger           agent.RunTrigger
	TenantInfo        pagination.TenantInfo
}

type AgentRunService interface {
	StartForDefinition(
		ctx context.Context,
		req *StartAgentRunForDefinitionRequest,
		actor *permissionports.RequestActor,
	) (*agent.AgentRun, error)
	// StartInline records a run for a deterministic agent that reasons in-process. Such
	// agents still need the run row so their proposals, decisions, and audit trail are
	// indistinguishable from an LLM-backed agent's.
	StartInline(
		ctx context.Context,
		req *StartInlineAgentRunRequest,
		actor *permissionports.RequestActor,
	) (*agent.AgentRun, error)
	ListConnection(
		ctx context.Context,
		req *ListAgentRunConnectionRequest,
	) (*pagination.CursorListResult[*agent.AgentRun], error)
	GetByID(
		ctx context.Context,
		req GetAgentRunByIDRequest,
	) (*agent.AgentRun, error)
}

type AgentRunTranscriptService interface {
	RunTranscript(
		ctx context.Context,
		req GetAgentRunByIDRequest,
	) (*kernelports.TranscriptFile, error)
}
