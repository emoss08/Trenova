package pagetourresolver

import (
	"math"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/pagetour"
)

func tourVersionOf(version int) int32 {
	if version < 0 || version > math.MaxInt32 {
		return 0
	}

	return int32(version)
}

func pageTourViewToModel(view *pagetour.View) *gqlmodel.PageTourView {
	return &gqlmodel.PageTourView{
		Tour:        view.Tour,
		TourVersion: int(view.TourVersion),
		Outcome:     gqlmodel.PageTourOutcome(view.Outcome),
		SeenAt:      int(view.SeenAt),
	}
}
