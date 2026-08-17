package tenderservice

import (
	"context"
	"fmt"
	"html/template"
	"net/url"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	portservices "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/shipmenteventservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/emoss08/trenova/shared/tokenutils"
	"go.uber.org/zap"
)

// DispatchOffer moves one offer Pending→Sent and delivers it on its channel.
// A channel failure is a business outcome (the offer is marked
// DeliveryFailed and the plan moves on), not an activity error.
func (s *Service) DispatchOffer(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	offerID pulid.ID,
) (*portservices.TenderDispatchResult, error) {
	offer, err := s.repo.GetOfferByID(ctx, repositories.GetTenderOfferByIDRequest{
		TenantInfo:    tenantInfo,
		OfferID:       offerID,
		IncludeTender: true,
	})
	if err != nil {
		return nil, err
	}

	// A Sent row on a fresh attempt means an earlier try crashed between the
	// CAS and the channel send. Delivery is re-driven rather than adopted
	// blindly: the email idempotency key and the stamped EDI message make a
	// duplicate send harmless, while a lost send would otherwise burn the
	// carrier's whole TTL in silence.
	if offer.Status == tender.OfferStatusSent && offer.ExpiresAt != nil {
		if err = s.deliverOffer(ctx, tenantInfo, offer, *offer.ExpiresAt); err != nil {
			return s.dispatchDeliveryOutcome(ctx, tenantInfo, offer, err)
		}
		return &portservices.TenderDispatchResult{
			Delivered: true,
			ExpiresAt: *offer.ExpiresAt,
		}, nil
	}
	if offer.Status != tender.OfferStatusPending {
		return &portservices.TenderDispatchResult{
			Delivered: false,
			Reason:    "Offer is no longer pending",
		}, nil
	}

	now := timeutils.NowUnix()
	expiresAt := now + int64(offer.OfferTTLSeconds)
	moved, err := s.repo.UpdateOfferStatus(ctx, &repositories.UpdateOfferStatusRequest{
		TenantInfo: tenantInfo,
		OfferID:    offerID,
		FromStatus: []tender.OfferStatus{tender.OfferStatusPending},
		ToStatus:   tender.OfferStatusSent,
		SentAt:     &now,
		ExpiresAt:  &expiresAt,
	})
	if err != nil {
		return nil, err
	}
	if !moved {
		return &portservices.TenderDispatchResult{
			Delivered: false,
			Reason:    "Offer is no longer pending",
		}, nil
	}

	rank := offer.Rank
	if _, err = s.repo.UpdateTenderStatus(ctx, &repositories.UpdateTenderStatusRequest{
		TenantInfo:  tenantInfo,
		TenderID:    offer.TenderID,
		FromStatus:  []tender.Status{tender.StatusActive},
		ToStatus:    tender.StatusActive,
		CurrentRank: &rank,
	}); err != nil {
		s.l.Warn("failed to advance tender current rank", zap.Error(err))
	}

	if err = s.deliverOffer(ctx, tenantInfo, offer, expiresAt); err != nil {
		return s.dispatchDeliveryOutcome(ctx, tenantInfo, offer, err)
	}

	s.recordEvent(ctx, shipmenteventservice.BuildTenderOffered(
		shipmenteventservice.TenantRefFor(tenantInfo),
		tenderRefForOffer(offer),
		carrierNameOf(offer),
		offer.Rank,
		offer.Channel,
		shipmenteventservice.ActorFor(tenantInfo),
	))
	s.publishTenderShipmentInvalidation(ctx, tenantInfo, offer, "tender_offer_sent")

	return &portservices.TenderDispatchResult{
		Delivered: true,
		ExpiresAt: expiresAt,
	}, nil
}

func (s *Service) deliverOffer(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	offer *tender.TenderOffer,
	expiresAt int64,
) error {
	switch offer.Channel {
	case tender.ChannelEmail:
		return s.deliverByEmail(ctx, tenantInfo, offer, expiresAt)
	case tender.ChannelEDI:
		if offer.EDIMessageID != nil && !offer.EDIMessageID.IsNil() {
			return nil
		}
		return s.deliverByEDI(ctx, tenantInfo, offer)
	default:
		return errortypes.NewBusinessError("Offer has no deliverable channel")
	}
}

// dispatchDeliveryOutcome separates deterministic delivery rejections (bad
// configuration, broken template — mark the offer failed, plan moves on)
// from transient infra failures, which propagate so the activity retries
// instead of irrevocably burning the carrier's rank on a blip.
func (s *Service) dispatchDeliveryOutcome(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	offer *tender.TenderOffer,
	deliveryErr error,
) (*portservices.TenderDispatchResult, error) {
	if !isAssignBusinessFailure(deliveryErr) {
		return nil, deliveryErr
	}

	s.markDeliveryFailed(ctx, tenantInfo, offer, deliveryErr.Error())
	return &portservices.TenderDispatchResult{
		Delivered: false,
		Reason:    deliveryErr.Error(),
	}, nil
}

func (s *Service) deliverByEmail(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	offer *tender.TenderOffer,
	expiresAt int64,
) error {
	if s.templates == nil || s.emailService == nil {
		return errortypes.NewBusinessError(
			"Email delivery is not configured for this environment",
		)
	}
	if s.cfg.Tendering.GetPublicBaseURL() == "" {
		return errortypes.NewBusinessError(
			"Tendering public base URL is not configured; offer links would be unreachable",
		)
	}

	token, tokenHash, err := tokenutils.New()
	if err != nil {
		return err
	}

	if _, err = s.repo.CreateToken(ctx, &tender.TenderOfferToken{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		TenderOfferID:  offer.ID,
		TokenHash:      tokenHash,
		Email:          offer.RecipientEmail,
		ExpiresAt:      expiresAt,
	}); err != nil {
		return err
	}

	data, err := s.buildOfferEmailContext(ctx, tenantInfo, offer, token, expiresAt)
	if err != nil {
		return err
	}

	rendered, err := s.templates.RenderMessage(ctx, &portservices.RenderMessageRequest{
		TenantInfo:        tenantInfo,
		Kind:              documenttemplate.KindTenderOfferEmail,
		Data:              data,
		ReferenceID:       offer.ID,
		FallbackToBuiltIn: true,
	})
	if err != nil {
		return errortypes.NewBusinessError(
			"The tender offer email could not be rendered: " + err.Error(),
		)
	}

	if _, err = s.emailService.Send(ctx, &portservices.SendEmailRequest{
		TenantInfo:     tenantInfo,
		Purpose:        email.PurposeGeneral,
		To:             []string{offer.RecipientEmail},
		Subject:        rendered.Subject,
		HTML:           rendered.HTML,
		Text:           rendered.Text,
		IdempotencyKey: "tender-offer-" + offer.ID.String(),
	}); err != nil {
		return fmt.Errorf("send tender offer email: %w", err)
	}

	return nil
}

func (s *Service) deliverByEDI(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	offer *tender.TenderOffer,
) error {
	if s.ediChannel == nil {
		return errortypes.NewBusinessError(
			"EDI delivery is not configured for this environment",
		)
	}

	result, err := s.ediChannel.SendTenderOffer(ctx, &portservices.SendTenderOfferEDIRequest{
		TenantInfo: tenantInfo,
		OfferID:    offer.ID,
	})
	if err != nil {
		return err
	}

	if result != nil && !result.MessageID.IsNil() {
		messageID := result.MessageID
		if _, uErr := s.repo.UpdateOfferStatus(ctx, &repositories.UpdateOfferStatusRequest{
			TenantInfo:   tenantInfo,
			OfferID:      offer.ID,
			FromStatus:   []tender.OfferStatus{tender.OfferStatusSent},
			ToStatus:     tender.OfferStatusSent,
			EDIMessageID: &messageID,
		}); uErr != nil {
			s.l.Warn("failed to stamp EDI message on tender offer", zap.Error(uErr))
		}
	}

	return nil
}

func (s *Service) markDeliveryFailed(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	offer *tender.TenderOffer,
	reason string,
) {
	now := timeutils.NowUnix()
	if _, err := s.repo.UpdateOfferStatus(ctx, &repositories.UpdateOfferStatusRequest{
		TenantInfo:    tenantInfo,
		OfferID:       offer.ID,
		FromStatus:    []tender.OfferStatus{tender.OfferStatusSent},
		ToStatus:      tender.OfferStatusDeliveryFailed,
		DeliveryError: reason,
	}); err != nil {
		s.l.Error("failed to mark tender offer delivery failed", zap.Error(err))
	}
	if err := s.repo.RevokeTokensForOffer(ctx, tenantInfo, offer.ID, now); err != nil {
		s.l.Warn("failed to revoke tokens for failed tender offer", zap.Error(err))
	}

	s.recordEvent(ctx, shipmenteventservice.BuildTenderDeliveryFailed(
		shipmenteventservice.TenantRefFor(tenantInfo),
		tenderRefForOffer(offer),
		carrierNameOf(offer),
		offer.Channel,
		reason,
		shipmenteventservice.ActorFor(tenantInfo),
	))
	s.notifyDispatch(ctx, tenantInfo, &dispatchNotification{
		EventType: "tender_delivery_failed",
		Priority:  notificationPriorityHigh,
		Title:     "Tender offer could not be delivered",
		Message: fmt.Sprintf(
			"The offer to %s could not be delivered: %s",
			carrierNameOf(offer), reason,
		),
		CorrelationID: offer.ID.String(),
		Link:          dispatchConsoleOfferLink(offer),
	})
}

// buildOfferEmailContext assembles the carrier-facing summary from the
// shipment the offer covers. Only lane, timing, equipment, weight, and the
// offered rate are exposed — never customer or revenue data.
func (s *Service) buildOfferEmailContext(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	offer *tender.TenderOffer,
	token string,
	expiresAt int64,
) (documenttemplate.TenderOfferEmailContext, error) {
	data := documenttemplate.TenderOfferEmailContext{
		CarrierName:     carrierNameOf(offer),
		CompanyName:     s.companyName(ctx, tenantInfo),
		RateAmount:      "$" + offer.Rate.StringFixed(2),
		RateMethodLabel: rateMethodLabel(offer.RateMethod),
		//nolint:gosec // G203: both components are trusted — an operator-configured
		// base URL and a server-minted, path-escaped token; nothing here is caller-supplied
		AcceptURL: template.URL(
			s.offerResponseURL(token, "accept"),
		),
		//nolint:gosec // G203: see AcceptURL
		DeclineURL: template.URL(
			s.offerResponseURL(token, "decline"),
		),
		ExpiresAt: timeutils.WindowLabelUTC(expiresAt, nil),
	}
	now := timeutils.NowUnix()
	if expiresAt > now {
		data.ExpiresInHours = int((expiresAt - now + 3599) / 3600)
	}

	if offer.Tender == nil {
		return data, nil
	}

	sh, err := s.shipmentRepo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID: offer.Tender.ShipmentID,
		TenantInfo: pagination.TenantInfo{
			OrgID: tenantInfo.OrgID,
			BuID:  tenantInfo.BuID,
		},
		ShipmentOptions: repositories.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		return data, err
	}

	data.ShipmentProNumber = sh.ProNumber
	if sh.Weight != nil && *sh.Weight > 0 {
		data.WeightSummary = fmt.Sprintf("%d lbs", *sh.Weight)
	}
	if sh.TrailerType != nil {
		data.EquipmentSummary = sh.TrailerType.Code
	}

	move := sh.FindMove(offer.Tender.ShipmentMoveID)
	if move == nil {
		return data, nil
	}
	for _, stop := range move.Stops {
		if stop == nil || stop.Location == nil {
			continue
		}
		summary := stop.Location.City
		if stop.Location.State != nil {
			summary += ", " + stop.Location.State.Abbreviation
		}
		window := timeutils.WindowLabelUTC(stop.ScheduledWindowStart, stop.ScheduledWindowEnd)
		if stop.IsOriginStop() && data.OriginSummary == "" {
			data.OriginSummary = summary
			data.PickupWindow = window
		}
		if stop.IsDestinationStop() {
			data.DestinationSummary = summary
			data.DeliveryWindow = window
		}
	}

	return data, nil
}

func (s *Service) companyName(ctx context.Context, tenantInfo pagination.TenantInfo) string {
	if s.orgRepo == nil {
		return ""
	}

	org, err := s.orgRepo.GetByID(ctx, repositories.GetOrganizationByIDRequest{
		TenantInfo: tenantInfo,
	})
	if err != nil || org == nil {
		return ""
	}
	return org.Name
}

func (s *Service) offerResponseURL(token, action string) string {
	path := "/tender-offer/" + url.PathEscape(token) + "/" + action
	base := s.cfg.Tendering.GetPublicBaseURL()
	if base == "" {
		return path
	}
	return base + path
}

// HashOfferToken derives the stored lookup hash for a raw token.
func HashOfferToken(token string) string {
	return tokenutils.Hash(token)
}

func rateMethodLabel(method shipment.CarrierRateMethod) string {
	return method.Label()
}

func carrierNameOf(offer *tender.TenderOffer) string {
	if offer == nil || offer.Carrier == nil {
		return "the carrier"
	}
	return offer.Carrier.Name
}

func tenderRefForOffer(offer *tender.TenderOffer) shipmenteventservice.TenderRef {
	ref := shipmenteventservice.TenderRef{
		TenderID: offer.TenderID,
		OfferID:  offer.ID,
	}
	if offer.Tender != nil {
		ref.ShipmentID = offer.Tender.ShipmentID
		ref.MoveID = offer.Tender.ShipmentMoveID
	}
	return ref
}

func (s *Service) publishTenderShipmentInvalidation(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	offer *tender.TenderOffer,
	action string,
) {
	if offer.Tender == nil {
		return
	}
	s.publishInvalidation(ctx, tenantInfo, offer.Tender.ShipmentID, action)
}
