package edi

import (
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type DocumentPayload struct {
	TransactionSet               TransactionSet                   `json:"transactionSet,omitempty"`
	LoadTender                   *LoadTenderPayload               `json:"loadTender,omitempty"`
	Shipment                     *LoadTenderPayload               `json:"shipment,omitempty"`
	FreightInvoice               *FreightInvoicePayload           `json:"invoice,omitempty"`
	ShipmentStatus               *ShipmentStatusPayload           `json:"shipmentStatus,omitempty"`
	TenderResponse               *TenderResponsePayload           `json:"tenderResponse,omitempty"`
	FunctionalAcknowledgment     *FunctionalAcknowledgmentPayload `json:"functionalAck,omitempty"`
	ImplementationAcknowledgment *ImplementationAckPayload        `json:"implementationAck,omitempty"`
}

type FreightInvoicePayload struct {
	InvoiceID          pulid.ID               `json:"invoiceId,omitempty"`
	InvoiceNumber      string                 `json:"invoiceNumber,omitempty"`
	InvoiceDate        int64                  `json:"invoiceDate,omitempty"`
	ShipmentID         pulid.ID               `json:"shipmentId,omitempty"`
	BOL                string                 `json:"bol,omitempty"`
	ProNumber          string                 `json:"proNumber,omitempty"`
	BillToName         string                 `json:"billToName,omitempty"`
	BillToAddressLine1 string                 `json:"billToAddressLine1,omitempty"`
	BillToAddressLine2 string                 `json:"billToAddressLine2,omitempty"`
	BillToCity         string                 `json:"billToCity,omitempty"`
	BillToStateCode    string                 `json:"billToStateCode,omitempty"`
	BillToPostalCode   string                 `json:"billToPostalCode,omitempty"`
	BillToCountry      string                 `json:"billToCountry,omitempty"`
	CurrencyCode       string                 `json:"currencyCode,omitempty"`
	TotalAmount        decimal.NullDecimal    `json:"totalAmount"`
	LineCharges        []FreightInvoiceCharge `json:"lineCharges,omitempty"`
	ReferenceNumbers   map[string]string      `json:"referenceNumbers,omitempty"`
}

type FreightInvoiceCharge struct {
	Sequence    int64           `json:"sequence"`
	Code        string          `json:"code,omitempty"`
	Description string          `json:"description,omitempty"`
	Amount      decimal.Decimal `json:"amount"`
}

type ShipmentStatusPayload struct {
	ShipmentID       pulid.ID          `json:"shipmentId,omitempty"`
	BOL              string            `json:"bol,omitempty"`
	StatusCode       string            `json:"statusCode,omitempty"`
	StatusReasonCode string            `json:"statusReasonCode,omitempty"`
	EventDate        int64             `json:"eventDate,omitempty"`
	EventTime        int64             `json:"eventTime,omitempty"`
	City             string            `json:"city,omitempty"`
	StateCode        string            `json:"stateCode,omitempty"`
	EquipmentNumber  string            `json:"equipmentNumber,omitempty"`
	References       map[string]string `json:"references,omitempty"`
}

type TenderResponsePayload struct {
	TransferID      pulid.ID       `json:"transferId,omitempty"`
	ShipmentID      pulid.ID       `json:"shipmentId,omitempty"`
	BOL             string         `json:"bol,omitempty"`
	ResponseCode    string         `json:"responseCode,omitempty"`
	ReasonCode      string         `json:"reasonCode,omitempty"`
	RejectionReason string         `json:"rejectionReason,omitempty"`
	Status          TransferStatus `json:"status,omitempty"`
}

type FunctionalAcknowledgmentPayload struct {
	SourceMessageID                  pulid.ID                   `json:"sourceMessageId,omitempty"`
	OriginalFunctionalGroupID        string                     `json:"originalFunctionalGroupId,omitempty"`
	OriginalGroupControlNumber       string                     `json:"originalGroupControlNumber,omitempty"`
	OriginalTransactionSet           TransactionSet             `json:"originalTransactionSet,omitempty"`
	OriginalTransactionControlNumber string                     `json:"originalTransactionControlNumber,omitempty"`
	GroupAcknowledgmentCode          string                     `json:"groupAcknowledgmentCode,omitempty"`
	TransactionAcknowledgmentCode    string                     `json:"transactionAcknowledgmentCode,omitempty"`
	AcceptedTransactionSetCount      int64                      `json:"acceptedTransactionSetCount,omitempty"`
	ReceivedTransactionSetCount      int64                      `json:"receivedTransactionSetCount,omitempty"`
	IncludedTransactionSetCount      int64                      `json:"includedTransactionSetCount,omitempty"`
	Diagnostics                      []AcknowledgmentDiagnostic `json:"diagnostics,omitempty"`
}

type ImplementationAckPayload struct {
	SourceMessageID                  pulid.ID                   `json:"sourceMessageId,omitempty"`
	OriginalFunctionalGroupID        string                     `json:"originalFunctionalGroupId,omitempty"`
	OriginalGroupControlNumber       string                     `json:"originalGroupControlNumber,omitempty"`
	OriginalTransactionSet           TransactionSet             `json:"originalTransactionSet,omitempty"`
	OriginalTransactionControlNumber string                     `json:"originalTransactionControlNumber,omitempty"`
	GroupAcknowledgmentCode          string                     `json:"groupAcknowledgmentCode,omitempty"`
	TransactionAcknowledgmentCode    string                     `json:"transactionAcknowledgmentCode,omitempty"`
	AcceptedTransactionSetCount      int64                      `json:"acceptedTransactionSetCount,omitempty"`
	ReceivedTransactionSetCount      int64                      `json:"receivedTransactionSetCount,omitempty"`
	IncludedTransactionSetCount      int64                      `json:"includedTransactionSetCount,omitempty"`
	Diagnostics                      []AcknowledgmentDiagnostic `json:"diagnostics,omitempty"`
}

type AcknowledgmentDiagnostic struct {
	SegmentID       string `json:"segmentId,omitempty"`
	SegmentPosition int64  `json:"segmentPosition,omitempty"`
	ElementPosition int64  `json:"elementPosition,omitempty"`
	ErrorCode       string `json:"errorCode,omitempty"`
	Message         string `json:"message,omitempty"`
}

func NewLoadTenderDocumentPayload(payload LoadTenderPayload) DocumentPayload {
	return DocumentPayload{
		TransactionSet: TransactionSet204,
		LoadTender:     &payload,
		Shipment:       &payload,
	}
}

func (p *DocumentPayload) Normalize() {
	if p.LoadTender == nil && p.Shipment != nil {
		p.LoadTender = p.Shipment
	}
	if p.Shipment == nil && p.LoadTender != nil {
		p.Shipment = p.LoadTender
	}
	if p.TransactionSet == "" {
		switch {
		case p.LoadTender != nil || p.Shipment != nil:
			p.TransactionSet = TransactionSet204
		case p.FreightInvoice != nil:
			p.TransactionSet = TransactionSet210
		case p.ShipmentStatus != nil:
			p.TransactionSet = TransactionSet214
		case p.TenderResponse != nil:
			p.TransactionSet = TransactionSet990
		case p.FunctionalAcknowledgment != nil:
			p.TransactionSet = TransactionSet997
		case p.ImplementationAcknowledgment != nil:
			p.TransactionSet = TransactionSet999
		}
	}
}

func (p *DocumentPayload) UnmarshalJSON(data []byte) error {
	type alias DocumentPayload
	var wrapped alias
	if err := sonic.Unmarshal(data, &wrapped); err != nil {
		return fmt.Errorf("decode document payload: %w", err)
	}
	*p = DocumentPayload(wrapped)
	p.Normalize()
	if p.HasBranch() {
		return nil
	}

	var legacy LoadTenderPayload
	if err := sonic.Unmarshal(data, &legacy); err != nil {
		return fmt.Errorf("decode legacy load tender payload: %w", err)
	}
	if legacy.ShipmentID.IsNotNil() {
		*p = NewLoadTenderDocumentPayload(legacy)
	}
	return nil
}

func (p DocumentPayload) HasBranch() bool {
	return p.LoadTender != nil ||
		p.Shipment != nil ||
		p.FreightInvoice != nil ||
		p.ShipmentStatus != nil ||
		p.TenderResponse != nil ||
		p.FunctionalAcknowledgment != nil ||
		p.ImplementationAcknowledgment != nil
}
