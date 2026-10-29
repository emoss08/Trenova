package carriersettlement

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edicarrierinvoice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ pagination.CursorEntity            = (*InvoiceMatch)(nil)
	_ validationframework.TenantedEntity = (*InvoiceMatch)(nil)
)

type InvoiceMatch struct {
	keel.Table `keel:"carrier_invoice_matches"`

	ID                     pulid.ID           `json:"id"                     keel:"id,pk,autoid:cim_"`
	BusinessUnitID         pulid.ID           `json:"businessUnitId"         keel:"business_unit_id,pk,tenant"`
	OrganizationID         pulid.ID           `json:"organizationId"         keel:"organization_id,pk,tenant"`
	EDICarrierInvoiceID    *pulid.ID          `json:"ediCarrierInvoiceId"    keel:"edi_carrier_invoice_id"`
	DocumentAIExtractionID *pulid.ID          `json:"documentAiExtractionId" keel:"document_ai_extraction_id"`
	CarrierID              pulid.ID           `json:"carrierId"              keel:"carrier_id"`
	CarrierAssignmentID    pulid.ID           `json:"carrierAssignmentId"    keel:"carrier_assignment_id"`
	CarrierSettlementID    *pulid.ID          `json:"carrierSettlementId"    keel:"carrier_settlement_id"`
	AdjustmentCostEventID  *pulid.ID          `json:"adjustmentCostEventId"  keel:"adjustment_cost_event_id"`
	Status                 InvoiceMatchStatus `json:"status"                 keel:"status"`
	MatchedVia             MatchVia           `json:"matchedVia"             keel:"matched_via"`
	InvoiceNumber          string             `json:"invoiceNumber"          keel:"invoice_number,nullzero"`
	InvoiceNumberKey       string             `json:"-"                      keel:"invoice_number_key,nullzero"`
	DuplicateOfMatchID     *pulid.ID          `json:"duplicateOfMatchId"     keel:"duplicate_of_match_id"`
	PossibleDuplicateOfID  *pulid.ID          `json:"possibleDuplicateOfId"  keel:"possible_duplicate_of_id"`
	InvoiceTotalMinor      int64              `json:"invoiceTotalMinor"      keel:"invoice_total_minor"`
	ExpectedTotalMinor     int64              `json:"expectedTotalMinor"     keel:"expected_total_minor"`
	VarianceMinor          int64              `json:"varianceMinor"          keel:"variance_minor"`
	CurrencyCode           string             `json:"currencyCode"           keel:"currency_code"`
	ResolutionNote         string             `json:"resolutionNote"         keel:"resolution_note,nullzero"`
	ResolvedByID           pulid.ID           `json:"resolvedById"           keel:"resolved_by_id,nullzero"`
	ResolvedAt             *int64             `json:"resolvedAt"             keel:"resolved_at"`
	Version                int64              `json:"version"                keel:"version,version"`
	CreatedAt              int64              `json:"createdAt"              keel:"created_at,autocreate"`
	UpdatedAt              int64              `json:"updatedAt"              keel:"updated_at,autoupdate"`

	Carrier           *carrier.Carrier            `json:"carrier,omitempty"           keel:"belongs:carrier_id"`
	CarrierAssignment *shipment.CarrierAssignment `json:"carrierAssignment,omitempty" keel:"belongs:carrier_assignment_id"`
}

func (m *InvoiceMatch) BeforeUpdate(_ context.Context) error {
	m.InvoiceNumberKey = edicarrierinvoice.InvoiceNumberKey(m.InvoiceNumber)

	return nil
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

func (m *InvoiceMatch) BeforeCreate(ctx context.Context) error {
	if m.Status == "" {
		m.Status = InvoiceMatchStatusSuggested
	}
	if m.MatchedVia == "" {
		m.MatchedVia = MatchViaManual
	}
	if m.CurrencyCode == "" {
		m.CurrencyCode = money.DefaultCurrencyCode
	}
	m.InvoiceNumberKey = edicarrierinvoice.InvoiceNumberKey(m.InvoiceNumber)

	return nil
}

func (m *InvoiceMatch) IsDuplicate() bool {
	return m.DuplicateOfMatchID != nil && m.DuplicateOfMatchID.IsNotNil()
}

func (m *InvoiceMatch) IsPossibleDuplicate() bool {
	return m.PossibleDuplicateOfID != nil && m.PossibleDuplicateOfID.IsNotNil()
}
