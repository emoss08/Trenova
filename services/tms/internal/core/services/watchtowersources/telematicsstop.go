package watchtowersources

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/watchtower"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
)

const stopVisitWindow = 7 * 24 * 60 * 60

type TelematicsStopVisitSource struct {
	repo repositories.TelematicsRepository
	now  func() int64
}

func NewTelematicsStopVisitSource(
	repo repositories.TelematicsRepository,
) services.WatchtowerSource {
	return &TelematicsStopVisitSource{repo: repo, now: timeutils.NowUnix}
}

func (s *TelematicsStopVisitSource) Kind() watchtower.SourceKind {
	return watchtower.SourceTelematicsStopVisit
}

func (s *TelematicsStopVisitSource) Snapshot(
	ctx context.Context,
	tenant pagination.TenantInfo,
) ([]services.WatchtowerItemInput, error) {
	reviews, err := s.repo.ListOpenStopReviews(ctx, &repositories.ListOpenStopReviewsRequest{
		TenantInfo: tenant,
		Since:      s.now() - stopVisitWindow,
		Limit:      snapshotLimit,
	})
	if err != nil {
		return nil, err
	}

	return DescribeTelematicsStopVisits(reviews), nil
}

func DescribeTelematicsStopVisits(
	reviews []*repositories.StopReview,
) []services.WatchtowerItemInput {
	items := make([]services.WatchtowerItemInput, 0, len(reviews))
	seen := make(map[string]struct{}, len(reviews))
	for _, review := range reviews {
		if review == nil || review.Event == nil {
			continue
		}
		item := DescribeTelematicsStopVisit(review)
		if _, dup := seen[item.SourceID]; dup {
			continue
		}
		seen[item.SourceID] = struct{}{}
		items = append(items, item)
	}

	return items
}
