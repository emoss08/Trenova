package driversettlement

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/paycode"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

var (
	_ domaintypes.PostgresSearchable     = (*Settlement)(nil)
	_ pagination.CursorEntity            = (*Settlement)(nil)
	_ validationframework.TenantedEntity = (*Settlement)(nil)
)

type Exception struct {
	Code     ExceptionCode     `json:"code"`
	Severity ExceptionSeverity `json:"severity"`
	Message  string            `json:"message"`
}

// Driver and owner-operator pay statements
type Settlement struct {
	keel.Table `keel:"driver_settlements,report=Payroll,label=Driver Settlement"`

	ID                     pulid.ID                      `json:"id"                     keel:"id,pk,autoid:dstl_"`
	BusinessUnitID         pulid.ID                      `json:"businessUnitId"         keel:"business_unit_id,pk,tenant"`
	OrganizationID         pulid.ID                      `json:"organizationId"         keel:"organization_id,pk,tenant"`
	WorkerID               pulid.ID                      `json:"workerId"               keel:"worker_id"`
	BatchID                *pulid.ID                     `json:"batchId"                keel:"batch_id"`
	PayProfileID           *pulid.ID                     `json:"payProfileId"           keel:"pay_profile_id"`
	SettlementNumber       string                        `json:"settlementNumber"       keel:"settlement_number"`
	Status                 Status                        `json:"status"                 keel:"status"`
	Classification         driverpay.PayeeClassification `json:"classification"         keel:"classification"`
	PayProfileName         string                        `json:"payProfileName"         keel:"pay_profile_name,nullzero,label=Pay Profile"`
	PeriodStart            int64                         `json:"periodStart"            keel:"period_start,type=epoch"`
	PeriodEnd              int64                         `json:"periodEnd"              keel:"period_end,type=epoch"`
	PayDate                int64                         `json:"payDate"                keel:"pay_date"`
	GrossEarningsMinor     int64                         `json:"grossEarningsMinor"     keel:"gross_earnings_minor,label=Gross Earnings (Minor)"`
	ReimbursementsMinor    int64                         `json:"reimbursementsMinor"    keel:"reimbursements_minor,label=Reimbursements (Minor)"`
	DeductionsMinor        int64                         `json:"deductionsMinor"        keel:"deductions_minor,label=Deductions (Minor)"`
	CarryForwardInMinor    int64                         `json:"carryForwardInMinor"    keel:"carry_forward_in_minor,label=Carry Forward In (Minor)"`
	CarryForwardOutMinor   int64                         `json:"carryForwardOutMinor"   keel:"carry_forward_out_minor,label=Carry Forward Out (Minor)"`
	NetPayMinor            int64                         `json:"netPayMinor"            keel:"net_pay_minor,label=Net Pay (Minor)"`
	TotalMiles             decimal.Decimal               `json:"totalMiles"             keel:"total_miles,format=distance"`
	ShipmentCount          int                           `json:"shipmentCount"          keel:"shipment_count,label=Shipments,format=count"`
	CurrencyCode           string                        `json:"currencyCode"           keel:"currency_code,label=Currency"`
	ExchangeRate           decimal.NullDecimal           `json:"exchangeRate"           keel:"exchange_rate"`
	ExchangeRateDate       *int64                        `json:"exchangeRateDate"       keel:"exchange_rate_date"`
	HasExceptions          bool                          `json:"hasExceptions"          keel:"has_exceptions"`
	Exceptions             []Exception                   `json:"exceptions"             keel:"exceptions,jsonb,nullzero,noreport"`
	Notes                  string                        `json:"notes"                  keel:"notes,nullzero"`
	SubmittedByID          pulid.ID                      `json:"submittedById"          keel:"submitted_by_id,nullzero"`
	SubmittedAt            *int64                        `json:"submittedAt"            keel:"submitted_at"`
	ApprovedByID           pulid.ID                      `json:"approvedById"           keel:"approved_by_id,nullzero"`
	ApprovedAt             *int64                        `json:"approvedAt"             keel:"approved_at"`
	PostedByID             pulid.ID                      `json:"postedById"             keel:"posted_by_id,nullzero"`
	PostedAt               *int64                        `json:"postedAt"               keel:"posted_at"`
	PostedJournalBatchID   *pulid.ID                     `json:"postedJournalBatchId"   keel:"posted_journal_batch_id"`
	PostedPayableAccountID *pulid.ID                     `json:"postedPayableAccountId" keel:"posted_payable_account_id"`
	PaidAt                 *int64                        `json:"paidAt"                 keel:"paid_at"`
	PaidByID               pulid.ID                      `json:"paidById"               keel:"paid_by_id,nullzero"`
	PaymentMethod          string                        `json:"paymentMethod"          keel:"payment_method,nullzero"`
	PaymentReference       string                        `json:"paymentReference"       keel:"payment_reference,nullzero"`
	PaidJournalBatchID     *pulid.ID                     `json:"paidJournalBatchId"     keel:"paid_journal_batch_id"`
	PaidExchangeRate       decimal.NullDecimal           `json:"paidExchangeRate"       keel:"paid_exchange_rate"`
	PaidExchangeRateDate   *int64                        `json:"paidExchangeRateDate"   keel:"paid_exchange_rate_date"`
	VoidedByID             pulid.ID                      `json:"voidedById"             keel:"voided_by_id,nullzero"`
	VoidedAt               *int64                        `json:"voidedAt"               keel:"voided_at"`
	VoidReason             string                        `json:"voidReason"             keel:"void_reason,nullzero"`
	VoidJournalBatchID     *pulid.ID                     `json:"voidJournalBatchId"     keel:"void_journal_batch_id"`
	Version                int64                         `json:"version"                keel:"version,version"`
	CreatedAt              int64                         `json:"createdAt"              keel:"created_at,autocreate"`
	UpdatedAt              int64                         `json:"updatedAt"              keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit  `json:"businessUnit,omitempty"`
	Organization *tenant.Organization  `json:"organization,omitempty"`
	Worker       *worker.Worker        `json:"worker,omitempty"       keel:"belongs:worker_id,label=Driver"`
	Batch        *SettlementBatch      `json:"batch,omitempty"        keel:"belongs:batch_id,label=Settlement Batch"`
	PayProfile   *driverpay.PayProfile `json:"payProfile,omitempty"   keel:"belongs:pay_profile_id,noreport"`
	Lines        []*SettlementLine     `json:"lines,omitempty"        keel:"many:settlement_id,label=Settlement Lines"`
}

// Each earning, deduction and reimbursement on a settlement
type SettlementLine struct {
	keel.Table `keel:"driver_settlement_lines,report=Payroll"`

	ID                   pulid.ID                `json:"id"                   keel:"id,pk,autoid:dstll_"`
	BusinessUnitID       pulid.ID                `json:"businessUnitId"       keel:"business_unit_id,pk,tenant"`
	OrganizationID       pulid.ID                `json:"organizationId"       keel:"organization_id,pk,tenant"`
	SettlementID         pulid.ID                `json:"settlementId"         keel:"settlement_id"`
	LineNumber           int                     `json:"lineNumber"           keel:"line_number,label=Line,dimension"`
	Category             LineCategory            `json:"category"             keel:"category"`
	ComponentKind        driverpay.ComponentKind `json:"componentKind"        keel:"component_kind,nullzero,label=Component"`
	Method               driverpay.CalcMethod    `json:"method"               keel:"method,nullzero,label=Calculation Method"`
	Description          string                  `json:"description"          keel:"description"`
	Quantity             decimal.Decimal         `json:"quantity"             keel:"quantity"`
	Rate                 decimal.Decimal         `json:"rate"                 keel:"rate"`
	AmountMinor          int64                   `json:"amountMinor"          keel:"amount_minor,label=Amount (Minor)"`
	ShipmentID           *pulid.ID               `json:"shipmentId"           keel:"shipment_id"`
	MoveID               *pulid.ID               `json:"moveId"               keel:"move_id"`
	PayEventID           *pulid.ID               `json:"payEventId"           keel:"pay_event_id"`
	RecurringDeductionID *pulid.ID               `json:"recurringDeductionId" keel:"recurring_deduction_id"`
	RecurringEarningID   *pulid.ID               `json:"recurringEarningId"   keel:"recurring_earning_id"`
	PayCodeID            *pulid.ID               `json:"payCodeId"            keel:"pay_code_id"`
	AdvanceID            *pulid.ID               `json:"advanceId"            keel:"advance_id"`
	EscrowAccountID      *pulid.ID               `json:"escrowAccountId"      keel:"escrow_account_id"`
	ProNumber            string                  `json:"proNumber"            keel:"pro_number,nullzero"`
	AddedByAgent         bool                    `json:"addedByAgent"         keel:"added_by_agent"`
	CreatedAt            int64                   `json:"createdAt"            keel:"created_at,autocreate"`
	UpdatedAt            int64                   `json:"updatedAt"            keel:"updated_at,autoupdate"`

	Settlement *Settlement        `json:"settlement,omitempty" keel:"belongs:settlement_id"`
	Shipment   *shipment.Shipment `json:"shipment,omitempty"   keel:"belongs:shipment_id"`
	PayCode    *paycode.PayCode   `json:"payCode,omitempty"    keel:"belongs:pay_code_id"`
}

func (s *Settlement) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(s,
		validation.Field(&s.WorkerID, validation.Required.Error("Worker is required")),
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
	if !s.Classification.IsValid() {
		multiErr.Add("classification", errortypes.ErrInvalid, "Classification is invalid")
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

func (l *SettlementLine) Validate(multiErr *errortypes.MultiError) {
	if !l.Category.IsValid() {
		multiErr.Add("category", errortypes.ErrInvalid, "Line category is invalid")
	}
	if l.Description == "" {
		multiErr.Add("description", errortypes.ErrRequired, "Description is required")
	}
	if l.Category.IsCredit() && l.AmountMinor < 0 {
		multiErr.Add(
			"amountMinor",
			errortypes.ErrInvalid,
			"Earnings, reimbursements, and guarantee top-ups must not be negative",
		)
	}
	switch l.Category {
	case LineCategoryDeduction, LineCategoryAdvanceRecovery, LineCategoryEscrowContribution:
		if l.AmountMinor > 0 {
			multiErr.Add(
				"amountMinor",
				errortypes.ErrInvalid,
				"Deduction lines must carry a negative amount",
			)
		}
	case LineCategoryEarning, LineCategoryReimbursement, LineCategoryGuaranteeTopUp,
		LineCategoryCarryForward, LineCategoryAdjustment:
	}
}

func (SettlementLine) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "uq_driver_settlement_lines_number",
			Field:   "lines",
			Message: "Settlement line numbers must be unique within a settlement",
		},
	}
}

func (s *Settlement) SyncTotals() {
	var gross, reimbursements, deductions, carryIn int64
	for idx, line := range s.Lines {
		if line == nil {
			continue
		}
		line.LineNumber = idx + 1
		switch line.Category {
		case LineCategoryEarning, LineCategoryGuaranteeTopUp:
			gross += line.AmountMinor
		case LineCategoryReimbursement:
			reimbursements += line.AmountMinor
		case LineCategoryDeduction, LineCategoryAdvanceRecovery,
			LineCategoryEscrowContribution:
			deductions += -line.AmountMinor
		case LineCategoryCarryForward:
			carryIn += line.AmountMinor
		case LineCategoryAdjustment:
			if line.AmountMinor >= 0 {
				gross += line.AmountMinor
			} else {
				deductions += -line.AmountMinor
			}
		}
	}
	s.GrossEarningsMinor = gross
	s.ReimbursementsMinor = reimbursements
	s.DeductionsMinor = deductions
	s.CarryForwardInMinor = carryIn
	net := gross + reimbursements + carryIn - deductions
	if net < 0 {
		s.CarryForwardOutMinor = net
		s.NetPayMinor = 0
	} else {
		s.CarryForwardOutMinor = 0
		s.NetPayMinor = net
	}
}

func (s *Settlement) AddException(code ExceptionCode, severity ExceptionSeverity, message string) {
	for _, existing := range s.Exceptions {
		if existing.Code == code {
			return
		}
	}
	s.Exceptions = append(s.Exceptions, Exception{
		Code:     code,
		Severity: severity,
		Message:  message,
	})
	s.HasExceptions = true
}

func (s *Settlement) ClearExceptions() {
	s.Exceptions = nil
	s.HasExceptions = false
}

func (s *Settlement) IsEditable() bool {
	return s.Status == StatusDraft || s.Status == StatusPendingApproval
}

func (s *Settlement) GetID() pulid.ID { return s.ID }

func (s *Settlement) GetCreatedAt() int64 { return s.CreatedAt }

func (s *Settlement) GetOrganizationID() pulid.ID { return s.OrganizationID }

func (s *Settlement) GetBusinessUnitID() pulid.ID { return s.BusinessUnitID }

func (s *Settlement) GetTableName() string { return "driver_settlements" }

func (s *Settlement) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "dstl",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{
				Name:   "settlement_number",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightA,
			},
			{
				Name:   "pay_profile_name",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
		},
	}
}

func (s *Settlement) BeforeCreate(ctx context.Context) error {
	if s.Status == "" {
		s.Status = StatusDraft
	}
	if s.CurrencyCode == "" {
		s.CurrencyCode = money.DefaultCurrencyCode
	}

	return nil
}

func (Settlement) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "uq_driver_settlements_number",
			Message: "Settlement with this number already exists in your organization",
		},
	}
}
