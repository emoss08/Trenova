package aiusageservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Usage repositories.AIUsageRepository
}

type Service struct {
	usage repositories.AIUsageRepository
}

func New(p Params) services.AIUsageService {
	return &Service{usage: p.Usage}
}

// maxWindow bounds a summary to a year: the table has no other index than
// time, and a window nobody asked for is a scan nobody wants.
const maxWindow = 366 * 24 * 60 * 60

func (s *Service) Summary(
	ctx context.Context,
	tenant pagination.TenantInfo,
	since int64,
) (*services.AIUsageSummary, error) {
	now := timeutils.NowUnix()
	if since <= 0 || since > now {
		since = now - 7*24*60*60
	}
	if now-since > maxWindow {
		since = now - maxWindow
	}

	summary, err := s.usage.Summary(ctx, repositories.AIUsageSummaryRequest{
		TenantInfo: tenant,
		Since:      since,
	})
	if err != nil {
		return nil, err
	}

	return services.SummaryFromRepository(since, summary), nil
}
