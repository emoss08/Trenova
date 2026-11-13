package demandforecastservice

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/demandfit"
	"github.com/emoss08/trenova/internal/modules/operations/domain/demandforecast"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchmarket"
	"github.com/emoss08/trenova/internal/modules/operations/ports/demandforecastports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchcontrolports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/pto"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/orgholidayports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/maypok86/otter/v2"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	historySpanWeeks   = demandfit.HistoryWeeks + demandfit.BacktestWeeks + 1
	bookedLookbackSecs = 2 * dispatchmarket.SlotSeconds
	bookedHorizonSecs  = int64(5 * 24 * 3600)
	windowSeconds      = (demandforecast.WindowSlots + 1) * dispatchmarket.SlotSeconds
	maxCachedWeight    = 2_000_000
	cachedIdle         = 6 * time.Hour
)

type modelKey struct {
	orgID pulid.ID
	buID  pulid.ID
}

type Params struct {
	fx.In

	Logger        *zap.Logger
	Repo          demandforecastports.Repository
	Fits          demandforecastports.FitStarter
	Holidays      orgholidayports.OrgHolidayRepository
	Controls      dispatchcontrolports.DispatchControlRepository
	Organizations tenantports.OrganizationRepository
}

type Service struct {
	l             *zap.Logger
	repo          demandforecastports.Repository
	fits          demandforecastports.FitStarter
	holidays      orgholidayports.OrgHolidayRepository
	models        *otter.Cache[modelKey, *demandforecast.Index]
	controls      dispatchcontrolports.DispatchControlRepository
	organizations tenantports.OrganizationRepository
	clock         func() int64
}

var _ demandforecastports.Service = (*Service)(nil)

func New(p Params) *Service {
	return &Service{
		l:        p.Logger.Named("service.demand-forecast"),
		repo:     p.Repo,
		fits:     p.Fits,
		holidays: p.Holidays,
		models: otter.Must(&otter.Options[modelKey, *demandforecast.Index]{
			MaximumWeight: maxCachedWeight,
			Weigher: func(_ modelKey, index *demandforecast.Index) uint32 {
				return uint32(min(index.Weight(), maxCachedWeight)) //nolint:gosec // clamped above
			},
			ExpiryCalculator: otter.ExpiryAccessing[modelKey, *demandforecast.Index](cachedIdle),
		}),
		controls:      p.Controls,
		organizations: p.Organizations,
		clock:         timeutils.NowUnix,
	}
}

func (s *Service) Fit(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*demandforecast.Quality, error) {
	now := s.clock()
	since := now - historySpanWeeks*dispatchmarket.WeekSeconds
	grid := dispatchmarket.GridIn(s.Location(ctx, tenantInfo))
	rows, err := s.repo.History(ctx, &demandforecastports.HistoryRequest{
		TenantInfo: tenantInfo,
		Since:      since,
		Until:      now,
		Timezone:   grid.Timezone(),
	})
	if err != nil {
		return nil, err
	}

	history := HistoryOf(now, rows)
	history.Grid = grid
	if history.Holidays, err = s.holidaysBetween(ctx, tenantInfo, grid, since, now); err != nil {
		return nil, err
	}
	trained := demandfit.Train(history)

	if err = s.repo.Replace(ctx, &demandforecastports.ReplaceRequest{
		TenantInfo: tenantInfo,
		Fitted:     trained.Fitted,
		Quality:    trained.Quality,
		Days:       trained.Days,
	}); err != nil {
		return nil, err
	}

	return trained.Quality, nil
}

func (s *Service) holidaysBetween(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	grid dispatchmarket.Grid,
	since, now int64,
) (demandforecast.Holidays, error) {
	rows, err := s.holidays.List(ctx, &orgholidayports.ListOrgHolidaysRequest{
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, fmt.Errorf("read the organization's holidays: %w", err)
	}
	observed := make([]*pto.OrgHoliday, 0, len(rows))
	for _, row := range rows {
		if row != nil && row.Kind == pto.HolidayKindHoliday {
			observed = append(observed, row)
		}
	}
	calendar := pto.NewHolidayCalendar(observed)

	holidays := demandforecast.Holidays{}
	last := grid.Day(now) + demandfit.HolidayHorizonDays
	for day := grid.Day(since); day <= last; day++ {
		if calendar.HolidayOn(time.Unix(day*dispatchmarket.DaySeconds, 0).UTC()) {
			holidays[day] = struct{}{}
		}
	}

	return holidays, nil
}

func HistoryOf(now int64, rows *demandforecastports.HistoryRows) *demandfit.History {
	cells := make(map[pulid.ID]int64, len(rows.Locations))
	type areaStats struct {
		label    string
		busiest  int32
		loads    int32
		revenue  float64
		priced   int32
		location pulid.ID
	}
	areas := map[int64]*areaStats{}
	for _, location := range rows.Locations {
		cell, ok := dispatchmarket.CellOf(dispatchmarket.Point{
			Latitude:  location.Latitude,
			Longitude: location.Longitude,
		})
		if !ok {
			continue
		}
		cells[location.LocationID] = cell
		stats := areas[cell]
		if stats == nil {
			stats = &areaStats{}
			areas[cell] = stats
		}
		stats.loads += location.Loads
		if location.MedianRevenueCents > 0 {
			stats.revenue += float64(location.MedianRevenueCents) * float64(location.Loads)
			stats.priced += location.Loads
		}
		if location.Loads > stats.busiest ||
			(location.Loads == stats.busiest && location.LocationID < stats.location) {
			stats.busiest, stats.location = location.Loads, location.LocationID
			stats.label = labelOf(location.City, location.State)
		}
	}

	history := &demandfit.History{
		Now:    now,
		Counts: make([]demandfit.LoadCount, 0, len(rows.Counts)),
		Leads:  rows.Leads,
		Areas:  make([]demandfit.AreaSummary, 0, len(areas)),
	}
	for _, count := range rows.Counts {
		cell, ok := cells[count.LocationID]
		if !ok {
			continue
		}
		history.Counts = append(history.Counts, demandfit.LoadCount{
			CustomerID: count.CustomerID,
			Cell:       cell,
			Week:       count.Week,
			Slot:       count.Slot,
			Loads:      count.Loads,
		})
	}
	for cell, stats := range areas {
		median := int64(0)
		if stats.priced > 0 {
			median = int64(stats.revenue / float64(stats.priced))
		}
		history.Areas = append(history.Areas, demandfit.AreaSummary{
			Cell:               cell,
			Label:              stats.label,
			MedianRevenueCents: median,
			Loads:              stats.loads,
		})
	}
	slices.SortFunc(history.Areas, func(a, b demandfit.AreaSummary) int {
		return cmp.Compare(a.Cell, b.Cell)
	})

	return history
}

func labelOf(city, state string) string {
	switch {
	case city != "" && state != "":
		return city + ", " + state
	default:
		return cmp.Or(city, state)
	}
}

func (s *Service) Standing(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*demandforecast.Standing, error) {
	quality, err := s.repo.Quality(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	standing := &demandforecast.Standing{Quality: quality}
	if !quality.Stale(s.clock()) {
		return standing, nil
	}

	if err = s.fits.StartFitting(ctx, tenantInfo); err != nil {
		s.l.Warn("the freight forecast will be fitted tonight instead of now",
			zap.String("organizationId", tenantInfo.OrgID.String()), zap.Error(err))
		return standing, nil
	}
	standing.Refitting = true

	return standing, nil
}

func (s *Service) Forecaster(
	ctx context.Context,
	req *demandforecastports.ForecasterRequest,
) (*demandforecast.Forecaster, error) {
	index, err := s.index(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	params := &demandforecast.ForecasterParams{
		Now:           req.Now,
		Index:         index,
		Supply:        req.Supply,
		Solving:       req.Solving,
		MarginPercent: req.MarginPercent,
		CapCents:      req.CapCents,
	}
	if !index.Ready() || len(index.Cells()) == 0 {
		return demandforecast.NewForecaster(params), nil
	}

	booked, err := s.repo.ListBooked(ctx, &demandforecastports.BookedRequest{
		TenantInfo: req.TenantInfo,
		From:       req.Now - bookedLookbackSecs,
		Until:      req.Now + max(req.Horizon, bookedHorizonSecs) + windowSeconds,
	})
	if err != nil {
		return nil, err
	}
	params.Booked = BookedOf(booked)

	return demandforecast.NewForecaster(params), nil
}

func (s *Service) index(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*demandforecast.Index, error) {
	quality, err := s.repo.Quality(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	key := modelKey{orgID: tenantInfo.OrgID, buID: tenantInfo.BuID}
	if cached, ok := s.models.GetIfPresent(key); ok {
		if fitted := cached.Quality(); fitted.FittedAt == quality.FittedAt &&
			fitted.Version == quality.Version {
			return cached, nil
		}
	}

	model, err := s.repo.Model(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	names, err := s.repo.CustomerNames(ctx, &demandforecastports.CustomerNamesRequest{
		TenantInfo:  tenantInfo,
		CustomerIDs: customersOf(model.Rates),
	})
	if err != nil {
		return nil, err
	}
	index := demandforecast.NewIndex(&demandforecast.IndexParams{
		Quality:       model.Quality,
		Rates:         model.Rates,
		Areas:         model.Areas,
		Curves:        model.Curves,
		CustomerNames: names,
	})
	s.models.Set(key, index)

	return index, nil
}

func BookedOf(slots []demandforecastports.BookedSlot) []demandforecast.Booked {
	out := make([]demandforecast.Booked, 0, len(slots))
	for _, slot := range slots {
		cell, ok := dispatchmarket.CellOf(dispatchmarket.Point{
			Latitude:  slot.Latitude,
			Longitude: slot.Longitude,
		})
		if !ok {
			continue
		}
		out = append(out, demandforecast.Booked{
			Cell:      cell,
			SlotStart: slot.SlotStart,
			Total:     int(slot.Total),
			Open:      int(slot.Open),
		})
	}

	return out
}

func customersOf(rates []*demandforecast.Rate) []pulid.ID {
	seen := map[pulid.ID]struct{}{}
	out := []pulid.ID{}
	for _, rate := range rates {
		if rate.CustomerID.IsNil() {
			continue
		}
		if _, dup := seen[rate.CustomerID]; dup {
			continue
		}
		seen[rate.CustomerID] = struct{}{}
		out = append(out, rate.CustomerID)
	}
	slices.Sort(out)

	return out
}

func (s *Service) Location(ctx context.Context, tenantInfo pagination.TenantInfo) *time.Location {
	org, err := s.organizations.GetByID(ctx, tenantports.GetOrganizationByIDRequest{
		TenantInfo: tenantInfo,
	})
	if err != nil {
		s.l.Warn("freight outlook times are in UTC: the organization could not be read",
			zap.String("organizationId", tenantInfo.OrgID.String()), zap.Error(err))
		return time.UTC
	}
	if org == nil || org.Timezone == "" {
		return time.UTC
	}
	location, err := time.LoadLocation(org.Timezone)
	if err != nil {
		s.l.Warn("freight outlook times are in UTC: the organization's time zone is unknown",
			zap.String("timezone", org.Timezone), zap.Error(err))
		return time.UTC
	}

	return location
}

func (s *Service) Economics(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (margin, capCents int64, err error) {
	control, err := s.controls.GetOrCreate(ctx, tenantInfo.OrgID, tenantInfo.BuID)
	if err != nil {
		return 0, 0, fmt.Errorf("read dispatch economics: %w", err)
	}
	economics := control.ResolvedEconomics()

	return economics.CoverageMarginPercent, economics.AnticipationMaxCents, nil
}
