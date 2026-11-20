package pagetourresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/pagetour"
)

func pageTourViewToModel(view *pagetour.View) *gqlmodel.PageTourView {
	return &gqlmodel.PageTourView{
		Tour:        view.Tour,
		TourVersion: int(view.TourVersion),
		Outcome:     gqlmodel.PageTourOutcome(view.Outcome),
		SeenAt:      int(view.SeenAt),
	}
}
