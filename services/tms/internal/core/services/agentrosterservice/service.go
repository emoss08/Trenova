package agentrosterservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agentroster"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Facts repositories.AIControlFactsRepository
}

type Service struct {
	facts repositories.AIControlFactsRepository
	now   func() int64
}

var _ services.AgentRosterService = (*Service)(nil)

func New(p Params) *Service {
	return &Service{facts: p.Facts, now: timeutils.NowUnix}
}

func (s *Service) Roster(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*agentroster.Stat, error) {
	now := s.now()
	return s.facts.AgentRoster(ctx, &repositories.AgentRosterRequest{
		TenantInfo:     tenantInfo,
		RunsSince:      agentroster.RunsSince(now),
		DecisionsSince: agentroster.DecisionsSince(now),
	})
}
