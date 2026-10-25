package aifeedbackports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aifeedback"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const MaxAIFeedbackTargetsPerRead = 200

type AIFeedbackTargetRef struct {
	TargetType aifeedback.TargetType
	TargetID   pulid.ID
	TargetPart string
}

type DeleteAIFeedbackRequest struct {
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
	Target     AIFeedbackTargetRef
}

type ListAIFeedbackForTargetsRequest struct {
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
	Targets    []AIFeedbackTargetRef
}

type ListAIFeedbackConnectionRequest struct {
	Filter  *pagination.QueryOptions `json:"filter"`
	Cursor  pagination.CursorInfo    `json:"-"`
	Columns []string                 `json:"-"`
}

type ListAIFeedbackByIDsRequest struct {
	TenantInfo pagination.TenantInfo
	IDs        []pulid.ID
}

type AIFeedbackWindowRequest struct {
	TenantInfo        pagination.TenantInfo
	AgentDefinitionID pulid.ID
	Since             int64
	Timezone          string
	Limit             int
	Offset            int
}

type AIFeedbackAgentTotalsRequest struct {
	TenantInfo         pagination.TenantInfo
	AgentDefinitionIDs []pulid.ID
	Since              int64
	Until              int64
}

type AIFeedbackAgentTotals struct {
	AgentDefinitionID pulid.ID
	Positive          int
	Negative          int
}

type AIFeedbackDay struct {
	Day      string
	Positive int
	Negative int
}

type AIFeedbackTargetScore struct {
	TargetType  aifeedback.TargetType
	TargetID    pulid.ID
	TargetPart  string
	Positive    int
	Negative    int
	LastRatedAt int64
	SampleID    pulid.ID
}

type ListNegativeAIFeedbackRequest struct {
	TenantInfo pagination.TenantInfo
	Since      int64
	Limit      int
}

type LinkAIFeedbackEvalCaseRequest struct {
	TenantInfo pagination.TenantInfo
	FeedbackID pulid.ID
	EvalCaseID pulid.ID
}

type PurgeAIFeedbackRequest struct {
	TenantInfo pagination.TenantInfo
	Before     int64
	Limit      int
}

type AIFeedbackRepository interface {
	Upsert(ctx context.Context, entity *aifeedback.Feedback) (*aifeedback.Feedback, error)
	Delete(ctx context.Context, req DeleteAIFeedbackRequest) (bool, error)
	ListForTargets(
		ctx context.Context,
		req ListAIFeedbackForTargetsRequest,
	) ([]*aifeedback.Feedback, error)
	ListByIDs(ctx context.Context, req ListAIFeedbackByIDsRequest) ([]*aifeedback.Feedback, error)
	ListConnection(
		ctx context.Context,
		req *ListAIFeedbackConnectionRequest,
	) (*pagination.CursorListResult[*aifeedback.Feedback], error)
	DailySatisfaction(ctx context.Context, req AIFeedbackWindowRequest) ([]*AIFeedbackDay, error)
	WorstRated(ctx context.Context, req AIFeedbackWindowRequest) ([]*AIFeedbackTargetScore, error)
	TotalsByAgent(
		ctx context.Context,
		req AIFeedbackAgentTotalsRequest,
	) ([]*AIFeedbackAgentTotals, error)
	ListNegativeSince(
		ctx context.Context,
		req ListNegativeAIFeedbackRequest,
	) ([]*aifeedback.Feedback, error)
	PurgeBefore(ctx context.Context, req PurgeAIFeedbackRequest) (int64, error)
	LinkEvalCase(ctx context.Context, req LinkAIFeedbackEvalCaseRequest) error
}

type GetAIFeedbackMessageRequest struct {
	TenantInfo pagination.TenantInfo
	MessageID  pulid.ID
}

type AIFeedbackMessageContext struct {
	Message  *conversation.Message
	Thread   *conversation.Thread
	Exchange []conversation.Message
	Turn     *conversation.AssistantTurn
}

type AIFeedbackSourceRepository interface {
	GetMessageContext(
		ctx context.Context,
		req GetAIFeedbackMessageRequest,
	) (*AIFeedbackMessageContext, error)
}
