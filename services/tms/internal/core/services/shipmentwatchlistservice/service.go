package shipmentwatchlistservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/detentionservice"
	"github.com/emoss08/trenova/internal/core/services/shipmentboardcache"
	"github.com/emoss08/trenova/internal/core/services/shipmentquickfilterservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const billingCustomerLimit = 4

type DetentionDesk interface {
	ListDesk(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) ([]*detentionservice.DeskEntry, error)
}

type BoardMoveLister interface {
	ListBoardMoves(
		ctx context.Context,
		filter *repositories.DispatchBoardFilter,
	) ([]*repositories.BoardMove, error)
}

type Params struct {
	fx.In

	Board        repositories.ShipmentBoardRepository
	Watchlist    repositories.ShipmentWatchlistRepository
	QuickFilters services.ShipmentQuickFilterBasisResolver
	Controls     repositories.ShipmentControlRepository
	Accessorials repositories.AccessorialChargeRepository
	Console      repositories.DispatchConsoleRepository
	Detention    *detentionservice.Service
	Cache        repositories.ShipmentBoardCache
	Logger       *zap.Logger
}

type Dependencies struct {
	Board        repositories.ShipmentBoardRepository
	Watchlist    repositories.ShipmentWatchlistRepository
	QuickFilters services.ShipmentQuickFilterBasisResolver
	Controls     repositories.ShipmentControlRepository
	Accessorials repositories.AccessorialChargeRepository
	Moves        BoardMoveLister
	Detention    DetentionDesk
	Cache        repositories.ShipmentBoardCache
	Logger       *zap.Logger
}

type Service struct {
	board        repositories.ShipmentBoardRepository
	watchlist    repositories.ShipmentWatchlistRepository
	quickFilters services.ShipmentQuickFilterBasisResolver
	controls     repositories.ShipmentControlRepository
	accessorials repositories.AccessorialChargeRepository
	moves        BoardMoveLister
	detention    DetentionDesk
	cache        repositories.ShipmentBoardCache
	l            *zap.Logger
}

var _ services.ShipmentWatchlistReader = (*Service)(nil)

//nolint:gocritic // dependency injection
func New(p Params) services.ShipmentWatchlistReader {
	return NewWithDependencies(&Dependencies{
		Board:        p.Board,
		Watchlist:    p.Watchlist,
		QuickFilters: p.QuickFilters,
		Controls:     p.Controls,
		Accessorials: p.Accessorials,
		Moves:        p.Console,
		Detention:    p.Detention,
		Cache:        p.Cache,
		Logger:       p.Logger,
	})
}

func NewWithDependencies(d *Dependencies) *Service {
	return &Service{
		board:        d.Board,
		watchlist:    d.Watchlist,
		quickFilters: d.QuickFilters,
		controls:     d.Controls,
		accessorials: d.Accessorials,
		moves:        d.Moves,
		detention:    d.Detention,
		cache:        d.Cache,
		l:            d.Logger.Named("service.shipment-watchlist"),
	}
}

func (s *Service) Watchlist(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	timezone string,
) (*services.ShipmentWatchlist, error) {
	return shipmentboardcache.GetOrCompute(
		ctx,
		s.cache,
		s.l,
		&shipmentboardcache.Request{
			Section:    repositories.ShipmentBoardSectionWatchlist,
			TenantInfo: tenantInfo,
			Timezone:   timezone,
		},
		func(ctx context.Context) (*services.ShipmentWatchlist, error) {
			return s.compute(ctx, tenantInfo, timezone)
		},
	)
}

type watchlistInputs struct {
	basis     *repositories.ShipmentQuickFilterBasis
	control   *tenant.ShipmentControl
	windows   []shipment.PickupWindowBounds
	totals    []repositories.ShipmentQuickFilterTotal
	rows      []*repositories.ShipmentDeliveryRow
	next      *repositories.ShipmentPickupRow
	customers []*repositories.ShipmentBillingCustomerRow
	detention services.ShipmentDetentionWatch
}

const (
	totalUncovered = iota
	totalReadyToBill
	totalWindowsOffset
)

func (s *Service) compute(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	timezone string,
) (*services.ShipmentWatchlist, error) {
	basis, err := s.quickFilters.Resolve(ctx, &services.ResolveShipmentQuickFilterBasisRequest{
		TenantInfo: tenantInfo,
		Timezone:   timezone,
	})
	if err != nil {
		return nil, err
	}

	control, err := s.controls.Get(
		ctx,
		repositories.GetShipmentControlRequest{TenantInfo: tenantInfo},
	)
	if err != nil {
		return nil, err
	}
	basis.Detention = shipmentquickfilterservice.DetentionBasisOf(
		control.UseDetentionPolicyEngine,
		control.DetentionThreshold,
	)

	in := &watchlistInputs{
		basis:   basis,
		control: control,
		windows: shipment.PickupWindowsAt(basis.Now, basis.Location),
	}
	req := &repositories.ShipmentWatchlistRequest{TenantInfo: tenantInfo, Basis: basis}

	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		totals, tErr := s.board.QuickFilterTotals(
			gctx,
			&repositories.CountShipmentQuickFiltersRequest{
				Scope: &repositories.ShipmentBoardScope{
					Filter:  &pagination.QueryOptions{TenantInfo: tenantInfo},
					Options: repositories.ShipmentOptions{QuickFilterBasis: basis},
				},
				Filters: totalSpecs(in.windows),
			},
		)
		in.totals = totals
		return tErr
	})
	g.Go(func() error {
		rows, rErr := s.watchlist.ListDeliveriesToday(gctx, req)
		in.rows = rows
		return rErr
	})
	g.Go(func() error {
		next, nErr := s.watchlist.NextUncoveredPickup(gctx, req)
		in.next = next
		return nErr
	})
	g.Go(func() error {
		customers, cErr := s.watchlist.ListReadyToBillCustomers(
			gctx,
			&repositories.ListReadyToBillCustomersRequest{
				TenantInfo: tenantInfo,
				Limit:      billingCustomerLimit,
			},
		)
		in.customers = customers
		return cErr
	})
	g.Go(func() error {
		watch, dErr := s.detentionWatch(gctx, req, control)
		in.detention = watch
		return dErr
	})
	if err = g.Wait(); err != nil {
		return nil, err
	}

	return &services.ShipmentWatchlist{
		Deliveries: BuildDeliveryWatch(in.rows, basis.Now.Unix(), basis.Location),
		Uncovered:  buildUncoveredWatch(in),
		Detention:  in.detention,
		Billing:    buildBillingWatch(in),
	}, nil
}

func totalSpecs(windows []shipment.PickupWindowBounds) []shipment.QuickFilterSpec {
	specs := make([]shipment.QuickFilterSpec, totalWindowsOffset, totalWindowsOffset+len(windows))
	specs[totalUncovered] = shipment.Quick(shipment.QuickFilterUncovered)
	specs[totalReadyToBill] = shipment.Quick(shipment.QuickFilterReadyToBill)
	for _, window := range windows {
		specs = append(specs, window.Spec())
	}
	return specs
}

func buildUncoveredWatch(in *watchlistInputs) services.ShipmentUncoveredWatch {
	watch := services.ShipmentUncoveredWatch{
		Count:   in.totals[totalUncovered].Count,
		Revenue: in.totals[totalUncovered].Revenue,
		Windows: make([]services.UncoveredWindowSummary, 0, len(in.windows)),
	}
	for i, window := range in.windows {
		total := in.totals[totalWindowsOffset+i]
		watch.Windows = append(watch.Windows, services.UncoveredWindowSummary{
			Window:       window.Window,
			StartMinutes: window.StartMinutes,
			EndMinutes:   window.EndMinutes,
			Count:        total.Count,
			Revenue:      total.Revenue,
		})
	}
	if in.next != nil {
		watch.Next = &services.NextUncoveredPickup{
			ShipmentID:      in.next.ShipmentID,
			PickupAt:        in.next.PickupAt,
			OriginCity:      in.next.OriginCity,
			DestinationCity: in.next.DestinationCity,
		}
	}
	return watch
}

func buildBillingWatch(in *watchlistInputs) services.ShipmentBillingWatch {
	watch := services.ShipmentBillingWatch{
		Count:     in.totals[totalReadyToBill].Count,
		Total:     in.totals[totalReadyToBill].Revenue,
		Customers: make([]services.ReadyToBillCustomer, 0, len(in.customers)),
	}
	for _, row := range in.customers {
		watch.Customers = append(watch.Customers, services.ReadyToBillCustomer{
			CustomerID: row.CustomerID,
			Name:       row.Name,
			Count:      row.Count,
			Total:      row.Total,
		})
		watch.MoreCustomers = max(row.TotalCustomers-len(in.customers), 0)
	}
	return watch
}
