package shipmentwatchlistservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentboardcache"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentquickfilterservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
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
		filter *shipmentports.DispatchBoardFilter,
	) ([]*shipmentports.BoardMove, error)
}

type Params struct {
	fx.In

	Board        shipmentports.ShipmentBoardRepository
	Watchlist    shipmentports.ShipmentWatchlistRepository
	QuickFilters shipmentports.ShipmentQuickFilterBasisResolver
	Controls     tenantports.ShipmentControlRepository
	Accessorials accessorialchargeports.AccessorialChargeRepository
	Console      shipmentports.DispatchConsoleRepository
	Detention    *detentionservice.Service
	Cache        shipmentports.ShipmentBoardCache
	Logger       *zap.Logger
}

type Dependencies struct {
	Board        shipmentports.ShipmentBoardRepository
	Watchlist    shipmentports.ShipmentWatchlistRepository
	QuickFilters shipmentports.ShipmentQuickFilterBasisResolver
	Controls     tenantports.ShipmentControlRepository
	Accessorials accessorialchargeports.AccessorialChargeRepository
	Moves        BoardMoveLister
	Detention    DetentionDesk
	Cache        shipmentports.ShipmentBoardCache
	Logger       *zap.Logger
}

type Service struct {
	board        shipmentports.ShipmentBoardRepository
	watchlist    shipmentports.ShipmentWatchlistRepository
	quickFilters shipmentports.ShipmentQuickFilterBasisResolver
	controls     tenantports.ShipmentControlRepository
	accessorials accessorialchargeports.AccessorialChargeRepository
	moves        BoardMoveLister
	detention    DetentionDesk
	cache        shipmentports.ShipmentBoardCache
	l            *zap.Logger
}

var _ shipmentports.ShipmentWatchlistReader = (*Service)(nil)

//nolint:gocritic // dependency injection
func New(p Params) shipmentports.ShipmentWatchlistReader {
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
) (*shipmentports.ShipmentWatchlist, error) {
	return shipmentboardcache.GetOrCompute(
		ctx,
		s.cache,
		s.l,
		&shipmentboardcache.Request{
			Section:    shipmentports.ShipmentBoardSectionWatchlist,
			TenantInfo: tenantInfo,
			Timezone:   timezone,
		},
		func(ctx context.Context) (*shipmentports.ShipmentWatchlist, error) {
			return s.compute(ctx, tenantInfo, timezone)
		},
	)
}

type watchlistInputs struct {
	basis     *shipmentports.ShipmentQuickFilterBasis
	control   *tenant.ShipmentControl
	windows   []shipment.PickupWindowBounds
	totals    []shipmentports.ShipmentQuickFilterTotal
	rows      []*shipmentports.ShipmentDeliveryRow
	next      *shipmentports.ShipmentPickupRow
	customers []*shipmentports.ShipmentBillingCustomerRow
	detention shipmentports.ShipmentDetentionWatch
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
) (*shipmentports.ShipmentWatchlist, error) {
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
	req := &shipmentports.ShipmentWatchlistRequest{TenantInfo: tenantInfo, Basis: basis}

	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		totals, tErr := s.board.QuickFilterTotals(
			gctx,
			&shipmentports.CountShipmentQuickFiltersRequest{
				Scope: &shipmentports.ShipmentBoardScope{
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
			&shipmentports.ListReadyToBillCustomersRequest{
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

	return &shipmentports.ShipmentWatchlist{
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

func buildUncoveredWatch(in *watchlistInputs) shipmentports.ShipmentUncoveredWatch {
	watch := shipmentports.ShipmentUncoveredWatch{
		Count:   in.totals[totalUncovered].Count,
		Revenue: in.totals[totalUncovered].Revenue,
		Windows: make([]shipmentports.UncoveredWindowSummary, 0, len(in.windows)),
	}
	for i, window := range in.windows {
		total := in.totals[totalWindowsOffset+i]
		watch.Windows = append(watch.Windows, shipmentports.UncoveredWindowSummary{
			Window:       window.Window,
			StartMinutes: window.StartMinutes,
			EndMinutes:   window.EndMinutes,
			Count:        total.Count,
			Revenue:      total.Revenue,
		})
	}
	if in.next != nil {
		watch.Next = &shipmentports.NextUncoveredPickup{
			ShipmentID:      in.next.ShipmentID,
			PickupAt:        in.next.PickupAt,
			OriginCity:      in.next.OriginCity,
			DestinationCity: in.next.DestinationCity,
		}
	}
	return watch
}

func buildBillingWatch(in *watchlistInputs) shipmentports.ShipmentBillingWatch {
	watch := shipmentports.ShipmentBillingWatch{
		Count:     in.totals[totalReadyToBill].Count,
		Total:     in.totals[totalReadyToBill].Revenue,
		Customers: make([]shipmentports.ReadyToBillCustomer, 0, len(in.customers)),
	}
	for _, row := range in.customers {
		watch.Customers = append(watch.Customers, shipmentports.ReadyToBillCustomer{
			CustomerID: row.CustomerID,
			Name:       row.Name,
			Count:      row.Count,
			Total:      row.Total,
		})
		watch.MoreCustomers = max(row.TotalCustomers-len(in.customers), 0)
	}
	return watch
}
