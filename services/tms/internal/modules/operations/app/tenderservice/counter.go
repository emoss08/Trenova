package tenderservice

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmenteventservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/operations/jobs/tenderjobs/tendercontract"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/tenderports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// validateCounter checks a counter-offer's own shape. Whether the offer can
// still take one is planResponse's call.
func validateCounter(req *tenderports.TenderResponseRequest) error {
	multiErr := errortypes.NewMultiError()
	switch {
	case !req.CounterRate.IsPositive():
		multiErr.Add(
			"counterRate",
			errortypes.ErrRequired,
			"A counter-offer needs a rate above zero",
		)
	case req.CounterRate.GreaterThanOrEqual(tender.MaxCounterRate):
		multiErr.Add("counterRate", errortypes.ErrInvalid, "Counter-offer rate is too large")
	}
	if utf8.RuneCountInString(req.CounterNote) > tender.MaxCounterNoteLength {
		multiErr.Add("counterNote", errortypes.ErrInvalid, "Note must be at most 500 characters")
	}
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

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

	policy := s.tenderPolicy(ctx, tenantInfo)
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
		s.recordLateOfferResponse(ctx, tenantInfo, offer, tender.ResponseActionCounter, req.Source)
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

type counterAnnouncement struct {
	offer    *tender.TenderOffer
	rate     decimal.Decimal
	note     string
	source   tender.ResponseSource
	verdict  tender.CounterVerdict
	deadline int64
}

func (s *Service) announceCounter(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	a *counterAnnouncement,
) {
	carrierName := carrierNameOf(a.offer)

	s.recordEvent(ctx, shipmenteventservice.BuildTenderCountered(
		shipmenteventservice.TenantRefFor(tenantInfo),
		tenderRefForOffer(a.offer),
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
		priority := notificationPriorityMed
		message := fmt.Sprintf(
			"%s countered at %s against %s offered. Take or refuse it by %s, or it lapses and the tender moves on",
			carrierName,
			a.rate.StringFixed(2),
			a.offer.Rate.StringFixed(2),
			timeutils.FormatInstantUTC(a.deadline),
		)
		if a.verdict.NeedsApproval {
			priority = notificationPriorityHigh
			message = fmt.Sprintf(
				"%s countered at %s against %s offered, and %s. Approving it needs the tender approve permission. Decide by %s, or it lapses and the tender moves on",
				carrierName,
				a.rate.StringFixed(2),
				a.offer.Rate.StringFixed(2),
				a.verdict.Explanation,
				timeutils.FormatInstantUTC(a.deadline),
			)
		}
		s.notifyDispatch(ctx, tenantInfo, &dispatchNotification{
			EventType:     "tender_countered",
			Priority:      priority,
			Title:         "Carrier counter-offer to decide",
			Message:       message,
			CorrelationID: a.offer.TenderID.String(),
			Link:          dispatchConsoleOfferLink(a.offer),
		})
	}

	comment := fmt.Sprintf(
		"Counter-offer from %s via %s at %s", carrierName, a.source.String(), a.rate.StringFixed(2),
	)
	if a.note != "" {
		comment += ": " + a.note
	}
	s.logTenderAudit(&tenderAuditParams{
		TenantInfo: tenantInfo,
		TenderID:   a.offer.TenderID,
		Operation:  permission.OpUpdate,
		UserID:     tenantInfo.UserID,
		Comment:    comment,
		Current:    a.offer.Tender,
	})
	s.publishTenderShipmentInvalidation(ctx, tenantInfo, a.offer, "tender_countered")
}

// DecideCounterRequest takes or refuses a carrier's counter-offer.
type DecideCounterRequest struct {
	TenantInfo pagination.TenantInfo         `json:"-"`
	Actor      *permissionports.RequestActor `json:"-"`
	OfferID    pulid.ID                      `json:"offerId"`
	Approve    bool                          `json:"approve"`
	Reason     string                        `json:"reason"`
}

func (r *DecideCounterRequest) Validate() *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	if r.OfferID.IsNil() {
		multiErr.Add("offerId", errortypes.ErrRequired, "Offer is required")
	}
	if utf8.RuneCountInString(r.Reason) > maxDeclineReasonLength {
		multiErr.Add("reason", errortypes.ErrInvalid, "Reason must be at most 500 characters")
	}
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func (r *DecideCounterRequest) actor() *permissionports.RequestActor {
	if r.Actor != nil {
		return r.Actor
	}
	return permissionports.UserActor(r.TenantInfo)
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
		out.TenderAfter = acceptedTender(entity, offer.ID, offer.ResponseSource, now)
		return out, nil
	}
	out.TenderAfter = copyTender(entity, func(o *tender.TenderOffer) {
		if o.ID == offer.ID {
			o.Status = tender.OfferStatusCounterRejected
			o.CounterDecidedAt = &now
			o.DeclineReason = strings.TrimSpace(req.Reason)
		}
	})
	out.NextOffer = nextOfferAfterDecline(out.TenderAfter)
	if out.NextOffer == nil && !hasOutstandingOffer(out.TenderAfter) {
		out.Exhausted = true
		out.TenderAfter.Status = tender.StatusExhausted
		out.TenderAfter.ExhaustedAt = &now
	}
	return out, nil
}

func decisionStamp(tenantInfo pagination.TenantInfo, at int64) *tenderports.DecisionStamp {
	return &tenderports.DecisionStamp{At: at, UserID: userIDPtr(tenantInfo)}
}

// refuseCounter closes an offer whose counter somebody refused, so the tender
// moves on to its next carrier.
func (s *Service) refuseCounter(
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
		CounterDecided: decisionStamp(tenantInfo, now),
	})
	if err != nil || !moved {
		return err
	}
	s.closeCounter(ctx, tenantInfo, offer, reason, "Counter-offer from %s refused")
	return nil
}

// lapseCounter closes an offer whose counter nobody decided before its
// deadline, so the tender moves on to its next carrier.
func (s *Service) lapseCounter(
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
	carrierName := carrierNameOf(offer)
	s.recordEvent(ctx, shipmenteventservice.BuildTenderCounterDecided(
		shipmenteventservice.TenantRefFor(tenantInfo),
		tenderRefForOffer(offer),
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
	s.logTenderAudit(&tenderAuditParams{
		TenantInfo: tenantInfo,
		TenderID:   offer.TenderID,
		Operation:  permission.OpUpdate,
		UserID:     tenantInfo.UserID,
		Comment:    comment,
		Current:    offer.Tender,
	})
	s.publishTenderShipmentInvalidation(ctx, tenantInfo, offer, "tender_counter_closed")
}

// counterDecisionFor reads a person's Accept or Decline on an offer that holds
// a counter as their decision on the counter: taking it, or refusing it. Any
// other answer is not a decision and is left to the response path.
func (s *Service) counterDecisionFor(
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
