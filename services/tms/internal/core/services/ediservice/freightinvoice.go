package ediservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type RecordInboundFreightInvoiceRequest struct {
	Partner   *edi.EDIPartner
	Message   *edi.EDIMessage
	Recipient *edi.TenderRecipient
	Payload   *edi.FreightInvoicePayload
}

type RecordInboundFreightInvoiceResult struct {
	Invoice  *edi.CarrierInvoice
	Warnings []string
}

func (s *Service) RecordInboundFreightInvoice(
	ctx context.Context,
	req *RecordInboundFreightInvoiceRequest,
) (*RecordInboundFreightInvoiceResult, error) {
	if req == nil || req.Partner == nil || req.Message == nil || req.Payload == nil {
		return nil, errortypes.NewValidationError(
			"payload",
			errortypes.ErrRequired,
			"Partner, message, and freight invoice payload are required",
		)
	}

	entity := carrierInvoiceFromPayload(req.Partner, req.Message, req.Payload)
	warnings := make([]string, 0, 2)
	notes := make([]string, 0, 2)

	customerResolved, err := s.resolveCarrierInvoiceCustomer(ctx, req.Partner, entity)
	if err != nil {
		return nil, err
	}

	if req.Recipient != nil {
		entity.TenderRecipientID = req.Recipient.ID
		entity.ShipmentID = req.Recipient.SourceShipmentID
		entity.ExpectedAmount = req.Recipient.LatestBaselinePayload.TotalChargeAmount
	}

	entity.ReconciliationStatus = reconcileCarrierInvoice(entity, customerResolved, &notes)
	entity.ReconciliationNotes = strings.Join(notes, "; ")
	warnings = append(warnings, notes...)

	invoice, err := s.carrierInvoiceRepo.CreateCarrierInvoice(ctx, entity)
	if err != nil {
		return nil, err
	}

	if req.Recipient != nil && req.Recipient.SourceShipmentID.IsNotNil() {
		s.recordCarrierInvoiceShipmentComment(ctx, req.Recipient, invoice)
	}
	return &RecordInboundFreightInvoiceResult{Invoice: invoice, Warnings: warnings}, nil
}

func carrierInvoiceFromPayload(
	partner *edi.EDIPartner,
	message *edi.EDIMessage,
	payload *edi.FreightInvoicePayload,
) *edi.CarrierInvoice {
	entity := &edi.CarrierInvoice{
		BusinessUnitID:    message.BusinessUnitID,
		OrganizationID:    message.OrganizationID,
		EDIPartnerID:      partner.ID,
		InboundMessageID:  message.ID,
		InvoiceNumber:     payload.InvoiceNumber,
		ShipmentReference: payload.ReferenceNumbers["shipmentId"],
		BOL:               payload.BOL,
		ProNumber:         payload.ProNumber,
		BillToName:        payload.BillToName,
		CurrencyCode:      payload.CurrencyCode,
		TotalAmount:       payload.TotalAmount,
		LineCharges:       payload.LineCharges,
		ReferenceNumbers:  payload.ReferenceNumbers,
	}
	if payload.InvoiceDate > 0 {
		invoiceDate := payload.InvoiceDate
		entity.InvoiceDate = &invoiceDate
	}
	if payload.DeliveryDate > 0 {
		deliveryDate := payload.DeliveryDate
		entity.DeliveryDate = &deliveryDate
	}
	return entity
}

func (s *Service) resolveCarrierInvoiceCustomer(
	ctx context.Context,
	partner *edi.EDIPartner,
	entity *edi.CarrierInvoice,
) (bool, error) {
	if entity.BillToName == "" {
		return false, nil
	}
	entity.BillToSourceID = edi.MappingSourceID(entity.BillToName)
	items, err := s.mappingProfileRepo.GetMappingItems(ctx, repositories.GetMappingItemsRequest{
		PartnerID: partner.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: partner.OrganizationID,
			BuID:  partner.BusinessUnitID,
		},
		EntityTypes: []edi.MappingEntityType{edi.MappingEntityTypeCustomer},
		SourceIDs:   []pulid.ID{entity.BillToSourceID},
	})
	if err != nil {
		return false, err
	}
	index := mappingIndex(items)
	if item := index[edi.MappingEntityTypeCustomer][entity.BillToSourceID]; item != nil &&
		item.TargetID.IsNotNil() {
		entity.CustomerID = item.TargetID
		return true, nil
	}
	return false, nil
}

func reconcileCarrierInvoice(
	entity *edi.CarrierInvoice,
	customerResolved bool,
	notes *[]string,
) edi.CarrierInvoiceReconciliationStatus {
	status := edi.CarrierInvoiceReconciliationStatusMatched
	if entity.TotalAmount.Valid && entity.ExpectedAmount.Valid {
		variance := entity.TotalAmount.Decimal.Sub(entity.ExpectedAmount.Decimal)
		entity.VarianceAmount = decimal.NullDecimal{Decimal: variance, Valid: true}
		if !variance.IsZero() {
			status = edi.CarrierInvoiceReconciliationStatusVariance
			*notes = append(*notes, fmt.Sprintf(
				"carrier invoice %s differs from the tendered amount by %s",
				entity.InvoiceNumber,
				variance.String(),
			))
		}
	} else {
		status = edi.CarrierInvoiceReconciliationStatusVariance
		*notes = append(*notes, fmt.Sprintf(
			"carrier invoice %s cannot be compared: invoice or tendered amount is missing",
			entity.InvoiceNumber,
		))
	}
	if entity.BillToName != "" && !customerResolved {
		status = edi.CarrierInvoiceReconciliationStatusMappingRequired
		*notes = append(*notes, fmt.Sprintf(
			"bill-to party %q has no customer mapping for this partner",
			entity.BillToName,
		))
	}
	if entity.ShipmentID.IsNil() {
		status = edi.CarrierInvoiceReconciliationStatusUnmatched
		*notes = append(*notes, fmt.Sprintf(
			"carrier invoice %s could not be matched to an outbound tender",
			entity.InvoiceNumber,
		))
	}
	return status
}

func (s *Service) recordCarrierInvoiceShipmentComment(
	ctx context.Context,
	recipient *edi.TenderRecipient,
	invoice *edi.CarrierInvoice,
) {
	comment := fmt.Sprintf(
		"EDI 210 carrier invoice %s received from trading partner",
		invoice.InvoiceNumber,
	)
	if invoice.TotalAmount.Valid {
		comment += ", total " + invoice.TotalAmount.Decimal.String()
		if invoice.CurrencyCode != "" {
			comment += " " + invoice.CurrencyCode
		}
	}
	switch invoice.ReconciliationStatus {
	case edi.CarrierInvoiceReconciliationStatusMatched:
		comment += "; amount matches the tendered rate"
	case edi.CarrierInvoiceReconciliationStatusVariance:
		if invoice.VarianceAmount.Valid {
			comment += "; variance of " + invoice.VarianceAmount.Decimal.String() +
				" against the tendered rate"
		} else {
			comment += "; amounts could not be compared"
		}
	case edi.CarrierInvoiceReconciliationStatusMappingRequired:
		comment += "; bill-to customer mapping is required"
	case edi.CarrierInvoiceReconciliationStatusUnmatched:
	}
	metadata := map[string]any{
		"carrierInvoiceId":     invoice.ID,
		"invoiceNumber":        invoice.InvoiceNumber,
		"reconciliationStatus": invoice.ReconciliationStatus,
	}
	if invoice.TotalAmount.Valid {
		metadata["totalAmount"] = invoice.TotalAmount.Decimal.String()
	}
	if invoice.VarianceAmount.Valid {
		metadata["varianceAmount"] = invoice.VarianceAmount.Decimal.String()
	}
	tenantInfo := pagination.TenantInfo{
		OrgID: recipient.SourceOrganizationID,
		BuID:  recipient.SourceBusinessUnitID,
	}
	if err := s.createSystemShipmentComment(
		ctx,
		recipient.SourceShipmentID,
		tenantInfo,
		comment,
		metadata,
	); err != nil {
		s.l.Warn(
			"failed to record EDI carrier invoice comment",
			zap.String("carrierInvoiceId", invoice.ID.String()),
			zap.Error(err),
		)
	}
}
