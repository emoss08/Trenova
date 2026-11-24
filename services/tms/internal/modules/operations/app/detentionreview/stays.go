package detentionreview

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/app/detentionengine"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const (
	defaultStays  = 25
	secondsInDay  = 24 * 60 * 60
	minuteSeconds = 60
)

type StaysRequest struct {
	TenantInfo pagination.TenantInfo
	Period     boardcatalog.Period
	Stage      detentionports.StayStage
	LocationID pulid.ID
	Limit      int
	Offset     int
}

type Stay struct {
	Occurrence      *detention.DetentionOccurrence
	MinutesOnSite   int32
	MinutesPastFree int32
	BillableAmount  decimal.Decimal
	PricedAt        int64
	HoldBack        detention.HoldBack
}

type StayList struct {
	Items []*Stay
	Total int
}

func (s *Service) ListStays(ctx context.Context, req *StaysRequest) (*StayList, error) {
	limit := req.Limit
	if limit == 0 {
		limit = defaultStays
	}
	if limit < 1 || limit > detentionports.MaxStays {
		return nil, errortypes.NewValidationError("limit", errortypes.ErrInvalid,
			"List between 1 and 100 detention stays at a time")
	}
	if req.Offset < 0 {
		return nil, errortypes.NewValidationError("offset", errortypes.ErrInvalid,
			"The offset cannot be negative")
	}
	location, err := s.effects.TenantLocation(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	now := s.now()
	first, last, ok := req.Period.Days(now, location)
	if !ok {
		return nil, errortypes.NewValidationError("period", errortypes.ErrInvalid,
			"Detention stays are listed by whole days: today, yesterday, this or last week, "+
				"month or quarter")
	}

	page, err := s.stays.ListStays(ctx, &detentionports.ListDetentionStaysRequest{
		TenantInfo: req.TenantInfo,
		From:       first.Unix(),
		To:         last.Unix() + secondsInDay - 1,
		Stage:      req.Stage,
		LocationID: req.LocationID,
		Limit:      limit,
		Offset:     req.Offset,
	})
	if err != nil {
		return nil, err
	}

	running := make([]*detention.DetentionOccurrence, 0)
	for _, stay := range page.Items {
		if stay.Occurrence.IsOpen {
			running = append(running, stay.Occurrence)
		}
	}
	priced, err := s.priceOpenClocks(ctx, req.TenantInfo, running, now)
	if err != nil {
		return nil, err
	}

	out := &StayList{Items: make([]*Stay, 0, len(page.Items)), Total: page.Total}
	for _, found := range page.Items {
		out.Items = append(out.Items, stayOf(found, priced, now))
	}

	return out, nil
}

func stayOf(
	found *detentionports.DetentionStay,
	priced map[pulid.ID]detentionengine.ComputeResult,
	now int64,
) *Stay {
	occurrence := found.Occurrence
	stay := &Stay{
		Occurrence:     occurrence,
		MinutesOnSite:  occurrence.RawDwellMinutes,
		BillableAmount: occurrence.BillableAmount,
		HoldBack:       found.HoldBack(now),
	}
	if occurrence.IsOpen && occurrence.ClockStartAt > 0 && now > occurrence.ClockStartAt {
		stay.MinutesOnSite = intutils.SafeToInt32((now - occurrence.ClockStartAt) / minuteSeconds)
		if current, ok := priced[occurrence.ID]; ok {
			stay.BillableAmount = current.BillableAmount
			stay.PricedAt = now
		}
	}
	stay.MinutesPastFree = max(stay.MinutesOnSite-occurrence.FreeMinutesGranted, 0)

	return stay
}
