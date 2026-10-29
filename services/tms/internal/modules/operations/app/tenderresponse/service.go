package tenderresponse

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/app/tenderbounce"
	"github.com/emoss08/trenova/internal/modules/operations/app/tenderdispatch"
	"github.com/emoss08/trenova/internal/modules/operations/app/tendereffects"
	"github.com/emoss08/trenova/internal/modules/operations/app/tendernegotiation"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/operations/jobs/tenderjobs/tendercontract"
	"github.com/emoss08/trenova/internal/modules/operations/ports/tenderports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/emoss08/trenova/shared/tokenutils"
	"go.uber.org/zap"
)

type Service struct {
	l          *zap.Logger
	repo       tenderports.TenderRepository
	workflows  kernelports.WorkflowStarter
	bouncer    *tenderbounce.Service
	dispatcher *tenderdispatch.Service
	effects    *tendereffects.Effects
	negotiator *tendernegotiation.Service
}

type Deps struct {
	Logger     *zap.Logger
	Repo       tenderports.TenderRepository
	Workflows  kernelports.WorkflowStarter
	Bouncer    *tenderbounce.Service
	Dispatcher *tenderdispatch.Service
	Effects    *tendereffects.Effects
	Negotiator *tendernegotiation.Service
}

func New(d Deps) *Service {
	return &Service{
		l:          d.Logger,
		repo:       d.Repo,
		workflows:  d.Workflows,
		bouncer:    d.Bouncer,
		dispatcher: d.Dispatcher,
		effects:    d.Effects,
		negotiator: d.Negotiator,
	}
}

// PreviewResponse is what RecordResponse would lead to, from the same checks,
// without signalling the workflow or filing a late answer: the transitions
// the workflow makes when it lands the answer.
func (s *Service) PreviewResponse(
	ctx context.Context,
	req *tenderports.TenderResponseRequest,
) (*tendernegotiation.ResponsePreview, error) {
	decision, err := s.negotiator.CounterDecisionFor(ctx, req)
	if err != nil {
		return nil, err
	}
	if decision != nil {
		return s.negotiator.PreviewDecideCounter(ctx, decision)
	}

	offer, err := s.planResponse(ctx, req)
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
	out := &tendernegotiation.ResponsePreview{Tender: entity, Offer: offer}
	if req.Action == tender.ResponseActionAccept {
		out.TenderAfter = tendernegotiation.AcceptedTender(entity, offer.ID, req.Source, now)

		return out, nil
	}

	if req.Action == tender.ResponseActionCounter {
		out.TenderAfter = counteredTender(entity, offer.ID, req, now)
		return out, nil
	}

	out.TenderAfter = declinedTender(entity, offer.ID, req, now)
	out.NextOffer = tendernegotiation.NextOfferAfterDecline(out.TenderAfter)
	if out.NextOffer == nil && !tendernegotiation.HasOutstandingOffer(out.TenderAfter) {
		out.Exhausted = true
		out.TenderAfter.Status = tender.StatusExhausted
		out.TenderAfter.ExhaustedAt = &now
	}

	return out, nil
}

// planResponse is everything RecordResponse decides before it signals: the
// answer is well formed and the offer can still take it. An offer that can
// no longer take it is returned with ErrOfferNoLongerAvailable, so the write
// can file the answer as late.
func (s *Service) planResponse(
	ctx context.Context,
	req *tenderports.TenderResponseRequest,
) (*tender.TenderOffer, error) {
	if req.OfferID.IsNil() {
		return nil, errortypes.NewValidationError(
			"offerId", errortypes.ErrRequired, "Offer is required",
		)
	}
	if !req.Action.IsValid() {
		return nil, errortypes.NewValidationError(
			"action", errortypes.ErrInvalid, "Response action is invalid",
		)
	}
	if !req.Source.IsValid() {
		return nil, errortypes.NewValidationError(
			"source", errortypes.ErrInvalid, "Response source is invalid",
		)
	}
	if len(req.DeclineReason) > tendernegotiation.MaxDeclineReasonLength {
		return nil, errortypes.NewValidationError(
			"declineReason",
			errortypes.ErrInvalid,
			"Decline reason must be at most 500 characters",
		)
	}
	if req.Action == tender.ResponseActionCounter {
		if err := validateCounter(req); err != nil {
			return nil, err
		}
		if !s.effects.Policy(ctx, req.TenantInfo).CountersEnabled {
			return nil, errortypes.NewBusinessError(
				"Counter-offers are turned off; accept or decline the offer",
			)
		}
	}

	offer, err := s.repo.GetOfferByID(ctx, tenderports.GetTenderOfferByIDRequest{
		TenantInfo:    req.TenantInfo,
		OfferID:       req.OfferID,
		IncludeTender: true,
	})
	if err != nil {
		return nil, err
	}

	if offer.Tender == nil || offer.Tender.Status != tender.StatusActive ||
		offer.Status != tender.OfferStatusSent ||
		offer.IsExpiredAt(timeutils.NowUnix()) {
		return offer, ErrOfferNoLongerAvailable
	}

	return offer, nil
}

// resolveToken maps a raw public token to its offer, enforcing usability with
// the single vague error.
func (s *Service) resolveToken(
	ctx context.Context,
	rawToken string,
) (context.Context, *tender.TenderOfferToken, *tender.TenderOffer, error) {
	if rawToken == "" {
		return ctx, nil, nil, invalidTokenError()
	}

	token, err := s.repo.GetTokenByHash(ctx, tokenutils.Hash(rawToken))
	if err != nil {
		return ctx, nil, nil, err
	}
	if token == nil || !token.IsUsable(timeutils.NowUnix()) {
		return ctx, nil, nil, invalidTokenError()
	}

	tenantInfo := pagination.TenantInfo{
		OrgID: token.OrganizationID,
		BuID:  token.BusinessUnitID,
	}
	ctx = dbscope.WithTenant(ctx, dbscope.Tenant{
		OrganizationID: token.OrganizationID,
		BusinessUnitID: token.BusinessUnitID,
	})
	offer, err := s.repo.GetOfferByID(ctx, tenderports.GetTenderOfferByIDRequest{
		TenantInfo:    tenantInfo,
		OfferID:       token.TenderOfferID,
		IncludeTender: true,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return ctx, nil, nil, invalidTokenError()
		}
		return ctx, nil, nil, err
	}

	return ctx, token, offer, nil
}

// PreviewByToken renders the offer for the public page. It NEVER mutates:
// email scanners prefetch GET links, and a prefetch must not consume the
// offer.
func (s *Service) PreviewByToken(
	ctx context.Context,
	rawToken string,
) (*PublicOfferView, error) {
	ctx, token, offer, err := s.resolveToken(ctx, rawToken)
	if err != nil {
		return nil, err
	}

	tenantInfo := pagination.TenantInfo{
		OrgID: token.OrganizationID,
		BuID:  token.BusinessUnitID,
	}

	emailCtx, err := s.dispatcher.BuildOfferEmailContext(
		ctx, tenantInfo, offer, "", token.ExpiresAt,
	)
	if err != nil {
		s.l.Warn("failed to build public tender offer view", zap.Error(err))
		return nil, invalidTokenError()
	}

	return &PublicOfferView{
		CarrierName:        emailCtx.CarrierName,
		ShipmentProNumber:  emailCtx.ShipmentProNumber,
		OriginSummary:      emailCtx.OriginSummary,
		DestinationSummary: emailCtx.DestinationSummary,
		PickupWindow:       emailCtx.PickupWindow,
		DeliveryWindow:     emailCtx.DeliveryWindow,
		EquipmentSummary:   emailCtx.EquipmentSummary,
		WeightSummary:      emailCtx.WeightSummary,
		RateAmount:         emailCtx.RateAmount,
		RateMethodLabel:    emailCtx.RateMethodLabel,
		ExpiresAt:          token.ExpiresAt,
		Responded:          offer.Status != tender.OfferStatusSent,
		RateValue:          offer.Rate.StringFixed(2),
		CounterAllowed: offer.Status == tender.OfferStatusSent &&
			s.effects.Policy(ctx, tenantInfo).CountersEnabled,
	}, nil
}

// RespondByToken records the carrier's answer from the public page. The
// response is funneled through RecordResponse first — the workflow's CAS is
// the hard duplicate gate — and only a recorded response consumes the token,
// so a transient failure never burns the carrier's only link.
func (s *Service) RespondByToken(
	ctx context.Context,
	rawToken string,
	response PublicResponse,
) error {
	if !response.Action.IsValid() {
		return errortypes.NewValidationError(
			"action", errortypes.ErrInvalid, "Response action is invalid",
		)
	}

	ctx, token, offer, err := s.resolveToken(ctx, rawToken)
	if err != nil {
		return err
	}

	tenantInfo := pagination.TenantInfo{
		OrgID: token.OrganizationID,
		BuID:  token.BusinessUnitID,
	}

	if err = s.RecordResponse(ctx, &tenderports.TenderResponseRequest{
		TenantInfo:    tenantInfo,
		OfferID:       offer.ID,
		Action:        response.Action,
		Source:        tender.ResponseSourceEmail,
		DeclineReason: response.DeclineReason,
		CounterRate:   response.CounterRate,
		CounterNote:   response.CounterNote,
	}); err != nil {
		return err
	}

	if _, err = s.repo.MarkTokenUsed(ctx, &tenderports.MarkTokenUsedRequest{
		TenantInfo: tenantInfo,
		TokenID:    token.ID,
		UsedAt:     timeutils.NowUnix(),
	}); err != nil {
		s.l.Warn("failed to mark tender offer token used", zap.Error(err))
	}

	return nil
}

// RecordResponse is the single funnel every response channel resolves
// through. It validates the response is still answerable, then signals the
// tender's workflow — which owns every state transition — so concurrent
// responses from different channels serialize in one place.
func (s *Service) RecordResponse(
	ctx context.Context,
	req *tenderports.TenderResponseRequest,
) error {
	if bounced, err := s.bouncer.BounceOnCarrierDecline(ctx, req); bounced || err != nil {
		return err
	}
	if decision, err := s.negotiator.CounterDecisionFor(ctx, req); decision != nil || err != nil {
		if err != nil {
			return err
		}
		return s.negotiator.DecideCounter(ctx, decision)
	}

	offer, err := s.planResponse(ctx, req)
	if isNoLongerAvailable(err) {
		s.recordLateResponse(ctx, req, offer)
		return err
	}
	if err != nil {
		return err
	}

	workflowID := tendercontract.WorkflowID(offer.TenderID, offer.Tender.BounceCount)
	err = s.workflows.SignalWorkflow(
		ctx,
		workflowID,
		"",
		tendercontract.TenderSignalName,
		tendercontract.Signal{
			Kind:          tendercontract.SignalKindResponse,
			OfferID:       offer.ID,
			Action:        req.Action,
			Source:        req.Source,
			DeclineReason: req.DeclineReason,
			CounterRate:   req.CounterRate,
			CounterNote:   strings.TrimSpace(req.CounterNote),
			ActorUserID:   req.TenantInfo.UserID,
		},
	)
	if err != nil {
		s.l.Error("failed to signal tender workflow with response",
			zap.Error(err),
			zap.String("offerId", offer.ID.String()),
			zap.String("workflowId", workflowID))
		return errortypes.NewBusinessError(
			"The response could not be recorded right now. Retry in a moment",
		)
	}

	return nil
}

// recordLateResponse appends a response that lost the race against expiry,
// supersession, or another channel — the offer's status never moves, but the
// attempt is preserved for the tender history.
func (s *Service) recordLateResponse(
	ctx context.Context,
	req *tenderports.TenderResponseRequest,
	offer *tender.TenderOffer,
) {
	if offer == nil {
		return
	}
	s.effects.RecordLateOfferResponse(ctx, req.TenantInfo, offer, req.Action, req.Source)
}

// RecordLateResponse preserves a response the workflow drained after its
// offer had already been expired, superseded, or answered — the race the
// service-side gate cannot see because the signal was already in flight.
func (s *Service) RecordLateResponse(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	offerID pulid.ID,
	action tender.ResponseAction,
	source tender.ResponseSource,
) error {
	offer, err := s.repo.GetOfferByID(ctx, tenderports.GetTenderOfferByIDRequest{
		TenantInfo:    tenantInfo,
		OfferID:       offerID,
		IncludeTender: true,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil
		}
		return err
	}
	if offer.Status.IsOutstanding() || offer.RespondedAt != nil {
		return nil
	}

	s.effects.RecordLateOfferResponse(ctx, tenantInfo, offer, action, source)
	return nil
}
