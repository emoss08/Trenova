package tenderdispatch

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"net/url"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/tenderingconfig"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/tendertemplate"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmenteventservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/tendereffects"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/tenderports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierintelports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/emoss08/trenova/shared/tokenutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Service struct {
	l               *zap.Logger
	repo            tenderports.TenderRepository
	carrierRepo     carrierports.CarrierRepository
	shipmentRepo    shipmentports.ShipmentRepository
	orgRepo         tenantports.OrganizationRepository
	appConfig       *appconfig.Config
	tenderingConfig *tenderingconfig.Config
	templates       documenttemplateports.DocumentTemplateResolver
	emailService    emailports.EmailService
	ediChannel      tenderports.EDITenderChannel
	intelGate       carrierintelports.CarrierIntelGate
	effects         *tendereffects.Effects
}

// DispatchOffer moves one offer Pending→Sent and delivers it on its channel.
// A channel failure is a business outcome (the offer is marked
// DeliveryFailed and the plan moves on), not an activity error.
func (s *Service) DispatchOffer(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	offerID pulid.ID,
) (*tenderports.TenderDispatchResult, error) {
	offer, err := s.repo.GetOfferByID(ctx, tenderports.GetTenderOfferByIDRequest{
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
		if offer.IsExpiredAt(timeutils.NowUnix()) {
			return &tenderports.TenderDispatchResult{
				Delivered: true,
				ExpiresAt: *offer.ExpiresAt,
			}, nil
		}
		if err = s.deliverOffer(ctx, tenantInfo, offer, *offer.ExpiresAt); err != nil {
			return s.dispatchDeliveryOutcome(ctx, tenantInfo, offer, err)
		}
		return &tenderports.TenderDispatchResult{
			Delivered: true,
			ExpiresAt: *offer.ExpiresAt,
		}, nil
	}
	if offer.Status != tender.OfferStatusPending {
		return &tenderports.TenderDispatchResult{
			Delivered: false,
			Reason:    "Offer is no longer pending",
		}, nil
	}

	skipped, skipErr := s.skipIneligibleOffer(ctx, tenantInfo, offer)
	if !errors.Is(skipErr, errOfferNotSkipped) {
		return skipped, skipErr
	}

	now := timeutils.NowUnix()
	expiresAt := now + int64(offer.OfferTTLSeconds)
	moved, err := s.repo.UpdateOfferStatus(ctx, &tenderports.UpdateOfferStatusRequest{
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
		return &tenderports.TenderDispatchResult{
			Delivered: false,
			Reason:    "Offer is no longer pending",
		}, nil
	}

	rank := offer.Rank
	if _, err = s.repo.UpdateTenderStatus(ctx, &tenderports.UpdateTenderStatusRequest{
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

	s.effects.RecordEvent(ctx, shipmenteventservice.BuildTenderOffered(
		shipmenteventservice.TenantRefFor(tenantInfo),
		tendereffects.TenderRefForOffer(offer),
		tendereffects.CarrierNameOf(offer),
		offer.Rank,
		offer.Channel,
		shipmenteventservice.ActorFor(tenantInfo),
	))
	s.effects.PublishShipmentInvalidation(ctx, tenantInfo, offer, "tender_offer_sent")

	return &tenderports.TenderDispatchResult{
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
) (*tenderports.TenderDispatchResult, error) {
	if !tendereffects.IsAssignBusinessFailure(deliveryErr) {
		return nil, deliveryErr
	}

	s.markDeliveryFailed(ctx, tenantInfo, offer, deliveryErr.Error())
	return &tenderports.TenderDispatchResult{
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
	if s.tenderingConfig.PublicLinkBaseURL(s.appConfig) == "" {
		return errortypes.NewBusinessError(
			"Public links need a base URL; set app.webBaseUrl so offer links can be reached",
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

	data, err := s.BuildOfferEmailContext(ctx, tenantInfo, offer, token, expiresAt)
	if err != nil {
		return err
	}

	rendered, err := s.templates.RenderMessage(ctx, &documenttemplateports.RenderMessageRequest{
		TenantInfo:        tenantInfo,
		Kind:              documenttemplate.KindTenderOfferEmail,
		Data:              data,
		ReferenceID:       offer.ID,
		FallbackToBuiltIn: true,
	})
	if err != nil {
		return errortypes.NewBusinessError(
			"The tender offer email could not be rendered: {0}", err.Error(),
		)
	}

	if _, err = s.emailService.Send(ctx, &emailports.SendEmailRequest{
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

	result, err := s.ediChannel.SendTenderOffer(ctx, &tenderports.SendTenderOfferEDIRequest{
		TenantInfo: tenantInfo,
		OfferID:    offer.ID,
	})
	if err != nil {
		return err
	}

	if result != nil && !result.MessageID.IsNil() {
		messageID := result.MessageID
		if _, uErr := s.repo.UpdateOfferStatus(ctx, &tenderports.UpdateOfferStatusRequest{
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
	if _, err := s.repo.UpdateOfferStatus(ctx, &tenderports.UpdateOfferStatusRequest{
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

	s.effects.RecordEvent(ctx, shipmenteventservice.BuildTenderDeliveryFailed(
		shipmenteventservice.TenantRefFor(tenantInfo),
		tendereffects.TenderRefForOffer(offer),
		tendereffects.CarrierNameOf(offer),
		offer.Channel,
		reason,
		shipmenteventservice.ActorFor(tenantInfo),
	))
	s.effects.NotifyDispatch(ctx, tenantInfo, &tendereffects.DispatchNotification{
		EventType: "tender_delivery_failed",
		Priority:  tendereffects.NotificationPriorityHigh,
		Title:     "Tender offer could not be delivered",
		Message: fmt.Sprintf(
			"The offer to %s could not be delivered: %s",
			tendereffects.CarrierNameOf(offer), reason,
		),
		CorrelationID: offer.ID.String(),
		Link:          tendereffects.DispatchConsoleOfferLink(offer),
	})
}

// BuildOfferEmailContext assembles the carrier-facing summary from the
// shipment the offer covers. Only lane, timing, equipment, weight, and the
// offered rate are exposed — never customer or revenue data.
func (s *Service) BuildOfferEmailContext(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	offer *tender.TenderOffer,
	token string,
	expiresAt int64,
) (tendertemplate.TenderOfferEmailContext, error) {
	data := tendertemplate.TenderOfferEmailContext{
		CarrierName:     tendereffects.CarrierNameOf(offer),
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

	sh, err := s.shipmentRepo.GetByID(ctx, &shipmentports.GetShipmentByIDRequest{
		ID: offer.Tender.ShipmentID,
		TenantInfo: pagination.TenantInfo{
			OrgID: tenantInfo.OrgID,
			BuID:  tenantInfo.BuID,
		},
		ShipmentOptions: shipmentports.ShipmentOptions{
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

	org, err := s.orgRepo.GetByID(ctx, tenantports.GetOrganizationByIDRequest{
		TenantInfo: tenantInfo,
	})
	if err != nil || org == nil {
		return ""
	}
	return org.Name
}

func (s *Service) offerResponseURL(token, action string) string {
	path := "/tender-offer/" + url.PathEscape(token) + "/" + action
	base := s.tenderingConfig.PublicLinkBaseURL(s.appConfig)
	if base == "" {
		return path
	}
	return base + path
}

type Params struct {
	fx.In

	Logger          *zap.Logger
	Repo            tenderports.TenderRepository
	CarrierRepo     carrierports.CarrierRepository
	ShipmentRepo    shipmentports.ShipmentRepository
	OrgRepo         tenantports.OrganizationRepository
	AppConfig       *appconfig.Config
	TenderingConfig *tenderingconfig.Config
	Templates       documenttemplateports.DocumentTemplateResolver `optional:"true"`
	EmailService    emailports.EmailService                        `optional:"true"`
	EDIChannel      tenderports.EDITenderChannel                   `optional:"true"`
	Effects         *tendereffects.Effects
}

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func New(p Params) *Service {
	return &Service{
		l:               p.Logger.Named("service.tender"),
		repo:            p.Repo,
		carrierRepo:     p.CarrierRepo,
		shipmentRepo:    p.ShipmentRepo,
		orgRepo:         p.OrgRepo,
		appConfig:       p.AppConfig,
		tenderingConfig: p.TenderingConfig,
		templates:       p.Templates,
		emailService:    p.EmailService,
		ediChannel:      p.EDIChannel,
		effects:         p.Effects,
	}
}

func (s *Service) SetIntelGate(value carrierintelports.CarrierIntelGate) {
	s.intelGate = value
}

func (s *Service) SetEDITenderChannel(value tenderports.EDITenderChannel) {
	s.ediChannel = value
}

var errOfferNotSkipped = errors.New("tender offer was not skipped")

func (s *Service) skipIneligibleOffer(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	offer *tender.TenderOffer,
) (*tenderports.TenderDispatchResult, error) {
	if s.intelGate == nil {
		return nil, errOfferNotSkipped
	}

	carriers, err := s.carrierRepo.GetByIDs(ctx, carrierports.GetCarriersByIDsRequest{
		TenantInfo: tenantInfo,
		CarrierIDs: []pulid.ID{offer.CarrierID},
		CarrierFilterOptions: carrierports.CarrierFilterOptions{
			IncludeInsurancePolicies: true,
		},
	})
	if err != nil {
		return nil, err
	}
	if len(carriers) == 0 {
		return nil, errOfferNotSkipped
	}

	gates := s.effects.IntelGates(ctx, tenantInfo, []pulid.ID{offer.CarrierID}, true)
	eligibility := carrier.EvaluateEligibility(carrier.EligibilityInput{
		Carrier: carriers[0],
		Now:     timeutils.NowUnix(),
		Intel:   gates[offer.CarrierID],
	})
	if !eligibility.IsBlocked() {
		return nil, errOfferNotSkipped
	}

	reason := "Carrier became ineligible before the offer was sent: " +
		strings.Join(eligibility.Blockers, "; ")
	moved, err := s.repo.UpdateOfferStatus(ctx, &tenderports.UpdateOfferStatusRequest{
		TenantInfo:    tenantInfo,
		OfferID:       offer.ID,
		FromStatus:    []tender.OfferStatus{tender.OfferStatusPending},
		ToStatus:      tender.OfferStatusSkipped,
		DeclineReason: reason,
	})
	if err != nil {
		return nil, err
	}
	if moved {
		s.l.Info("skipped tender offer for ineligible carrier",
			zap.String("offerId", offer.ID.String()),
			zap.String("carrierId", offer.CarrierID.String()))
		s.effects.PublishShipmentInvalidation(ctx, tenantInfo, offer, "tender_offer_skipped")
	}
	return &tenderports.TenderDispatchResult{Delivered: false, Reason: reason}, nil
}

var _ tenderports.TenderOfferDispatcher = (*Service)(nil)
