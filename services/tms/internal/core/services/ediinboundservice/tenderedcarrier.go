package ediinboundservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/ediservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.uber.org/zap"
)

// splitTenderOfferCandidates scans reference candidates for an echoed tof_
// offer id and returns it alongside the remaining non-offer references.
func splitTenderOfferCandidates(candidates []string) (pulid.ID, []string) {
	offerID := pulid.Nil
	references := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if strings.HasPrefix(candidate, tenderOfferReferencePrefix) {
			if offerID.IsNil() {
				if id, parseErr := pulid.Parse(candidate); parseErr == nil {
					offerID = id
				}
			}
			continue
		}
		references = append(references, candidate)
	}
	return offerID, references
}

// resolveAcceptedTenderOffer matches an inbound carrier document to the
// accepted tender offer it answers, for outbound carrier tenders that
// deliberately have no tender recipient row.
func (s *Service) resolveAcceptedTenderOffer(
	ctx context.Context,
	file *edi.EDIInboundFile,
	partner *edi.EDIPartner,
	offerID pulid.ID,
	references []string,
) (*tender.TenderOffer, error) {
	if s.tenderRepo == nil || (offerID.IsNil() && len(references) == 0) {
		return nil, errortypes.NewNotFoundError("Tender offer")
	}
	return s.tenderRepo.GetAcceptedOfferForPartnerReference(
		ctx,
		repositories.GetAcceptedOfferForPartnerReferenceRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: file.OrganizationID,
				BuID:  file.BusinessUnitID,
			},
			PartnerID:  partner.ID,
			OfferID:    offerID,
			References: references,
		},
	)
}

func (s *Service) routeTenderedShipmentStatus(
	ctx context.Context,
	file *edi.EDIInboundFile,
	partner *edi.EDIPartner,
	details *shipmentStatusDetails,
	reference string,
) ([]string, error) {
	candidates := make([]string, 0, len(details.references)+2)
	candidates = append(candidates, details.shipmentRef, details.referenceID)
	candidates = append(candidates, details.references...)
	offerID, references := splitTenderOfferCandidates(candidates)
	offer, err := s.resolveAcceptedTenderOffer(ctx, file, partner, offerID, references)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return []string{fmt.Sprintf(
				"shipment status for reference %s could not be matched to an outbound tender or an accepted tender offer",
				reference,
			)}, nil
		}
		return nil, err
	}
	warnings, err := s.ediService.RecordTenderedShipmentStatus(
		ctx,
		&ediservice.RecordTenderedShipmentStatusRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: file.OrganizationID,
				BuID:  file.BusinessUnitID,
			},
			Offer:       offer,
			StatusCode:  details.statusCode,
			ReasonCode:  details.reasonCode,
			ReferenceID: details.referenceID,
			EventAt:     details.eventAt,
		},
	)
	if err != nil {
		if errortypes.IsBusinessError(err) {
			s.l.Info(
				"EDI tendered carrier shipment status was not applied",
				zap.String("offerId", offer.ID.String()),
				zap.String("fileId", file.ID.String()),
				zap.Error(err),
			)
			return []string{fmt.Sprintf(
				"shipment status for tender offer %s was not applied: %s",
				offer.ID,
				err.Error(),
			)}, nil
		}
		return nil, err
	}
	if len(warnings) > 0 {
		s.l.Info(
			"EDI tendered carrier shipment status produced stop actual warnings",
			zap.String("offerId", offer.ID.String()),
			zap.String("fileId", file.ID.String()),
			zap.Strings("warnings", warnings),
		)
	}
	return warnings, nil
}

func (s *Service) resolveFreightInvoiceOffer(
	ctx context.Context,
	file *edi.EDIInboundFile,
	partner *edi.EDIPartner,
	payload *edi.FreightInvoicePayload,
) (*tender.TenderOffer, []string, error) {
	referenceCandidates := make([]string, 0, len(payload.ReferenceNumbers))
	for _, value := range payload.ReferenceNumbers {
		referenceCandidates = append(referenceCandidates, value)
	}
	offerID, _ := splitTenderOfferCandidates(referenceCandidates)
	references := stringutils.NonEmptyStrings(
		payload.ReferenceNumbers["shipmentId"],
		payload.BOL,
		payload.ProNumber,
	)
	if offerID.IsNil() && len(references) == 0 {
		return nil, nil, nil
	}
	offer, err := s.resolveAcceptedTenderOffer(ctx, file, partner, offerID, references)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, []string{fmt.Sprintf(
				"freight invoice %s could not be matched to an outbound tender or an accepted tender offer",
				payload.InvoiceNumber,
			)}, nil
		}
		return nil, nil, err
	}
	return offer, nil, nil
}

// autoMatchInboundFreightInvoice hands a pre-linked tendered-carrier 210 to the
// settlement auto-matcher. Failures degrade to file warnings — inbound document
// routing never fails because reconciliation could not run.
func (s *Service) autoMatchInboundFreightInvoice(
	ctx context.Context,
	file *edi.EDIInboundFile,
	invoice *edi.CarrierInvoice,
) []string {
	if s.invoiceMatcher == nil || invoice == nil {
		return nil
	}
	if invoice.CarrierID.IsNil() ||
		(invoice.ShipmentID.IsNil() && invoice.ProNumber == "") {
		return nil
	}
	result, err := s.invoiceMatcher.AutoMatchInboundInvoice(
		ctx,
		&services.AutoMatchInboundInvoiceRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: file.OrganizationID,
				BuID:  file.BusinessUnitID,
			},
			EDICarrierInvoiceID: invoice.ID,
		},
	)
	if err != nil {
		s.l.Warn(
			"failed to auto-match inbound EDI carrier invoice",
			zap.String("invoiceId", invoice.ID.String()),
			zap.String("fileId", file.ID.String()),
			zap.Error(err),
		)
		return []string{fmt.Sprintf(
			"carrier invoice %s could not be auto-matched: %s",
			invoice.InvoiceNumber,
			err.Error(),
		)}
	}
	if result.Matched {
		s.l.Info(
			"auto-matched inbound EDI carrier invoice",
			zap.String("invoiceId", invoice.ID.String()),
			zap.String("fileId", file.ID.String()),
			zap.String("status", result.Status),
			zap.Bool("autoAccepted", result.AutoAccepted),
		)
	}
	return result.Warnings
}
