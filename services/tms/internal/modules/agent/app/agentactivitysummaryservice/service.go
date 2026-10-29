package agentactivitysummaryservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentactivityports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
)

const secondsPerDay = int64(24 * 60 * 60)

type Params struct {
	fx.In

	Repo agentactivityports.AgentActivityRepository
}

type Service struct {
	repo agentactivityports.AgentActivityRepository
	now  func() int64
}

var _ agentactivityports.AgentActivityService = (*Service)(nil)

func New(p Params) agentactivityports.AgentActivityService {
	return &Service{repo: p.Repo, now: timeutils.NowUnix}
}

func (s *Service) Summary(
	ctx context.Context,
	req *agentactivityports.AgentActivitySummaryRequest,
) (*agentactivityports.AgentActivitySummary, error) {
	now := s.now()
	if req.Since > now ||
		req.Since < now-agentactivityports.MaxAgentActivityWindowDays*secondsPerDay {
		return nil, errortypes.NewValidationError(
			"since",
			errortypes.ErrInvalid,
			"Activity can be summarized from at most {0} days ago",
			agentactivityports.MaxAgentActivityWindowDays,
		)
	}

	totals, err := s.repo.Totals(ctx, agentactivityports.AgentActivityTotalsRequest{
		TenantInfo:     req.TenantInfo,
		RunsSince:      req.Since,
		DecisionsSince: now - agentactivityports.AgentActivityDecisionWindowDays*secondsPerDay,
	})
	if err != nil {
		return nil, err
	}

	decided := totals.DecisionsAccepted + totals.DecisionsModified + totals.DecisionsRejected
	summary := &agentactivityports.AgentActivitySummary{
		Since:              req.Since,
		Runs:               totals.Runs,
		RunsFailed:         totals.RunsFailed,
		RunsWorking:        totals.RunsWorking,
		RunsAwaiting:       totals.RunsAwaiting,
		PendingProposals:   totals.PendingProposals,
		OldestPendingAt:    totals.OldestPendingAt,
		OpenExceptions:     totals.OpenExceptions,
		DecisionWindowDays: agentactivityports.AgentActivityDecisionWindowDays,
		Decided:            decided,
	}
	if decided > 0 {
		share := float64(totals.DecisionsAccepted) / float64(decided)
		summary.ApprovedAsProposed = &share
	}

	return summary, nil
}
