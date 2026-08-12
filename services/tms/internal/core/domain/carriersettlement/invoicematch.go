package carriersettlement

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*InvoiceMatch)(nil)
	_ pagination.CursorEntity            = (*InvoiceMatch)(nil)
	_ validationframework.TenantedEntity = (*InvoiceMatch)(nil)
)

type InvoiceMatch struct {
	bun.BaseModel             `bun:"table:carrier_invoice_matches,alias:cim" json:"-"`
	pagination.CursorValueSet `bun:",embed"                                  json:"-"`

	ID                     pulid.ID           `json:"id"                      bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID         pulid.ID           `json:"businessUnitId"          bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID         pulid.ID           `json:"organizationId"          bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	EDICarrierInvoiceID    *pulid.ID          `json:"ediCarrierInvoiceId"     bun:"edi_carrier_invoice_id,type:VARCHAR(100),nullzero"`
	DocumentAIExtractionID *pulid.ID          `json:"documentAiExtractionId"  bun:"document_ai_extraction_id,type:VARCHAR(100),nullzero"`
	CarrierID              pulid.ID           `json:"carrierId"               bun:"carrier_id,type:VARCHAR(100),notnull"`
	CarrierAssignmentID    pulid.ID           `json:"carrierAssignmentId"     bun:"carrier_assignment_id,type:VARCHAR(100),notnull"`
	CarrierSettlementID    *pulid.ID          `json:"carrierSettlementId"     bun:"carrier_settlement_id,type:VARCHAR(100),nullzero"`
	AdjustmentCostEventID  *pulid.ID          `json:"adjustmentCostEventId"   bun:"adjustment_cost_event_id,type:VARCHAR(100),nullzero"`
	Status                 InvoiceMatchStatus `json:"status"                  bun:"status,type:VARCHAR(50),notnull,default:'Suggested'"`
	MatchedVia             MatchVia           `json:"matchedVia"              bun:"matched_via,type:VARCHAR(50),notnull,default:'Manual'"`
	InvoiceNumber          string             `json:"invoiceNumber"           bun:"invoice_number,type:VARCHAR(100),nullzero"`
	InvoiceTotalMinor      int64              `json:"invoiceTotalMinor"       bun:"invoice_total_minor,type:BIGINT,notnull,default:0"`
	ExpectedTotalMinor     int64              `json:"expectedTotalMinor"      bun:"expected_total_minor,type:BIGINT,notnull,default:0"`
	VarianceMinor          int64              `json:"varianceMinor"           bun:"variance_minor,type:BIGINT,notnull,default:0"`
	CurrencyCode           string             `json:"currencyCode"            bun:"currency_code,type:VARCHAR(3),notnull,default:'USD'"`
	ResolutionNote         string             `json:"resolutionNote"          bun:"resolution_note,type:TEXT,nullzero"`
	ResolvedByID           pulid.ID           `json:"resolvedById"            bun:"resolved_by_id,type:VARCHAR(100),nullzero"`
	ResolvedAt             *int64             `json:"resolvedAt"              bun:"resolved_at,type:BIGINT,nullzero"`
	Version                int64              `json:"version"                 bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt              int64              `json:"createdAt"               bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt              int64              `json:"updatedAt"               bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Carrier           *carrier.Carrier            `json:"carrier,omitempty"           bun:"rel:belongs-to,join:carrier_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	CarrierAssignment *shipment.CarrierAssignment `json:"carrierAssignment,omitempty" bun:"rel:belongs-to,join:carrier_assignment_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
}

func (m *InvoiceMatch) HasEDISource() bool {
	return m.EDICarrierInvoiceID != nil && !m.EDICarrierInvoiceID.IsNil()
}

func (m *InvoiceMatch) HasDocumentAISource() bool {
	return m.DocumentAIExtractionID != nil && !m.DocumentAIExtractionID.IsNil()
}

func (m *InvoiceMatch) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(m,
		validation.Field(&m.CarrierID, validation.Required.Error("Carrier is required")),
		validation.Field(
			&m.CarrierAssignmentID,
			validation.Required.Error("Carrier assignment is required"),
		),
		validation.Field(&m.CurrencyCode,
			validation.Required.Error("Currency code is required"),
			validation.Length(3, 3).Error("Currency code must be 3 characters"),
		),
	))

	if !m.Status.IsValid() {
		multiErr.Add("status", errortypes.ErrInvalid, "Invoice match status is invalid")
	}
	if !m.MatchedVia.IsValid() {
		multiErr.Add("matchedVia", errortypes.ErrInvalid, "Invoice match provenance is invalid")
	}
	if m.HasEDISource() == m.HasDocumentAISource() {
		multiErr.Add(
			"ediCarrierInvoiceId",
			errortypes.ErrInvalid,
			"A match must reference exactly one source: an EDI carrier invoice or a document AI extraction",
		)
	}
	if m.VarianceMinor != m.InvoiceTotalMinor-m.ExpectedTotalMinor {
		multiErr.Add(
			"varianceMinor",
			errortypes.ErrInvalid,
			"Variance must equal the invoice total minus the expected total",
		)
	}
}

func (m *InvoiceMatch) GetID() pulid.ID { return m.ID }

func (m *InvoiceMatch) GetCreatedAt() int64 { return m.CreatedAt }

func (m *InvoiceMatch) GetOrganizationID() pulid.ID { return m.OrganizationID }

func (m *InvoiceMatch) GetBusinessUnitID() pulid.ID { return m.BusinessUnitID }

func (m *InvoiceMatch) GetTableName() string { return "carrier_invoice_matches" }

func (m *InvoiceMatch) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if m.ID.IsNil() {
			m.ID = pulid.MustNew("cim_")
		}
		m.CreatedAt = now
	case *bun.UpdateQuery:
		m.UpdatedAt = now
	}
	return nil
}
