package shipmentcapacityservice

import (
	"context"
	"errors"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/ports/dispatchconsoleports"
	"github.com/emoss08/trenova/internal/modules/operations/app/tenderservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentboardports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentcapacityports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carriercapacityports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierports"
	"github.com/emoss08/trenova/internal/modules/rating/app/ratequoteservice"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratequoteports"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	performanceLookback     = 90 * 24 * time.Hour
	carrierWindow           = 48 * time.Hour
	withinTwoHours          = 2 * time.Hour
	usuallyAcceptPercent    = 80.0
	usuallyAcceptMinAnswers = 3
	lowAcceptancePercent    = 50.0
	hosMaxHours             = 11.0
	hosLowHours             = 4.0
	defaultPickupRadius     = 50.0
	maxMatches              = 10
	defaultMatches          = 2
	coverageSuggestions     = 3
	maxTenderItems          = 100
	driverMatchScan         = 25
)

var ErrUnknownKind = errors.New("capacity kind must be Driver or Carrier")

type shopper interface {
	Shop(ctx context.Context, req *ratequoteservice.ShopRequest) (*ratequoteports.ShopResult, error)
}

type tenderer interface {
	CreateSpot(
		ctx context.Context,
		req *tenderservice.CreateSpotTenderRequest,
	) (*tender.Tender, error)
	CreateWaterfall(
		ctx context.Context,
		req *tenderservice.CreateWaterfallTenderRequest,
	) (*tenderservice.CreateWaterfallResult, error)
}

type Params struct {
	fx.In

	Console      dispatchconsoleports.DispatchConsoleService
	Capabilities shipmentboardports.ShipmentBoardCapabilitiesReader
	Postings     carriercapacityports.CarrierCapacityPostingRepository
	Performance  shipmentboardports.CarrierPerformanceRepository
	Carriers     carrierports.CarrierRepository
	Shipments    shipmentports.ShipmentRepository
	RateQuotes   *ratequoteservice.Service
	Tenders      *tenderservice.Service
	Logger       *zap.Logger
}

type Dependencies struct {
	Console      dispatchconsoleports.DispatchConsoleService
	Capabilities shipmentboardports.ShipmentBoardCapabilitiesReader
	Postings     carriercapacityports.CarrierCapacityPostingRepository
	Performance  shipmentboardports.CarrierPerformanceRepository
	Carriers     carrierports.CarrierRepository
	Shipments    shipmentports.ShipmentRepository
	Shopper      shopper
	Tenderer     tenderer
	Logger       *zap.Logger
	Now          func() time.Time
}

type Service struct {
	console      dispatchconsoleports.DispatchConsoleService
	capabilities shipmentboardports.ShipmentBoardCapabilitiesReader
	postings     carriercapacityports.CarrierCapacityPostingRepository
	performance  shipmentboardports.CarrierPerformanceRepository
	carriers     carrierports.CarrierRepository
	shipments    shipmentports.ShipmentRepository
	shopper      shopper
	tenderer     tenderer
	l            *zap.Logger
	now          func() time.Time
}

var (
	_ shipmentcapacityports.ShipmentCapacityReader    = (*Service)(nil)
	_ shipmentcapacityports.ShipmentCoverageSuggester = (*Service)(nil)
	_ shipmentcapacityports.ShipmentTenderer          = (*Service)(nil)
)

//nolint:gocritic // dependency injection
func New(p Params) *Service {
	return NewWithDependencies(&Dependencies{
		Console:      p.Console,
		Capabilities: p.Capabilities,
		Postings:     p.Postings,
		Performance:  p.Performance,
		Carriers:     p.Carriers,
		Shipments:    p.Shipments,
		Shopper:      p.RateQuotes,
		Tenderer:     p.Tenders,
		Logger:       p.Logger,
	})
}

func NewWithDependencies(d *Dependencies) *Service {
	now := d.Now
	if now == nil {
		now = time.Now
	}
	logger := d.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	return &Service{
		console:      d.Console,
		capabilities: d.Capabilities,
		postings:     d.Postings,
		performance:  d.Performance,
		carriers:     d.Carriers,
		shipments:    d.Shipments,
		shopper:      d.Shopper,
		tenderer:     d.Tenderer,
		l:            logger.Named("service.shipment-capacity"),
		now:          now,
	}
}

func NewCapacityReader(s *Service) shipmentcapacityports.ShipmentCapacityReader { return s }

func NewCoverageSuggester(s *Service) shipmentcapacityports.ShipmentCoverageSuggester { return s }

func NewTenderer(s *Service) shipmentcapacityports.ShipmentTenderer { return s }

func (s *Service) Capacity(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	kind shipmentcapacityports.CapacityUnitKind,
) (*shipmentcapacityports.ShipmentCapacity, error) {
	switch kind {
	case shipmentcapacityports.CapacityUnitDriver:
		return s.driverCapacity(ctx, tenantInfo)
	case shipmentcapacityports.CapacityUnitCarrier:
		return s.carrierCapacity(ctx, tenantInfo)
	default:
		return nil, ErrUnknownKind
	}
}

func (s *Service) Matches(
	ctx context.Context,
	req *shipmentcapacityports.CapacityMatchesRequest,
) ([]*shipmentcapacityports.CapacityMatch, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = defaultMatches
	}
	limit = min(limit, maxMatches)

	switch req.Kind {
	case shipmentcapacityports.CapacityUnitDriver:
		return s.driverMatches(ctx, req.TenantInfo, req.UnitID, limit)
	case shipmentcapacityports.CapacityUnitCarrier:
		return s.carrierMatches(ctx, req.TenantInfo, req.UnitID, limit)
	default:
		return nil, ErrUnknownKind
	}
}
