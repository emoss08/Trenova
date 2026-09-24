package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/aifeedback"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	DefaultAgentFeedbackWindowDays = 30
	MaxAgentFeedbackWindowDays     = 365
	DefaultAgentFeedbackWorstLimit = 10
)

type SetAIFeedbackRequest struct {
	TenantInfo pagination.TenantInfo
	Target     repositories.AIFeedbackTargetRef
	Rating     aifeedback.Rating
	Reasons    []aifeedback.Reason
	Comment    string
}

type ClearAIFeedbackRequest struct {
	TenantInfo pagination.TenantInfo
	Target     repositories.AIFeedbackTargetRef
}

type ListMyAIFeedbackRequest struct {
	TenantInfo pagination.TenantInfo
	Targets    []repositories.AIFeedbackTargetRef
}

type AgentFeedbackSummaryRequest struct {
	TenantInfo        pagination.TenantInfo
	AgentDefinitionID pulid.ID
	WindowDays        int
	WorstLimit        int
}

type AgentFeedbackDay struct {
	Day          string   `json:"day"`
	Positive     int      `json:"positive"`
	Negative     int      `json:"negative"`
	Satisfaction *float64 `json:"satisfaction"`
}

type AgentFeedbackWorstRated struct {
	TargetType  aifeedback.TargetType `json:"targetType"`
	TargetID    pulid.ID              `json:"targetId"`
	TargetPart  string                `json:"targetPart"`
	Positive    int                   `json:"positive"`
	Negative    int                   `json:"negative"`
	LastRatedAt int64                 `json:"lastRatedAt"`
	Sample      *aifeedback.Feedback  `json:"sample"`
}

type AgentFeedbackSummary struct {
	AgentDefinitionID pulid.ID                   `json:"agentDefinitionId"`
	WindowDays        int                        `json:"windowDays"`
	Since             int64                      `json:"since"`
	Positive          int                        `json:"positive"`
	Negative          int                        `json:"negative"`
	Satisfaction      *float64                   `json:"satisfaction"`
	Days              []*AgentFeedbackDay        `json:"days"`
	WorstRated        []*AgentFeedbackWorstRated `json:"worstRated"`
}

type SuggestAgentMemoriesRequest struct {
	TenantInfo pagination.TenantInfo
	Now        int64
}

type SuggestAgentMemoriesResult struct {
	Suggested  int `json:"suggested"`
	Covered    int `json:"covered"`
	BelowFloor int `json:"belowFloor"`
}

type PurgeExpiredAIFeedbackRequest struct {
	TenantInfo pagination.TenantInfo
	Now        int64
}

type AIFeedbackService interface {
	SetMine(
		ctx context.Context,
		req *SetAIFeedbackRequest,
		actor *RequestActor,
	) (*aifeedback.Feedback, error)
	ClearMine(ctx context.Context, req ClearAIFeedbackRequest, actor *RequestActor) (bool, error)
	ListMine(
		ctx context.Context,
		req ListMyAIFeedbackRequest,
		actor *RequestActor,
	) ([]*aifeedback.Feedback, error)
	ListConnection(
		ctx context.Context,
		req *repositories.ListAIFeedbackConnectionRequest,
	) (*pagination.CursorListResult[*aifeedback.Feedback], error)
	AgentSummary(
		ctx context.Context,
		req AgentFeedbackSummaryRequest,
	) (*AgentFeedbackSummary, error)
}

type AIFeedbackMaintenance interface {
	SuggestMemories(
		ctx context.Context,
		req SuggestAgentMemoriesRequest,
	) (*SuggestAgentMemoriesResult, error)
	PurgeExpired(ctx context.Context, req PurgeExpiredAIFeedbackRequest) (int64, error)
}
