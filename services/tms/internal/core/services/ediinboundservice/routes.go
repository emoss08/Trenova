package ediinboundservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/ediservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
)

func (s *Service) routeAcknowledgment(
	ctx context.Context,
	file *edi.EDIInboundFile,
	partner *edi.EDIPartner,
	message *edi.EDIMessage,
	transaction *parsedTransaction,
) []string {
	entries := parseAcknowledgments(transaction)
	if len(entries) == 0 {
		return []string{fmt.Sprintf(
			"acknowledgment %s/%s does not reference any transactions",
			transaction.set,
			transaction.controlNumber,
		)}
	}
	tenantInfo := pagination.TenantInfo{
		OrgID: file.OrganizationID,
		BuID:  file.BusinessUnitID,
	}
	warnings := make([]string, 0)
	now := timeutils.NowUnix()
	for index := range entries {
		entry := &entries[index]
		if entry.originalControlNumber == "" {
			warnings = append(warnings, fmt.Sprintf(
				"acknowledgment %s/%s is missing the original transaction control number",
				transaction.set,
				transaction.controlNumber,
			))
			continue
		}
		original, err := s.messageRepo.GetOutboundMessageForAck(
			ctx,
			repositories.GetEDIOutboundMessageForAckRequest{
				TenantInfo:               tenantInfo,
				PartnerID:                partner.ID,
				TransactionSet:           transactionSetFromCode(entry.originalTransactionSet),
				GroupControlNumber:       entry.originalGroupControl,
				TransactionControlNumber: entry.originalControlNumber,
			},
		)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf(
				"acknowledgment for control number %s could not be matched to an outbound message",
				entry.originalControlNumber,
			))
			continue
		}
		ackStatus, ackError := acknowledgmentResolution(entry)
		if _, err = s.messageRepo.UpdateMessageAcknowledgment(
			ctx,
			&repositories.UpdateEDIMessageAcknowledgmentRequest{
				ID:            original.ID,
				TenantInfo:    tenantInfo,
				AckStatus:     ackStatus,
				AckMessageID:  message.ID,
				AckReceivedAt: &now,
				AckLastError:  ackError,
			},
		); err != nil {
			warnings = append(warnings, fmt.Sprintf(
				"failed to record acknowledgment for message %s: %s",
				original.ID,
				err.Error(),
			))
		}
	}
	return warnings
}

func acknowledgmentResolution(
	entry *acknowledgmentEntry,
) (status edi.MessageAcknowledgmentStatus, ackError string) {
	code := entry.acknowledgmentCode
	if code == "" {
		code = entry.groupAcknowledgmentCode
	}
	switch code {
	case "A", "E":
		return edi.MessageAcknowledgmentStatusAccepted, ""
	default:
		details := make([]string, 0, len(entry.diagnostics)+1)
		details = append(details, "acknowledgment code "+stringutils.FirstNonEmpty(code, "missing"))
		for _, diagnostic := range entry.diagnostics {
			detail := diagnostic.Message
			if diagnostic.ErrorCode != "" {
				detail += " (code " + diagnostic.ErrorCode + ")"
			}
			if diagnostic.SegmentID != "" {
				detail += " segment " + diagnostic.SegmentID
			}
			details = append(details, detail)
		}
		return edi.MessageAcknowledgmentStatusRejected, joinDetails(details)
	}
}

func (s *Service) routeTenderResponse(
	ctx context.Context,
	file *edi.EDIInboundFile,
	partner *edi.EDIPartner,
	transaction *parsedTransaction,
) ([]string, error) {
	details := parseTenderResponse(transaction)
	if details.shipmentRef == "" {
		return []string{fmt.Sprintf(
			"tender response %s/%s does not carry a shipment reference",
			transaction.set,
			transaction.controlNumber,
		)}, nil
	}
	recipient, err := s.tenderRecipientRepo.GetActiveExternalRecipientByShipmentReference(
		ctx,
		repositories.GetActiveExternalEDITenderRecipientByReferenceRequest{
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
			return []string{fmt.Sprintf(
				"tender response for reference %s could not be matched to an outbound tender",
				details.shipmentRef,
			)}, nil
		}
		return nil, err
	}
	return nil, s.ediService.ApplyExternalTenderResponse(
		ctx,
		&ediservice.ApplyExternalTenderResponseRequest{
			Recipient:       recipient,
			Accepted:        details.reservationCode == "A",
			ReasonCode:      details.reservationCode,
			RejectionReason: details.remarks,
		},
	)
}

func (s *Service) routeShipmentStatus(
	ctx context.Context,
	file *edi.EDIInboundFile,
	partner *edi.EDIPartner,
	transaction *parsedTransaction,
) ([]string, error) {
	details := parseShipmentStatus(transaction)
	reference := stringutils.FirstNonEmpty(details.shipmentRef, details.referenceID)
	if reference == "" {
		return []string{fmt.Sprintf(
			"shipment status %s/%s does not carry a shipment reference",
			transaction.set,
			transaction.controlNumber,
		)}, nil
	}
	if details.statusCode == "" {
		return []string{fmt.Sprintf(
			"shipment status %s/%s does not carry an AT7 status code",
			transaction.set,
			transaction.controlNumber,
		)}, nil
	}
	recipient, err := s.tenderRecipientRepo.GetActiveExternalRecipientByShipmentReference(
		ctx,
		repositories.GetActiveExternalEDITenderRecipientByReferenceRequest{
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
			return []string{fmt.Sprintf(
				"shipment status for reference %s could not be matched to an outbound tender",
				reference,
			)}, nil
		}
		return nil, err
	}
	return nil, s.ediService.RecordExternalShipmentStatus(
		ctx,
		&ediservice.RecordExternalShipmentStatusRequest{
			Recipient:   recipient,
			StatusCode:  details.statusCode,
			ReasonCode:  details.reasonCode,
			ReferenceID: details.referenceID,
			EventAt:     details.eventAt,
		},
	)
}

func (s *Service) routeFreightInvoice(
	ctx context.Context,
	file *edi.EDIInboundFile,
	partner *edi.EDIPartner,
	message *edi.EDIMessage,
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
	result, err := s.ediService.RecordInboundFreightInvoice(
		ctx,
		&ediservice.RecordInboundFreightInvoiceRequest{
			Partner:   partner,
			Message:   message,
			Recipient: recipient,
			Payload:   payload,
		},
	)
	if err != nil {
		return warnings, err
	}
	return append(warnings, result.Warnings...), nil
}

func (s *Service) freightInvoiceRecipient(
	ctx context.Context,
	file *edi.EDIInboundFile,
	partner *edi.EDIPartner,
	payload *edi.FreightInvoicePayload,
) (*edi.TenderRecipient, []string, error) {
	references := stringutils.NonEmptyStrings(
		payload.ReferenceNumbers["shipmentId"],
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
			repositories.GetActiveExternalEDITenderRecipientByReferenceRequest{
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
	file *edi.EDIInboundFile,
	partner *edi.EDIPartner,
	message *edi.EDIMessage,
	transaction *parsedTransaction,
) ([]string, error) {
	payload := message.PayloadSnapshot.LoadTender
	if payload == nil {
		return nil, fmt.Errorf(
			"load tender %s/%s could not be parsed into a tender payload",
			transaction.set,
			transaction.controlNumber,
		)
	}
	if !partner.EnabledForInbound {
		return []string{fmt.Sprintf(
			"load tender %s/%s recorded without processing: partner %s is disabled for inbound",
			transaction.set,
			transaction.controlNumber,
			partner.Code,
		)}, nil
	}
	tenantInfo := pagination.TenantInfo{
		OrgID: file.OrganizationID,
		BuID:  file.BusinessUnitID,
	}
	warnings := make([]string, 0)
	if payload.PurposeCode == edi.LoadTenderPurposeChange {
		supersedeWarnings, err := s.supersedePriorInboundTransfer(
			ctx,
			tenantInfo,
			partner,
			payload,
			transaction,
		)
		if err != nil {
			return warnings, err
		}
		warnings = append(warnings, supersedeWarnings...)
	}

	preview, err := s.ediService.BuildInboundMappingPreview(ctx, partner, *payload)
	if err != nil {
		return warnings, err
	}
	status := edi.TransferStatusPendingApproval
	if len(preview.Unresolved) > 0 {
		status = edi.TransferStatusMappingRequired
	}
	transfer := &edi.EDITransfer{
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
	if _, err = s.transferRepo.CreateTransfer(ctx, transfer); err != nil {
		return warnings, err
	}
	if status == edi.TransferStatusMappingRequired {
		warnings = append(warnings, fmt.Sprintf(
			"load tender %s/%s requires mapping before approval (%d unresolved entities)",
			transaction.set,
			transaction.controlNumber,
			len(preview.Unresolved),
		))
	}
	return warnings, nil
}

func (s *Service) supersedePriorInboundTransfer(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	partner *edi.EDIPartner,
	payload *edi.LoadTenderPayload,
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
		repositories.GetActionableInboundEDITransferByExternalReferenceRequest{
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
	prior.Status = edi.TransferStatusCanceled
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
