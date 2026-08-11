package tenderservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/dispatchguard"
	"github.com/emoss08/trenova/internal/core/services/shipmenteventservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs/tenderjobs"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

type CreateWaterfallTenderRequest struct {
	TenantInfo     pagination.TenantInfo `json:"-"`
	ShipmentMoveID pulid.ID              `json:"shipmentMoveId"`
	RoutingGuideID *pulid.ID             `json:"routingGuideId"`
}

func (r *CreateWaterfallTenderRequest) Validate() *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	if r.ShipmentMoveID.IsNil() {
		multiErr.Add("shipmentMoveId", errortypes.ErrRequired, "Shipment move is required")
	}
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

type SpotTenderLine struct {
	CarrierID       pulid.ID                   `json:"carrierId"`
	RateMethod      shipment.CarrierRateMethod `json:"rateMethod"`
	Rate            decimal.Decimal            `json:"rate"`
	OfferTTLSeconds int32                      `json:"offerTtlSeconds"`
	Channel         tender.Channel             `json:"channel"`
	Email           string                     `json:"email"`
}

type CreateSpotTenderRequest struct {
	TenantInfo                pagination.TenantInfo `json:"-"`
	ShipmentMoveID            pulid.ID              `json:"shipmentMoveId"`
	Mode                      tender.Mode           `json:"mode"`
	Lines                     []SpotTenderLine      `json:"lines"`
	OverrideInsuranceWarnings bool                  `json:"overrideInsuranceWarnings"`
}

func (r *CreateSpotTenderRequest) Validate() *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	if r.ShipmentMoveID.IsNil() {
		multiErr.Add("shipmentMoveId", errortypes.ErrRequired, "Shipment move is required")
	}
	if r.Mode != tender.ModeSpotBroadcast && r.Mode != tender.ModeSpotSequential {
		multiErr.Add(
			"mode",
			errortypes.ErrInvalid,
			"Spot tenders must be broadcast or sequential",
		)
	}
	if len(r.Lines) == 0 {
		multiErr.Add("lines", errortypes.ErrRequired, "At least one carrier line is required")
	}
	seen := make(map[pulid.ID]struct{}, len(r.Lines))
	for idx, line := range r.Lines {
		lineErr := multiErr.WithIndex("lines", idx)
		if line.CarrierID.IsNil() {
			lineErr.Add("carrierId", errortypes.ErrRequired, "Carrier is required")
		}
		if _, dup := seen[line.CarrierID]; dup {
			lineErr.Add("carrierId", errortypes.ErrInvalid, "Carrier is already on this tender")
		}
		seen[line.CarrierID] = struct{}{}
		if line.RateMethod != "" && !line.RateMethod.IsValid() {
			lineErr.Add("rateMethod", errortypes.ErrInvalid, "Rate method is invalid")
		}
		if line.Rate.IsNegative() {
			lineErr.Add("rate", errortypes.ErrInvalid, "Rate cannot be negative")
		}
		if line.Channel != "" && !line.Channel.IsValid() {
			lineErr.Add("channel", errortypes.ErrInvalid, "Channel is invalid")
		}
	}
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

// CreateWaterfall starts a routing-guide-driven tender for a move. The guide
// is auto-matched from the move's lane unless one is named explicitly.
func (s *Service) CreateWaterfall(
	ctx context.Context,
	req *CreateWaterfallTenderRequest,
) (*tender.Tender, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	move, sh, err := s.loadTenderableMove(ctx, req.TenantInfo, req.ShipmentMoveID)
	if err != nil {
		return nil, err
	}

	guide, err := s.resolveGuide(ctx, req, move, sh)
	if err != nil {
		return nil, err
	}

	plan, err := s.buildGuideOffers(ctx, req.TenantInfo, guide, move)
	if err != nil {
		return nil, err
	}

	guideID := guide.ID
	entity := &tender.Tender{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		ShipmentID:     move.ShipmentID,
		ShipmentMoveID: move.ID,
		RoutingGuideID: &guideID,
		Mode:           tender.ModeWaterfall,
		Status:         tender.StatusActive,
		CreatedByID:    userIDPtr(req.TenantInfo),
		Offers:         plan.offers,
	}

	created, err := s.persistAndStart(ctx, req.TenantInfo, entity)
	if err != nil {
		return nil, err
	}

	s.auditGuideScreening(ctx, req.TenantInfo, created, plan)

	return created, nil
}

// CreateSpot starts an ad-hoc tender for a move with a dispatcher-picked
// carrier list, either broadcast simultaneously or worked in sequence.
func (s *Service) CreateSpot(
	ctx context.Context,
	req *CreateSpotTenderRequest,
) (*tender.Tender, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	move, _, err := s.loadTenderableMove(ctx, req.TenantInfo, req.ShipmentMoveID)
	if err != nil {
		return nil, err
	}

	offers, err := s.buildSpotOffers(ctx, req, move)
	if err != nil {
		return nil, err
	}

	entity := &tender.Tender{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		ShipmentID:     move.ShipmentID,
		ShipmentMoveID: move.ID,
		Mode:           req.Mode,
		Status:         tender.StatusActive,
		CreatedByID:    userIDPtr(req.TenantInfo),
		Offers:         offers,
	}

	return s.persistAndStart(ctx, req.TenantInfo, entity)
}

// loadTenderableMove enforces every precondition shared by both tender modes:
// the move must be assignable, hold-free, and not already covered by a driver,
// a carrier, or another live tender.
func (s *Service) loadTenderableMove(
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

	live, err := s.repo.GetLiveByMoveID(ctx, repositories.GetLiveTenderByMoveRequest{
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

	sh, err := s.shipmentRepo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID: move.ShipmentID,
		TenantInfo: pagination.TenantInfo{
			OrgID: tenantInfo.OrgID,
			BuID:  tenantInfo.BuID,
		},
		ShipmentOptions: repositories.ShipmentOptions{
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
		guide, err := s.guideRepo.GetByID(ctx, repositories.GetRoutingGuideByIDRequest{
			ID:         *req.RoutingGuideID,
			TenantInfo: req.TenantInfo,
			RoutingGuideFilterOptions: repositories.RoutingGuideFilterOptions{
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
		).WithParam("shipmentMoveId", move.ID.String())
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
) *repositories.MatchLaneRequest {
	req := &repositories.MatchLaneRequest{TenantInfo: tenantInfo}

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

// guideEntryScreening captures one routing guide entry's eligibility verdict
// so the audit trail can be written after the tender row exists.
type guideEntryScreening struct {
	carrierName string
	rank        int16
	reasons     []string
}

// guideOfferPlan is the screened offer plan for a waterfall tender: the offers
// that survived the eligibility gate plus the entries skipped or warned on.
type guideOfferPlan struct {
	offers  []*tender.TenderOffer
	skipped []guideEntryScreening
	warned  []guideEntryScreening
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
) (*guideOfferPlan, error) {
	carrierIDs := make([]pulid.ID, 0, len(guide.Entries))
	for _, entry := range guide.Entries {
		if entry == nil {
			continue
		}
		carrierIDs = append(carrierIDs, entry.CarrierID)
	}
	carriers, err := s.carrierRepo.GetByIDs(ctx, repositories.GetCarriersByIDsRequest{
		TenantInfo: tenantInfo,
		CarrierIDs: carrierIDs,
		CarrierFilterOptions: repositories.CarrierFilterOptions{
			IncludeInsurancePolicies: true,
		},
	})
	if err != nil {
		return nil, err
	}
	carrierByID := make(map[pulid.ID]*carrier.Carrier, len(carriers))
	for _, c := range carriers {
		carrierByID[c.ID] = c
	}

	now := timeutils.NowUnix()
	plan := &guideOfferPlan{
		offers: make([]*tender.TenderOffer, 0, len(guide.Entries)),
	}
	multiErr := errortypes.NewMultiError()

	for idx, entry := range guide.Entries {
		if entry == nil {
			continue
		}
		entryErr := multiErr.WithIndex("entries", idx)

		carrierEntity, ok := carrierByID[entry.CarrierID]
		if !ok {
			plan.skipped = append(plan.skipped, guideEntryScreening{
				carrierName: guideEntryCarrierName(entry),
				rank:        entry.Rank,
				reasons:     []string{"Carrier does not exist within your organization"},
			})
			continue
		}

		eligibility := carrier.EvaluateCarrierEligibility(carrierEntity, now)
		if eligibility.IsBlocked() {
			plan.skipped = append(plan.skipped, guideEntryScreening{
				carrierName: carrierEntity.Name,
				rank:        entry.Rank,
				reasons:     eligibility.Blockers,
			})
			continue
		}

		if entry.RateMethod == shipment.CarrierRateMethodPerMile &&
			(move.Distance == nil || *move.Distance <= 0) {
			entryErr.Add(
				"rateMethod",
				errortypes.ErrInvalid,
				"Per-mile entries require a move with a computed distance",
			)
			continue
		}

		offer := &tender.TenderOffer{
			OrganizationID:  tenantInfo.OrgID,
			BusinessUnitID:  tenantInfo.BuID,
			CarrierID:       entry.CarrierID,
			Rank:            entry.Rank,
			RateMethod:      entry.RateMethod,
			Rate:            entry.Rate,
			OfferTTLSeconds: entry.OfferTTLSeconds,
			Channel:         entry.Channel,
			Status:          tender.OfferStatusPending,
		}
		if err = s.resolveOfferChannel(ctx, tenantInfo, offer, carrierEntity, ""); err != nil {
			entryErr.Add("channel", errortypes.ErrInvalid, err.Error())
			continue
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

func guideEntryCarrierName(entry *tender.RoutingGuideEntry) string {
	if entry.Carrier != nil && entry.Carrier.Name != "" {
		return entry.Carrier.Name
	}
	return entry.CarrierID.String()
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
		s.recordEvent(ctx, shipmenteventservice.BuildTenderEntrySkipped(
			tenant, ref, skip.carrierName, skip.rank, skip.reasons, actor,
		))
	}
	for _, warned := range plan.warned {
		s.recordEvent(ctx, shipmenteventservice.BuildTenderEntryWarned(
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
	s.notifyDispatch(ctx, tenantInfo, &dispatchNotification{
		EventType: "tender_entries_skipped",
		Priority:  notificationPriorityMed,
		Title:     "Ineligible carriers skipped on tender",
		Message: fmt.Sprintf(
			"%d of %d routing guide carriers were skipped for eligibility: %s. "+
				"The remaining carriers are being offered the move",
			len(plan.skipped),
			len(plan.skipped)+len(plan.offers),
			strings.Join(names, ", "),
		),
		CorrelationID: entity.ID.String(),
	})
}

func (s *Service) buildSpotOffers(
	ctx context.Context,
	req *CreateSpotTenderRequest,
	move *shipment.ShipmentMove,
) ([]*tender.TenderOffer, error) {
	carrierIDs := make([]pulid.ID, 0, len(req.Lines))
	for _, line := range req.Lines {
		carrierIDs = append(carrierIDs, line.CarrierID)
	}
	carriers, err := s.carrierRepo.GetByIDs(ctx, repositories.GetCarriersByIDsRequest{
		TenantInfo: req.TenantInfo,
		CarrierIDs: carrierIDs,
		CarrierFilterOptions: repositories.CarrierFilterOptions{
			IncludeInsurancePolicies: true,
		},
	})
	if err != nil {
		return nil, err
	}
	carrierByID := make(map[pulid.ID]*carrier.Carrier, len(carriers))
	for _, c := range carriers {
		carrierByID[c.ID] = c
	}

	now := timeutils.NowUnix()
	offers := make([]*tender.TenderOffer, 0, len(req.Lines))
	warnings := make([]string, 0, len(req.Lines))
	multiErr := errortypes.NewMultiError()

	for idx, line := range req.Lines {
		lineErr := multiErr.WithIndex("lines", idx)

		carrierEntity, ok := carrierByID[line.CarrierID]
		if !ok {
			lineErr.Add(
				"carrierId",
				errortypes.ErrInvalid,
				"Carrier does not exist within your organization",
			)
			continue
		}

		eligibility := carrier.EvaluateCarrierEligibility(carrierEntity, now)
		if eligibility.IsBlocked() {
			lineErr.Add(
				"carrierId",
				errortypes.ErrInvalid,
				"Carrier is not eligible for tendering: "+
					strings.Join(eligibility.Blockers, "; "),
			)
			continue
		}
		if eligibility.HasWarnings() {
			for _, warning := range eligibility.Warnings {
				warnings = append(warnings, carrierEntity.Name+" — "+warning)
			}
		}

		rateMethod := line.RateMethod
		if rateMethod == "" {
			rateMethod = shipment.CarrierRateMethodFlat
		}
		if rateMethod == shipment.CarrierRateMethodPerMile &&
			(move.Distance == nil || *move.Distance <= 0) {
			lineErr.Add(
				"rateMethod",
				errortypes.ErrInvalid,
				"Per-mile lines require a move with a computed distance",
			)
			continue
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
			OrganizationID:  req.TenantInfo.OrgID,
			BusinessUnitID:  req.TenantInfo.BuID,
			CarrierID:       line.CarrierID,
			Rank:            int16(idx + 1), //nolint:gosec // bounded by request size
			RateMethod:      rateMethod,
			Rate:            line.Rate,
			OfferTTLSeconds: ttl,
			Channel:         channel,
			Status:          tender.OfferStatusPending,
		}
		if err = s.resolveOfferChannel(
			ctx, req.TenantInfo, offer, carrierEntity, line.Email,
		); err != nil {
			lineErr.Add("channel", errortypes.ErrInvalid, err.Error())
			continue
		}
		offers = append(offers, offer)
	}

	if multiErr.HasErrors() {
		return nil, multiErr
	}
	if len(warnings) > 0 && !req.OverrideInsuranceWarnings {
		return nil, errortypes.NewBusinessError(
			"Carrier has insurance warnings: "+strings.Join(warnings, "; ")+
				". Confirm the override to proceed",
		).WithParam("overridable", "true")
	}
	return offers, nil
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
			repositories.GetCarrierEDIChannelRequest{
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

// persistAndStart writes the tender with its offer plan in one transaction,
// then starts the owning workflow. A workflow that cannot start cancels the
// tender immediately rather than leaving an orphaned Active row.
func (s *Service) persistAndStart(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	entity *tender.Tender,
) (*tender.Tender, error) {
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
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

	workflowID := tenderjobs.WorkflowID(entity.ID)
	_, err = s.workflows.StartWorkflow(ctx, client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: temporaltype.TaskQueueDispatch.String(),
	}, tenderjobs.CarrierTenderWorkflowName, tenderjobs.WorkflowInput{
		TenderID:       entity.ID,
		OrganizationID: entity.OrganizationID,
		BusinessUnitID: entity.BusinessUnitID,
		Mode:           entity.Mode,
	})
	if err != nil {
		s.l.Error("failed to start tender workflow", zap.Error(err),
			zap.String("tenderId", entity.ID.String()))
		s.cancelAfterStartFailure(ctx, tenantInfo, entity)
		return nil, errortypes.NewBusinessError(
			"Tendering is unavailable right now. Retry the request",
		)
	}

	if err = s.repo.SetWorkflowID(ctx, tenantInfo, entity.ID, workflowID); err != nil {
		return nil, err
	}
	entity.WorkflowID = workflowID

	s.publishInvalidation(ctx, tenantInfo, entity.ShipmentID, "tender_created")

	return entity, nil
}

func (s *Service) cancelAfterStartFailure(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	entity *tender.Tender,
) {
	now := timeutils.NowUnix()
	if _, err := s.repo.UpdateTenderStatus(ctx, &repositories.UpdateTenderStatusRequest{
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

	if _, err := s.repo.BulkUpdateOfferStatus(ctx, &repositories.BulkOfferStatusRequest{
		TenantInfo: tenantInfo,
		TenderID:   entity.ID,
		FromStatus: []tender.OfferStatus{tender.OfferStatusPending, tender.OfferStatusSent},
		ToStatus:   tender.OfferStatusSkipped,
	}); err != nil {
		s.l.Error("failed to skip offers after workflow start failure",
			zap.Error(err), zap.String("tenderId", entity.ID.String()))
	}
}

func userIDPtr(tenantInfo pagination.TenantInfo) *pulid.ID {
	if tenantInfo.UserID.IsNil() {
		return nil
	}
	userID := tenantInfo.UserID
	return &userID
}
