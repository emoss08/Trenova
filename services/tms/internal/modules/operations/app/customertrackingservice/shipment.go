package customertrackingservice

import (
	"context"
	"slices"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

func (s *Service) loadShipment(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
) (*shipment.Shipment, error) {
	return s.shipmentRepo.GetByID(ctx, &shipmentports.GetShipmentByIDRequest{
		ID:         shipmentID,
		TenantInfo: tenantInfo,
		ShipmentOptions: shipmentports.ShipmentOptions{
			IncludeCustomer: true,
			IncludeRoute:    true,
		},
	})
}

func (s *Service) loadCustomer(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	customerID pulid.ID,
) (*customer.Customer, error) {
	return s.customerRepo.GetByID(ctx, customerports.GetCustomerByIDRequest{
		ID:         customerID,
		TenantInfo: tenantInfo,
		CustomerFilterOptions: customerports.CustomerFilterOptions{
			IncludeEmailProfile: true,
		},
	})
}

func (s *Service) companyName(ctx context.Context, tenantInfo pagination.TenantInfo) string {
	if s.orgRepo == nil {
		return ""
	}
	org, err := s.orgRepo.GetByID(
		ctx,
		tenantports.GetOrganizationByIDRequest{TenantInfo: tenantInfo},
	)
	if err != nil || org == nil {
		return ""
	}
	return org.Name
}

// routeStop is a stop in travel order: moves by sequence, then stops by
// sequence, canceled moves and stops left out.
type routeStop struct {
	stop *shipment.Stop
}

func route(sh *shipment.Shipment) []routeStop {
	moves := make([]*shipment.ShipmentMove, 0, len(sh.Moves))
	for _, move := range sh.Moves {
		if move != nil && move.Status != shipmentvocab.MoveStatusCanceled {
			moves = append(moves, move)
		}
	}
	slices.SortStableFunc(moves, func(a, b *shipment.ShipmentMove) int {
		switch {
		case a.Sequence < b.Sequence:
			return -1
		case a.Sequence > b.Sequence:
			return 1
		default:
			return 0
		}
	})
	stops := make([]routeStop, 0, len(moves)*2)
	for _, move := range moves {
		for _, stop := range move.ActiveStopsBySequence() {
			stops = append(stops, routeStop{stop: stop})
		}
	}
	return stops
}

func place(stop *shipment.Stop) (city, state string) {
	if stop == nil || stop.Location == nil {
		return "", ""
	}
	city = stop.Location.City
	if stop.Location.State != nil {
		state = stop.Location.State.Abbreviation
	}
	return city, state
}

func placeSummary(stop *shipment.Stop) string {
	city, state := place(stop)
	switch {
	case city != "" && state != "":
		return city + ", " + state
	default:
		return city + state
	}
}

func originAndDestination(stops []routeStop) (origin, destination *shipment.Stop) {
	for _, rs := range stops {
		if rs.stop.IsOriginStop() && origin == nil {
			origin = rs.stop
		}
		if rs.stop.IsDestinationStop() {
			destination = rs.stop
		}
	}
	return origin, destination
}

func isClosed(sh *shipment.Shipment) bool {
	switch sh.Status {
	case shipmentvocab.StatusCompleted,
		shipmentvocab.StatusReadyToInvoice,
		shipmentvocab.StatusInvoiced,
		shipmentvocab.StatusCanceled:
		return true
	case shipmentvocab.StatusNew,
		shipmentvocab.StatusPartiallyAssigned,
		shipmentvocab.StatusAssigned,
		shipmentvocab.StatusInTransit,
		shipmentvocab.StatusDelayed,
		shipmentvocab.StatusPartiallyCompleted:
		return false
	default:
		return false
	}
}

// stopZone is the zone a stop's times are read in: the location's own, or
// UTC when the location has none.
func stopZone(stop *shipment.Stop) *time.Location {
	if stop == nil || stop.Location == nil {
		return time.UTC
	}
	return timeutils.LoadLocation(stop.Location.Timezone)
}

func stopWindowLabel(stop *shipment.Stop) string {
	if stop == nil {
		return ""
	}
	return timeutils.WindowLabelIn(
		stop.ScheduledWindowStart,
		stop.ScheduledWindowEnd,
		stopZone(stop),
	)
}

func instantLabel(at int64, stop *shipment.Stop) string {
	return timeutils.WindowLabelIn(at, nil, stopZone(stop))
}
