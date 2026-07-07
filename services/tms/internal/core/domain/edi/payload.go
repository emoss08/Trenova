//nolint:gocritic // EDI payload value APIs are shared contracts across services and renderers.
package edi

import (
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type DocumentPayload struct {
	TransactionSet               TransactionSet                   `json:"transactionSet,omitempty"`
	PurposeCode                  LoadTenderPurposeCode            `json:"purposeCode,omitempty"`
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
	DeliveryDate       int64                  `json:"deliveryDate,omitempty"`
	ShipmentID         pulid.ID               `json:"shipmentId,omitempty"`
	BOL                string                 `json:"bol,omitempty"`
	ProNumber          string                 `json:"proNumber,omitempty"`
	Weight             *int64                 `json:"weight,omitempty"`
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
	Sequence    int64               `json:"sequence"`
	Code        string              `json:"code,omitempty"`
	Description string              `json:"description,omitempty"`
	Amount      decimal.Decimal     `json:"amount"`
	Rate        decimal.NullDecimal `json:"rate,omitempty"`
	Weight      *int64              `json:"weight,omitempty"`
}

type ShipmentStatusPayload struct {
	ShipmentID                 pulid.ID          `json:"shipmentId,omitempty"`
	BOL                        string            `json:"bol,omitempty"`
	ProNumber                  string            `json:"proNumber,omitempty"`
	StatusCode                 string            `json:"statusCode,omitempty"`
	StatusReasonCode           string            `json:"statusReasonCode,omitempty"`
	EventDate                  int64             `json:"eventDate,omitempty"`
	EventTime                  int64             `json:"eventTime,omitempty"`
	EventTimeCode              string            `json:"eventTimeCode,omitempty"`
	StopID                     pulid.ID          `json:"stopId,omitempty"`
	StopType                   string            `json:"stopType,omitempty"`
	StopSequence               int64             `json:"stopSequence,omitempty"`
	LocationID                 pulid.ID          `json:"locationId,omitempty"`
	LocationName               string            `json:"locationName,omitempty"`
	LocationCode               string            `json:"locationCode,omitempty"`
	AddressLine                string            `json:"addressLine,omitempty"`
	City                       string            `json:"city,omitempty"`
	StateCode                  string            `json:"stateCode,omitempty"`
	PostalCode                 string            `json:"postalCode,omitempty"`
	CountryCode                string            `json:"countryCode,omitempty"`
	AppointmentNumber          string            `json:"appointmentNumber,omitempty"`
	ScheduledWindowStart       int64             `json:"scheduledWindowStart,omitempty"`
	ScheduledWindowEnd         *int64            `json:"scheduledWindowEnd,omitempty"`
	ActualArrival              *int64            `json:"actualArrival,omitempty"`
	ActualDeparture            *int64            `json:"actualDeparture,omitempty"`
	EquipmentNumber            string            `json:"equipmentNumber,omitempty"`
	EquipmentType              string            `json:"equipmentType,omitempty"`
	ExceptionCode              string            `json:"exceptionCode,omitempty"`
	ReasonCode                 string            `json:"reasonCode,omitempty"`
	ReasonDescription          string            `json:"reasonDescription,omitempty"`
	LateMinutes                *int64            `json:"lateMinutes,omitempty"`
	ServiceFailureID           *pulid.ID         `json:"serviceFailureId,omitempty"`
	ServiceFailureNumber       string            `json:"serviceFailureNumber,omitempty"`
	ServiceFailureReasonCodeID *pulid.ID         `json:"serviceFailureReasonCodeId,omitempty"`
	ServiceFailureReasonCode   string            `json:"serviceFailureReasonCode,omitempty"`
	References                 map[string]string `json:"references,omitempty"`
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
	if payload.PurposeCode == "" {
		payload.PurposeCode = LoadTenderPurposeOriginal
	}
	return DocumentPayload{
		TransactionSet: TransactionSet204,
		PurposeCode:    payload.PurposeCode,
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
	if p.PurposeCode == "" {
		if p.LoadTender != nil && p.LoadTender.PurposeCode != "" {
			p.PurposeCode = p.LoadTender.PurposeCode
		} else if p.Shipment != nil && p.Shipment.PurposeCode != "" {
			p.PurposeCode = p.Shipment.PurposeCode
		}
	}
	if p.PurposeCode != "" {
		if p.LoadTender != nil && p.LoadTender.PurposeCode == "" {
			p.LoadTender.PurposeCode = p.PurposeCode
		}
		if p.Shipment != nil && p.Shipment.PurposeCode == "" {
			p.Shipment.PurposeCode = p.PurposeCode
		}
	}
	if (p.LoadTender != nil || p.Shipment != nil) && p.PurposeCode == "" {
		p.PurposeCode = LoadTenderPurposeOriginal
		if p.LoadTender != nil {
			p.LoadTender.PurposeCode = p.PurposeCode
		}
		if p.Shipment != nil {
			p.Shipment.PurposeCode = p.PurposeCode
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
