package agentreflectionports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentreflection"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetAgentReflectionRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
}

type ThreadReflectedThroughRequest struct {
	ThreadID   pulid.ID
	TenantInfo pagination.TenantInfo
}

type ListAgentReflectionConnectionRequest struct {
	Filter  *pagination.QueryOptions `json:"filter"`
	Cursor  pagination.CursorInfo    `json:"-"`
	Columns []string                 `json:"-"`
}

// FailInterruptedRequest marks a conversation's look backs that were claimed
// and never finished as failed, all but the one being prepared now.
type FailInterruptedRequest struct {
	TenantInfo pagination.TenantInfo
	ThreadID   pulid.ID
	ExceptID   pulid.ID
	Message    string
	At         int64
}

// ListLearningTriggersRequest asks what the look backs that called a model
// since a time cost and found, by what made each one look.
type ListLearningTriggersRequest struct {
	TenantInfo pagination.TenantInfo
	Since      int64
}

// LearningTrigger is the look backs one trigger started. A look back with
// several signals is counted under its first, which is its strongest: what a
// person corrected or asked for comes before what they only mentioned.
type LearningTrigger struct {
	Trigger       string
	LookBacks     int64
	TaughtNothing int64
	ToldAgain     int64
	InputTokens   int64
	OutputTokens  int64
}

type AgentReflectionRepository interface {
	Claim(
		ctx context.Context,
		entity *agentreflection.Reflection,
	) (*agentreflection.Reflection, error)
	GetByID(ctx context.Context, req GetAgentReflectionRequest) (*agentreflection.Reflection, error)
	Update(
		ctx context.Context,
		entity *agentreflection.Reflection,
	) (*agentreflection.Reflection, error)
	ThreadReflectedThrough(
		ctx context.Context,
		req ThreadReflectedThroughRequest,
	) (int, bool, error)
	ListConnection(
		ctx context.Context,
		req *ListAgentReflectionConnectionRequest,
	) (*pagination.CursorListResult[*agentreflection.Reflection], error)
	FailInterrupted(ctx context.Context, req *FailInterruptedRequest) error
	ListCorrectionLookBacks(
		ctx context.Context,
		req *ListCorrectionLookBacksRequest,
	) ([]string, error)
	ListLearningTriggers(
		ctx context.Context,
		req *ListLearningTriggersRequest,
	) ([]LearningTrigger, error)
}
