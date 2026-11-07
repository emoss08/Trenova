package demandforecastservice

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/demandforecast"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchmarket"
	"github.com/emoss08/trenova/internal/modules/operations/ports/demandforecastports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchcontrolports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	historySpanWeeks   = demandforecast.HistoryWeeks + demandforecast.BacktestWeeks + 1
	bookedLookbackSecs = dispatchmarket.SlotSeconds
	bookedHorizonSecs  = int64(5 * 24 * 3600)
	windowSeconds      = demandforecast.WindowSlots * dispatchmarket.SlotSeconds
)

type Params struct {
	fx.In

	Logger        *zap.Logger
	Repo          demandforecastports.Repository
	Controls      dispatchcontrolports.DispatchControlRepository
	Organizations tenantports.OrganizationRepository
}

type Service struct {
	l             *zap.Logger
	repo          demandforecastports.Repository
	controls      dispatchcontrolports.DispatchControlRepository
	organizations tenantports.OrganizationRepository
	clock         func() int64
}

var _ demandforecastports.Service = (*Service)(nil)

func New(p Params) *Service {
	return &Service{
		l:             p.Logger.Named("service.demand-forecast"),
		repo:          p.Repo,
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
	rows, err := s.repo.History(ctx, &demandforecastports.HistoryRequest{
		TenantInfo: tenantInfo,
		Since:      now - historySpanWeeks*dispatchmarket.WeekSeconds,
		Until:      now,
	})
	if err != nil {
		return nil, err
	}

	history := HistoryOf(now, rows)
	fitted := demandforecast.Fit(history)
	quality := demandforecast.Backtest(history)

	if err = s.repo.Replace(ctx, &demandforecastports.ReplaceRequest{
		TenantInfo: tenantInfo,
		Fitted:     fitted,
		Quality:    quality,
		Days:       demandforecast.TrackRecord(history),
	}); err != nil {
		return nil, err
	}

	return quality, nil
}

func HistoryOf(now int64, rows *demandforecastports.HistoryRows) *demandforecast.History {
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

	history := &demandforecast.History{
		Now:    now,
		Counts: make([]demandforecast.LoadCount, 0, len(rows.Counts)),
		Leads:  rows.Leads,
		Areas:  make([]demandforecast.AreaSummary, 0, len(areas)),
	}
	for _, count := range rows.Counts {
		cell, ok := cells[count.LocationID]
		if !ok {
			continue
		}
		history.Counts = append(history.Counts, demandforecast.LoadCount{
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
		history.Areas = append(history.Areas, demandforecast.AreaSummary{
			Cell:               cell,
			Label:              stats.label,
			MedianRevenueCents: median,
			Loads:              stats.loads,
		})
	}
	slices.SortFunc(history.Areas, func(a, b demandforecast.AreaSummary) int {
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

func (s *Service) Quality(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*demandforecast.Quality, error) {
	return s.repo.Quality(ctx, tenantInfo)
}

func (s *Service) Forecaster(
	ctx context.Context,
	req *demandforecastports.ForecasterRequest,
) (*demandforecast.Forecaster, error) {
	var model *demandforecastports.Model
	var err error
	if req.AllCells {
		model, err = s.repo.Model(ctx, req.TenantInfo)
	} else {
		model, err = s.repo.ModelForCells(ctx, &demandforecastports.CellsRequest{
			TenantInfo: req.TenantInfo,
			Cells:      req.Cells,
		})
	}
	if err != nil {
		return nil, err
	}

	return s.forecaster(ctx, &forecasterInputs{
		tenantInfo: req.TenantInfo,
		now:        req.Now,
		model:      model,
		supply:     req.Supply,
		solving:    req.Solving,
		margin:     req.MarginPercent,
		capCents:   req.CapCents,
		horizon:    req.Horizon,
	})
}

type forecasterInputs struct {
	tenantInfo pagination.TenantInfo
	now        int64
	model      *demandforecastports.Model
	supply     []demandforecast.Driver
	solving    []demandforecast.Booked
	margin     int64
	capCents   int64
	horizon    int64
}

func (s *Service) forecaster(
	ctx context.Context,
	in *forecasterInputs,
) (*demandforecast.Forecaster, error) {
	params := &demandforecast.ForecasterParams{
		Now:           in.now,
		Quality:       in.model.Quality,
		Rates:         in.model.Rates,
		Areas:         in.model.Areas,
		Curves:        in.model.Curves,
		Supply:        in.supply,
		Solving:       in.solving,
		MarginPercent: in.margin,
		CapCents:      in.capCents,
	}
	if !in.model.Quality.Ready() || len(in.model.Areas) == 0 {
		return demandforecast.NewForecaster(params), nil
	}

	booked, err := s.repo.ListBooked(ctx, &demandforecastports.BookedRequest{
		TenantInfo: in.tenantInfo,
		From:       dispatchmarket.SlotStart(in.now) - bookedLookbackSecs,
		Until:      in.now + max(in.horizon, bookedHorizonSecs) + windowSeconds,
	})
	if err != nil {
		return nil, err
	}
	params.Booked = BookedOf(booked)

	names, err := s.repo.CustomerNames(ctx, &demandforecastports.CustomerNamesRequest{
		TenantInfo:  in.tenantInfo,
		CustomerIDs: customersOf(in.model.Rates),
	})
	if err != nil {
		return nil, err
	}
	params.CustomerNames = names

	return demandforecast.NewForecaster(params), nil
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
