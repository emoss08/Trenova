package freightoutlookservice

import (
	"cmp"
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/modules/operations/domain/demandforecast"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatcheconomics"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchmarket"
	"github.com/emoss08/trenova/internal/modules/operations/domain/freightmove"
	"github.com/emoss08/trenova/internal/modules/operations/ports/demandforecastports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchcontrolports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/driverpositioningports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/freightoutlookports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const horizonSeconds = int64(demandforecast.OutlookSlots) * dispatchmarket.SlotSeconds

type Params struct {
	fx.In

	Logger       *zap.Logger
	Forecasts    demandforecastports.Service
	Repo         demandforecastports.Repository
	Controls     dispatchcontrolports.DispatchControlRepository
	Positionings driverpositioningports.LiveReader
	Clock        timeutils.Clock
}

type Service struct {
	l            *zap.Logger
	forecasts    demandforecastports.Service
	repo         demandforecastports.Repository
	controls     dispatchcontrolports.DispatchControlRepository
	positionings driverpositioningports.LiveReader
	clock        timeutils.Clock
}

var _ freightoutlookports.Service = (*Service)(nil)

func New(p Params) *Service { //nolint:gocritic // fx requires an fx.In parameter struct by value
	return &Service{
		l:            p.Logger.Named("service.freight-outlook"),
		forecasts:    p.Forecasts,
		repo:         p.Repo,
		controls:     p.Controls,
		positionings: p.Positionings,
		clock:        p.Clock,
	}
}

type snapshot struct {
	now        int64
	quality    *demandforecast.Quality
	refitting  bool
	forecaster *demandforecast.Forecaster
	starts     []int64
	balances   []demandforecast.AreaBalance
	inputs     *freightmove.Inputs
	names      map[pulid.ID]string
}

func (s *Service) Outlook(
	ctx context.Context,
	req *freightoutlookports.Request,
) (*freightoutlookports.Outlook, error) {
	snap, err := s.snapshot(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	out := &freightoutlookports.Outlook{
		Quality:     snap.quality,
		Refitting:   snap.refitting,
		GeneratedAt: snap.now,
		Starts: demandforecast.OutlookStarts(
			dispatchmarket.GridNamed(snap.quality.Timezone),
			snap.now,
		),
		Network:      []demandforecast.NetworkSlot{},
		Areas:        []*freightoutlookports.Area{},
		Steps:        []freightmove.Step{},
		Pace:         []freightmove.Pace{},
		Days:         []*demandforecast.Day{},
		Positionings: []*freightoutlookports.Positioning{},
	}
	if snap.forecaster == nil {
		return out, nil
	}

	out.Network = snap.forecaster.Network(snap.balances, snap.starts)
	out.Steps = freightmove.Plan(snap.inputs)
	out.Pace = freightmove.Paces(snap.balances)
	out.AreaCount = len(snap.balances)
	for _, balance := range relevant(snap.balances, out.Steps) {
		out.Areas = append(out.Areas, s.area(snap, balance))
	}

	if out.Days, err = s.repo.Days(ctx, req.TenantInfo); err != nil {
		return nil, err
	}
	if out.Positionings, err = s.live(ctx, req.TenantInfo, snap); err != nil {
		return nil, err
	}

	return out, nil
}

func (s *Service) Area(
	ctx context.Context,
	req *freightoutlookports.AreaRequest,
) (*freightoutlookports.AreaDetail, error) {
	snap, err := s.snapshot(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	if snap.forecaster == nil {
		return nil, errortypes.NewBusinessError(
			"The freight forecast is not ready, so there is no outlook for this area yet.",
		)
	}
	index := slices.IndexFunc(snap.balances, func(b demandforecast.AreaBalance) bool {
		return b.Cell == req.Cell
	})
	if index < 0 {
		return nil, errortypes.NewNotFoundError("That area is not in the freight forecast.")
	}

	balance := snap.balances[index]
	at := slices.IndexFunc(balance.Slots, func(slot demandforecast.SlotBalance) bool {
		return slot.Start == req.SlotStart
	})
	if req.SlotStart == 0 || at < 0 {
		at, _ = balance.Worst()
	}
	slotStart := balance.Slots[at].Start

	detail := &freightoutlookports.AreaDetail{
		Quality:     snap.quality,
		Refitting:   snap.refitting,
		GeneratedAt: snap.now,
		Area:        s.area(snap, balance),
		Bounds:      dispatchmarket.BoundsOf(req.Cell),
		SlotStart:   slotStart,
		SlotEnd:     balance.Slots[at].End,
		Reach:       freightmove.Reach(snap.inputs, req.Cell, slotStart),
		Steps:       []freightmove.Step{},
		Pace:        []freightmove.Pace{},
		Recent:      []int32{},
	}
	for _, step := range freightmove.Plan(snap.inputs) {
		if step.ToCell == req.Cell || step.FromCell == req.Cell {
			detail.Steps = append(detail.Steps, step)
		}
	}
	detail.Pace = freightmove.Paces([]demandforecast.AreaBalance{balance})
	if rate, ok := snap.forecaster.RateAt(req.Cell, slotStart); ok {
		detail.Recent = rate.Recent
		typical := demandforecast.GammaPoisson(rate.Alpha, rate.Beta)
		detail.RecentLow = typical.Quantile(demandforecast.IntervalLow)
		detail.RecentHigh = typical.Quantile(demandforecast.IntervalHigh)
	}

	live, err := s.live(ctx, req.TenantInfo, snap)
	if err != nil {
		return nil, err
	}
	detail.Positionings = []*freightoutlookports.Positioning{}
	for _, positioning := range live {
		if positioning.ToCell == req.Cell || positioning.FromCell == req.Cell {
			detail.Positionings = append(detail.Positionings, positioning)
		}
	}
	detail.DeliveryLocationIDs, err = s.repo.ListAreaLocations(
		ctx,
		&demandforecastports.AreaLocationsRequest{TenantInfo: req.TenantInfo, Cell: req.Cell},
	)
	if err != nil {
		return nil, err
	}

	return detail, nil
}

func (s *Service) snapshot(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*snapshot, error) {
	now := s.clock()
	standing, err := s.forecasts.Standing(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	quality := standing.Quality
	snap := &snapshot{
		now:       now,
		quality:   quality,
		refitting: standing.Refitting,
		names:     map[pulid.ID]string{},
	}
	if !quality.Ready() {
		return snap, nil
	}

	starts, err := s.repo.ListDriverStarts(ctx, &demandforecastports.DriverStartsRequest{
		TenantInfo: tenantInfo,
		Now:        now,
		Location:   dispatchmarket.GridNamed(quality.Timezone).Location(),
	})
	if err != nil {
		return nil, err
	}
	economics, err := s.economics(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	supply, drivers := supplyOf(starts, snap.names)
	forecaster, err := s.forecasts.Forecaster(ctx, &demandforecastports.ForecasterRequest{
		TenantInfo:    tenantInfo,
		Now:           now,
		Horizon:       horizonSeconds,
		Supply:        supply,
		MarginPercent: economics.CoverageMarginPercent,
		CapCents:      economics.AnticipationMaxCents,
	})
	if err != nil {
		return nil, err
	}
	if !forecaster.Ready() {
		return snap, nil
	}

	window := &demandforecastports.WindowRequest{
		TenantInfo: tenantInfo,
		From:       now,
		Until:      now + horizonSeconds,
	}
	lanes, err := s.repo.ListOpenLanes(ctx, window)
	if err != nil {
		return nil, err
	}
	departures, err := s.repo.ListDepartures(ctx, window)
	if err != nil {
		return nil, err
	}

	snap.forecaster = forecaster
	snap.starts = forecaster.OutlookStarts()
	for _, cell := range forecaster.Cells() {
		snap.balances = append(snap.balances, forecaster.Balance(cell, snap.starts))
	}
	snap.inputs = &freightmove.Inputs{
		Now:            now,
		Grid:           forecaster.Grid(),
		Areas:          snap.balances,
		Drivers:        drivers,
		Lanes:          lanesOf(lanes),
		Departures:     departuresOf(departures),
		EmptyMileCents: economics.EmptyMileCents,
		MarginPercent:  economics.CoverageMarginPercent,
		Valuer:         forecaster,
	}

	return snap, nil
}

func (s *Service) economics(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (dispatcheconomics.Economics, error) {
	control, err := s.controls.GetOrCreate(ctx, tenantInfo.OrgID, tenantInfo.BuID)
	if err != nil {
		return dispatcheconomics.Economics{}, err
	}

	return control.ResolvedEconomics(), nil
}

func (s *Service) live(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	snap *snapshot,
) ([]*freightoutlookports.Positioning, error) {
	rows, err := s.positionings.ListLive(ctx, &driverpositioningports.LiveRequest{
		TenantInfo: tenantInfo,
		Now:        snap.now,
	})
	if err != nil {
		return nil, err
	}
	out := make([]*freightoutlookports.Positioning, 0, len(rows))
	for _, row := range rows {
		out = append(out, &freightoutlookports.Positioning{
			Positioning: row,
			WorkerName:  snap.names[row.WorkerID],
		})
	}

	return out, nil
}

func (s *Service) area(
	snap *snapshot,
	balance demandforecast.AreaBalance,
) *freightoutlookports.Area {
	return &freightoutlookports.Area{
		Balance:    balance,
		Point:      dispatchmarket.CenterOf(balance.Cell),
		ValueCents: snap.forecaster.OneMoreDriver(&balance),
		Customers:  customersOf(balance),
	}
}

func relevant(
	balances []demandforecast.AreaBalance,
	steps []freightmove.Step,
) []demandforecast.AreaBalance {
	named := map[int64]bool{}
	for _, step := range steps {
		named[step.ToCell], named[step.FromCell] = true, true
	}
	type scored struct {
		balance demandforecast.AreaBalance
		named   bool
		gap     float64
		need    float64
	}
	rows := make([]scored, 0, len(balances))
	for _, balance := range balances {
		_, short := balance.Worst()
		_, spare := balance.MostSpare()
		need := 0.0
		for _, slot := range balance.Slots {
			need += slot.Need
		}
		rows = append(rows, scored{
			balance: balance,
			named:   named[balance.Cell],
			gap:     short + spare,
			need:    need,
		})
	}
	slices.SortStableFunc(rows, func(a, b scored) int {
		if a.named != b.named {
			if a.named {
				return -1
			}
			return 1
		}
		if c := cmp.Compare(b.gap, a.gap); c != 0 {
			return c
		}
		if c := cmp.Compare(b.need, a.need); c != 0 {
			return c
		}
		return cmp.Compare(a.balance.Cell, b.balance.Cell)
	})

	out := make([]demandforecast.AreaBalance, 0, min(len(rows), freightoutlookports.MaxAreas))
	for _, row := range rows[:min(len(rows), freightoutlookports.MaxAreas)] {
		out = append(out, row.balance)
	}

	return out
}

func customersOf(balance demandforecast.AreaBalance) []freightoutlookports.AreaCustomer {
	usual := map[pulid.ID]*freightoutlookports.AreaCustomer{}
	total := 0.0
	for _, slot := range balance.Slots {
		for _, customer := range slot.Customers {
			row := usual[customer.CustomerID]
			if row == nil {
				row = &freightoutlookports.AreaCustomer{
					CustomerID: customer.CustomerID,
					Name:       customer.Name,
				}
				usual[customer.CustomerID] = row
			}
			row.Share += customer.Usual
			total += customer.Usual
		}
	}
	out := make([]freightoutlookports.AreaCustomer, 0, len(usual))
	for _, row := range usual {
		if total > 0 {
			row.Share /= total
		}
		if row.Name == "" {
			continue
		}
		out = append(out, *row)
	}
	slices.SortFunc(out, func(a, b freightoutlookports.AreaCustomer) int {
		if c := cmp.Compare(b.Share, a.Share); c != 0 {
			return c
		}
		return cmp.Compare(a.CustomerID, b.CustomerID)
	})

	return out[:min(len(out), freightoutlookports.MaxAreaCustomers)]
}

func supplyOf(
	starts []demandforecastports.DriverStart,
	names map[pulid.ID]string,
) ([]demandforecast.Driver, []freightmove.Driver) {
	supply := make([]demandforecast.Driver, 0, len(starts))
	drivers := make([]freightmove.Driver, 0, len(starts))
	for _, start := range starts {
		names[start.WorkerID] = start.Name
		point := dispatchmarket.Point{Latitude: start.Latitude, Longitude: start.Longitude}
		cell, ok := dispatchmarket.CellOf(point)
		if !ok {
			continue
		}
		supply = append(supply, demandforecast.Driver{
			WorkerID: start.WorkerID,
			Cell:     cell,
			FreeAt:   start.FreeAt,
		})
		drivers = append(drivers, freightmove.Driver{
			WorkerID: start.WorkerID,
			Name:     start.Name,
			Point:    point,
			Cell:     cell,
			FreeAt:   start.FreeAt,
		})
	}

	return supply, drivers
}

func lanesOf(rows []demandforecastports.OpenLane) []freightmove.Lane {
	out := make([]freightmove.Lane, 0, len(rows))
	for _, row := range rows {
		out = append(out, freightmove.Lane{
			MoveID:    row.MoveID,
			ProNumber: row.ProNumber,
			Origin: dispatchmarket.Point{
				Latitude:  row.OriginLatitude,
				Longitude: row.OriginLongitude,
			},
			Destination: dispatchmarket.Point{
				Latitude:  row.DestinationLatitude,
				Longitude: row.DestinationLongitude,
			},
			PickupStart: row.PickupStart,
			PickupEnd:   row.PickupEnd,
			DeliveryEnd: row.DeliveryEnd,
		})
	}

	return out
}

func departuresOf(rows []demandforecastports.Departure) []freightmove.Departure {
	out := make([]freightmove.Departure, 0, len(rows))
	for _, row := range rows {
		out = append(out, freightmove.Departure{
			MoveID:    row.MoveID,
			ProNumber: row.ProNumber,
			WorkerID:  row.WorkerID,
			Origin: dispatchmarket.Point{
				Latitude:  row.OriginLatitude,
				Longitude: row.OriginLongitude,
			},
			PickupStart:      row.PickupStart,
			PickupEnd:        row.PickupEnd,
			DestinationLabel: row.DestinationLabel,
		})
	}

	return out
}
