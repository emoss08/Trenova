package edi

import (
	"context"

	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
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
	bun.BaseModel             `json:"-" bun:"table:edi_carrier_invoices,alias:ecinv"`
	pagination.CursorValueSet `json:"-" bun:",embed"`

	ID                   pulid.ID                           `json:"id"                   bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID       pulid.ID                           `json:"businessUnitId"       bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID       pulid.ID                           `json:"organizationId"       bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	EDIPartnerID         pulid.ID                           `json:"ediPartnerId"         bun:"edi_partner_id,type:VARCHAR(100),notnull"`
	InboundMessageID     pulid.ID                           `json:"inboundMessageId"     bun:"inbound_message_id,type:VARCHAR(100),notnull"`
	ShipmentID           pulid.ID                           `json:"shipmentId"           bun:"shipment_id,type:VARCHAR(100),nullzero"`
	TenderRecipientID    pulid.ID                           `json:"tenderRecipientId"    bun:"tender_recipient_id,type:VARCHAR(100),nullzero"`
	CustomerID           pulid.ID                           `json:"customerId"           bun:"customer_id,type:VARCHAR(100),nullzero"`
	InvoiceNumber        string                             `json:"invoiceNumber"        bun:"invoice_number,type:VARCHAR(100),notnull"`
	InvoiceDate          *int64                             `json:"invoiceDate"          bun:"invoice_date,type:BIGINT,nullzero"`
	DeliveryDate         *int64                             `json:"deliveryDate"         bun:"delivery_date,type:BIGINT,nullzero"`
	ShipmentReference    string                             `json:"shipmentReference"    bun:"shipment_reference,type:VARCHAR(100),nullzero"`
	BOL                  string                             `json:"bol"                  bun:"bol,type:VARCHAR(100),nullzero"`
	ProNumber            string                             `json:"proNumber"            bun:"pro_number,type:VARCHAR(100),nullzero"`
	BillToName           string                             `json:"billToName"           bun:"bill_to_name,type:VARCHAR(200),nullzero"`
	BillToSourceID       pulid.ID                           `json:"billToSourceId"       bun:"bill_to_source_id,type:VARCHAR(100),nullzero"`
	CurrencyCode         string                             `json:"currencyCode"         bun:"currency_code,type:VARCHAR(3),nullzero"`
	TotalAmount          decimal.NullDecimal                `json:"totalAmount"          bun:"total_amount,type:NUMERIC(19,4),nullzero"`
	ExpectedAmount       decimal.NullDecimal                `json:"expectedAmount"       bun:"expected_amount,type:NUMERIC(19,4),nullzero"`
	VarianceAmount       decimal.NullDecimal                `json:"varianceAmount"       bun:"variance_amount,type:NUMERIC(19,4),nullzero"`
	LineCharges          []FreightInvoiceCharge             `json:"lineCharges"          bun:"line_charges,type:JSONB,notnull,default:'[]'::jsonb"`
	ReferenceNumbers     map[string]string                  `json:"referenceNumbers"     bun:"reference_numbers,type:JSONB,notnull,default:'{}'::jsonb"`
	ReconciliationStatus CarrierInvoiceReconciliationStatus `json:"reconciliationStatus" bun:"reconciliation_status,type:edi_carrier_invoice_reconciliation_status_enum,notnull,default:'Unmatched'"`
	ReconciliationNotes  string                             `json:"reconciliationNotes"  bun:"reconciliation_notes,type:TEXT,nullzero"`
	Version              int64                              `json:"version"              bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt            int64                              `json:"createdAt"            bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt            int64                              `json:"updatedAt"            bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Partner *EDIPartner `json:"partner,omitempty" bun:"rel:belongs-to,join:edi_partner_id=id"`
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

func (i *CarrierInvoice) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	if i.LineCharges == nil {
		i.LineCharges = []FreightInvoiceCharge{}
	}
	if i.ReferenceNumbers == nil {
		i.ReferenceNumbers = map[string]string{}
	}
	if i.ReconciliationStatus == "" {
		i.ReconciliationStatus = CarrierInvoiceReconciliationStatusUnmatched
	}
	switch query.(type) {
	case *bun.InsertQuery:
		if i.ID.IsNil() {
			i.ID = pulid.MustNew("edici_")
		}
		i.CreatedAt = now
	case *bun.UpdateQuery:
		i.UpdatedAt = now
	}
	return nil
}

func (i *CarrierInvoice) GetCreatedAt() int64 {
	return i.CreatedAt
}
