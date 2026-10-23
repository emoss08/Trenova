package tariff

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/app/orgzone"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/places/domain/rategeo"
	"github.com/emoss08/trenova/internal/modules/rating/app/tariff/record"
	"github.com/emoss08/trenova/internal/modules/rating/ports/fuelsurchargeports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratequoteports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratezoneports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/tariffports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
)

var _ tariffports.RecordBuilder = (*RecordSource)(nil)

type RecordSourceParams struct {
	fx.In

	Organizations tenantports.OrganizationRepository
	Zones         ratezoneports.RateZoneRepository
	Fuel          fuelsurchargeports.FuelSurchargeResolver
	Settings      *Settings
	Shipments     shipmentports.ShipmentRepository
	Hydrator      ratequoteports.ShipmentRatingHydrator
}

type RecordSource struct {
	organizations tenantports.OrganizationRepository
	zones         ratezoneports.RateZoneRepository
	fuel          fuelsurchargeports.FuelSurchargeResolver
	settings      *Settings
	shipments     shipmentports.ShipmentRepository
	hydrator      ratequoteports.ShipmentRatingHydrator
}

//nolint:gocritic // fx param structs are passed by value
func NewRecordSource(p RecordSourceParams) *RecordSource {
	return &RecordSource{
		organizations: p.Organizations,
		zones:         p.Zones,
		fuel:          p.Fuel,
		settings:      p.Settings,
		shipments:     p.Shipments,
		hydrator:      p.Hydrator,
	}
}

type RecordRequest struct {
	TenantInfo       pagination.TenantInfo
	Shipment         *shipment.Shipment
	OriginZones      []string
	DestinationZones []string
	ZonesResolved    bool
	Mode             string
	SellTotal        decimal.NullDecimal
	Currency         string
	AsOf             int64
	CustomerID       pulid.ID
	FuelOverride     *fuelsurchargeports.FuelProgramOverride
}

func (s *RecordSource) Build(ctx context.Context, req *RecordRequest) (rules.RawJSON, error) {
	if req.Shipment == nil {
		return nil, record.ErrNoShipment
	}
	zone, err := orgzone.Location(ctx, s.organizations, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	settings, err := s.settings.For(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	origin := record.PlaceOf(req.Shipment.ShipperStop())
	destination := record.PlaceOf(req.Shipment.DeliveryStop())
	origin.Zones, destination.Zones = req.OriginZones, req.DestinationZones
	if !req.ZonesResolved {
		lane, laneErr := s.LaneZones(ctx, req.TenantInfo, req.Shipment)
		if laneErr != nil {
			return nil, laneErr
		}
		origin.Zones = ratezoneports.ZoneCodes(lane.Origin)
		destination.Zones = ratezoneports.ZoneCodes(lane.Destination)
	}

	at := time.Unix(req.AsOf, 0).In(zone)
	customerID := req.CustomerID
	if customerID.IsNil() {
		customerID = req.Shipment.CustomerID
	}
	fuel, err := s.fuel.FuelPriceAsOf(ctx, &fuelsurchargeports.FuelPriceAsOfRequest{
		TenantInfo: req.TenantInfo,
		CustomerID: customerID,
		Override:   req.FuelOverride,
		Date:       at.Format(time.DateOnly),
	})
	if err != nil {
		return nil, err
	}

	input := &record.Input{
		Shipment:     req.Shipment,
		Origin:       origin,
		Destination:  destination,
		DistanceUnit: settings.DistanceUnit,
		Rating: record.Rating{
			At:        at,
			Mode:      req.Mode,
			SellTotal: req.SellTotal,
			Currency:  req.Currency,
		},
	}
	if fuel != nil {
		input.Fuel = record.Fuel{
			Price:     decimal.NewNullDecimal(fuel.Price),
			PriceDate: fuel.PriceDate,
			Index:     fuel.IndexCode,
		}
	}
	return record.Build(input)
}

func (s *RecordSource) LaneZones(
	ctx context.Context,
	tenant pagination.TenantInfo,
	entity *shipment.Shipment,
) (*ratezoneports.LaneZones, error) {
	origin := placeOf(entity.ShipperStop())
	destination := placeOf(entity.DeliveryStop())
	lane, err := rategeo.MatchLane(ctx, &origin, &destination)
	if err != nil {
		return nil, err
	}
	return s.zones.ResolveLaneZones(ctx, &ratezoneports.ResolveLaneZonesRequest{
		TenantInfo:      tenant,
		OriginKeys:      lane.OriginKeys,
		DestinationKeys: lane.DestinationKeys,
	})
}

func placeOf(stop *shipment.Stop) rategeo.Place {
	if stop == nil {
		return rategeo.Place{}
	}
	place := rategeo.Place{LocationID: stop.LocationID}
	if stop.Location == nil {
		return place
	}
	place.PostalCode = stop.Location.PostalCode
	place.City = stop.Location.City
	place.StateID = stop.Location.StateID
	if stop.Location.State != nil {
		place.CountryISO3 = stop.Location.State.CountryIso3
	}
	return place
}

func (s *RecordSource) ShipmentRecord(
	ctx context.Context,
	req *tariffports.ShipmentRecordRequest,
) (rules.RawJSON, error) {
	entity, err := s.shipments.GetByID(ctx, &shipmentports.GetShipmentByIDRequest{
		ID:         req.ShipmentID,
		TenantInfo: req.TenantInfo,
		ShipmentOptions: shipmentports.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		return nil, err
	}
	if err = s.hydrator.HydrateShipment(ctx, &ratequoteports.HydrateShipmentRequest{
		Shipment:   entity,
		TenantInfo: req.TenantInfo,
	}); err != nil {
		return nil, err
	}
	asOf := req.AsOf
	if asOf == 0 {
		asOf = entity.RatingAsOf(timeutils.NowUnix())
	}
	return s.Build(ctx, &RecordRequest{
		TenantInfo: req.TenantInfo,
		Shipment:   entity,
		Mode:       req.Mode,
		Currency:   BillingCurrency(entity),
		AsOf:       asOf,
	})
}

func BillingCurrency(entity *shipment.Shipment) string {
	if entity != nil && entity.Customer != nil && entity.Customer.BillingProfile != nil &&
		entity.Customer.BillingProfile.BillingCurrency != "" {
		return entity.Customer.BillingProfile.BillingCurrency
	}
	return money.DefaultCurrencyCode
}
