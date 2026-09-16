package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// StartThreadRequest opens a conversation with a configured agent.
type StartThreadRequest struct {
	AgentDefinitionID pulid.ID
	Title             string
	TenantInfo        pagination.TenantInfo
}

// SendMessageRequest is one turn from a person.
type SendMessageRequest struct {
	ThreadID   pulid.ID
	Content    string
	TenantInfo pagination.TenantInfo
}

// SendMessageResult is what the turn produced and saved.
type SendMessageResult struct {
	Thread   *conversation.Thread   `json:"thread"`
	Messages []conversation.Message `json:"messages"`
	// Reply is the assistant's answer, or the refusal explaining why there is
	// none.
	Reply string `json:"reply"`
	// Refused separates "the assistant declined" from "the assistant answered",
	// which a client renders differently.
	Refused bool `json:"refused"`
	// Proposals are writes the agent asked for. Nothing here has run.
	Proposals []AssistantProposal `json:"proposals"`
}

// AssistantProposal is a write the agent proposed during a conversation.
type AssistantProposal struct {
	ToolName  string         `json:"toolName"`
	Arguments map[string]any `json:"arguments"`
	Rationale string         `json:"rationale"`
}

type AssistantService interface {
	StartThread(
		ctx context.Context,
		req *StartThreadRequest,
		actor *RequestActor,
	) (*conversation.Thread, error)
	ListThreads(
		ctx context.Context,
		req repositories.ListThreadsRequest,
	) (*pagination.ListResult[*conversation.Thread], error)
	GetThread(
		ctx context.Context,
		req repositories.GetThreadRequest,
	) (*conversation.Thread, error)
	ListMessages(
		ctx context.Context,
		req repositories.GetThreadRequest,
	) ([]conversation.Message, error)
	DeleteThread(ctx context.Context, req repositories.GetThreadRequest) error
	// SendMessage runs a guarded turn and persists it.
	SendMessage(
		ctx context.Context,
		req *SendMessageRequest,
		actor *RequestActor,
	) (*SendMessageResult, error)
}
