package ediservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/shopspring/decimal"
)

var _ services.EDITenderChannel = (*Service)(nil)

func (s *Service) SendTenderOffer(
	ctx context.Context,
	req *services.SendTenderOfferEDIRequest,
) (*services.SendTenderOfferEDIResult, error) {
	if req == nil || req.OfferID.IsNil() {
		return nil, errortypes.NewValidationError(
			"offerId",
			errortypes.ErrRequired,
			"Tender offer is required",
		)
	}
	if s.tenderRepo == nil || s.carrierRepo == nil {
		return nil, errortypes.NewBusinessError(
			"EDI tender delivery is not configured for this environment",
		)
	}

	offer, err := s.tenderRepo.GetOfferByID(ctx, repositories.GetTenderOfferByIDRequest{
		TenantInfo:    req.TenantInfo,
		OfferID:       req.OfferID,
		IncludeTender: true,
	})
	if err != nil {
		return nil, err
	}
	if offer.Channel != tender.ChannelEDI {
		return nil, errortypes.NewBusinessError("Tender offer is not an EDI offer")
	}
	if offer.Tender == nil {
		return nil, errortypes.NewBusinessError("Tender offer is missing its tender")
	}
	if offer.EDIPartnerID == nil || offer.EDIPartnerID.IsNil() {
		return nil, errortypes.NewBusinessError("Tender offer has no EDI partner")
	}

	channel, err := s.resolveCarrierEDIChannel(ctx, req.TenantInfo, offer)
	if err != nil {
		return nil, err
	}
	profileID, err := s.resolveCarrierTenderProfileID(ctx, req.TenantInfo, channel)
	if err != nil {
		return nil, err
	}

	source, err := s.shipmentSvc.Get(ctx, &repositories.GetShipmentByIDRequest{
		ID:         offer.Tender.ShipmentID,
		TenantInfo: req.TenantInfo,
		ShipmentOptions: repositories.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		return nil, err
	}

	payload, err := buildCarrierOfferDocumentPayload(offer, source, channel)
	if err != nil {
		return nil, err
	}

	message, err := s.GenerateDocument(ctx, &GenerateEDIDocumentRequest{
		TenantInfo:                    req.TenantInfo,
		PartnerDocumentProfileID:      profileID,
		EDIPartnerID:                  channel.EDIPartnerID,
		TransactionSet:                edi.TransactionSet204,
		Direction:                     edi.DocumentDirectionOutbound,
		Payload:                       payload,
		CarrierSCAC:                   carrierOfferSCAC(offer, channel),
		GeneratedByID:                 req.TenantInfo.UserID,
		SuppressTenderRecipientUpsert: true,
	})
	if err != nil {
		return nil, err
	}

	return &services.SendTenderOfferEDIResult{MessageID: message.ID}, nil
}

func (s *Service) resolveCarrierEDIChannel(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	offer *tender.TenderOffer,
) (*carrier.CarrierEDIChannel, error) {
	partnerID := pulid.ConvertFromPtr(offer.EDIPartnerID)
	channels, err := s.carrierRepo.ListEDIChannels(ctx, repositories.GetCarrierEDIChannelRequest{
		TenantInfo: tenantInfo,
		CarrierID:  offer.CarrierID,
	})
	if err != nil {
		return nil, err
	}
	for _, channel := range channels {
		if channel != nil && channel.EDIPartnerID == partnerID {
			return channel, nil
		}
	}
	channel, err := s.carrierRepo.GetDefaultEDIChannel(
		ctx,
		repositories.GetCarrierEDIChannelRequest{
			TenantInfo: tenantInfo,
			CarrierID:  offer.CarrierID,
		},
	)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, errortypes.NewBusinessError(
				"Carrier has no EDI channel for the tendered partner",
			)
		}
		return nil, err
	}
	if channel == nil || channel.EDIPartnerID != partnerID {
		return nil, errortypes.NewBusinessError(
			"Carrier has no EDI channel for the tendered partner",
		)
	}
	return channel, nil
}

func (s *Service) resolveCarrierTenderProfileID(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	channel *carrier.CarrierEDIChannel,
) (pulid.ID, error) {
	if channel.PartnerDocumentProfileID != nil && channel.PartnerDocumentProfileID.IsNotNil() {
		return *channel.PartnerDocumentProfileID, nil
	}
	profile, err := s.EnsureOutboundDocumentProfile(
		ctx,
		tenantInfo,
		channel.EDIPartnerID,
		edi.TransactionSet204,
	)
	if err != nil {
		return pulid.Nil, err
	}
	return profile.ID, nil
}

// buildCarrierOfferDocumentPayload assembles the COST-side 204 for a carrier
// offer: lane, stops, equipment, weight, and the offered rate. Customer
// identity and revenue amounts from the shipment never enter this payload.
func buildCarrierOfferDocumentPayload(
	offer *tender.TenderOffer,
	source *shipment.Shipment,
	channel *carrier.CarrierEDIChannel,
) (*edi.DocumentPayload, error) {
	move := source.FindMove(offer.Tender.ShipmentMoveID)
	if move == nil {
		return nil, errortypes.NewBusinessError(
			"Tendered move is no longer part of the shipment",
		)
	}

	offeredTotal := carrierOfferTotal(offer, move.Distance)
	payload := edi.LoadTenderPayload{
		PurposeCode:         edi.LoadTenderPurposeOriginal,
		TenderOfferID:       offer.ID,
		ShipmentID:          source.ID,
		BusinessUnitID:      source.BusinessUnitID,
		OrganizationID:      source.OrganizationID,
		BOL:                 source.BOL,
		Pieces:              source.Pieces,
		Weight:              source.Weight,
		TemperatureMin:      source.TemperatureMin,
		TemperatureMax:      source.TemperatureMax,
		FreightChargeAmount: offeredTotal,
		TotalChargeAmount:   offeredTotal,
		RatingDetail:        carrierOfferRatingDetail(offer, source),
		Moves:               []edi.LoadTenderMove{loadTenderMoveFromShipmentMove(move)},
		Commodities:         make([]edi.LoadTenderCommodity, 0, len(source.Commodities)),
	}
	for _, commodity := range source.Commodities {
		if commodity == nil {
			continue
		}
		payload.Commodities = append(
			payload.Commodities,
			loadTenderCommodityFromShipmentCommodity(commodity),
		)
	}

	documentPayload := edi.NewLoadTenderDocumentPayload(payload)
	if channel.CommunicationProfileID != nil && channel.CommunicationProfileID.IsNotNil() {
		documentPayload.DeliveryCommunicationProfileID = *channel.CommunicationProfileID
	}
	return &documentPayload, nil
}

func carrierOfferTotal(offer *tender.TenderOffer, distance *float64) decimal.NullDecimal {
	return decimal.NullDecimal{
		Decimal: shipment.CarrierBaseAmount(offer.RateMethod, offer.Rate, distance),
		Valid:   true,
	}
}

func carrierOfferRatingDetail(
	offer *tender.TenderOffer,
	source *shipment.Shipment,
) map[string]any {
	detail := map[string]any{
		"paymentMethod": "PP",
		"rateMethod":    string(offer.RateMethod),
		"rate":          offer.Rate.StringFixed(2),
	}
	if source.TrailerType != nil && source.TrailerType.Code != "" {
		detail["equipmentType"] = source.TrailerType.Code
		detail["note"] = "Equipment: " + source.TrailerType.Code
	}
	return detail
}

func carrierOfferSCAC(offer *tender.TenderOffer, channel *carrier.CarrierEDIChannel) string {
	carrierSCAC := ""
	if offer.Carrier != nil {
		carrierSCAC = offer.Carrier.SCAC
	}
	return stringutils.FirstNonEmpty(channel.SCACOverride, carrierSCAC)
}
