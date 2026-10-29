package tenderbounce

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmenteventservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/tendercreate"
	"github.com/emoss08/trenova/internal/modules/operations/app/tendereffects"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/operations/ports/carrierassignmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentinvalidationports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/tenderports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// planBounce is every check Bounce makes before it acts, so a preview refuses
// exactly what the write would.
func (s *Service) planBounce(ctx context.Context, req *BounceTenderRequest) (*bouncePlan, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}
	if req.TenderID.IsNil() {
		offer, err := s.repo.GetOfferByID(ctx, tenderports.GetTenderOfferByIDRequest{
			TenantInfo: req.TenantInfo,
			OfferID:    req.OfferID,
		})
		if err != nil {
			return nil, err
		}
		req.TenderID = offer.TenderID
	}

	entity, err := s.repo.GetByID(ctx, tenderports.GetTenderByIDRequest{
		TenantInfo:    req.TenantInfo,
		TenderID:      req.TenderID,
		IncludeOffers: true,
	})
	if err != nil {
		return nil, err
	}
	revoked, ok := entity.CanBounce()
	if !ok || (!req.OfferID.IsNil() && revoked.ID != req.OfferID) {
		return nil, errortypes.NewBusinessError(
			"Only an offer a carrier accepted can be backed out of",
		).WithParam("status", entity.Status.String())
	}
	if entity.BounceCount >= tender.MaxBounces {
		return nil, errortypes.NewBusinessError(
			"This tender has been bounced too many times; start a new tender for the move",
		)
	}

	live, err := s.repo.GetLiveByMoveID(ctx, tenderports.GetLiveTenderByMoveRequest{
		TenantInfo: req.TenantInfo,
		MoveID:     entity.ShipmentMoveID,
	})
	if err != nil && !errortypes.IsNotFoundError(err) {
		return nil, err
	}
	if live != nil && live.ID != entity.ID {
		return nil, errortypes.NewBusinessError(
			"The move already has another tender in progress",
		)
	}

	assignment, err := s.carrierAssignmentRepo.GetActiveByMoveID(
		ctx, req.TenantInfo, entity.ShipmentMoveID,
	)
	if err != nil {
		return nil, err
	}
	if assignment != nil && assignment.CarrierID != revoked.CarrierID {
		return nil, errortypes.NewBusinessError(
			"The move is covered by a different carrier than the one that accepted the tender",
		)
	}
	if assignment != nil && s.assigner == nil {
		return nil, errortypes.NewBusinessError(
			"Carrier assignment is unavailable right now. Retry the request",
		)
	}

	return &bouncePlan{
		entity:     entity,
		revoked:    revoked,
		assignment: assignment,
		reoffers:   entity.ReofferPlan(revoked),
	}, nil
}

// PreviewBounce is what Bounce would do, from the same checks, without
// writing.
func (s *Service) PreviewBounce(
	ctx context.Context,
	req *BounceTenderRequest,
) (*BouncePreview, error) {
	plan, err := s.planBounce(ctx, req)
	if err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	after := bouncedTender(plan, req, now)
	return &BouncePreview{
		Before:      plan.entity,
		After:       after,
		Revoked:     after.FindOffer(plan.revoked.ID),
		Reoffers:    plan.reoffers,
		Unassigns:   plan.assignment != nil,
		WillExhaust: len(plan.reoffers) == 0,
	}, nil
}

// Bounce walks back a carrier's acceptance and sends the tender on to the
// carriers the acceptance cut short, in rank order. The carrier comes off the
// move first, voiding its rate confirmation; a move already under way cannot
// be bounced, because the assignment can no longer be canceled. With nobody
// left to ask, the new round exhausts at once and the organization's
// escalation policy takes over.
//
// It is safe to retry: a retry that finds the carrier already off the move
// carries on from there.
func (s *Service) Bounce(ctx context.Context, req *BounceTenderRequest) (*tender.Tender, error) {
	plan, err := s.planBounce(ctx, req)
	if err != nil {
		return nil, err
	}
	if req.Source == "" {
		req.Source = tender.ResponseSourceManual
	}
	reason := strings.TrimSpace(req.Reason)

	if plan.assignment != nil {
		if err = s.assigner.Cancel(ctx, &carrierassignmentports.CancelCarrierAssignmentRequest{
			TenantInfo:     req.TenantInfo,
			ShipmentMoveID: plan.entity.ShipmentMoveID,
			Reason:         "Carrier backed out of the accepted tender: " + reason,
		}); err != nil {
			return nil, err
		}
	}

	now := timeutils.NowUnix()
	reopened, err := s.repo.Reopen(ctx, &tenderports.ReopenTenderRequest{
		TenantInfo:     req.TenantInfo,
		TenderID:       plan.entity.ID,
		RevokedOfferID: plan.revoked.ID,
		Reason:         reason,
		RevokedByID:    tendereffects.UserIDPtr(req.TenantInfo),
		At:             now,
		Reoffers:       plan.reoffers,
	})
	if err != nil {
		return nil, err
	}
	if !reopened {
		return nil, errortypes.NewBusinessError("The tender has already moved on")
	}

	entity, err := s.repo.GetByID(ctx, tenderports.GetTenderByIDRequest{
		TenantInfo:    req.TenantInfo,
		TenderID:      plan.entity.ID,
		IncludeOffers: true,
	})
	if err != nil {
		return nil, err
	}
	if err = s.repo.RevokeTokensForOffer(ctx, req.TenantInfo, plan.revoked.ID, now); err != nil {
		s.l.Warn("failed to revoke tokens for bounced tender offer", zap.Error(err))
	}
	if err = s.creator.StartRound(ctx, req.TenantInfo, entity); err != nil {
		return nil, err
	}

	s.announceBounce(ctx, req, plan, entity)
	return entity, nil
}

func (s *Service) announceBounce(
	ctx context.Context,
	req *BounceTenderRequest,
	plan *bouncePlan,
	entity *tender.Tender,
) {
	offer := plan.revoked
	offer.Tender = entity
	carrierName := tendereffects.CarrierNameOf(offer)
	reason := strings.TrimSpace(req.Reason)

	s.effects.RecordEvent(ctx, shipmenteventservice.BuildTenderBounced(
		shipmenteventservice.TenantRefFor(req.TenantInfo),
		tendereffects.TenderRefForOffer(offer),
		shipmenteventservice.TenderBounce{
			CarrierName: carrierName,
			Reason:      reason,
			Source:      req.Source,
			Reoffers:    len(plan.reoffers),
			Round:       entity.BounceCount,
		},
		shipmenteventservice.ActorFor(req.TenantInfo),
	))

	message := fmt.Sprintf(
		"%s backed out of the tender after accepting it (%s). The tender goes on to %d more %s",
		carrierName, reason, len(plan.reoffers), tendercreate.PluralCarriers(len(plan.reoffers)),
	)
	if len(plan.reoffers) == 0 {
		message = fmt.Sprintf(
			"%s backed out of the tender after accepting it (%s). Nobody is left to ask, so the move needs coverage",
			carrierName,
			reason,
		)
	}
	s.effects.NotifyDispatch(ctx, req.TenantInfo, &tendereffects.DispatchNotification{
		EventType:     "tender_bounced",
		Priority:      tendereffects.NotificationPriorityHigh,
		Title:         "Carrier backed out of a tender",
		Message:       message,
		CorrelationID: entity.ID.String(),
		Link:          tendereffects.DispatchConsoleMoveLink(entity.ShipmentMoveID),
	})
	s.effects.LogAudit(&tendereffects.AuditParams{
		TenantInfo: req.TenantInfo,
		TenderID:   entity.ID,
		Operation:  permission.OpUpdate,
		UserID:     req.TenantInfo.UserID,
		Comment: fmt.Sprintf(
			"Tender bounced by %s via %s: %s", carrierName, req.Source.String(), reason,
		),
		Previous: plan.entity,
		Current:  entity,
	})
	shipmentinvalidationports.InvalidateShipments(
		ctx,
		s.invalidator,
		shipmentinvalidationports.ShipmentInvalidationByUser(
			req.TenantInfo,
			entity.ShipmentID,
			"tender_bounced",
		),
	)
}

// BounceOnCarrierDecline treats a carrier's EDI decline of an offer it had
// already accepted as the carrier backing out: a 990 is the carrier's own word,
// so the tender is bounced on it. Every other answer to an accepted offer is
// late and is filed as such by the caller.
func (s *Service) BounceOnCarrierDecline(
	ctx context.Context,
	req *tenderports.TenderResponseRequest,
) (bool, error) {
	if req.Source != tender.ResponseSourceEDI || req.Action != tender.ResponseActionDecline ||
		req.OfferID.IsNil() {
		return false, nil
	}
	offer, err := s.repo.GetOfferByID(ctx, tenderports.GetTenderOfferByIDRequest{
		TenantInfo:    req.TenantInfo,
		OfferID:       req.OfferID,
		IncludeTender: true,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return false, nil
		}
		return false, err
	}
	if offer.Tender == nil || offer.Status != tender.OfferStatusAccepted ||
		offer.Tender.AcceptedOfferID == nil || *offer.Tender.AcceptedOfferID != offer.ID {
		return false, nil
	}

	reason := strings.TrimSpace(req.DeclineReason)
	if reason == "" {
		reason = "The carrier declined by EDI after accepting"
	}
	reason = stringutils.TruncateRunes(reason, maxBounceReasonLength)
	_, err = s.Bounce(ctx, &BounceTenderRequest{
		TenantInfo: req.TenantInfo,
		TenderID:   offer.TenderID,
		Reason:     reason,
		Source:     tender.ResponseSourceEDI,
	})
	return true, err
}

type Service struct {
	l                     *zap.Logger
	repo                  tenderports.TenderRepository
	carrierAssignmentRepo carrierassignmentports.CarrierAssignmentRepository
	invalidator           shipmentinvalidationports.ShipmentInvalidator
	assigner              tenderports.CarrierMoveAssigner
	creator               *tendercreate.Service
	effects               *tendereffects.Effects
}

type Params struct {
	fx.In

	Logger                *zap.Logger
	Repo                  tenderports.TenderRepository
	CarrierAssignmentRepo carrierassignmentports.CarrierAssignmentRepository
	Invalidator           shipmentinvalidationports.ShipmentInvalidator
	Creator               *tendercreate.Service
	Effects               *tendereffects.Effects
}

func New(p Params) *Service {
	return &Service{
		l:                     p.Logger.Named("service.tender"),
		repo:                  p.Repo,
		carrierAssignmentRepo: p.CarrierAssignmentRepo,
		invalidator:           p.Invalidator,
		creator:               p.Creator,
		effects:               p.Effects,
	}
}

func (s *Service) SetCarrierMoveAssigner(value tenderports.CarrierMoveAssigner) {
	s.assigner = value
}
