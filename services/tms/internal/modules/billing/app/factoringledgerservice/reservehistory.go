package factoringledgerservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringledgerports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/timeutils"
)

const MaxReserveHistoryDays = 365

func (s *LedgerService) ReserveHistory(
	ctx context.Context,
	req factoringledgerports.ReserveHistoryRequest,
) ([]*factoringledgerports.ReserveDay, error) {
	if req.FactoringCompanyID.IsNil() {
		return nil, errortypes.NewValidationError(
			"factoringCompanyId",
			errortypes.ErrRequired,
			"Choose the factor whose reserve to read",
		)
	}
	if req.Days < 1 || req.Days > MaxReserveHistoryDays {
		return nil, errortypes.NewValidationError(
			"days",
			errortypes.ErrInvalid,
			"The reserve history covers between a day and a year",
		)
	}

	today := timeutils.DayStartUTC(s.now())
	first := today - int64(req.Days-1)*timeutils.SecondsPerDay
	read, err := s.entries.GetHistory(ctx, factoringledgerports.GetReserveHistoryRequest{
		TenantInfo:         req.TenantInfo,
		FactoringCompanyID: req.FactoringCompanyID,
		Since:              first,
	})
	if err != nil {
		return nil, err
	}

	return walkBack(read, first, req.Days), nil
}

func walkBack(
	read *factoringledgerports.ReserveHistoryRead,
	first int64,
	days int,
) []*factoringledgerports.ReserveDay {
	after := make(map[int64]int64, len(read.Changes))
	var beyond int64
	last := first + int64(days-1)*timeutils.SecondsPerDay
	for _, change := range read.Changes {
		day := timeutils.DayStartUTC(change.Day)
		if day > last {
			beyond += change.ChangeMinor
			continue
		}
		after[day] += change.ChangeMinor
	}

	out := make([]*factoringledgerports.ReserveDay, days)
	balance := read.CurrentMinor - beyond
	for i := days - 1; i >= 0; i-- {
		day := first + int64(i)*timeutils.SecondsPerDay
		out[i] = &factoringledgerports.ReserveDay{Day: day, ReserveMinor: balance}
		balance -= after[day]
	}

	return out
}
