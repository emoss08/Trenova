package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// AIUsageAttribution is what a completion request says about who it is for.
// Everything is optional: a probe has no user, a chat turn has no run.
type AIUsageAttribution struct {
	UserID            pulid.ID
	AgentDefinitionID pulid.ID
	ThreadID          pulid.ID
	RunID             pulid.ID
	Purpose           AIUsagePurpose
}

type AIUsagePurpose string

const (
	AIUsagePurposeLive       = AIUsagePurpose("")
	AIUsagePurposeEvaluation = AIUsagePurpose("Evaluation")
)

func UsagePurposeForRun(runID pulid.ID) AIUsagePurpose {
	if runID.Prefix() == agent.EvaluationIDPrefix {
		return AIUsagePurposeEvaluation
	}

	return AIUsagePurposeLive
}

func (a AIUsageAttribution) ResolvedPurpose() AIUsagePurpose {
	if a.Purpose != AIUsagePurposeLive {
		return a.Purpose
	}

	return UsagePurposeForRun(a.RunID)
}

func (a AIUsageAttribution) Evaluates() bool {
	return a.ResolvedPurpose() == AIUsagePurposeEvaluation
}

// AIUsageProviderSlice is one provider's share of a window.
type AIUsageProviderSlice struct {
	ProviderID      pulid.ID `json:"providerId"`
	ProviderName    string   `json:"providerName"`
	Model           string   `json:"model"`
	Calls           int      `json:"calls"`
	Failed          int      `json:"failed"`
	InputTokens     int64    `json:"inputTokens"`
	OutputTokens    int64    `json:"outputTokens"`
	ReasoningTokens int64    `json:"reasoningTokens"`
	CostUSD         string   `json:"costUsd"`
	PricedCalls     int      `json:"pricedCalls"`
	LatencyP50Ms    int64    `json:"latencyP50Ms"`
	LatencyP95Ms    int64    `json:"latencyP95Ms"`
}

// AIUsageSummary is what the organization's models did over a window: how
// many calls, how many failed, what they consumed, what it cost where the
// price is known, and how long a person waited.
type AIUsageSummary struct {
	Since           int64                  `json:"since"`
	Calls           int                    `json:"calls"`
	Failed          int                    `json:"failed"`
	InputTokens     int64                  `json:"inputTokens"`
	OutputTokens    int64                  `json:"outputTokens"`
	ReasoningTokens int64                  `json:"reasoningTokens"`
	CostUSD         string                 `json:"costUsd"`
	PricedCalls     int                    `json:"pricedCalls"`
	LatencyP50Ms    int64                  `json:"latencyP50Ms"`
	LatencyP95Ms    int64                  `json:"latencyP95Ms"`
	ByProvider      []AIUsageProviderSlice `json:"byProvider"`
	// RecentFailures are the newest failed calls in the window, so the
	// reason a provider keeps failing is on the screen that counts the
	// failures rather than only in the server log.
	RecentFailures []AIUsageFailure `json:"recentFailures"`
}

// AIUsageFailure is one failed model call as the overview shows it.
type AIUsageFailure struct {
	ProviderID   pulid.ID `json:"providerId"`
	ProviderName string   `json:"providerName"`
	Model        string   `json:"model"`
	Task         string   `json:"task"`
	ErrorClass   string   `json:"errorClass"`
	Message      string   `json:"message"`
	At           int64    `json:"at"`
}

type AIUsageService interface {
	Summary(ctx context.Context, tenant pagination.TenantInfo, since int64) (*AIUsageSummary, error)
}

// SummaryFromRepository shapes a repository summary for the API.
func SummaryFromRepository(since int64, summary *repositories.AIUsageSummary) *AIUsageSummary {
	out := &AIUsageSummary{
		Since:           since,
		Calls:           summary.Totals.Calls,
		Failed:          summary.Totals.Failed,
		InputTokens:     summary.Totals.InputTokens,
		OutputTokens:    summary.Totals.OutputTokens,
		ReasoningTokens: summary.Totals.ReasoningTokens,
		CostUSD:         summary.Totals.CostUSD,
		PricedCalls:     summary.Totals.PricedCalls,
		LatencyP50Ms:    summary.Totals.LatencyP50,
		LatencyP95Ms:    summary.Totals.LatencyP95,
		ByProvider:      make([]AIUsageProviderSlice, 0, len(summary.ByProvider)),
		RecentFailures:  make([]AIUsageFailure, 0, len(summary.RecentFailures)),
	}
	for _, failure := range summary.RecentFailures {
		out.RecentFailures = append(out.RecentFailures, AIUsageFailure{
			ProviderID:   failure.ProviderID,
			ProviderName: failure.ProviderName,
			Model:        failure.Model,
			Task:         failure.Task,
			ErrorClass:   failure.ErrorClass,
			Message:      failure.Message,
			At:           failure.At,
		})
	}
	for _, slice := range summary.ByProvider {
		out.ByProvider = append(out.ByProvider, AIUsageProviderSlice{
			ProviderID:      slice.ProviderID,
			ProviderName:    slice.ProviderName,
			Model:           slice.Model,
			Calls:           slice.Calls,
			Failed:          slice.Failed,
			InputTokens:     slice.InputTokens,
			OutputTokens:    slice.OutputTokens,
			ReasoningTokens: slice.ReasoningTokens,
			CostUSD:         slice.CostUSD,
			PricedCalls:     slice.PricedCalls,
			LatencyP50Ms:    slice.LatencyP50,
			LatencyP95Ms:    slice.LatencyP95,
		})
	}

	return out
}
