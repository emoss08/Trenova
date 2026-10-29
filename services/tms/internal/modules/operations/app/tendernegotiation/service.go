package tendernegotiation

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmenteventservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/tendereffects"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/operations/jobs/tenderjobs/tendercontract"
	"github.com/emoss08/trenova/internal/modules/operations/ports/assignmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/tenderports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// RecordCounter lands a carrier's counter-offer. The offer moves to
// Countered and holds until the decision deadline; the verdict against the
// margin floor says whether the policy takes it at once, or who has to decide
// it. Losing the CAS means the offer moved on first, and the counter is filed
// as late.
func (s *Service) RecordCounter(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	req *tenderports.RecordCounterRequest,
) (*tenderports.TenderCounterResult, error) {
	offer, err := s.repo.GetOfferByID(ctx, tenderports.GetTenderOfferByIDRequest{
		TenantInfo:    tenantInfo,
		OfferID:       req.OfferID,
		IncludeTender: true,
	})
	if err != nil {
		return nil, err
	}
	if offer.Tender == nil {
		return nil, errortypes.NewBusinessError("Offer is not attached to a tender")
	}

	policy := s.effects.Policy(ctx, tenantInfo)
	verdict := s.evaluateCounter(ctx, tenantInfo, offer, req.Rate, &policy)
	now := timeutils.NowUnix()
	deadline := now + policy.CounterDecisionSeconds
	note := strings.TrimSpace(req.Note)

	moved, err := s.repo.UpdateOfferStatus(ctx, &tenderports.UpdateOfferStatusRequest{
		TenantInfo:     tenantInfo,
		OfferID:        offer.ID,
		FromStatus:     []tender.OfferStatus{tender.OfferStatusSent},
		ToStatus:       tender.OfferStatusCountered,
		RespondedAt:    &now,
		ResponseSource: req.Source,
		ExpiresAt:      &deadline,
		Counter: &tenderports.OfferCounter{
			Rate:          req.Rate,
			Note:          note,
			At:            now,
			MarginPercent: verdict.MarginPercent,
			NeedsApproval: verdict.NeedsApproval,
		},
	})
	if err != nil {
		return nil, err
	}
	if !moved {
		s.effects.RecordLateOfferResponse(
			ctx,
			tenantInfo,
			offer,
			tender.ResponseActionCounter,
			req.Source,
		)
		return &tenderports.TenderCounterResult{Ignored: true}, nil
	}

	if err = s.repo.RevokeTokensForOffer(ctx, tenantInfo, offer.ID, now); err != nil {
		s.l.Warn("failed to revoke tokens for countered tender offer", zap.Error(err))
	}

	s.announceCounter(ctx, tenantInfo, &counterAnnouncement{
		offer:    offer,
		rate:     req.Rate,
		note:     note,
		source:   req.Source,
		verdict:  verdict,
		deadline: deadline,
	})

	return &tenderports.TenderCounterResult{
		AutoAccept:       verdict.AutoAccept,
		DecisionDeadline: deadline,
	}, nil
}

// evaluateCounter judges a counter against the floor on the same terms an
// assignment's margin is judged: what the shipment sells for against what this
// carrier would be paid on this move. A sell price or distance that cannot be
// read leaves the margin unknown, and the counter waits for a person.
func (s *Service) evaluateCounter(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	offer *tender.TenderOffer,
	rate decimal.Decimal,
	policy *dispatchcontrol.TenderPolicy,
) tender.CounterVerdict {
	terms := tender.CounterTerms{
		Floor:      policy.CounterMarginFloor,
		AutoAccept: policy.AutoAcceptCounters,
	}

	move, err := s.assignmentRepo.GetMoveByID(ctx, tenantInfo, offer.Tender.ShipmentMoveID)
	if err != nil {
		s.l.Warn("failed to read the move to judge a counter-offer", zap.Error(err))
		return tender.EvaluateCounter(terms)
	}
	sh, err := s.shipmentRepo.GetByID(ctx, &shipmentports.GetShipmentByIDRequest{
		ID:         offer.Tender.ShipmentID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		s.l.Warn("failed to read the shipment to judge a counter-offer", zap.Error(err))
		return tender.EvaluateCounter(terms)
	}

	if sh.TotalChargeAmount.Valid {
		terms.Sell = sh.TotalChargeAmount.Decimal
	}
	terms.Buy = shipment.CarrierBaseAmount(offer.RateMethod, rate, move.Distance)
	return tender.EvaluateCounter(terms)
}

func (s *Service) announceCounter(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	a *counterAnnouncement,
) {
	carrierName := tendereffects.CarrierNameOf(a.offer)

	s.effects.RecordEvent(ctx, shipmenteventservice.BuildTenderCountered(
		shipmenteventservice.TenantRefFor(tenantInfo),
		tendereffects.TenderRefForOffer(a.offer),
		&shipmenteventservice.TenderCounter{
			CarrierName:   carrierName,
			OfferedRate:   a.offer.Rate.StringFixed(2),
			CounterRate:   a.rate.StringFixed(2),
			NeedsApproval: a.verdict.NeedsApproval,
			AutoAccepted:  a.verdict.AutoAccept,
			Source:        a.source,
		},
		shipmenteventservice.ActorFor(tenantInfo),
	))

	if !a.verdict.AutoAccept {
		priority := tendereffects.NotificationPriorityMed
		message := fmt.Sprintf(
			"%s countered at %s against %s offered. Take or refuse it by %s, or it lapses and the tender moves on",
			carrierName,
			a.rate.StringFixed(2),
			a.offer.Rate.StringFixed(2),
			timeutils.FormatInstantUTC(a.deadline),
		)
		if a.verdict.NeedsApproval {
			priority = tendereffects.NotificationPriorityHigh
			message = fmt.Sprintf(
				"%s countered at %s against %s offered, and %s. Approving it needs the tender approve permission. Decide by %s, or it lapses and the tender moves on",
				carrierName,
				a.rate.StringFixed(2),
				a.offer.Rate.StringFixed(2),
				a.verdict.Explanation,
				timeutils.FormatInstantUTC(a.deadline),
			)
		}
		s.effects.NotifyDispatch(ctx, tenantInfo, &tendereffects.DispatchNotification{
			EventType:     "tender_countered",
			Priority:      priority,
			Title:         "Carrier counter-offer to decide",
			Message:       message,
			CorrelationID: a.offer.TenderID.String(),
			Link:          tendereffects.DispatchConsoleOfferLink(a.offer),
		})
	}

	comment := fmt.Sprintf(
		"Counter-offer from %s via %s at %s", carrierName, a.source.String(), a.rate.StringFixed(2),
	)
	if a.note != "" {
		comment += ": " + a.note
	}
	s.effects.LogAudit(&tendereffects.AuditParams{
		TenantInfo: tenantInfo,
		TenderID:   a.offer.TenderID,
		Operation:  permission.OpUpdate,
		UserID:     tenantInfo.UserID,
		Comment:    comment,
		Current:    a.offer.Tender,
	})
	s.effects.PublishShipmentInvalidation(ctx, tenantInfo, a.offer, "tender_countered")
}

// planCounterDecision is every check DecideCounter makes before it signals:
// the offer holds a counter that has not lapsed on a live tender, and taking
// a counter below the floor is left to somebody who may approve it.
func (s *Service) planCounterDecision(
	ctx context.Context,
	req *DecideCounterRequest,
) (*tender.TenderOffer, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	offer, err := s.repo.GetOfferByID(ctx, tenderports.GetTenderOfferByIDRequest{
		TenantInfo:    req.TenantInfo,
		OfferID:       req.OfferID,
		IncludeTender: true,
	})
	if err != nil {
		return nil, err
	}
	now := timeutils.NowUnix()
	if offer.Tender == nil || offer.Tender.Status != tender.StatusActive ||
		offer.Status != tender.OfferStatusCountered ||
		(offer.ExpiresAt != nil && *offer.ExpiresAt <= now) {
		return nil, errortypes.NewBusinessError(
			"This counter-offer is no longer waiting on a decision",
		)
	}

	if req.Approve && offer.CounterNeedsApproval {
		allowed, cerr := s.mayApproveCounters(ctx, req.actor())
		if cerr != nil {
			return nil, cerr
		}
		if !allowed {
			return nil, errortypes.NewAuthorizationError(
				"This counter-offer is below the margin floor; taking it needs the approve permission on tenders",
			)
		}
	}

	return offer, nil
}

func (s *Service) mayApproveCounters(
	ctx context.Context,
	actor *permissionports.RequestActor,
) (bool, error) {
	if s.permissions == nil || actor == nil || actor.PrincipalID.IsNil() {
		return false, nil
	}
	result, err := s.permissions.Check(ctx, &permissionports.PermissionCheckRequest{
		PrincipalType:  actor.PrincipalType,
		PrincipalID:    actor.PrincipalID,
		UserID:         actor.UserID,
		APIKeyID:       actor.APIKeyID,
		BusinessUnitID: actor.BusinessUnitID,
		OrganizationID: actor.OrganizationID,
		Resource:       permission.ResourceTender.String(),
		Operation:      permission.OpApprove,
	})
	if err != nil {
		return false, err
	}
	return result != nil && result.Allowed, nil
}

// DecideCounter takes or refuses a counter-offer by signalling the tender's
// workflow, which owns the transition: taking it accepts the offer at the
// counter rate, refusing it closes the offer and the tender moves on.
func (s *Service) DecideCounter(ctx context.Context, req *DecideCounterRequest) error {
	offer, err := s.planCounterDecision(ctx, req)
	if err != nil {
		return err
	}

	action := tender.ResponseActionDecline
	if req.Approve {
		action = tender.ResponseActionAccept
	}
	workflowID := tendercontract.WorkflowID(offer.TenderID, offer.Tender.BounceCount)
	if err = s.workflows.SignalWorkflow(
		ctx,
		workflowID,
		"",
		tendercontract.TenderSignalName,
		tendercontract.Signal{
			Kind:          tendercontract.SignalKindResponse,
			OfferID:       offer.ID,
			Action:        action,
			Source:        tender.ResponseSourceManual,
			DeclineReason: strings.TrimSpace(req.Reason),
			ActorUserID:   req.TenantInfo.UserID,
		},
	); err != nil {
		s.l.Error("failed to signal tender workflow with a counter decision",
			zap.Error(err),
			zap.String("offerId", offer.ID.String()),
			zap.String("workflowId", workflowID))
		return errortypes.NewBusinessError(
			"The decision could not be recorded right now. Retry in a moment",
		)
	}
	return nil
}

// PreviewDecideCounter is what DecideCounter would lead to, from the same
// checks, without signalling the workflow.
func (s *Service) PreviewDecideCounter(
	ctx context.Context,
	req *DecideCounterRequest,
) (*ResponsePreview, error) {
	offer, err := s.planCounterDecision(ctx, req)
	if err != nil {
		return nil, err
	}
	entity, err := s.repo.GetByID(ctx, tenderports.GetTenderByIDRequest{
		TenantInfo:    req.TenantInfo,
		TenderID:      offer.TenderID,
		IncludeOffers: true,
	})
	if err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	out := &ResponsePreview{Tender: entity, Offer: offer}
	if req.Approve {
		out.TenderAfter = AcceptedTender(entity, offer.ID, offer.ResponseSource, now)
		return out, nil
	}
	out.TenderAfter = tendereffects.CopyTender(entity, func(o *tender.TenderOffer) {
		if o.ID == offer.ID {
			o.Status = tender.OfferStatusCounterRejected
			o.CounterDecidedAt = &now
			o.DeclineReason = strings.TrimSpace(req.Reason)
		}
	})
	out.NextOffer = NextOfferAfterDecline(out.TenderAfter)
	if out.NextOffer == nil && !HasOutstandingOffer(out.TenderAfter) {
		out.Exhausted = true
		out.TenderAfter.Status = tender.StatusExhausted
		out.TenderAfter.ExhaustedAt = &now
	}
	return out, nil
}

// RefuseCounter closes an offer whose counter somebody refused, so the tender
// moves on to its next carrier.
func (s *Service) RefuseCounter(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	offer *tender.TenderOffer,
	reason string,
) error {
	now := timeutils.NowUnix()
	moved, err := s.repo.UpdateOfferStatus(ctx, &tenderports.UpdateOfferStatusRequest{
		TenantInfo:     tenantInfo,
		OfferID:        offer.ID,
		FromStatus:     []tender.OfferStatus{tender.OfferStatusCountered},
		ToStatus:       tender.OfferStatusCounterRejected,
		DeclineReason:  reason,
		CounterDecided: DecisionStamp(tenantInfo, now),
	})
	if err != nil || !moved {
		return err
	}
	s.closeCounter(ctx, tenantInfo, offer, reason, "Counter-offer from %s refused")
	return nil
}

// LapseCounter closes an offer whose counter nobody decided before its
// deadline, so the tender moves on to its next carrier.
func (s *Service) LapseCounter(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	offer *tender.TenderOffer,
) error {
	moved, err := s.repo.UpdateOfferStatus(ctx, &tenderports.UpdateOfferStatusRequest{
		TenantInfo: tenantInfo,
		OfferID:    offer.ID,
		FromStatus: []tender.OfferStatus{tender.OfferStatusCountered},
		ToStatus:   tender.OfferStatusExpired,
	})
	if err != nil || !moved {
		return err
	}
	s.closeCounter(ctx, tenantInfo, offer,
		"Nobody decided on the counter-offer before it lapsed",
		"Counter-offer from %s lapsed undecided",
	)
	return nil
}

func (s *Service) closeCounter(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	offer *tender.TenderOffer,
	reason string,
	auditFormat string,
) {
	carrierName := tendereffects.CarrierNameOf(offer)
	s.effects.RecordEvent(ctx, shipmenteventservice.BuildTenderCounterDecided(
		shipmenteventservice.TenantRefFor(tenantInfo),
		tendereffects.TenderRefForOffer(offer),
		carrierName,
		offer.CounterRate.Decimal.StringFixed(2),
		reason,
		false,
		shipmenteventservice.ActorFor(tenantInfo),
	))
	comment := fmt.Sprintf(auditFormat, carrierName)
	if reason != "" {
		comment += ": " + reason
	}
	s.effects.LogAudit(&tendereffects.AuditParams{
		TenantInfo: tenantInfo,
		TenderID:   offer.TenderID,
		Operation:  permission.OpUpdate,
		UserID:     tenantInfo.UserID,
		Comment:    comment,
		Current:    offer.Tender,
	})
	s.effects.PublishShipmentInvalidation(ctx, tenantInfo, offer, "tender_counter_closed")
}

// CounterDecisionFor reads a person's Accept or Decline on an offer that holds
// a counter as their decision on the counter: taking it, or refusing it. Any
// other answer is not a decision and is left to the response path.
func (s *Service) CounterDecisionFor(
	ctx context.Context,
	req *tenderports.TenderResponseRequest,
) (*DecideCounterRequest, error) {
	if req.Source != tender.ResponseSourceManual || req.OfferID.IsNil() ||
		(req.Action != tender.ResponseActionAccept && req.Action != tender.ResponseActionDecline) {
		return nil, nil //nolint:nilnil // no decision is not an error
	}
	offer, err := s.repo.GetOfferByID(ctx, tenderports.GetTenderOfferByIDRequest{
		TenantInfo: req.TenantInfo,
		OfferID:    req.OfferID,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, nil //nolint:nilnil // the response path reports the missing offer
		}
		return nil, err
	}
	if offer.Status != tender.OfferStatusCountered {
		return nil, nil //nolint:nilnil // an offer with no counter takes an ordinary answer
	}
	return &DecideCounterRequest{
		TenantInfo: req.TenantInfo,
		Actor:      req.Actor,
		OfferID:    req.OfferID,
		Approve:    req.Action == tender.ResponseActionAccept,
		Reason:     req.DeclineReason,
	}, nil
}

type Service struct {
	l              *zap.Logger
	repo           tenderports.TenderRepository
	assignmentRepo assignmentports.AssignmentRepository
	shipmentRepo   shipmentports.ShipmentRepository
	workflows      kernelports.WorkflowStarter
	permissions    permissionports.PermissionEngine
	effects        *tendereffects.Effects
}

type Deps struct {
	Logger         *zap.Logger
	Repo           tenderports.TenderRepository
	AssignmentRepo assignmentports.AssignmentRepository
	ShipmentRepo   shipmentports.ShipmentRepository
	Workflows      kernelports.WorkflowStarter
	Permissions    permissionports.PermissionEngine
	Effects        *tendereffects.Effects
}

func New(d Deps) *Service {
	return &Service{
		l:              d.Logger,
		repo:           d.Repo,
		assignmentRepo: d.AssignmentRepo,
		shipmentRepo:   d.ShipmentRepo,
		workflows:      d.Workflows,
		permissions:    d.Permissions,
		effects:        d.Effects,
	}
}
