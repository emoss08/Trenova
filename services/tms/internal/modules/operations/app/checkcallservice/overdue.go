package checkcallservice

import (
	"cmp"
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/modules/operations/ports/checkcallports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchcontrolports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

func (s *Service) ListOverdue(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	now int64,
) ([]*checkcallports.OverdueMove, error) {
	control, err := s.dispatchControlRepo.GetByOrgID(
		ctx,
		dispatchcontrolports.GetDispatchControlRequest{
			TenantInfo: tenantInfo,
		},
	)
	if err != nil && !errortypes.IsNotFoundError(err) {
		return nil, err
	}
	if err != nil || control == nil || !control.EnableCheckCallMonitoring {
		return []*checkcallports.OverdueMove{}, nil
	}

	monitored, err := s.repo.ListMonitoredMoves(ctx, tenantInfo, 0)
	if err != nil {
		return nil, err
	}

	overdue := make([]*checkcallports.OverdueMove, 0, len(monitored))
	for _, move := range monitored {
		if move.LastContactAt <= 0 {
			continue
		}
		hours := control.CheckCallWindowHours(move.CustomerIntervalHours)
		deadline := move.LastContactAt + int64(hours)*secondsPerHour
		if now <= deadline {
			continue
		}
		overdue = append(overdue, &checkcallports.OverdueMove{
			MoveID:        move.MoveID,
			ShipmentID:    move.ShipmentID,
			ProNumber:     move.ProNumber,
			CarrierID:     move.CarrierID,
			CarrierName:   move.CarrierName,
			LastContactAt: move.LastContactAt,
			IntervalHours: hours,
			OverdueBy:     now - deadline,
		})
	}
	slices.SortStableFunc(overdue, func(a, b *checkcallports.OverdueMove) int {
		return cmp.Compare(b.OverdueBy, a.OverdueBy)
	})
	return overdue, nil
}
