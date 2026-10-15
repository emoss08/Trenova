package tenderservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/operations/ports/tenderports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/emoss08/trenova/shared/tokenutils"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// invalidTokenError uses one piece of copy for every failure mode — missing,
// expired, used, revoked — so the public endpoint leaks nothing about which
// one applied.
func invalidTokenError() error {
	return errortypes.NewValidationError(
		"token",
		errortypes.ErrInvalid,
		"This offer link is no longer valid",
	)
}

// PublicOfferView is everything a carrier without a login may see about an
// offer: the lane, the timing, and the money — never customer data.
type PublicOfferView struct {
	CarrierName        string `json:"carrierName"`
	ShipmentProNumber  string `json:"shipmentProNumber"`
	OriginSummary      string `json:"originSummary"`
	DestinationSummary string `json:"destinationSummary"`
	PickupWindow       string `json:"pickupWindow"`
	DeliveryWindow     string `json:"deliveryWindow"`
	EquipmentSummary   string `json:"equipmentSummary"`
	WeightSummary      string `json:"weightSummary"`
	RateAmount         string `json:"rateAmount"`
	RateMethodLabel    string `json:"rateMethodLabel"`
	ExpiresAt          int64  `json:"expiresAt"`
	Responded          bool   `json:"responded"`
	// RateValue is the offered rate as a plain number, for a counter-offer
	// form to start from.
	RateValue string `json:"rateValue"`
	// CounterAllowed says whether the carrier may answer with a rate of its
	// own instead of accepting or declining.
	CounterAllowed bool `json:"counterAllowed"`
}

// PublicResponse is a carrier's answer from the public page.
type PublicResponse struct {
	Action        tender.ResponseAction
	DeclineReason string
	CounterRate   decimal.Decimal
	CounterNote   string
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

	emailCtx, err := s.buildOfferEmailContext(
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
			s.tenderPolicy(ctx, tenantInfo).CountersEnabled,
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
