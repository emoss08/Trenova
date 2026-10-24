package carriersettlement

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
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
	keel.Table                `keel:"carrier_invoice_matches"`
	pagination.CursorValueSet `bun:",embed"                                  json:"-"`

	ID                     pulid.ID           `json:"id"                     bun:"id,pk,type:VARCHAR(100),notnull" keel:"id,pk,autoid:cim_"`
	BusinessUnitID         pulid.ID           `json:"businessUnitId"         bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	OrganizationID         pulid.ID           `json:"organizationId"         bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`
	EDICarrierInvoiceID    *pulid.ID          `json:"ediCarrierInvoiceId"    bun:"edi_carrier_invoice_id,type:VARCHAR(100),nullzero" keel:"edi_carrier_invoice_id"`
	DocumentAIExtractionID *pulid.ID          `json:"documentAiExtractionId" bun:"document_ai_extraction_id,type:VARCHAR(100),nullzero" keel:"document_ai_extraction_id"`
	CarrierID              pulid.ID           `json:"carrierId"              bun:"carrier_id,type:VARCHAR(100),notnull" keel:"carrier_id"`
	CarrierAssignmentID    pulid.ID           `json:"carrierAssignmentId"    bun:"carrier_assignment_id,type:VARCHAR(100),notnull" keel:"carrier_assignment_id"`
	CarrierSettlementID    *pulid.ID          `json:"carrierSettlementId"    bun:"carrier_settlement_id,type:VARCHAR(100),nullzero" keel:"carrier_settlement_id"`
	AdjustmentCostEventID  *pulid.ID          `json:"adjustmentCostEventId"  bun:"adjustment_cost_event_id,type:VARCHAR(100),nullzero" keel:"adjustment_cost_event_id"`
	Status                 InvoiceMatchStatus `json:"status"                 bun:"status,type:VARCHAR(50),notnull,default:'Suggested'" keel:"status"`
	MatchedVia             MatchVia           `json:"matchedVia"             bun:"matched_via,type:VARCHAR(50),notnull,default:'Manual'" keel:"matched_via"`
	InvoiceNumber          string             `json:"invoiceNumber"          bun:"invoice_number,type:VARCHAR(100),nullzero" keel:"invoice_number,nullzero"`
	InvoiceNumberKey       string             `json:"-"                      bun:"invoice_number_key,type:VARCHAR(100),nullzero" keel:"invoice_number_key,nullzero"`
	DuplicateOfMatchID     *pulid.ID          `json:"duplicateOfMatchId"     bun:"duplicate_of_match_id,type:VARCHAR(100),nullzero" keel:"duplicate_of_match_id"`
	PossibleDuplicateOfID  *pulid.ID          `json:"possibleDuplicateOfId"  bun:"possible_duplicate_of_id,type:VARCHAR(100),nullzero" keel:"possible_duplicate_of_id"`
	InvoiceTotalMinor      int64              `json:"invoiceTotalMinor"      bun:"invoice_total_minor,type:BIGINT,notnull,default:0" keel:"invoice_total_minor"`
	ExpectedTotalMinor     int64              `json:"expectedTotalMinor"     bun:"expected_total_minor,type:BIGINT,notnull,default:0" keel:"expected_total_minor"`
	VarianceMinor          int64              `json:"varianceMinor"          bun:"variance_minor,type:BIGINT,notnull,default:0" keel:"variance_minor"`
	CurrencyCode           string             `json:"currencyCode"           bun:"currency_code,type:VARCHAR(3),notnull,default:'USD'" keel:"currency_code"`
	ResolutionNote         string             `json:"resolutionNote"         bun:"resolution_note,type:TEXT,nullzero" keel:"resolution_note,nullzero"`
	ResolvedByID           pulid.ID           `json:"resolvedById"           bun:"resolved_by_id,type:VARCHAR(100),nullzero" keel:"resolved_by_id,nullzero"`
	ResolvedAt             *int64             `json:"resolvedAt"             bun:"resolved_at,type:BIGINT,nullzero" keel:"resolved_at"`
	Version                int64              `json:"version"                bun:"version,type:BIGINT,notnull,default:0" keel:"version,version"`
	CreatedAt              int64              `json:"createdAt"              bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt              int64              `json:"updatedAt"              bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`

	Carrier           *carrier.Carrier            `json:"carrier,omitempty"           bun:"rel:belongs-to,join:carrier_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id" keel:"belongs:carrier_id"`
	CarrierAssignment *shipment.CarrierAssignment `json:"carrierAssignment,omitempty" bun:"rel:belongs-to,join:carrier_assignment_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id" keel:"belongs:carrier_assignment_id"`
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
	m.InvoiceNumberKey = edi.InvoiceNumberKey(m.InvoiceNumber)
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

func (m *InvoiceMatch) IsDuplicate() bool {
	return m.DuplicateOfMatchID != nil && m.DuplicateOfMatchID.IsNotNil()
}

func (m *InvoiceMatch) IsPossibleDuplicate() bool {
	return m.PossibleDuplicateOfID != nil && m.PossibleDuplicateOfID.IsNotNil()
}
