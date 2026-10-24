package carriersettlement

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

var (
	_ domaintypes.PostgresSearchable     = (*CarrierSettlement)(nil)
	_ pagination.CursorEntity            = (*CarrierSettlement)(nil)
	_ validationframework.TenantedEntity = (*CarrierSettlement)(nil)
)

// Carrier payable statements for purchased transportation
type CarrierSettlement struct {
	keel.Table `                                                      keel:"carrier_settlements,report=Billing"`

	ID                     pulid.ID            `json:"id" keel:"id,pk,autoid:carstl_"`
	BusinessUnitID         pulid.ID            `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID         pulid.ID            `json:"organizationId" keel:"organization_id,pk,tenant"`
	CarrierID              pulid.ID            `json:"carrierId" keel:"carrier_id"`
	BatchID                *pulid.ID           `json:"batchId" keel:"batch_id"`
	SettlementNumber       string              `json:"settlementNumber" keel:"settlement_number"`
	Status                 Status              `json:"status" keel:"status"`
	PeriodStart            int64               `json:"periodStart" keel:"period_start,type=epoch"`
	PeriodEnd              int64               `json:"periodEnd" keel:"period_end,type=epoch"`
	PayDate                int64               `json:"payDate" keel:"pay_date"`
	GrossCostMinor         int64               `json:"grossCostMinor" keel:"gross_cost_minor,label=Gross Cost (Minor)"`
	AdjustmentsMinor       int64               `json:"adjustmentsMinor" keel:"adjustments_minor,label=Adjustments (Minor)"`
	NetPayableMinor        int64               `json:"netPayableMinor" keel:"net_payable_minor,label=Net Payable (Minor)"`
	ShipmentCount          int                 `json:"shipmentCount" keel:"shipment_count"`
	CurrencyCode           string              `json:"currencyCode" keel:"currency_code"`
	ExchangeRate           decimal.NullDecimal `json:"exchangeRate" keel:"exchange_rate"`
	ExchangeRateDate       *int64              `json:"exchangeRateDate" keel:"exchange_rate_date"`
	Notes                  string              `json:"notes" keel:"notes,nullzero"`
	SubmittedByID          pulid.ID            `json:"submittedById" keel:"submitted_by_id,nullzero"`
	SubmittedAt            *int64              `json:"submittedAt" keel:"submitted_at"`
	ApprovedByID           pulid.ID            `json:"approvedById" keel:"approved_by_id,nullzero"`
	ApprovedAt             *int64              `json:"approvedAt" keel:"approved_at"`
	PostedByID             pulid.ID            `json:"postedById" keel:"posted_by_id,nullzero"`
	PostedAt               *int64              `json:"postedAt" keel:"posted_at"`
	PostedJournalBatchID   *pulid.ID           `json:"postedJournalBatchId" keel:"posted_journal_batch_id"`
	PostedExpenseAccountID *pulid.ID           `json:"postedExpenseAccountId" keel:"posted_expense_account_id"`
	PostedAPAccountID      *pulid.ID           `json:"postedApAccountId" keel:"posted_ap_account_id"`
	PaidAt                 *int64              `json:"paidAt" keel:"paid_at"`
	PaidByID               pulid.ID            `json:"paidById" keel:"paid_by_id,nullzero"`
	PaymentMethod          string              `json:"paymentMethod" keel:"payment_method,nullzero"`
	PaymentReference       string              `json:"paymentReference" keel:"payment_reference,nullzero"`
	PaidJournalBatchID     *pulid.ID           `json:"paidJournalBatchId" keel:"paid_journal_batch_id"`
	PaidExchangeRate       decimal.NullDecimal `json:"paidExchangeRate" keel:"paid_exchange_rate"`
	PaidExchangeRateDate   *int64              `json:"paidExchangeRateDate" keel:"paid_exchange_rate_date"`
	VoidedByID             pulid.ID            `json:"voidedById" keel:"voided_by_id,nullzero"`
	VoidedAt               *int64              `json:"voidedAt" keel:"voided_at"`
	VoidReason             string              `json:"voidReason" keel:"void_reason,nullzero"`
	VoidJournalBatchID     *pulid.ID           `json:"voidJournalBatchId" keel:"void_journal_batch_id"`
	Version                int64               `json:"version" keel:"version,version"`
	CreatedAt              int64               `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt              int64               `json:"updatedAt" keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit     `json:"businessUnit,omitempty"`
	Organization *tenant.Organization     `json:"organization,omitempty"`
	Carrier      *carrier.Carrier         `json:"carrier,omitempty" keel:"belongs:carrier_id"`
	Batch        *CarrierSettlementBatch  `json:"batch,omitempty" keel:"belongs:batch_id,noreport"`
	Lines        []*CarrierSettlementLine `json:"lines,omitempty" keel:"many:settlement_id,noreport"`
}

type CarrierSettlementLine struct {
	keel.Table `                                                            keel:"carrier_settlement_lines"`

	ID             pulid.ID      `json:"id" keel:"id,pk,autoid:carstll_"`
	BusinessUnitID pulid.ID      `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID      `json:"organizationId" keel:"organization_id,pk,tenant"`
	SettlementID   pulid.ID      `json:"settlementId" keel:"settlement_id"`
	LineNumber     int           `json:"lineNumber" keel:"line_number"`
	EventType      CostEventType `json:"eventType" keel:"event_type"`
	Description    string        `json:"description" keel:"description"`
	AmountMinor    int64         `json:"amountMinor" keel:"amount_minor"`
	CostEventID    *pulid.ID     `json:"costEventId" keel:"cost_event_id"`
	GLAccountID    *pulid.ID     `json:"glAccountId" keel:"gl_account_id"`
	ShipmentID     *pulid.ID     `json:"shipmentId" keel:"shipment_id"`
	MoveID         *pulid.ID     `json:"moveId" keel:"move_id"`
	ProNumber      string        `json:"proNumber" keel:"pro_number,nullzero"`
	AddedByAgent   bool          `json:"addedByAgent" keel:"added_by_agent"`
	CreatedAt      int64         `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt      int64         `json:"updatedAt" keel:"updated_at,autoupdate"`
}

func (s *CarrierSettlement) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(s,
		validation.Field(&s.CarrierID, validation.Required.Error("Carrier is required")),
		validation.Field(&s.PeriodStart, validation.Required.Error("Period start is required")),
		validation.Field(&s.PeriodEnd, validation.Required.Error("Period end is required")),
		validation.Field(&s.PayDate, validation.Required.Error("Pay date is required")),
		validation.Field(&s.CurrencyCode,
			validation.Required.Error("Currency code is required"),
			validation.Length(3, 3).Error("Currency code must be 3 characters"),
		),
	))

	if !s.Status.IsValid() {
		multiErr.Add("status", errortypes.ErrInvalid, "Settlement status is invalid")
	}
	if s.PeriodEnd <= s.PeriodStart {
		multiErr.Add(
			"periodEnd",
			errortypes.ErrInvalid,
			"Period end must be after the period start",
		)
	}
	for idx, line := range s.Lines {
		if line == nil {
			multiErr.Add(
				"lines",
				errortypes.ErrInvalid,
				"Settlement lines must not contain null values",
			)
			continue
		}
		line.Validate(multiErr.WithIndex("lines", idx))
	}
}

func (l *CarrierSettlementLine) Validate(multiErr *errortypes.MultiError) {
	if !l.EventType.IsValid() {
		multiErr.Add("eventType", errortypes.ErrInvalid, "Line event type is invalid")
	}
	if l.Description == "" {
		multiErr.Add("description", errortypes.ErrRequired, "Description is required")
	}
	if l.EventType != CostEventTypeAdjustment && l.AmountMinor < 0 {
		multiErr.Add(
			"amountMinor",
			errortypes.ErrInvalid,
			"Only adjustment lines may carry a negative amount",
		)
	}
}

func (l *CarrierSettlementLine) IsManualAdjustment() bool {
	return l.EventType == CostEventTypeAdjustment &&
		(l.CostEventID == nil || l.CostEventID.IsNil())
}

func (s *CarrierSettlement) SyncTotals() {
	var gross, adjustments int64
	for idx, line := range s.Lines {
		if line == nil {
			continue
		}
		line.LineNumber = idx + 1
		switch line.EventType {
		case CostEventTypeLinehaulCost, CostEventTypeFuelSurcharge,
			CostEventTypeAccessorial:
			gross += line.AmountMinor
		case CostEventTypeAdjustment:
			adjustments += line.AmountMinor
		}
	}
	s.GrossCostMinor = gross
	s.AdjustmentsMinor = adjustments
	s.NetPayableMinor = gross + adjustments
}

func (s *CarrierSettlement) IsEditable() bool {
	return s.Status == StatusDraft || s.Status == StatusPendingApproval
}

func (s *CarrierSettlement) GetID() pulid.ID { return s.ID }

func (s *CarrierSettlement) GetCreatedAt() int64 { return s.CreatedAt }

func (s *CarrierSettlement) GetOrganizationID() pulid.ID { return s.OrganizationID }

func (s *CarrierSettlement) GetBusinessUnitID() pulid.ID { return s.BusinessUnitID }

func (s *CarrierSettlement) GetTableName() string { return "carrier_settlements" }

func (s *CarrierSettlement) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "carstl",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{
				Name:   "settlement_number",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightA,
			},
		},
	}
}

func (s *CarrierSettlement) BeforeCreate(ctx context.Context) error {
	if s.Status == "" {
		s.Status = StatusDraft
	}
	if s.CurrencyCode == "" {
		s.CurrencyCode = "USD"
	}

	return nil
}
