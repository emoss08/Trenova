package shipmentcapacityservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/ratequote"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/ratequoteservice"
	"github.com/emoss08/trenova/internal/core/services/tenderservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type shipmentBook struct {
	repositories.ShipmentRepository
	byID map[pulid.ID]*shipment.Shipment
}

func (f *shipmentBook) GetByID(
	_ context.Context,
	req *repositories.GetShipmentByIDRequest,
) (*shipment.Shipment, error) {
	if entity, ok := f.byID[req.ID]; ok {
		return entity, nil
	}

	return nil, errortypes.NewNotFoundError("Shipment not found")
}

type fakeShopper struct {
	options  []*services.ShopOption
	requests []*ratequoteservice.ShopRequest
}

func (f *fakeShopper) Shop(
	_ context.Context,
	req *ratequoteservice.ShopRequest,
) (*services.ShopResult, error) {
	f.requests = append(f.requests, req)

	return &services.ShopResult{Options: f.options}, nil
}

type fakeTenderer struct {
	waterfallErr error
	spots        []*tenderservice.CreateSpotTenderRequest
	waterfalls   []*tenderservice.CreateWaterfallTenderRequest
}

func (f *fakeTenderer) CreateSpot(
	_ context.Context,
	req *tenderservice.CreateSpotTenderRequest,
) (*tender.Tender, error) {
	f.spots = append(f.spots, req)

	return &tender.Tender{ID: pulid.MustNew("tnd_")}, nil
}

func (f *fakeTenderer) CreateWaterfall(
	_ context.Context,
	req *tenderservice.CreateWaterfallTenderRequest,
) (*tenderservice.CreateWaterfallResult, error) {
	f.waterfalls = append(f.waterfalls, req)
	if f.waterfallErr != nil {
		return nil, f.waterfallErr
	}
	guideCarrier := &carrier.Carrier{ID: pulid.MustNew("car_"), Name: "Werner"}

	return &tenderservice.CreateWaterfallResult{Tender: &tender.Tender{
		ID: pulid.MustNew("tnd_"),
		Offers: []*tender.TenderOffer{
			{CarrierID: guideCarrier.ID, Carrier: guideCarrier},
		},
	}}, nil
}

func uncoveredShipment(moves ...*shipment.ShipmentMove) *shipment.Shipment {
	return &shipment.Shipment{ID: pulid.MustNew("shp_"), Moves: moves}
}

func move(sequence int64, coverage shipment.MoveCoverageType) *shipment.ShipmentMove {
	return &shipment.ShipmentMove{
		ID:           pulid.MustNew("smv_"),
		Sequence:     sequence,
		CoverageType: coverage,
	}
}

type tenderFixture struct {
	service  *Service
	shopper  *fakeShopper
	tenderer *fakeTenderer
	book     *shipmentBook
}

func newTenderFixture(shipments ...*shipment.Shipment) *tenderFixture {
	book := &shipmentBook{byID: map[pulid.ID]*shipment.Shipment{}}
	for _, sp := range shipments {
		book.byID[sp.ID] = sp
	}
	f := &tenderFixture{
		shopper: &fakeShopper{options: []*services.ShopOption{
			{CarrierID: pulid.MustNew("car_"), CarrierName: "No rate", Outcome: ratequote.OutcomeNoRateFound},
			{
				CarrierID:   pulid.MustNew("car_"),
				CarrierName: "Prime Inc.",
				Outcome:     ratequote.OutcomeRated,
				Cost:        decimal.RequireFromString("620"),
			},
		}},
		tenderer: &fakeTenderer{},
		book:     book,
	}
	f.service = NewWithDependencies(&Dependencies{
		Shipments: book,
		Shopper:   f.shopper,
		Tenderer:  f.tenderer,
	})

	return f
}

func tenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
}

func TestTenderShipments_NamedCarrierGetsASpotTenderAtItsPricedRate(t *testing.T) {
	t.Parallel()

	first := move(0, shipment.MoveCoverageTypeDriver)
	second := move(1, shipment.MoveCoverageTypeUnassigned)
	sp := uncoveredShipment(second, first)
	f := newTenderFixture(sp)
	carrierID := pulid.MustNew("car_")

	result, err := f.service.TenderShipments(t.Context(), &services.TenderShipmentsRequest{
		TenantInfo: tenant(),
		Items:      []services.TenderShipmentItem{{ShipmentID: sp.ID, CarrierID: carrierID}},
	})

	require.NoError(t, err)
	require.Len(t, result.Tendered, 1)
	assert.Empty(t, result.Failed)
	require.Len(t, f.shopper.requests, 1)
	assert.Equal(t, []pulid.ID{carrierID}, f.shopper.requests[0].CarrierIDs)
	require.Len(t, f.tenderer.spots, 1)
	spot := f.tenderer.spots[0]
	assert.Equal(t, second.ID, spot.ShipmentMoveID, "the first move still without coverage")
	assert.Equal(t, tender.ModeSpotBroadcast, spot.Mode)
	require.Len(t, spot.Lines, 1)
	assert.Equal(t, shipment.CarrierRateMethodFlat, spot.Lines[0].RateMethod)
	assert.Equal(t, "620", spot.Lines[0].Rate.String(), "the first priced option, never an unpriced one")
	assert.Equal(t, "Prime Inc.", result.Tendered[0].CarrierName)
	assert.Empty(t, f.tenderer.waterfalls)
}

func TestTenderShipments_UsesTheRoutingGuideWhenNoCarrierIsNamed(t *testing.T) {
	t.Parallel()

	sp := uncoveredShipment(move(0, shipment.MoveCoverageTypeUnassigned))
	f := newTenderFixture(sp)

	result, err := f.service.TenderShipments(t.Context(), &services.TenderShipmentsRequest{
		TenantInfo: tenant(),
		Items:      []services.TenderShipmentItem{{ShipmentID: sp.ID}},
	})

	require.NoError(t, err)
	require.Len(t, result.Tendered, 1)
	assert.Equal(t, "Werner", result.Tendered[0].CarrierName)
	assert.Len(t, f.tenderer.waterfalls, 1)
	assert.Empty(t, f.tenderer.spots)
}

func TestTenderShipments_FallsBackToTheBestPricedCarrierOnlyWhenNoGuideMatches(t *testing.T) {
	t.Parallel()

	sp := uncoveredShipment(move(0, shipment.MoveCoverageTypeUnassigned))
	noGuide := newTenderFixture(sp)
	noGuide.tenderer.waterfallErr = errortypes.NewBusinessError("No routing guide matches this lane").
		WithInternal(tenderservice.ErrNoRoutingGuideMatch)

	result, err := noGuide.service.TenderShipments(t.Context(), &services.TenderShipmentsRequest{
		TenantInfo: tenant(),
		Items:      []services.TenderShipmentItem{{ShipmentID: sp.ID}},
	})
	require.NoError(t, err)
	require.Len(t, result.Tendered, 1)
	assert.Len(t, noGuide.tenderer.spots, 1)
	assert.Nil(t, noGuide.shopper.requests[0].CarrierIDs, "the best match across the network")

	held := newTenderFixture(sp)
	held.tenderer.waterfallErr = errortypes.NewBusinessError("The shipment is on hold")
	result, err = held.service.TenderShipments(t.Context(), &services.TenderShipmentsRequest{
		TenantInfo: tenant(),
		Items:      []services.TenderShipmentItem{{ShipmentID: sp.ID}},
	})
	require.NoError(t, err)
	assert.Empty(t, result.Tendered)
	require.Len(t, result.Failed, 1)
	assert.Equal(t, "The shipment is on hold", result.Failed[0].Message)
	assert.Empty(t, held.tenderer.spots, "any other refusal is reported, not worked around")
}

func TestTenderShipments_ReportsEachShipmentItCouldNotTender(t *testing.T) {
	t.Parallel()

	covered := uncoveredShipment(move(0, shipment.MoveCoverageTypeCarrier))
	open := uncoveredShipment(move(0, shipment.MoveCoverageTypeUnassigned))
	f := newTenderFixture(covered, open)
	missing := pulid.MustNew("shp_")

	result, err := f.service.TenderShipments(t.Context(), &services.TenderShipmentsRequest{
		TenantInfo: tenant(),
		Items: []services.TenderShipmentItem{
			{ShipmentID: covered.ID},
			{ShipmentID: missing},
			{ShipmentID: open.ID},
		},
	})

	require.NoError(t, err)
	require.Len(t, result.Tendered, 1)
	assert.Equal(t, open.ID, result.Tendered[0].ShipmentID)
	require.Len(t, result.Failed, 2)
	assert.Equal(t, errAlreadyCovered.Error(), result.Failed[0].Message)
	assert.Equal(t, "Shipment not found", result.Failed[1].Message)
}

func TestTenderShipments_RefusesAnEmptyOrOversizedBatch(t *testing.T) {
	t.Parallel()

	f := newTenderFixture()
	_, err := f.service.TenderShipments(t.Context(), &services.TenderShipmentsRequest{TenantInfo: tenant()})
	require.Error(t, err)

	items := make([]services.TenderShipmentItem, maxTenderItems+1)
	_, err = f.service.TenderShipments(t.Context(), &services.TenderShipmentsRequest{
		TenantInfo: tenant(),
		Items:      items,
	})
	var validation *errortypes.Error
	require.True(t, errors.As(err, &validation))
}
