package edi

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

type CarrierInvoiceReconciliationStatus string

const (
	CarrierInvoiceReconciliationStatusUnmatched = CarrierInvoiceReconciliationStatus(
		"Unmatched",
	)
	CarrierInvoiceReconciliationStatusMappingRequired = CarrierInvoiceReconciliationStatus(
		"MappingRequired",
	)
	CarrierInvoiceReconciliationStatusMatched = CarrierInvoiceReconciliationStatus(
		"Matched",
	)
	CarrierInvoiceReconciliationStatusVariance = CarrierInvoiceReconciliationStatus(
		"Variance",
	)
)

func (s CarrierInvoiceReconciliationStatus) NeedsAttention() bool {
	return s == CarrierInvoiceReconciliationStatusUnmatched ||
		s == CarrierInvoiceReconciliationStatusMappingRequired ||
		s == CarrierInvoiceReconciliationStatusVariance
}

type CarrierInvoice struct {
	keel.Table `keel:"edi_carrier_invoices"`

	ID                   pulid.ID                           `json:"id"                   keel:"id,pk,autoid:edici_"`
	BusinessUnitID       pulid.ID                           `json:"businessUnitId"       keel:"business_unit_id,pk,tenant"`
	OrganizationID       pulid.ID                           `json:"organizationId"       keel:"organization_id,pk,tenant"`
	EDIPartnerID         pulid.ID                           `json:"ediPartnerId"         keel:"edi_partner_id"`
	InboundMessageID     pulid.ID                           `json:"inboundMessageId"     keel:"inbound_message_id"`
	ShipmentID           pulid.ID                           `json:"shipmentId"           keel:"shipment_id,nullzero"`
	TenderRecipientID    pulid.ID                           `json:"tenderRecipientId"    keel:"tender_recipient_id,nullzero"`
	CustomerID           pulid.ID                           `json:"customerId"           keel:"customer_id,nullzero"`
	CarrierID            pulid.ID                           `json:"carrierId"            keel:"carrier_id,nullzero"`
	InvoiceNumber        string                             `json:"invoiceNumber"        keel:"invoice_number"`
	InvoiceNumberKey     string                             `json:"-"                    keel:"invoice_number_key"`
	DuplicateOfID        pulid.ID                           `json:"duplicateOfId"        keel:"duplicate_of_id,nullzero"`
	InvoiceDate          *int64                             `json:"invoiceDate"          keel:"invoice_date"`
	DeliveryDate         *int64                             `json:"deliveryDate"         keel:"delivery_date"`
	ShipmentReference    string                             `json:"shipmentReference"    keel:"shipment_reference,nullzero"`
	BOL                  string                             `json:"bol"                  keel:"bol,nullzero"`
	ProNumber            string                             `json:"proNumber"            keel:"pro_number,nullzero"`
	BillToName           string                             `json:"billToName"           keel:"bill_to_name,nullzero"`
	BillToSourceID       pulid.ID                           `json:"billToSourceId"       keel:"bill_to_source_id,nullzero"`
	CurrencyCode         string                             `json:"currencyCode"         keel:"currency_code,nullzero"`
	TotalAmount          decimal.NullDecimal                `json:"totalAmount"          keel:"total_amount"`
	ExpectedAmount       decimal.NullDecimal                `json:"expectedAmount"       keel:"expected_amount"`
	VarianceAmount       decimal.NullDecimal                `json:"varianceAmount"       keel:"variance_amount"`
	LineCharges          []FreightInvoiceCharge             `json:"lineCharges"          keel:"line_charges,jsonb"`
	ReferenceNumbers     map[string]string                  `json:"referenceNumbers"     keel:"reference_numbers,jsonb"`
	ReconciliationStatus CarrierInvoiceReconciliationStatus `json:"reconciliationStatus" keel:"reconciliation_status"`
	ReconciliationNotes  string                             `json:"reconciliationNotes"  keel:"reconciliation_notes,nullzero"`
	Version              int64                              `json:"version"              keel:"version,version"`
	CreatedAt            int64                              `json:"createdAt"            keel:"created_at,autocreate"`
	UpdatedAt            int64                              `json:"updatedAt"            keel:"updated_at,autoupdate"`

	Partner *EDIPartner `json:"partner,omitempty" keel:"belongs:edi_partner_id"`
}

func (i *CarrierInvoice) BeforeUpdate(ctx context.Context) error {
	if err := i.BeforeSave(ctx); err != nil {
		return err
	}

	return nil
}

func (i *CarrierInvoice) BeforeCreate(ctx context.Context) error {
	if err := i.BeforeSave(ctx); err != nil {
		return err
	}
	if i.ReconciliationStatus == "" {
		i.ReconciliationStatus = CarrierInvoiceReconciliationStatusUnmatched
	}

	return nil
}

func (i *CarrierInvoice) GetID() pulid.ID {
	return i.ID
}

func (i *CarrierInvoice) GetTableName() string {
	return "edi_carrier_invoices"
}

func (i *CarrierInvoice) GetOrganizationID() pulid.ID {
	return i.OrganizationID
}

func (i *CarrierInvoice) GetBusinessUnitID() pulid.ID {
	return i.BusinessUnitID
}

func (i *CarrierInvoice) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias: "ecinv",
		SearchableFields: []domaintypes.SearchableField{
			{
				Name:   "invoice_number",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightA,
			},
			{
				Name:   "bill_to_name",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
			{
				Name:   "reconciliation_status",
				Type:   domaintypes.FieldTypeEnum,
				Weight: domaintypes.SearchWeightB,
			},
			{Name: "bol", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightC},
		},
	}
}

func (i *CarrierInvoice) BeforeSave(_ context.Context) error {
	if i.LineCharges == nil {
		i.LineCharges = []FreightInvoiceCharge{}
	}
	if i.ReferenceNumbers == nil {
		i.ReferenceNumbers = map[string]string{}
	}
	if i.ReconciliationStatus == "" {
		i.ReconciliationStatus = CarrierInvoiceReconciliationStatusUnmatched
	}
	i.InvoiceNumberKey = InvoiceNumberKey(i.InvoiceNumber)
	return nil
}

func (i *CarrierInvoice) GetCreatedAt() int64 {
	return i.CreatedAt
}

func (v CarrierInvoiceReconciliationStatus) IsValid() bool {
	switch v {
	case CarrierInvoiceReconciliationStatusUnmatched,
		CarrierInvoiceReconciliationStatusMappingRequired,
		CarrierInvoiceReconciliationStatusMatched,
		CarrierInvoiceReconciliationStatusVariance:
		return true
	default:
		return false
	}
}

func (i *CarrierInvoice) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(i,
		validation.Field(&i.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&i.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&i.EDIPartnerID, validation.Required.Error("Partner is required")),
		// The inbound message is the evidence for what the carrier billed, so a
		// disputed charge stays traceable to the interchange that carried it.
		validation.Field(&i.InboundMessageID,
			validation.Required.Error("Inbound message is required"),
		),
		validation.Field(&i.InvoiceNumber,
			validation.Required.Error("Invoice number is required"),
			validation.Length(1, maxInvoiceFieldLength).
				Error("Invoice number cannot be longer than 100 characters"),
		),
		validation.Field(&i.ShipmentReference,
			validation.Length(0, maxInvoiceFieldLength).
				Error("Shipment reference cannot be longer than 100 characters"),
		),
		validation.Field(&i.BOL,
			validation.Length(0, maxInvoiceFieldLength).
				Error("BOL cannot be longer than 100 characters"),
		),
		validation.Field(&i.ProNumber,
			validation.Length(0, maxInvoiceFieldLength).
				Error("Pro number cannot be longer than 100 characters"),
		),
		validation.Field(&i.BillToName,
			validation.Length(0, maxBillToNameLength).
				Error("Bill to name cannot be longer than 200 characters"),
		),
		validation.Field(&i.CurrencyCode,
			validation.Length(0, invoiceCurrencyCodeLength).
				Error("Currency must be a three letter code"),
		),
		validation.Field(&i.ReconciliationStatus,
			validation.Required.Error("Reconciliation status is required"),
			domainvalidation.ValidEnum[CarrierInvoiceReconciliationStatus](
				"Reconciliation status is invalid",
			),
		),
	))
}

const (
	maxInvoiceFieldLength     = 100
	maxBillToNameLength       = 200
	invoiceCurrencyCodeLength = 3
)

func InvoiceNumberKey(invoiceNumber string) string {
	return stringutils.CollapseUpper(invoiceNumber)
}
