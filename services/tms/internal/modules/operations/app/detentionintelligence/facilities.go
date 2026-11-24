package detentionintelligence

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/protectedfield"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

const (
	defaultFacilities = 10
	maxFacilities     = 50
)

type FacilitiesRequest struct {
	TenantInfo pagination.TenantInfo
	Period     boardcatalog.Period
	Limit      int
	Visible    protectedfield.GroupSet
}

func (s *Service) Facilities(
	ctx context.Context,
	req *FacilitiesRequest,
) ([]detentionports.FacilityIntelligence, error) {
	limit := req.Limit
	if limit == 0 {
		limit = defaultFacilities
	}
	if limit < 1 || limit > maxFacilities {
		return nil, errortypes.NewValidationError("limit", errortypes.ErrInvalid,
			"Rank between 1 and 50 facilities at a time")
	}
	zone, loc, err := s.zone(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	first, last, ok := req.Period.Days(s.now(), loc)
	if !ok {
		return nil, errortypes.NewValidationError("period", errortypes.ErrInvalid,
			"Facilities are ranked over whole days: today, yesterday, this or last week, "+
				"month or quarter")
	}

	facilities, err := s.repo.FacilityFigures(ctx, &detentionports.IntelligenceRequest{
		TenantInfo: req.TenantInfo,
		Current:    detentionports.NewIntelligenceWindow(first, last),
		Timezone:   zone,
		Limit:      limit,
	})
	if err != nil {
		return nil, err
	}
	for idx := range facilities {
		facilities[idx].WithholdFields(req.Visible)
	}

	return facilities, nil
}
