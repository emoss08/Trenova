package demandforecastservice

import (
	"cmp"
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/modules/operations/domain/demandforecast"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchmarket"
	"github.com/emoss08/trenova/internal/modules/operations/ports/demandforecastports"
	"github.com/emoss08/trenova/shared/pulid"
)

func (s *Service) Outlook(
	ctx context.Context,
	req *demandforecastports.OutlookRequest,
) (*demandforecastports.Outlook, error) {
	now := s.clock()
	hours := clamp(req.Hours, demandforecastports.DefaultOutlookHours,
		demandforecastports.MaxOutlookHours)
	limit := clamp(req.Limit, demandforecastports.DefaultOutlookLimit,
		demandforecastports.MaxOutlookLimit)

	model, err := s.repo.Model(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	outlook := &demandforecastports.Outlook{
		Quality:     model.Quality,
		GeneratedAt: now,
		Windows:     []*demandforecastports.OutlookWindow{},
	}
	if !model.Quality.Ready() || len(model.Areas) == 0 {
		return outlook, nil
	}

	starts, err := s.repo.ListDriverStarts(ctx, &demandforecastports.DriverStartsRequest{
		TenantInfo: req.TenantInfo,
		Now:        now,
	})
	if err != nil {
		return nil, err
	}
	margin, capCents, err := s.economics(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	horizon := int64(hours) * 3600
	location := s.location(ctx, req.TenantInfo)
	forecaster, err := s.forecaster(ctx, &forecasterInputs{
		tenantInfo: req.TenantInfo,
		now:        now,
		model:      model,
		supply:     SupplyOf(starts),
		margin:     margin,
		capCents:   capCents,
		location:   location,
		horizon:    horizon,
	})
	if err != nil {
		return nil, err
	}

	for _, cell := range forecaster.Cells() {
		first := dispatchmarket.SlotStart(now)
		for start := first; start < now+horizon; start += windowSeconds {
			valuation := forecaster.ValueAt(
				demandforecast.WindowKey{Cell: cell, Start: start},
				pulid.Nil,
			)
			reason := valuation.Reason
			if reason.TypicalHigh == 0 && reason.Booked == 0 && reason.ExpectedUnbooked < 0.5 {
				continue
			}
			outlook.Windows = append(outlook.Windows, &demandforecastports.OutlookWindow{
				Cell:             cell,
				AreaLabel:        reason.AreaLabel,
				WindowStart:      reason.WindowStart,
				WindowEnd:        reason.WindowEnd,
				TypicalLow:       reason.TypicalLow,
				TypicalHigh:      reason.TypicalHigh,
				LikelyLow:        reason.LikelyLow,
				LikelyHigh:       reason.LikelyHigh,
				Booked:           reason.Booked,
				ExpectedUnbooked: reason.ExpectedUnbooked,
				DriversExpected:  reason.DriversExpected,
				Shortfall:        reason.Shortfall(),
				ValueCents:       valuation.Cents,
				Customers:        reason.Customers,
				Explanation:      demandforecast.Explain(reason, location),
			})
		}
	}

	slices.SortStableFunc(outlook.Windows, func(a, b *demandforecastports.OutlookWindow) int {
		if c := cmp.Compare(b.Shortfall, a.Shortfall); c != 0 {
			return c
		}
		if c := cmp.Compare(b.ValueCents, a.ValueCents); c != 0 {
			return c
		}
		return cmp.Compare(a.WindowStart, b.WindowStart)
	})
	outlook.Windows = outlook.Windows[:min(len(outlook.Windows), limit)]

	return outlook, nil
}

func SupplyOf(starts []demandforecastports.DriverStart) []demandforecast.Driver {
	out := make([]demandforecast.Driver, 0, len(starts))
	for _, start := range starts {
		cell, ok := dispatchmarket.CellOf(dispatchmarket.Point{
			Latitude:  start.Latitude,
			Longitude: start.Longitude,
		})
		if !ok {
			continue
		}
		out = append(out, demandforecast.Driver{
			WorkerID: start.WorkerID,
			Cell:     cell,
			FreeAt:   start.FreeAt,
		})
	}

	return out
}

func clamp(value, fallback, ceiling int) int {
	if value <= 0 {
		return fallback
	}

	return min(value, ceiling)
}
