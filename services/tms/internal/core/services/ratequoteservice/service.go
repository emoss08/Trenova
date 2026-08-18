package ratequoteservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/domain/ratequote"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger             *zap.Logger
	Repo               repositories.RateQuoteRepository
	ShipmentRepo       repositories.ShipmentRepository
	BillingControlRepo repositories.BillingControlRepository
	Engine             services.RateEngine
}

type Service struct {
	l                  *zap.Logger
	repo               repositories.RateQuoteRepository
	shipmentRepo       repositories.ShipmentRepository
	billingControlRepo repositories.BillingControlRepository
	engine             services.RateEngine
}

func New(p Params) *Service {
	return &Service{
		l:                  p.Logger.Named("service.rate-quote"),
		repo:               p.Repo,
		shipmentRepo:       p.ShipmentRepo,
		billingControlRepo: p.BillingControlRepo,
		engine:             p.Engine,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListRateQuotesRequest,
) (*pagination.ListResult[*ratequote.RateQuote], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) ListConnection(
	ctx context.Context,
	req *repositories.ListRateQuoteConnectionRequest,
) (*pagination.CursorListResult[*ratequote.RateQuote], error) {
	return s.repo.ListConnection(ctx, req)
}

func (s *Service) GetByID(
	ctx context.Context,
	req *repositories.GetRateQuoteByIDRequest,
) (*ratequote.RateQuote, error) {
	return s.repo.GetByID(ctx, req)
}

// ListForShipment returns a shipment's rating history, newest first.
//
// A shipment is re-rated whenever a stop time, an assignment or a fuel price
// changes, so the history is how somebody sees that the rate moved and when.
func (s *Service) ListForShipment(
	ctx context.Context,
	req *repositories.ListShipmentRateQuotesRequest,
) ([]*ratequote.RateQuote, error) {
	return s.repo.ListForShipment(ctx, req)
}

// GetAppliedForShipment returns the quote currently governing a shipment, which
// is what the "why this rate" panel reads.
func (s *Service) GetAppliedForShipment(
	ctx context.Context,
	req *repositories.GetShipmentRateQuoteRequest,
) (*ratequote.RateQuote, error) {
	return s.repo.GetAppliedForShipment(ctx, req)
}

// ExplainRequest asks what a shipment would rate at, optionally on another day.
type ExplainRequest struct {
	TenantInfo pagination.TenantInfo
	ShipmentID pulid.ID
	PartyType  rateagreement.PartyType
	PartyID    pulid.ID
	// AsOf re-resolves against the terms effective on that date. Zero uses the
	// shipment's own rating date, which reproduces its current rate.
	AsOf int64
}

// Explain re-rates a saved shipment and returns the full trace without touching
// what it is currently billed at.
//
// This is the answer to "what would this have cost under next month's tariff",
// and to "show me why", and the two are the same question asked on different
// dates. The quote it produces is a what-if: it is recorded so the comparison
// can be cited later, but it never supersedes the shipment's applied rate.
func (s *Service) Explain(
	ctx context.Context,
	req *ExplainRequest,
) (*services.RatedShipment, error) {
	log := s.l.With(
		zap.String("operation", "Explain"),
		zap.String("shipmentId", req.ShipmentID.String()),
	)

	entity, err := s.shipmentRepo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID:         req.ShipmentID,
		TenantInfo: req.TenantInfo,
		ShipmentOptions: repositories.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		log.Error("failed to load shipment for explanation", zap.Error(err))
		return nil, err
	}

	return s.rate(ctx, entity, req, ratequote.PurposeWhatIf, true)
}

// QuoteRequest prices a shipment that has not been saved, which is what a
// pre-booking quote is.
type QuoteRequest struct {
	TenantInfo pagination.TenantInfo
	Shipment   *shipment.Shipment
	PartyType  rateagreement.PartyType
	PartyID    pulid.ID
	AsOf       int64
	// Persist keeps the quote so it can be sent to the customer and referred to
	// when the load is booked. A screen that re-prices on every keystroke
	// leaves it off.
	Persist bool
}

// Quote prices a hypothetical shipment.
func (s *Service) Quote(
	ctx context.Context,
	req *QuoteRequest,
) (*services.RatedShipment, error) {
	if req.Shipment == nil {
		return nil, errortypes.NewValidationError(
			"shipment",
			errortypes.ErrRequired,
			"A shipment is required to produce a quote",
		)
	}

	req.Shipment.OrganizationID = req.TenantInfo.OrgID
	req.Shipment.BusinessUnitID = req.TenantInfo.BuID

	return s.rate(ctx, req.Shipment, &ExplainRequest{
		TenantInfo: req.TenantInfo,
		PartyType:  req.PartyType,
		PartyID:    req.PartyID,
		AsOf:       req.AsOf,
	}, ratequote.PurposeQuote, req.Persist)
}

func (s *Service) rate(
	ctx context.Context,
	entity *shipment.Shipment,
	req *ExplainRequest,
	purpose ratequote.Purpose,
	persist bool,
) (*services.RatedShipment, error) {
	partyType := req.PartyType
	if partyType == "" {
		partyType = rateagreement.PartyTypeCustomer
	}

	return s.engine.RateShipment(ctx, &services.RateShipmentRequest{
		Shipment:       entity,
		TenantInfo:     req.TenantInfo,
		PartyType:      partyType,
		PartyID:        req.PartyID,
		AsOf:           req.AsOf,
		BillingControl: s.billingControl(ctx, req.TenantInfo.OrgID),
		Purpose:        purpose,
		Persist:        persist,
		UserID:         req.TenantInfo.UserID,
	})
}

// billingControl carries the organization's unrated-shipment decision into the
// engine. Failing to read it is not worth failing the request over: the default
// it falls back to is the behaviour that existed before rate agreements.
func (s *Service) billingControl(ctx context.Context, orgID pulid.ID) *tenant.BillingControl {
	control, err := s.billingControlRepo.GetByOrgID(ctx, orgID)
	if err != nil {
		return nil
	}

	return control
}

// ShopRequest asks what several carriers would charge to haul a saved shipment.
type ShopRequest struct {
	TenantInfo pagination.TenantInfo
	ShipmentID pulid.ID
	Strategy   services.ShopStrategy
	// CarrierIDs is an explicit shortlist. Left empty, the shipment's routing
	// guide supplies the candidates.
	CarrierIDs []pulid.ID
	AsOf       int64
	Limit      int
	// Persist keeps each option's quote, so the carrier that was picked and the
	// ones that were not are both answerable later.
	Persist bool
}

// Shop prices a saved shipment against several carriers and ranks them.
//
// The sell price it measures margin against is the shipment's own total, which
// is the number the customer is being charged. Taking it from the request would
// let a screen ask "what is my margin if I pretend this sold for more", which
// is a question worth asking but not one the record of a decision should carry.
func (s *Service) Shop(
	ctx context.Context,
	req *ShopRequest,
) (*services.ShopResult, error) {
	log := s.l.With(
		zap.String("operation", "Shop"),
		zap.String("shipmentId", req.ShipmentID.String()),
	)

	entity, err := s.shipmentRepo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID:         req.ShipmentID,
		TenantInfo: req.TenantInfo,
		ShipmentOptions: repositories.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		log.Error("failed to load shipment for shopping", zap.Error(err))
		return nil, err
	}

	return s.engine.Shop(ctx, &services.ShopRequest{
		Shipment:       entity,
		TenantInfo:     req.TenantInfo,
		Strategy:       req.Strategy,
		CarrierIDs:     req.CarrierIDs,
		AsOf:           req.AsOf,
		BillingControl: s.billingControl(ctx, req.TenantInfo.OrgID),
		SellTotal:      entity.TotalChargeAmount,
		Limit:          req.Limit,
		Persist:        req.Persist,
		UserID:         req.TenantInfo.UserID,
	})
}
