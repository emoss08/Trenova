package agentactivityports

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
)

const AgentActivityDecisionWindowDays = 7

const MaxAgentActivityWindowDays = 31

type AgentActivitySummaryRequest struct {
	TenantInfo pagination.TenantInfo
	Since      int64
}

type AgentActivitySummary struct {
	Since              int64
	Runs               int
	RunsFailed         int
	RunsWorking        int
	RunsAwaiting       int
	PendingProposals   int
	OldestPendingAt    *int64
	OpenExceptions     int
	DecisionWindowDays int
	Decided            int
	ApprovedAsProposed *float64
}

type AgentActivityService interface {
	Summary(ctx context.Context, req *AgentActivitySummaryRequest) (*AgentActivitySummary, error)
}
