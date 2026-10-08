package agentactivitysummaryservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
)

const secondsPerDay = int64(24 * 60 * 60)

type Params struct {
	fx.In

	Repo repositories.AgentActivityRepository
}

type Service struct {
	repo repositories.AgentActivityRepository
	now  func() int64
}

var _ services.AgentActivityService = (*Service)(nil)

func New(p Params) services.AgentActivityService {
	return &Service{repo: p.Repo, now: timeutils.NowUnix}
}

func (s *Service) Summary(
	ctx context.Context,
	req *services.AgentActivitySummaryRequest,
) (*services.AgentActivitySummary, error) {
	now := s.now()
	if req.Since > now || req.Since < now-services.MaxAgentActivityWindowDays*secondsPerDay {
		return nil, errortypes.NewValidationError(
			"since",
			errortypes.ErrInvalid,
			"Activity can be summarized from at most {0} days ago",
			services.MaxAgentActivityWindowDays,
		)
	}

	totals, err := s.repo.Totals(ctx, repositories.AgentActivityTotalsRequest{
		TenantInfo:     req.TenantInfo,
		RunsSince:      req.Since,
		DecisionsSince: now - services.AgentActivityDecisionWindowDays*secondsPerDay,
	})
	if err != nil {
		return nil, err
	}

	decided := totals.DecisionsAccepted + totals.DecisionsModified + totals.DecisionsRejected
	summary := &services.AgentActivitySummary{
		Since:              req.Since,
		Runs:               totals.Runs,
		RunsFailed:         totals.RunsFailed,
		RunsWorking:        totals.RunsWorking,
		RunsAwaiting:       totals.RunsAwaiting,
		PendingProposals:   totals.PendingProposals,
		OldestPendingAt:    totals.OldestPendingAt,
		OpenExceptions:     totals.OpenExceptions,
		DecisionWindowDays: services.AgentActivityDecisionWindowDays,
		Decided:            decided,
	}
	if decided > 0 {
		share := float64(totals.DecisionsAccepted) / float64(decided)
		summary.ApprovedAsProposed = &share
	}

	return summary, nil
}
