package agentreflectionports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemory"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentreflection"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const AgentMemoryResource = "agent_memory"

type ReflectOnThreadRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	ThreadID   pulid.ID              `json:"threadId"`
	UserID     pulid.ID              `json:"userId"`
	TurnID     pulid.ID              `json:"turnId,omitempty"`
}

type ReflectOnRunRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	RunID      pulid.ID              `json:"runId"`
}

type ReflectionSubjectRef struct {
	Type agentmemory.MemorySubjectType `json:"type"`
	ID   pulid.ID                      `json:"id"`
}

type ReflectionPlan struct {
	TenantInfo        pagination.TenantInfo                 `json:"tenantInfo"`
	ReflectionID      pulid.ID                              `json:"reflectionId"`
	Subject           agentreflection.ReflectionSubject     `json:"subject"`
	AgentDefinitionID pulid.ID                              `json:"agentDefinitionId"`
	ThreadID          pulid.ID                              `json:"threadId,omitempty"`
	RunID             pulid.ID                              `json:"runId,omitempty"`
	PersonUserID      pulid.ID                              `json:"personUserId,omitempty"`
	ReplyMessageID    pulid.ID                              `json:"replyMessageId,omitempty"`
	Taint             *agent.RunTaint                       `json:"taint,omitempty"`
	Signals           agentreflection.ReflectionSignals     `json:"signals"`
	Subjects          []ReflectionSubjectRef                `json:"subjects,omitempty"`
	ToolNames         []string                              `json:"toolNames,omitempty"`
	KnownMemoryIDs    []pulid.ID                            `json:"knownMemoryIds,omitempty"`
	Request           *llmports.StructuredCompletionRequest `json:"request,omitempty"`
}

func (p *ReflectionPlan) Ready() bool {
	return p != nil && p.ReflectionID.IsNotNil() && p.Request != nil
}

func (p *ReflectionPlan) Tainted() bool {
	return p != nil && (p.Taint == nil || p.Taint.Tainted())
}

type ReflectionOutcome struct {
	ReflectionID pulid.ID                         `json:"reflectionId"`
	Status       agentreflection.ReflectionStatus `json:"status"`
	Kept         int                              `json:"kept"`
	Offered      int                              `json:"offered"`
	Refused      int                              `json:"refused"`
}

type FailReflectionRequest struct {
	Plan    *ReflectionPlan `json:"plan"`
	Message string          `json:"message"`
}

type AgentReflectionService interface {
	PrepareThread(ctx context.Context, req *ReflectOnThreadRequest) (*ReflectionPlan, error)
	PrepareRun(ctx context.Context, req *ReflectOnRunRequest) (*ReflectionPlan, error)
	Finish(
		ctx context.Context,
		plan *ReflectionPlan,
		result *llmports.StructuredCompletionResult,
	) (*ReflectionOutcome, error)
	Fail(ctx context.Context, req *FailReflectionRequest) error
	ListConnection(
		ctx context.Context,
		req *ListAgentReflectionConnectionRequest,
	) (*pagination.CursorListResult[*agentreflection.Reflection], error)
}

type AgentReflectionScheduler interface {
	AfterTurn(ctx context.Context, req *ReflectOnThreadRequest)
	AfterRun(ctx context.Context, req *ReflectOnRunRequest)
}
