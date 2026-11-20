package ediinboundservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/edi/app/ediinboundapply"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edicarrierinvoice"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ediinboundfile"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edipayload"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/operations/ports/tenderports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

var acceptedOfferStatuses = []tender.OfferStatus{tender.OfferStatusAccepted}

func (s *Service) resolvePartnerOffer(
	ctx context.Context,
	file *ediinboundfile.EDIInboundFile,
	partner *edi.EDIPartner,
	references []string,
	statuses ...tender.OfferStatus,
) (*tender.TenderOffer, error) {
	keys := tender.ParsePartnerKeys(references...)
	if len(keys) == 0 {
		return nil, errortypes.NewNotFoundError("Tender offer")
	}
	return s.tenderRepo.GetOfferForPartnerKey(ctx, tenderports.GetOfferForPartnerKeyRequest{
		TenantInfo: inboundFileTenant(file),
		PartnerID:  partner.ID,
		Keys:       keys,
		Statuses:   statuses,
	})
}

func (s *Service) routeTenderedShipmentStatus(
	ctx context.Context,
	file *ediinboundfile.EDIInboundFile,
	partner *edi.EDIPartner,
	details *shipmentStatusDetails,
	reference string,
) ([]string, error) {
	references := make([]string, 0, len(details.references)+1)
	references = append(references, details.shipmentRef)
	references = append(references, details.references...)
	offer, err := s.resolvePartnerOffer(ctx, file, partner, references, acceptedOfferStatuses...)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return []string{
				fmt.Sprintf(
					"shipment status for reference %s could not be matched to an outbound tender or an accepted tender offer",
					reference,
				),
			}, nil
		}
		return nil, err
	}
	warnings := make([]string, 0, len(details.events))
	for index := range details.events {
		event := &details.events[index]
		eventWarnings, recordErr := s.recordTenderedStatusEvent(ctx, file, offer, details, event)
		if recordErr != nil {
			return nil, recordErr
		}
		warnings = append(warnings, eventWarnings...)
	}
	if len(warnings) == 0 {
		return nil, nil
	}
	s.l.Info(
		"EDI tendered carrier shipment status produced stop actual warnings",
		zap.String("offerId", offer.ID.String()),
		zap.String("fileId", file.ID.String()),
		zap.Strings("warnings", warnings),
	)
	return warnings, nil
}

func (s *Service) recordTenderedStatusEvent(
	ctx context.Context,
	file *ediinboundfile.EDIInboundFile,
	offer *tender.TenderOffer,
	details *shipmentStatusDetails,
	event *edipayload.ShipmentStatusEvent,
) ([]string, error) {
	statusCode := statusEventCode(event)
	warnings, err := s.inboundApply.RecordTenderedShipmentStatus(
		ctx,
		&ediinboundapply.RecordTenderedShipmentStatusRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: file.OrganizationID,
				BuID:  file.BusinessUnitID,
			},
			Offer:        offer,
			StatusCode:   statusCode,
			ReasonCode:   statusEventReason(event),
			ReferenceID:  details.referenceID,
			EventAt:      event.EventAt,
			Event:        event,
			RawReference: statusEventReference(file, details, event),
		},
	)
	if err != nil {
		if errortypes.IsBusinessError(err) {
			s.l.Info(
				"EDI tendered carrier shipment status was not applied",
				zap.String("offerId", offer.ID.String()),
				zap.String("fileId", file.ID.String()),
				zap.String("statusCode", statusCode),
				zap.Error(err),
			)
			return []string{fmt.Sprintf(
				"shipment status %s for tender offer %s was not applied: %s",
				statusCode,
				offer.ID,
				err.Error(),
			)}, nil
		}
		return nil, err
	}
	return warnings, nil
}

func (s *Service) resolveFreightInvoiceOffer(
	ctx context.Context,
	file *ediinboundfile.EDIInboundFile,
	partner *edi.EDIPartner,
	payload *edipayload.FreightInvoicePayload,
) (*tender.TenderOffer, []string, error) {
	references := make([]string, 0, len(payload.ReferenceNumbers)+2)
	references = append(references, payload.Key, payload.BOL)
	for _, value := range payload.ReferenceNumbers {
		references = append(references, value)
	}
	if len(tender.ParsePartnerKeys(references...)) == 0 {
		return nil, nil, nil
	}
	offer, err := s.resolvePartnerOffer(ctx, file, partner, references, acceptedOfferStatuses...)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, []string{
				fmt.Sprintf(
					"freight invoice %s could not be matched to an outbound tender or an accepted tender offer",
					payload.InvoiceNumber,
				),
			}, nil
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
	file *ediinboundfile.EDIInboundFile,
	invoice *edicarrierinvoice.CarrierInvoice,
) []string {
	if invoice == nil {
		return nil
	}
	if invoice.CarrierID.IsNil() ||
		(invoice.ShipmentID.IsNil() && invoice.ProNumber == "") {
		return nil
	}
	result, err := s.invoiceMatcher.AutoMatchInboundInvoice(
		ctx,
		&carrierports.AutoMatchInboundInvoiceRequest{
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
