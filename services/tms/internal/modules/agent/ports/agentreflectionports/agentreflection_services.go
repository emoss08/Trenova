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

// LearningRow is one way agents came to hold memories over a stretch of days,
// and what became of them: what a person typed, what an agent saved when
// asked, or what a look back kept after one kind of trigger. The look back
// figures are nought for a source that is not a look back.
type LearningRow struct {
	Source        agentmemory.MemorySource
	Trigger       *agentreflection.ReflectionSignalKind
	LookBacks     int
	TaughtNothing int
	ToldAgain     int
	InputTokens   int
	OutputTokens  int
	Kept          int
	Waiting       int
	Dismissed     int
	Lapsed        int
	Retired       int
	Used          int
}

type LearningStats struct {
	Days  int
	Since int64
	Rows  []*LearningRow
}

type LearningStatsRequest struct {
	TenantInfo pagination.TenantInfo
	Days       int
}

type AgentReflectionService interface {
	// LearningStats says, for the last days, how each way of learning did:
	// what it cost, what it offered and what people did with the offers.
	LearningStats(ctx context.Context, req *LearningStatsRequest) (*LearningStats, error)
	PrepareThread(ctx context.Context, req *ReflectOnThreadRequest) (*ReflectionPlan, error)
	PrepareRun(ctx context.Context, req *ReflectOnRunRequest) (*ReflectionPlan, error)
	// PlanCorrections finds the habits of correction worth a look back now:
	// a field an agent set that people changed by hand on several records,
	// that no look back has read lately.
	PlanCorrections(
		ctx context.Context,
		tenant pagination.TenantInfo,
		now int64,
	) ([]*ReflectOnCorrectionsRequest, error)
	PrepareCorrections(
		ctx context.Context,
		req *ReflectOnCorrectionsRequest,
	) (*ReflectionPlan, error)
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
	AfterCorrections(ctx context.Context, req *ReflectOnCorrectionsRequest)
}
