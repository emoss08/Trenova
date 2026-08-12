package rateconfirmationservice

import (
	"context"
	"html/template"
	"net/url"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/rateconfirmation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/emoss08/trenova/shared/tokenutils"
	"go.uber.org/zap"
)

// signTokenTTLSeconds is how long a mailed sign link stays live: 14 days,
// long enough for a carrier's back office, short enough that a stale link in
// a forwarded inbox dies on its own.
const signTokenTTLSeconds = int64(14 * 24 * 3600)

// invalidTokenError uses one piece of copy for every failure mode — missing,
// expired, used, revoked, voided — so the public endpoint leaks nothing about
// which one applied.
func invalidTokenError() error {
	return errortypes.NewValidationError(
		"token",
		errortypes.ErrInvalid,
		"This rate confirmation link is no longer valid",
	)
}

// PublicRateConfirmationView is everything a carrier without a login may see
// about the agreement: the routing, the money, and where it stands — built
// purely from the frozen snapshot, never customer data.
type PublicRateConfirmationView struct {
	CarrierName       string          `json:"carrierName"`
	CompanyName       string          `json:"companyName"`
	ShipmentProNumber string          `json:"shipmentProNumber"`
	RevisionLabel     string          `json:"revisionLabel"`
	RateMethodLabel   string          `json:"rateMethodLabel"`
	BaseRateLabel     string          `json:"baseRateLabel"`
	TotalCost         string          `json:"totalCost"`
	Currency          string          `json:"currency"`
	PaymentTermsLabel string          `json:"paymentTermsLabel"`
	Stops             []PublicStopRow `json:"stops"`
	Status            string          `json:"status"`
	ExpiresAt         int64           `json:"expiresAt"`
	Confirmed         bool            `json:"confirmed"`
}

// PublicStopRow is one stop of the move being agreed to.
type PublicStopRow struct {
	Sequence int    `json:"sequence"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	Address  string `json:"address"`
	Window   string `json:"window"`
}

// resolveToken maps a raw public token to its rate confirmation, enforcing
// usability with the single vague error. A voided agreement invalidates every
// outstanding link even if revocation was missed.
func (s *Service) resolveToken(
	ctx context.Context,
	rawToken string,
) (*rateconfirmation.RateConfirmationToken, *rateconfirmation.RateConfirmation, error) {
	if rawToken == "" {
		return nil, nil, invalidTokenError()
	}

	token, err := s.repo.GetTokenByHash(ctx, tokenutils.Hash(rawToken))
	if err != nil {
		return nil, nil, err
	}
	if token == nil || !token.IsUsable(timeutils.NowUnix()) {
		return nil, nil, invalidTokenError()
	}

	tenantInfo := pagination.TenantInfo{
		OrgID: token.OrganizationID,
		BuID:  token.BusinessUnitID,
	}
	entity, err := s.repo.GetByID(ctx, &repositories.GetRateConfirmationByIDRequest{
		TenantInfo:         tenantInfo,
		RateConfirmationID: token.RateConfirmationID,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, nil, invalidTokenError()
		}
		return nil, nil, err
	}
	if entity.Status == rateconfirmation.StatusVoided {
		return nil, nil, invalidTokenError()
	}

	return token, entity, nil
}

// PreviewByToken renders the agreement for the public page. It NEVER mutates:
// email scanners prefetch GET links, and a prefetch must not burn the
// carrier's only link.
func (s *Service) PreviewByToken(
	ctx context.Context,
	rawToken string,
) (*PublicRateConfirmationView, error) {
	token, entity, err := s.resolveToken(ctx, rawToken)
	if err != nil {
		return nil, err
	}

	templateContext, err := contextFromSnapshot(entity.PayloadSnapshot)
	if err != nil {
		s.l.Warn("failed to build public rate confirmation view", zap.Error(err))
		return nil, invalidTokenError()
	}

	view := &PublicRateConfirmationView{
		CarrierName:       templateContext.CarrierName,
		CompanyName:       templateContext.CompanyName,
		ShipmentProNumber: templateContext.ShipmentProNumber,
		RevisionLabel:     templateContext.RevisionLabel,
		RateMethodLabel:   templateContext.RateMethodLabel,
		BaseRateLabel:     templateContext.BaseRateLabel,
		TotalCost:         templateContext.TotalCost,
		Currency:          templateContext.Currency,
		PaymentTermsLabel: templateContext.PaymentTermsLabel,
		Status:            entity.Status.String(),
		ExpiresAt:         token.ExpiresAt,
		Confirmed:         entity.Status == rateconfirmation.StatusConfirmed,
	}
	view.Stops = make([]PublicStopRow, 0, len(templateContext.Stops))
	for _, stop := range templateContext.Stops {
		view.Stops = append(view.Stops, PublicStopRow{
			Sequence: stop.Sequence,
			Type:     stop.Type,
			Name:     stop.Name,
			Address:  stop.Address,
			Window:   stop.Window,
		})
	}

	return view, nil
}

// ConfirmByToken records the carrier's signature from the public page. The
// confirmation is written first and the token burned after — a transient
// failure between the two leaves the link usable rather than eating the
// carrier's only way to sign.
func (s *Service) ConfirmByToken(
	ctx context.Context,
	rawToken, signerName, signerTitle string,
) error {
	if signerName == "" {
		return errortypes.NewValidationError(
			"signerName", errortypes.ErrRequired,
			"The signer's name is required",
		)
	}

	token, entity, err := s.resolveToken(ctx, rawToken)
	if err != nil {
		return err
	}

	tenantInfo := pagination.TenantInfo{
		OrgID: token.OrganizationID,
		BuID:  token.BusinessUnitID,
	}

	if entity.Status != rateconfirmation.StatusConfirmed {
		if !entity.CanConfirm() {
			return invalidTokenError()
		}
		previous := *entity
		confirmed, confirmErr := s.markConfirmed(ctx, tenantInfo, entity, confirmParams{
			Name:  signerName,
			Title: signerTitle,
			Via:   rateconfirmation.ViaPublicSignature,
			At:    timeutils.NowUnix(),
		})
		if confirmErr != nil {
			return confirmErr
		}
		s.logRateConfirmationAudit(&rateConfirmationAuditParams{
			TenantInfo: tenantInfo,
			Operation:  permission.OpApprove,
			Comment:    "Confirmed via public signing token by " + signerName,
			Previous:   &previous,
			Current:    confirmed,
		})
	}

	if _, err = s.repo.MarkTokenUsed(ctx, &repositories.MarkRateConfirmationTokenUsedRequest{
		TenantInfo: tenantInfo,
		TokenID:    token.ID,
		UsedAt:     timeutils.NowUnix(),
	}); err != nil {
		s.l.Warn("failed to mark rate confirmation token used", zap.Error(err))
	}

	return nil
}

// applySignLink mints the single-use sign token for an outgoing revision and
// injects the link into the email context. Prior live links are revoked so
// exactly one current link stands. Without a configured public base URL the
// email still carries the attached PDF; the link is skipped, not broken.
func (s *Service) applySignLink(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	entity *rateconfirmation.RateConfirmation,
	recipientEmail string,
	templateContext *documenttemplate.RateConfirmationContext,
) error {
	if s.cfg == nil || s.cfg.Tendering.GetPublicBaseURL() == "" {
		s.l.Warn("tendering public base URL is not configured; the rate confirmation email carries no sign link",
			zap.String("rateConfirmationId", entity.ID.String()))
		return nil
	}

	s.revokeTokens(ctx, tenantInfo, entity.ID)

	rawToken, tokenHash, err := tokenutils.New()
	if err != nil {
		return err
	}

	expiresAt := timeutils.NowUnix() + signTokenTTLSeconds
	if _, err = s.repo.CreateToken(ctx, &rateconfirmation.RateConfirmationToken{
		OrganizationID:     tenantInfo.OrgID,
		BusinessUnitID:     tenantInfo.BuID,
		RateConfirmationID: entity.ID,
		TokenHash:          tokenHash,
		Email:              recipientEmail,
		ExpiresAt:          expiresAt,
	}); err != nil {
		return err
	}

	//nolint:gosec // built from config + a generated token
	templateContext.SignURL = template.URL(s.signURL(rawToken))
	templateContext.SignExpiresLabel = timeutils.WindowLabelUTC(expiresAt, nil)

	return nil
}

func (s *Service) signURL(rawToken string) string {
	return s.cfg.Tendering.GetPublicBaseURL() + "/rate-confirmation/" + url.PathEscape(rawToken)
}
