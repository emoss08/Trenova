package tendercreate

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/app/dispatchguard"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmenteventservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/tendereffects"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/operations/jobs/tenderjobs/tendercontract"
	"github.com/emoss08/trenova/internal/modules/operations/ports/assignmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/carrierassignmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentholdports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentinvalidationports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/tenderports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierports"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratequote"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratequoteports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Service struct {
	l                     *zap.Logger
	db                    kernelports.Transactor
	repo                  tenderports.TenderRepository
	guideRepo             tenderports.RoutingGuideRepository
	carrierRepo           carrierports.CarrierRepository
	carrierAssignmentRepo carrierassignmentports.CarrierAssignmentRepository
	assignmentRepo        assignmentports.AssignmentRepository
	shipmentRepo          shipmentports.ShipmentRepository
	holdRepo              shipmentholdports.ShipmentHoldRepository
	workflows             kernelports.WorkflowStarter
	invalidator           shipmentinvalidationports.ShipmentInvalidator
	rateEngine            ratequoteports.RateEngine
	effects               *tendereffects.Effects
}

// priceGuideEntry decides what to offer, reading the carrier's contract when
// the entry asks for it.
//
// An entry that did not ask keeps the rate the guide carries, so turning this
// feature on changes nothing about the guides an organization already wrote.
//
// A contract-priced entry that cannot be priced is refused rather than offered
// at the frozen rate. Somebody marked that entry to follow the contract, and
// quietly tendering last year's number instead is the opposite of what they
// asked for. The one exception is a deployment with no engine wired at all,
// where the frozen rate is the only rate there is.
func (s *Service) priceGuideEntry(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	entry *tender.RoutingGuideEntry,
	entity *shipment.Shipment,
) offerPrice {
	frozen := offerPrice{method: entry.RateMethod, rate: entry.Rate}

	if !entry.UseContractRate || s.rateEngine == nil || entity == nil {
		return frozen
	}

	rated, err := s.rateEngine.RateShipment(ctx, &ratequoteports.RateShipmentRequest{
		Shipment:   entity,
		TenantInfo: tenantInfo,
		PartyType:  rateagreement.PartyTypeCarrier,
		PartyID:    entry.CarrierID,
		Purpose:    ratequote.PurposeShopping,
		UserID:     tenantInfo.UserID,
	})
	if err != nil {
		s.l.Warn("failed to price a routing guide entry from its carrier's contract",
			zap.String("carrierId", entry.CarrierID.String()),
			zap.Error(err),
		)

		return offerPrice{reason: "The carrier's contract could not be read to price this offer"}
	}

	if !rated.Outcome.Priced() {
		return offerPrice{reason: "No carrier agreement covers this lane"}
	}

	// The contract produced one number, already through its own minimums and
	// rounding. Offering it as a per-mile rate would invite the offer to be
	// recomputed into something the contract never said.
	return offerPrice{method: shipmentvocab.CarrierRateMethodFlat, rate: rated.Total}
}

// shipmentForContractPricing loads the shipment a contract-priced entry needs.
//
// It returns nothing when no entry asks for contract pricing, so a guide that
// has always used its own rates never pays for a read it does not use. A read
// that fails is logged and treated as absent, which sends every contract-priced
// entry down the "cannot be priced" path rather than tendering the wrong number.
func (s *Service) shipmentForContractPricing(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	guide *tender.RoutingGuide,
	move *shipment.ShipmentMove,
) *shipment.Shipment {
	if s.rateEngine == nil || move == nil || move.ShipmentID.IsNil() {
		return nil
	}

	wanted := false
	for _, entry := range guide.Entries {
		if entry != nil && entry.UseContractRate {
			wanted = true
			break
		}
	}

	if !wanted {
		return nil
	}

	entity, err := s.shipmentRepo.GetByID(ctx, &shipmentports.GetShipmentByIDRequest{
		ID:         move.ShipmentID,
		TenantInfo: tenantInfo,
		ShipmentOptions: shipmentports.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		s.l.Warn("failed to load the shipment for contract priced tender offers",
			zap.String("shipmentId", move.ShipmentID.String()),
			zap.Error(err),
		)

		return nil
	}

	return entity
}

// CreateWaterfall starts a routing-guide-driven tender for a move. The guide
// is auto-matched from the move's lane unless one is named explicitly.
func (s *Service) CreateWaterfall(
	ctx context.Context,
	req *CreateWaterfallTenderRequest,
) (*CreateWaterfallResult, error) {
	waterfall, err := s.planWaterfall(ctx, req, true)
	if err != nil {
		return nil, err
	}
	guide, plan := waterfall.guide, waterfall.offers

	created, err := s.PersistAndStart(ctx, req.TenantInfo, waterfall.tender)
	if err != nil {
		return nil, err
	}

	s.effects.LogAudit(&tendereffects.AuditParams{
		TenantInfo: req.TenantInfo,
		TenderID:   created.ID,
		Operation:  permission.OpCreate,
		UserID:     req.TenantInfo.UserID,
		Comment:    fmt.Sprintf("Waterfall tender created from routing guide %s", guide.Name),
		Current:    created,
	})
	s.auditGuideScreening(ctx, req.TenantInfo, created, plan)

	return &CreateWaterfallResult{Tender: created, Screening: plan.screeningSummary()}, nil
}

// CreateSpot starts an ad-hoc tender for a move with a dispatcher-picked
// carrier list, either broadcast simultaneously or worked in sequence.
func (s *Service) CreateSpot(
	ctx context.Context,
	req *CreateSpotTenderRequest,
) (*tender.Tender, error) {
	spot, err := s.planSpot(ctx, req, true)
	if err != nil {
		return nil, err
	}
	if err = spot.refusal(req); err != nil {
		return nil, err
	}
	offers := spot.tender.Offers

	created, err := s.PersistAndStart(ctx, req.TenantInfo, spot.tender)
	if err != nil {
		return nil, err
	}

	s.effects.LogAudit(&tendereffects.AuditParams{
		TenantInfo: req.TenantInfo,
		TenderID:   created.ID,
		Operation:  permission.OpCreate,
		UserID:     req.TenantInfo.UserID,
		Comment: fmt.Sprintf(
			"Spot tender created (%s, %d carriers)", req.Mode.String(), len(offers),
		),
		Current: created,
	})

	return created, nil
}

// LoadTenderableMove enforces every precondition shared by both tender modes:
// the move must be assignable, hold-free, and not already covered by a driver,
// a carrier, or another live tender.
func (s *Service) LoadTenderableMove(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	moveID pulid.ID,
) (*shipment.ShipmentMove, *shipment.Shipment, error) {
	move, err := s.assignmentRepo.GetMoveByID(ctx, tenantInfo, moveID)
	if err != nil {
		return nil, nil, err
	}
	if err = move.EnsureAssignable(); err != nil {
		return nil, nil, err
	}
	if err = dispatchguard.EnsureNoDispatchHold(
		ctx, s.holdRepo, tenantInfo, move.ShipmentID,
	); err != nil {
		return nil, nil, err
	}

	driverAssignment, err := s.assignmentRepo.GetByMoveID(ctx, tenantInfo, moveID)
	if err != nil {
		return nil, nil, err
	}
	if driverAssignment != nil {
		return nil, nil, errortypes.NewBusinessError(
			"Shipment move already has a driver assignment",
		).WithParam("shipmentMoveId", moveID.String())
	}

	carrierAssignment, err := s.carrierAssignmentRepo.GetActiveByMoveID(ctx, tenantInfo, moveID)
	if err != nil {
		return nil, nil, err
	}
	if carrierAssignment != nil {
		return nil, nil, errortypes.NewBusinessError(
			"Shipment move already has a carrier assignment",
		).WithParam("shipmentMoveId", moveID.String())
	}

	live, err := s.repo.GetLiveByMoveID(ctx, tenderports.GetLiveTenderByMoveRequest{
		TenantInfo: tenantInfo,
		MoveID:     moveID,
	})
	if err != nil {
		return nil, nil, err
	}
	if live != nil {
		return nil, nil, errortypes.NewBusinessError(
			"Shipment move already has a tender in progress",
		).WithParam("tenderId", live.ID.String())
	}

	sh, err := s.shipmentRepo.GetByID(ctx, &shipmentports.GetShipmentByIDRequest{
		ID: move.ShipmentID,
		TenantInfo: pagination.TenantInfo{
			OrgID: tenantInfo.OrgID,
			BuID:  tenantInfo.BuID,
		},
		ShipmentOptions: shipmentports.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		return nil, nil, err
	}

	return move, sh, nil
}

func (s *Service) resolveGuide(
	ctx context.Context,
	req *CreateWaterfallTenderRequest,
	move *shipment.ShipmentMove,
	sh *shipment.Shipment,
) (*tender.RoutingGuide, error) {
	if req.RoutingGuideID != nil && !req.RoutingGuideID.IsNil() {
		guide, err := s.guideRepo.GetByID(ctx, tenderports.GetRoutingGuideByIDRequest{
			ID:         *req.RoutingGuideID,
			TenantInfo: req.TenantInfo,
			RoutingGuideFilterOptions: tenderports.RoutingGuideFilterOptions{
				IncludeEntries: true,
			},
		})
		if err != nil {
			return nil, err
		}
		if guide.Status != domaintypes.StatusActive {
			return nil, errortypes.NewBusinessError("Routing guide is inactive").
				WithParam("routingGuideId", guide.ID.String())
		}
		if len(guide.Entries) == 0 {
			return nil, errortypes.NewBusinessError("Routing guide has no carrier entries").
				WithParam("routingGuideId", guide.ID.String())
		}
		return guide, nil
	}

	matchReq := s.buildMatchLaneRequest(req.TenantInfo, move, sh)
	guide, err := s.guideRepo.MatchLane(ctx, matchReq)
	if err != nil {
		return nil, err
	}
	if guide == nil {
		return nil, errortypes.NewBusinessError(
			"No routing guide matches this lane. Create one or use a spot tender",
		).WithParam("shipmentMoveId", move.ID.String()).WithInternal(ErrNoRoutingGuideMatch)
	}
	if len(guide.Entries) == 0 {
		return nil, errortypes.NewBusinessError("Matched routing guide has no carrier entries").
			WithParam("routingGuideId", guide.ID.String())
	}
	return guide, nil
}

// buildMatchLaneRequest derives the move's lane from its stops: the first
// origin stop and the last destination stop, falling back to the shipment's
// full stop set when the move itself carries none.
func (s *Service) buildMatchLaneRequest(
	tenantInfo pagination.TenantInfo,
	move *shipment.ShipmentMove,
	sh *shipment.Shipment,
) *tenderports.MatchLaneRequest {
	req := &tenderports.MatchLaneRequest{TenantInfo: tenantInfo}

	stops := move.Stops
	if len(stops) == 0 && sh != nil {
		target := sh.FindMove(move.ID)
		if target != nil {
			stops = target.Stops
		}
	}

	for _, stop := range stops {
		if stop == nil {
			continue
		}
		if stop.IsOriginStop() && req.OriginLocationID.IsNil() {
			req.OriginLocationID = stop.LocationID
			if stop.Location != nil {
				req.OriginCity = stop.Location.City
				if stop.Location.State != nil {
					req.OriginState = stop.Location.State.Abbreviation
				}
			}
		}
		if stop.IsDestinationStop() {
			req.DestinationLocationID = stop.LocationID
			if stop.Location != nil {
				req.DestinationCity = stop.Location.City
				if stop.Location.State != nil {
					req.DestinationState = stop.Location.State.Abbreviation
				}
			}
		}
	}

	return req
}

// buildGuideOffers screens every guide entry's carrier before it can be
// offered: blocked carriers are skipped so one lapsed certificate cannot stall
// the lane, carriers with insurance warnings pass through flagged, and a guide
// with nobody eligible fails creation outright.
func (s *Service) buildGuideOffers(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	guide *tender.RoutingGuide,
	move *shipment.ShipmentMove,
	refreshIntel bool,
) (*guideOfferPlan, error) {
	carrierIDs := make([]pulid.ID, 0, len(guide.Entries))
	for _, entry := range guide.Entries {
		if entry == nil {
			continue
		}
		carrierIDs = append(carrierIDs, entry.CarrierID)
	}
	carriers, carrierByID, err := s.loadTenderCarriers(ctx, tenantInfo, carrierIDs)
	if err != nil {
		return nil, err
	}

	screen := &offerScreen{
		tenantInfo:  tenantInfo,
		carrierByID: carrierByID,
		now:         timeutils.NowUnix(),
		gates:       s.effects.IntelGates(ctx, tenantInfo, carrierIDs, refreshIntel),
		move:        move,
		// The shipment is read once, and only when some entry actually asks for
		// contract pricing. Reading it for every guide would put a query on a path
		// that has never needed one.
		pricing: s.shipmentForContractPricing(ctx, tenantInfo, guide, move),
	}
	plan := &guideOfferPlan{
		offers:   make([]*tender.TenderOffer, 0, len(guide.Entries)),
		carriers: carrierNames(carriers),
	}
	multiErr := errortypes.NewMultiError()

	for idx, entry := range guide.Entries {
		if entry == nil {
			continue
		}
		s.screenGuideEntry(ctx, screen, plan, entry, multiErr.WithIndex("entries", idx))
	}

	if multiErr.HasErrors() {
		return nil, multiErr
	}
	if len(plan.offers) == 0 && len(plan.skipped) > 0 {
		return nil, errortypes.NewBusinessError(
			"No eligible carriers on the routing guide. Every entry is blocked by carrier eligibility",
		).WithParam("routingGuideId", guide.ID.String())
	}
	return plan, nil
}

type offerScreen struct {
	tenantInfo  pagination.TenantInfo
	carrierByID map[pulid.ID]*carrier.Carrier
	gates       map[pulid.ID]*carrier.IntelGate
	now         int64
	move        *shipment.ShipmentMove
	pricing     *shipment.Shipment
}

func (s *Service) loadTenderCarriers(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	carrierIDs []pulid.ID,
) ([]*carrier.Carrier, map[pulid.ID]*carrier.Carrier, error) {
	carriers, err := s.carrierRepo.GetByIDs(ctx, carrierports.GetCarriersByIDsRequest{
		TenantInfo: tenantInfo,
		CarrierIDs: carrierIDs,
		CarrierFilterOptions: carrierports.CarrierFilterOptions{
			IncludeInsurancePolicies: true,
		},
	})
	if err != nil {
		return nil, nil, err
	}
	carrierByID := make(map[pulid.ID]*carrier.Carrier, len(carriers))
	for _, c := range carriers {
		carrierByID[c.ID] = c
	}
	return carriers, carrierByID, nil
}

func (s *Service) screenGuideEntry(
	ctx context.Context,
	screen *offerScreen,
	plan *guideOfferPlan,
	entry *tender.RoutingGuideEntry,
	entryErr *errortypes.MultiError,
) {
	carrierEntity, ok := screen.carrierByID[entry.CarrierID]
	if !ok {
		plan.skipped = append(plan.skipped, guideEntryScreening{
			carrierName: guideEntryCarrierName(entry),
			rank:        entry.Rank,
			reasons:     []string{"Carrier does not exist within your organization"},
		})
		return
	}

	eligibility := carrier.EvaluateEligibility(carrier.EligibilityInput{
		Carrier: carrierEntity,
		Now:     screen.now,
		Intel:   screen.gates[carrierEntity.ID],
	})
	if eligibility.IsBlocked() {
		plan.skipped = append(plan.skipped, guideEntryScreening{
			carrierName: carrierEntity.Name,
			rank:        entry.Rank,
			reasons:     eligibility.Blockers,
		})
		return
	}

	price := s.priceGuideEntry(ctx, screen.tenantInfo, entry, screen.pricing)
	if price.reason != "" {
		plan.skipped = append(plan.skipped, guideEntryScreening{
			carrierName: carrierEntity.Name,
			rank:        entry.Rank,
			reasons:     []string{price.reason},
		})
		return
	}

	if price.method == shipmentvocab.CarrierRateMethodPerMile &&
		(screen.move.Distance == nil || *screen.move.Distance <= 0) {
		entryErr.Add(
			"rateMethod",
			errortypes.ErrInvalid,
			"Per-mile entries require a move with a computed distance",
		)
		return
	}

	offer := &tender.TenderOffer{
		OrganizationID:  screen.tenantInfo.OrgID,
		BusinessUnitID:  screen.tenantInfo.BuID,
		CarrierID:       entry.CarrierID,
		Rank:            entry.Rank,
		RateMethod:      price.method,
		Rate:            price.rate,
		OfferTTLSeconds: entry.OfferTTLSeconds,
		Channel:         entry.Channel,
		Status:          tender.OfferStatusPending,
	}
	if err := s.resolveOfferChannel(
		ctx, screen.tenantInfo, offer, carrierEntity, "",
	); err != nil {
		entryErr.Add("channel", errortypes.ErrInvalid, err.Error())
		return
	}
	if eligibility.HasWarnings() {
		plan.warned = append(plan.warned, guideEntryScreening{
			carrierName: carrierEntity.Name,
			rank:        entry.Rank,
			reasons:     eligibility.Warnings,
		})
	}
	plan.offers = append(plan.offers, offer)
}

// auditGuideScreening writes the skip-with-audit trail after the tender row
// exists: one shipment event per screened entry and a single dispatch
// notification summarizing the skips. Failures never unwind the created
// tender.
func (s *Service) auditGuideScreening(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	entity *tender.Tender,
	plan *guideOfferPlan,
) {
	if len(plan.skipped) == 0 && len(plan.warned) == 0 {
		return
	}

	ref := shipmenteventservice.TenderRef{
		ShipmentID: entity.ShipmentID,
		MoveID:     entity.ShipmentMoveID,
		TenderID:   entity.ID,
	}
	tenant := shipmenteventservice.TenantRefFor(tenantInfo)
	actor := shipmenteventservice.ActorFor(tenantInfo)

	for _, skip := range plan.skipped {
		s.effects.RecordEvent(ctx, shipmenteventservice.BuildTenderEntrySkipped(
			tenant, ref, skip.carrierName, skip.rank, skip.reasons, actor,
		))
	}
	for _, warned := range plan.warned {
		s.effects.RecordEvent(ctx, shipmenteventservice.BuildTenderEntryWarned(
			tenant, ref, warned.carrierName, warned.rank, warned.reasons, actor,
		))
	}

	if len(plan.skipped) == 0 {
		return
	}
	names := make([]string, 0, len(plan.skipped))
	for _, skip := range plan.skipped {
		names = append(names, skip.carrierName)
	}
	s.effects.NotifyDispatch(ctx, tenantInfo, &tendereffects.DispatchNotification{
		EventType: "tender_entries_skipped",
		Priority:  tendereffects.NotificationPriorityMed,
		Title:     "Ineligible carriers skipped on tender",
		Message: fmt.Sprintf(
			"%d of %d routing guide carriers were skipped for eligibility: %s. "+
				"The remaining carriers are being offered the move",
			len(plan.skipped),
			len(plan.skipped)+len(plan.offers),
			strings.Join(names, ", "),
		),
		CorrelationID: entity.ID.String(),
		Link:          tendereffects.DispatchConsoleMoveLink(entity.ShipmentMoveID),
	})
}

func (s *Service) buildSpotOffers(
	ctx context.Context,
	req *CreateSpotTenderRequest,
	move *shipment.ShipmentMove,
	refreshIntel bool,
) (*spotOfferPlan, error) {
	carrierIDs := make([]pulid.ID, 0, len(req.Lines))
	for _, line := range req.Lines {
		carrierIDs = append(carrierIDs, line.CarrierID)
	}
	carriers, carrierByID, err := s.loadTenderCarriers(ctx, req.TenantInfo, carrierIDs)
	if err != nil {
		return nil, err
	}

	screen := &offerScreen{
		tenantInfo:  req.TenantInfo,
		carrierByID: carrierByID,
		now:         timeutils.NowUnix(),
		gates:       s.effects.IntelGates(ctx, req.TenantInfo, carrierIDs, refreshIntel),
		move:        move,
	}
	plan := &spotOfferPlan{
		offers:   make([]*tender.TenderOffer, 0, len(req.Lines)),
		warnings: make([]string, 0, len(req.Lines)),
		carriers: carrierNames(carriers),
	}
	multiErr := errortypes.NewMultiError()

	for idx := range req.Lines {
		s.screenSpotLine(ctx, screen, plan, idx, &req.Lines[idx], multiErr.WithIndex("lines", idx))
	}

	if multiErr.HasErrors() {
		return nil, multiErr
	}
	return plan, nil
}

func (s *Service) screenSpotLine(
	ctx context.Context,
	screen *offerScreen,
	plan *spotOfferPlan,
	idx int,
	line *SpotTenderLine,
	lineErr *errortypes.MultiError,
) {
	carrierEntity, ok := screen.carrierByID[line.CarrierID]
	if !ok {
		lineErr.Add(
			"carrierId",
			errortypes.ErrInvalid,
			"Carrier does not exist within your organization",
		)
		return
	}

	eligibility := carrier.EvaluateEligibility(carrier.EligibilityInput{
		Carrier: carrierEntity,
		Now:     screen.now,
		Intel:   screen.gates[carrierEntity.ID],
	})
	if eligibility.IsBlocked() {
		lineErr.Add(
			"carrierId",
			errortypes.ErrInvalid,
			"Carrier is not eligible for tendering: {0}",
			strings.Join(eligibility.Blockers, "; "),
		)
		return
	}
	if eligibility.HasWarnings() {
		for _, warning := range eligibility.Warnings {
			plan.warnings = append(plan.warnings, carrierEntity.Name+" — "+warning)
		}
	}

	rateMethod := line.RateMethod
	if rateMethod == "" {
		rateMethod = shipmentvocab.CarrierRateMethodFlat
	}
	if rateMethod == shipmentvocab.CarrierRateMethodPerMile &&
		(screen.move.Distance == nil || *screen.move.Distance <= 0) {
		lineErr.Add(
			"rateMethod",
			errortypes.ErrInvalid,
			"Per-mile lines require a move with a computed distance",
		)
		return
	}

	channel := line.Channel
	if channel == "" {
		channel = tender.ChannelEmail
	}
	ttl := line.OfferTTLSeconds
	if ttl == 0 {
		ttl = tender.DefaultOfferTTLSeconds
	}

	offer := &tender.TenderOffer{
		OrganizationID:  screen.tenantInfo.OrgID,
		BusinessUnitID:  screen.tenantInfo.BuID,
		CarrierID:       line.CarrierID,
		Rank:            intutils.SafeToInt16(idx + 1),
		RateMethod:      rateMethod,
		Rate:            line.Rate,
		OfferTTLSeconds: ttl,
		Channel:         channel,
		Status:          tender.OfferStatusPending,
	}
	if err := s.resolveOfferChannel(
		ctx, screen.tenantInfo, offer, carrierEntity, line.Email,
	); err != nil {
		lineErr.Add("channel", errortypes.ErrInvalid, err.Error())
		return
	}
	plan.offers = append(plan.offers, offer)
}

// resolveOfferChannel fills the channel-specific delivery fields: the
// recipient address for email offers, the default EDI partner for EDI offers.
func (s *Service) resolveOfferChannel(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	offer *tender.TenderOffer,
	carrierEntity *carrier.Carrier,
	emailOverride string,
) error {
	switch offer.Channel {
	case tender.ChannelEmail:
		email := emailOverride
		if email == "" && carrierEntity != nil {
			email = carrierEntity.Email
		}
		if email == "" {
			return errortypes.NewBusinessError(
				"Carrier has no email address on file. Provide one or use another channel",
			)
		}
		offer.RecipientEmail = email
	case tender.ChannelEDI:
		channel, err := s.carrierRepo.GetDefaultEDIChannel(
			ctx,
			carrierports.GetCarrierEDIChannelRequest{
				TenantInfo: tenantInfo,
				CarrierID:  offer.CarrierID,
			},
		)
		if err != nil {
			return err
		}
		if channel == nil {
			return errortypes.NewBusinessError(
				"Carrier has no default EDI channel. Link an EDI partner on the carrier or use the Email channel",
			)
		}
		partnerID := channel.EDIPartnerID
		offer.EDIPartnerID = &partnerID
	}
	return nil
}

// PersistAndStart writes the tender with its offer plan in one transaction,
// then starts the owning workflow. A workflow that cannot start cancels the
// tender immediately rather than leaving an orphaned Active row.
func (s *Service) PersistAndStart(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	entity *tender.Tender,
) (*tender.Tender, error) {
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	err := s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		_, txErr := s.repo.Create(txCtx, entity)
		if dberror.IsUniqueConstraintViolation(txErr) {
			return errortypes.NewBusinessError(
				"Shipment move already has a tender in progress",
			).WithParam("shipmentMoveId", entity.ShipmentMoveID.String())
		}
		return txErr
	})
	if err != nil {
		return nil, err
	}

	if err = s.StartRound(ctx, tenantInfo, entity); err != nil {
		return nil, err
	}

	shipmentinvalidationports.InvalidateShipments(
		ctx,
		s.invalidator,
		shipmentinvalidationports.ShipmentInvalidationByUser(
			tenantInfo,
			entity.ShipmentID,
			"tender_created",
		),
	)
	s.effects.MarkCarriersUsed(ctx, tenantInfo, entity.Offers)

	return entity, nil
}

// StartRound starts the workflow that drives the tender's current round and
// records its identifier. A tender whose workflow cannot be started is
// canceled, so it never occupies the move's live-tender slot with nobody
// driving it.
func (s *Service) StartRound(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	entity *tender.Tender,
) error {
	workflowID := tendercontract.WorkflowID(entity.ID, entity.BounceCount)
	if _, err := s.workflows.StartWorkflow(ctx, client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: temporaltype.TaskQueueDispatch.String(),
	}, tendercontract.CarrierTenderWorkflowName, tendercontract.WorkflowInput{
		TenderID:       entity.ID,
		OrganizationID: entity.OrganizationID,
		BusinessUnitID: entity.BusinessUnitID,
		Mode:           entity.Mode,
	}); err != nil {
		s.l.Error("failed to start tender workflow", zap.Error(err),
			zap.String("tenderId", entity.ID.String()),
			zap.String("workflowId", workflowID))
		s.cancelAfterStartFailure(ctx, tenantInfo, entity)
		return errortypes.NewBusinessError(
			"Tendering is unavailable right now. Retry the request",
		)
	}

	if err := s.repo.SetWorkflowID(ctx, tenantInfo, entity.ID, workflowID); err != nil {
		return err
	}
	entity.WorkflowID = workflowID
	return nil
}

func (s *Service) cancelAfterStartFailure(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	entity *tender.Tender,
) {
	now := timeutils.NowUnix()
	if _, err := s.repo.UpdateTenderStatus(ctx, &tenderports.UpdateTenderStatusRequest{
		TenantInfo:         tenantInfo,
		TenderID:           entity.ID,
		FromStatus:         []tender.Status{tender.StatusActive},
		ToStatus:           tender.StatusCanceled,
		CancellationReason: "Tender workflow could not be started",
		CanceledAt:         &now,
	}); err != nil {
		s.l.Error("failed to cancel tender after workflow start failure",
			zap.Error(err), zap.String("tenderId", entity.ID.String()))
	}

	if _, err := s.repo.BulkUpdateOfferStatus(ctx, &tenderports.BulkOfferStatusRequest{
		TenantInfo: tenantInfo,
		TenderID:   entity.ID,
		FromStatus: []tender.OfferStatus{tender.OfferStatusPending, tender.OfferStatusSent},
		ToStatus:   tender.OfferStatusSkipped,
	}); err != nil {
		s.l.Error("failed to skip offers after workflow start failure",
			zap.Error(err), zap.String("tenderId", entity.ID.String()))
	}
}

type Params struct {
	fx.In

	Logger                *zap.Logger
	DB                    kernelports.Transactor
	Repo                  tenderports.TenderRepository
	GuideRepo             tenderports.RoutingGuideRepository
	CarrierRepo           carrierports.CarrierRepository
	CarrierAssignmentRepo carrierassignmentports.CarrierAssignmentRepository
	AssignmentRepo        assignmentports.AssignmentRepository
	ShipmentRepo          shipmentports.ShipmentRepository
	HoldRepo              shipmentholdports.ShipmentHoldRepository
	Workflows             kernelports.WorkflowStarter
	Invalidator           shipmentinvalidationports.ShipmentInvalidator
	RateEngine            ratequoteports.RateEngine `optional:"true"`
	Effects               *tendereffects.Effects
}

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func New(p Params) *Service {
	return &Service{
		l:                     p.Logger.Named("service.tender"),
		db:                    p.DB,
		repo:                  p.Repo,
		guideRepo:             p.GuideRepo,
		carrierRepo:           p.CarrierRepo,
		carrierAssignmentRepo: p.CarrierAssignmentRepo,
		assignmentRepo:        p.AssignmentRepo,
		shipmentRepo:          p.ShipmentRepo,
		holdRepo:              p.HoldRepo,
		workflows:             p.Workflows,
		invalidator:           p.Invalidator,
		rateEngine:            p.RateEngine,
		effects:               p.Effects,
	}
}

// PreviewWaterfall is CreateWaterfall without the save: the guide the lane
// matches and the offers its entries become, with the carriers it skips.
func (s *Service) PreviewWaterfall(
	ctx context.Context,
	req *CreateWaterfallTenderRequest,
) (*TenderPreview, error) {
	waterfall, err := s.planWaterfall(ctx, req, false)
	if err != nil {
		return nil, err
	}

	return &TenderPreview{
		Tender:    waterfall.tender,
		Carriers:  waterfall.offers.carriers,
		Guide:     waterfall.guide,
		Screening: waterfall.offers.screeningSummary(),
	}, nil
}

// PreviewSpot is CreateSpot without the save. Insurance warnings the call
// does not override are returned as the refusal CreateSpot would make.
func (s *Service) PreviewSpot(
	ctx context.Context,
	req *CreateSpotTenderRequest,
) (*TenderPreview, error) {
	spot, err := s.planSpot(ctx, req, false)
	if err != nil {
		return nil, err
	}

	return &TenderPreview{
		Tender:       spot.tender,
		Carriers:     spot.offers.carriers,
		Warnings:     spot.offers.warnings,
		RefusalError: spot.refusal(req),
	}, nil
}

// planWaterfall is everything CreateWaterfall decides before it saves: the
// move is tenderable, the guide resolves, and its entries are screened into
// offers. refreshIntel refreshes carrier intelligence before screening,
// which writes, so only the write asks for it.
func (s *Service) planWaterfall(
	ctx context.Context,
	req *CreateWaterfallTenderRequest,
	refreshIntel bool,
) (*waterfallPlan, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	move, sh, err := s.LoadTenderableMove(ctx, req.TenantInfo, req.ShipmentMoveID)
	if err != nil {
		return nil, err
	}

	guide, err := s.resolveGuide(ctx, req, move, sh)
	if err != nil {
		return nil, err
	}

	offers, err := s.buildGuideOffers(ctx, req.TenantInfo, guide, move, refreshIntel)
	if err != nil {
		return nil, err
	}

	guideID := guide.ID

	return &waterfallPlan{
		guide:  guide,
		offers: offers,
		tender: &tender.Tender{
			OrganizationID: req.TenantInfo.OrgID,
			BusinessUnitID: req.TenantInfo.BuID,
			ShipmentID:     move.ShipmentID,
			ShipmentMoveID: move.ID,
			RoutingGuideID: &guideID,
			Mode:           tender.ModeWaterfall,
			Status:         tender.StatusActive,
			CreatedByID:    tendereffects.UserIDPtr(req.TenantInfo),
			Offers:         offers.offers,
		},
	}, nil
}

// planSpot is everything CreateSpot decides before it saves.
func (s *Service) planSpot(
	ctx context.Context,
	req *CreateSpotTenderRequest,
	refreshIntel bool,
) (*spotPlan, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	move, _, err := s.LoadTenderableMove(ctx, req.TenantInfo, req.ShipmentMoveID)
	if err != nil {
		return nil, err
	}

	offers, err := s.buildSpotOffers(ctx, req, move, refreshIntel)
	if err != nil {
		return nil, err
	}

	return &spotPlan{
		offers: offers,
		tender: &tender.Tender{
			OrganizationID: req.TenantInfo.OrgID,
			BusinessUnitID: req.TenantInfo.BuID,
			ShipmentID:     move.ShipmentID,
			ShipmentMoveID: move.ID,
			Mode:           req.Mode,
			Status:         tender.StatusActive,
			CreatedByID:    tendereffects.UserIDPtr(req.TenantInfo),
			Offers:         offers.offers,
		},
	}, nil
}
