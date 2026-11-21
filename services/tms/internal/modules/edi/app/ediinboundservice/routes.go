package ediinboundservice

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/edi/app/ediinboundapply"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ediinboundfile"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edimessage"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edipayload"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editenderchange"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editenderports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/operations/ports/tenderports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

func (s *Service) routeTenderResponse(
	ctx context.Context,
	file *ediinboundfile.EDIInboundFile,
	partner *edi.EDIPartner,
	transaction *parsedTransaction,
	details *tenderResponseDetails,
) ([]string, error) {
	outcome := s.routeTenderOfferResponse(ctx, file, partner, transaction, details)
	if outcome.handled || outcome.err != nil {
		return outcome.warnings, outcome.err
	}
	warnings := outcome.warnings
	if details.shipmentRef == "" {
		return append(warnings, fmt.Sprintf(
			"tender response %s/%s does not carry a shipment reference",
			transaction.set,
			transaction.controlNumber,
		)), nil
	}
	recipient, err := s.tenderRecipientRepo.GetActiveExternalRecipientByShipmentReference(
		ctx,
		editenderports.GetActiveExternalEDITenderRecipientByReferenceRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: file.OrganizationID,
				BuID:  file.BusinessUnitID,
			},
			PartnerID: partner.ID,
			Reference: details.shipmentRef,
		},
	)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return append(warnings, fmt.Sprintf(
				"tender response for reference %s could not be matched to an outbound tender",
				details.shipmentRef,
			)), nil
		}
		return warnings, err
	}
	return warnings, s.inboundApply.ApplyExternalTenderResponse(
		ctx,
		&ediinboundapply.ApplyExternalTenderResponseRequest{
			Recipient:       recipient,
			Accepted:        details.reservationCode == "A",
			ReasonCode:      details.reservationCode,
			RejectionReason: details.remarks,
		},
	)
}

type tenderOfferRouteOutcome struct {
	handled  bool
	warnings []string
	err      error
}

func (s *Service) routeTenderOfferResponse(
	ctx context.Context,
	file *ediinboundfile.EDIInboundFile,
	partner *edi.EDIPartner,
	transaction *parsedTransaction,
	details *tenderResponseDetails,
) tenderOfferRouteOutcome {
	references := make([]string, 0, len(details.references)+1)
	references = append(references, details.shipmentRef)
	references = append(references, details.references...)
	offer, err := s.resolvePartnerOffer(ctx, file, partner, references)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return tenderOfferRouteOutcome{}
		}
		return tenderOfferRouteOutcome{handled: true, err: err}
	}

	var action tender.ResponseAction
	declineReason := ""
	switch {
	case details.reservationCode == reservationCodeAccepted:
		action = tender.ResponseActionAccept
	case isReservationDecline(details.reservationCode):
		action = tender.ResponseActionDecline
		declineReason = stringutils.FirstNonEmpty(details.remarks, details.reservationCode)
	case details.reservationCode == "":
		return tenderOfferRouteOutcome{handled: true, warnings: []string{fmt.Sprintf(
			"tender response %s/%s for %s carries no reservation code; response not applied",
			transaction.set,
			transaction.controlNumber,
			details.shipmentRef,
		)}}
	default:
		return tenderOfferRouteOutcome{handled: true, warnings: []string{
			fmt.Sprintf(
				"tender response %s/%s for %s carries unknown reservation code %q; response not applied",
				transaction.set,
				transaction.controlNumber,
				details.shipmentRef,
				details.reservationCode,
			),
		}}
	}
	err = s.tenderResponses.RecordResponse(ctx, &tenderports.TenderResponseRequest{
		TenantInfo:    inboundFileTenant(file),
		OfferID:       offer.ID,
		Action:        action,
		Source:        tender.ResponseSourceEDI,
		DeclineReason: declineReason,
	})
	if err != nil {
		if errortypes.IsBusinessError(err) {
			s.l.Info(
				"EDI tender offer response was not applied",
				zap.String("offerId", offer.ID.String()),
				zap.String("fileId", file.ID.String()),
				zap.Error(err),
			)
			return tenderOfferRouteOutcome{handled: true, warnings: []string{fmt.Sprintf(
				"tender response for %s was not applied: %s",
				details.shipmentRef,
				err.Error(),
			)}}
		}
		return tenderOfferRouteOutcome{handled: true, err: err}
	}
	return tenderOfferRouteOutcome{handled: true}
}

func (s *Service) routeShipmentStatus(
	ctx context.Context,
	file *ediinboundfile.EDIInboundFile,
	partner *edi.EDIPartner,
	transaction *parsedTransaction,
	details *shipmentStatusDetails,
) ([]string, error) {
	reference := stringutils.FirstNonEmpty(details.shipmentRef, details.referenceID)
	if reference == "" {
		return []string{fmt.Sprintf(
			"shipment status %s/%s does not carry a shipment reference",
			transaction.set,
			transaction.controlNumber,
		)}, nil
	}
	if len(details.events) == 0 {
		return []string{fmt.Sprintf(
			"shipment status %s/%s does not carry an AT7 status code",
			transaction.set,
			transaction.controlNumber,
		)}, nil
	}
	recipient, err := s.tenderRecipientRepo.GetActiveExternalRecipientByShipmentReference(
		ctx,
		editenderports.GetActiveExternalEDITenderRecipientByReferenceRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: file.OrganizationID,
				BuID:  file.BusinessUnitID,
			},
			PartnerID: partner.ID,
			Reference: reference,
		},
	)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return s.routeTenderedShipmentStatus(ctx, file, partner, details, reference)
		}
		return nil, err
	}
	for index := range details.events {
		event := &details.events[index]
		if err = s.inboundApply.RecordExternalShipmentStatus(
			ctx,
			&ediinboundapply.RecordExternalShipmentStatusRequest{
				Recipient:   recipient,
				StatusCode:  statusEventCode(event),
				ReasonCode:  statusEventReason(event),
				ReferenceID: details.referenceID,
				EventAt:     event.EventAt,
			},
		); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func statusEventCode(event *edipayload.ShipmentStatusEvent) string {
	return stringutils.FirstNonEmpty(event.StatusCode, event.AppointmentCode)
}

func statusEventReason(event *edipayload.ShipmentStatusEvent) string {
	if event.StatusCode != "" {
		return event.ReasonCode
	}
	return event.AppointmentReasonCode
}

func statusEventReference(
	file *ediinboundfile.EDIInboundFile,
	details *shipmentStatusDetails,
	event *edipayload.ShipmentStatusEvent,
) string {
	return "edi_inbound_file:" + file.ID.String() + ":" + details.controlNumber + ":" +
		strconv.Itoa(event.Index)
}

func (s *Service) routeFreightInvoice(
	ctx context.Context,
	file *ediinboundfile.EDIInboundFile,
	partner *edi.EDIPartner,
	message *edimessage.EDIMessage,
	transaction *parsedTransaction,
) ([]string, error) {
	payload := message.PayloadSnapshot.FreightInvoice
	if payload == nil {
		return nil, fmt.Errorf(
			"freight invoice %s/%s could not be parsed into an invoice payload",
			transaction.set,
			transaction.controlNumber,
		)
	}
	if !partner.EnabledForInbound {
		return []string{fmt.Sprintf(
			"freight invoice %s/%s recorded without processing: partner %s is disabled for inbound",
			transaction.set,
			transaction.controlNumber,
			partner.Code,
		)}, nil
	}
	if payload.InvoiceNumber == "" {
		return []string{fmt.Sprintf(
			"freight invoice %s/%s does not carry a B3 invoice number",
			transaction.set,
			transaction.controlNumber,
		)}, nil
	}
	recipient, warnings, err := s.freightInvoiceRecipient(ctx, file, partner, payload)
	if err != nil {
		return warnings, err
	}
	var acceptedOffer *tender.TenderOffer
	if recipient == nil {
		offer, offerWarnings, offerErr := s.resolveFreightInvoiceOffer(ctx, file, partner, payload)
		if offerErr != nil {
			return warnings, offerErr
		}
		warnings = append(warnings, offerWarnings...)
		acceptedOffer = offer
	}
	result, err := s.inboundApply.RecordInboundFreightInvoice(
		ctx,
		&ediinboundapply.RecordInboundFreightInvoiceRequest{
			Partner:       partner,
			Message:       message,
			Recipient:     recipient,
			AcceptedOffer: acceptedOffer,
			Payload:       payload,
		},
	)
	if err != nil {
		return warnings, err
	}
	warnings = append(warnings, result.Warnings...)
	if acceptedOffer != nil {
		warnings = append(
			warnings,
			s.autoMatchInboundFreightInvoice(ctx, file, result.Invoice)...)
	}
	return warnings, nil
}

func (s *Service) freightInvoiceRecipient(
	ctx context.Context,
	file *ediinboundfile.EDIInboundFile,
	partner *edi.EDIPartner,
	payload *edipayload.FreightInvoicePayload,
) (*editenderchange.TenderRecipient, []string, error) {
	references := stringutils.NonEmptyStrings(
		payload.Key,
		payload.BOL,
		payload.ProNumber,
	)
	tenantInfo := pagination.TenantInfo{
		OrgID: file.OrganizationID,
		BuID:  file.BusinessUnitID,
	}
	for _, reference := range references {
		recipient, err := s.tenderRecipientRepo.GetActiveExternalRecipientByShipmentReference(
			ctx,
			editenderports.GetActiveExternalEDITenderRecipientByReferenceRequest{
				TenantInfo: tenantInfo,
				PartnerID:  partner.ID,
				Reference:  reference,
			},
		)
		if err != nil {
			if errortypes.IsNotFoundError(err) {
				continue
			}
			return nil, nil, err
		}
		return recipient, nil, nil
	}
	if len(references) == 0 {
		return nil, []string{fmt.Sprintf(
			"freight invoice %s does not carry a shipment reference",
			payload.InvoiceNumber,
		)}, nil
	}
	return nil, nil, nil
}

func (s *Service) routeLoadTender(
	ctx context.Context,
	file *ediinboundfile.EDIInboundFile,
	partner *edi.EDIPartner,
	message *edimessage.EDIMessage,
	transaction *parsedTransaction,
) (*editransfer.EDITransfer, []string, error) {
	payload := message.PayloadSnapshot.LoadTender
	if payload == nil {
		return nil, nil, fmt.Errorf(
			"load tender %s/%s could not be parsed into a tender payload",
			transaction.set,
			transaction.controlNumber,
		)
	}
	if !partner.EnabledForInbound {
		return nil, []string{fmt.Sprintf(
			"load tender %s/%s recorded without processing: partner %s is disabled for inbound",
			transaction.set,
			transaction.controlNumber,
			partner.Code,
		)}, nil
	}
	tenantInfo := inboundFileTenant(file)
	warnings := make([]string, 0)
	if payload.PurposeCode == editransfer.LoadTenderPurposeChange {
		supersedeWarnings, err := s.supersedePriorInboundTransfer(
			ctx,
			tenantInfo,
			partner,
			payload,
			transaction,
		)
		if err != nil {
			return nil, warnings, err
		}
		warnings = append(warnings, supersedeWarnings...)
	}

	preview, err := s.inboundApply.BuildInboundMappingPreview(ctx, partner, payload)
	if err != nil {
		return nil, warnings, err
	}
	status := editransfer.TransferStatusPendingApproval
	if len(preview.Unresolved) > 0 {
		status = editransfer.TransferStatusMappingRequired
	}
	transfer := &editransfer.EDITransfer{
		SourceOrganizationID: file.OrganizationID,
		SourceBusinessUnitID: file.BusinessUnitID,
		TargetOrganizationID: file.OrganizationID,
		TargetBusinessUnitID: file.BusinessUnitID,
		SourcePartnerID:      partner.ID,
		TargetPartnerID:      partner.ID,
		Status:               status,
		TenderPayload:        *payload,
		MappingSnapshot:      preview.All,
		InboundMessageID:     message.ID,
		SubmittedAt:          timeutils.NowUnix(),
	}
	created, err := s.transferRepo.CreateTransfer(ctx, transfer)
	if err != nil {
		return nil, warnings, err
	}
	if status == editransfer.TransferStatusMappingRequired {
		warnings = append(warnings, fmt.Sprintf(
			"load tender %s/%s requires mapping before approval (%d unresolved entities)",
			transaction.set,
			transaction.controlNumber,
			len(preview.Unresolved),
		))
	}
	return created, warnings, nil
}

func (s *Service) supersedePriorInboundTransfer(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	partner *edi.EDIPartner,
	payload *editransfer.LoadTenderPayload,
	transaction *parsedTransaction,
) ([]string, error) {
	externalRef, _ := payload.RatingDetail["externalShipmentId"].(string)
	if externalRef == "" {
		return []string{
			fmt.Sprintf(
				"load tender change %s/%s has no shipment identification number; created a new transfer",
				transaction.set,
				transaction.controlNumber,
			),
		}, nil
	}
	prior, err := s.transferRepo.GetActionableInboundTransferByExternalReference(
		ctx,
		editenderports.GetActionableInboundEDITransferByExternalReferenceRequest{
			TenantInfo:        tenantInfo,
			PartnerID:         partner.ID,
			ExternalReference: externalRef,
		},
	)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return []string{fmt.Sprintf(
				"load tender change %s/%s did not match a pending tender for %s; review manually",
				transaction.set,
				transaction.controlNumber,
				externalRef,
			)}, nil
		}
		return nil, err
	}
	now := timeutils.NowUnix()
	prior.Status = editransfer.TransferStatusCanceled
	prior.FailureReason = fmt.Sprintf(
		"Superseded by inbound load tender change %s",
		transaction.controlNumber,
	)
	prior.ProcessedAt = &now
	if _, err = s.transferRepo.UpdateTransfer(ctx, prior); err != nil {
		return nil, err
	}
	return nil, nil
}

func joinDetails(details []string) string {
	return strings.Join(details, "; ")
}
