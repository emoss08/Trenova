package agentrosterservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentroster"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentrosterports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Facts agentrosterports.AIControlFactsRepository
}

type Service struct {
	facts agentrosterports.AIControlFactsRepository
	now   func() int64
}

var _ agentrosterports.AgentRosterService = (*Service)(nil)

func New(p Params) *Service {
	return &Service{facts: p.Facts, now: timeutils.NowUnix}
}

func (s *Service) Roster(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*agentroster.Stat, error) {
	now := s.now()
	return s.facts.AgentRoster(ctx, &agentrosterports.AgentRosterRequest{
		TenantInfo:     tenantInfo,
		RunsSince:      agentroster.RunsSince(now),
		DecisionsSince: agentroster.DecisionsSince(now),
	})
}
