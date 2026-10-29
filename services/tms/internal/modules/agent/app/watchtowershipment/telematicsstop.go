package watchtowershipment

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/watchtoweritem"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/telematicsports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
)

const stopVisitWindow = 7 * 24 * 60 * 60

type TelematicsStopVisitSource struct {
	repo telematicsports.TelematicsRepository
	now  func() int64
}

func NewTelematicsStopVisitSource(
	repo telematicsports.TelematicsRepository,
) watchtowerports.WatchtowerSource {
	return &TelematicsStopVisitSource{repo: repo, now: timeutils.NowUnix}
}

func (s *TelematicsStopVisitSource) Kind() watchtower.SourceKind {
	return watchtower.SourceTelematicsStopVisit
}

func (s *TelematicsStopVisitSource) Snapshot(
	ctx context.Context,
	tenant pagination.TenantInfo,
) ([]watchtowerports.WatchtowerItemInput, error) {
	reviews, err := s.repo.ListOpenStopReviews(ctx, &telematicsports.ListOpenStopReviewsRequest{
		TenantInfo: tenant,
		Since:      s.now() - stopVisitWindow,
		Limit:      watchtoweritem.SnapshotLimit,
	})
	if err != nil {
		return nil, err
	}

	return DescribeTelematicsStopVisits(reviews), nil
}

func DescribeTelematicsStopVisits(
	reviews []*telematicsports.StopReview,
) []watchtowerports.WatchtowerItemInput {
	items := make([]watchtowerports.WatchtowerItemInput, 0, len(reviews))
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
