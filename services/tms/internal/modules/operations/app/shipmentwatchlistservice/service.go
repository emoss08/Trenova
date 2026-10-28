package shipmentwatchlistservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentboardcache"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentquickfilterservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchboardports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentboardports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/accessorialchargeports"
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
		filter *dispatchboardports.DispatchBoardFilter,
	) ([]*dispatchboardports.BoardMove, error)
}

type Params struct {
	fx.In

	Board        shipmentboardports.ShipmentBoardRepository
	Watchlist    shipmentboardports.ShipmentWatchlistRepository
	QuickFilters shipmentports.ShipmentQuickFilterBasisResolver
	Controls     tenantports.ShipmentControlRepository
	Accessorials accessorialchargeports.AccessorialChargeRepository
	Console      dispatchboardports.DispatchConsoleRepository
	Detention    *detentionservice.Service
	Cache        shipmentboardports.ShipmentBoardCache
	Logger       *zap.Logger
}

type Dependencies struct {
	Board        shipmentboardports.ShipmentBoardRepository
	Watchlist    shipmentboardports.ShipmentWatchlistRepository
	QuickFilters shipmentports.ShipmentQuickFilterBasisResolver
	Controls     tenantports.ShipmentControlRepository
	Accessorials accessorialchargeports.AccessorialChargeRepository
	Moves        BoardMoveLister
	Detention    DetentionDesk
	Cache        shipmentboardports.ShipmentBoardCache
	Logger       *zap.Logger
}

type Service struct {
	board        shipmentboardports.ShipmentBoardRepository
	watchlist    shipmentboardports.ShipmentWatchlistRepository
	quickFilters shipmentports.ShipmentQuickFilterBasisResolver
	controls     tenantports.ShipmentControlRepository
	accessorials accessorialchargeports.AccessorialChargeRepository
	moves        BoardMoveLister
	detention    DetentionDesk
	cache        shipmentboardports.ShipmentBoardCache
	l            *zap.Logger
}

var _ shipmentboardports.ShipmentWatchlistReader = (*Service)(nil)

//nolint:gocritic // dependency injection
func New(p Params) shipmentboardports.ShipmentWatchlistReader {
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
) (*shipmentboardports.ShipmentWatchlist, error) {
	return shipmentboardcache.GetOrCompute(
		ctx,
		s.cache,
		s.l,
		&shipmentboardcache.Request{
			Section:    shipmentboardports.ShipmentBoardSectionWatchlist,
			TenantInfo: tenantInfo,
			Timezone:   timezone,
		},
		func(ctx context.Context) (*shipmentboardports.ShipmentWatchlist, error) {
			return s.compute(ctx, tenantInfo, timezone)
		},
	)
}

type watchlistInputs struct {
	basis     *shipmentports.ShipmentQuickFilterBasis
	control   *tenant.ShipmentControl
	windows   []shipment.PickupWindowBounds
	totals    []shipmentboardports.ShipmentQuickFilterTotal
	rows      []*shipmentboardports.ShipmentDeliveryRow
	next      *shipmentboardports.ShipmentPickupRow
	customers []*shipmentboardports.ShipmentBillingCustomerRow
	detention shipmentboardports.ShipmentDetentionWatch
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
) (*shipmentboardports.ShipmentWatchlist, error) {
	basis, err := s.quickFilters.Resolve(ctx, &shipmentports.ResolveShipmentQuickFilterBasisRequest{
		TenantInfo: tenantInfo,
		Timezone:   timezone,
	})
	if err != nil {
		return nil, err
	}

	control, err := s.controls.Get(
		ctx,
		tenantports.GetShipmentControlRequest{TenantInfo: tenantInfo},
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
	req := &shipmentboardports.ShipmentWatchlistRequest{TenantInfo: tenantInfo, Basis: basis}

	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		totals, tErr := s.board.QuickFilterTotals(
			gctx,
			&shipmentboardports.CountShipmentQuickFiltersRequest{
				Scope: &shipmentboardports.ShipmentBoardScope{
					Filter:  &pagination.QueryOptions{TenantInfo: tenantInfo},
					Options: shipmentports.ShipmentOptions{QuickFilterBasis: basis},
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
			&shipmentboardports.ListReadyToBillCustomersRequest{
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

	return &shipmentboardports.ShipmentWatchlist{
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

func buildUncoveredWatch(in *watchlistInputs) shipmentboardports.ShipmentUncoveredWatch {
	watch := shipmentboardports.ShipmentUncoveredWatch{
		Count:   in.totals[totalUncovered].Count,
		Revenue: in.totals[totalUncovered].Revenue,
		Windows: make([]shipmentboardports.UncoveredWindowSummary, 0, len(in.windows)),
	}
	for i, window := range in.windows {
		total := in.totals[totalWindowsOffset+i]
		watch.Windows = append(watch.Windows, shipmentboardports.UncoveredWindowSummary{
			Window:       window.Window,
			StartMinutes: window.StartMinutes,
			EndMinutes:   window.EndMinutes,
			Count:        total.Count,
			Revenue:      total.Revenue,
		})
	}
	if in.next != nil {
		watch.Next = &shipmentboardports.NextUncoveredPickup{
			ShipmentID:      in.next.ShipmentID,
			PickupAt:        in.next.PickupAt,
			OriginCity:      in.next.OriginCity,
			DestinationCity: in.next.DestinationCity,
		}
	}
	return watch
}

func buildBillingWatch(in *watchlistInputs) shipmentboardports.ShipmentBillingWatch {
	watch := shipmentboardports.ShipmentBillingWatch{
		Count:     in.totals[totalReadyToBill].Count,
		Total:     in.totals[totalReadyToBill].Revenue,
		Customers: make([]shipmentboardports.ReadyToBillCustomer, 0, len(in.customers)),
	}
	for _, row := range in.customers {
		watch.Customers = append(watch.Customers, shipmentboardports.ReadyToBillCustomer{
			CustomerID: row.CustomerID,
			Name:       row.Name,
			Count:      row.Count,
			Total:      row.Total,
		})
		watch.MoreCustomers = max(row.TotalCustomers-len(in.customers), 0)
	}
	return watch
}
