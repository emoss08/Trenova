package driversettlement

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*Settlement)(nil)
	_ domaintypes.PostgresSearchable     = (*Settlement)(nil)
	_ pagination.CursorEntity            = (*Settlement)(nil)
	_ validationframework.TenantedEntity = (*Settlement)(nil)
	_ bun.BeforeAppendModelHook          = (*SettlementLine)(nil)
)

type Exception struct {
	Code     ExceptionCode     `json:"code"`
	Severity ExceptionSeverity `json:"severity"`
	Message  string            `json:"message"`
}

// Driver and owner-operator pay statements
type Settlement struct {
	bun.BaseModel `bun:"table:driver_settlements,alias:dstl" json:"-"`
	keel.Table    `                                                   keel:"driver_settlements,report=Payroll,label=Driver Settlement"`

	ID                     pulid.ID                      `json:"id"                     bun:"id,pk,type:VARCHAR(100),notnull"                                                      keel:"id,pk,autoid:dstl_"`
	BusinessUnitID         pulid.ID                      `json:"businessUnitId"         bun:"business_unit_id,pk,type:VARCHAR(100),notnull"                                        keel:"business_unit_id,pk,tenant"`
	OrganizationID         pulid.ID                      `json:"organizationId"         bun:"organization_id,pk,type:VARCHAR(100),notnull"                                         keel:"organization_id,pk,tenant"`
	WorkerID               pulid.ID                      `json:"workerId"               bun:"worker_id,type:VARCHAR(100),notnull"                                                  keel:"worker_id"`
	BatchID                *pulid.ID                     `json:"batchId"                bun:"batch_id,type:VARCHAR(100),nullzero"                                                  keel:"batch_id"`
	PayProfileID           *pulid.ID                     `json:"payProfileId"           bun:"pay_profile_id,type:VARCHAR(100),nullzero"                                            keel:"pay_profile_id"`
	SettlementNumber       string                        `json:"settlementNumber"       bun:"settlement_number,type:VARCHAR(100),notnull"                                          keel:"settlement_number"`
	Status                 Status                        `json:"status"                 bun:"status,type:VARCHAR(50),notnull,default:'Draft'"                                      keel:"status"`
	Classification         driverpay.PayeeClassification `json:"classification"         bun:"classification,type:VARCHAR(50),notnull"                                              keel:"classification"`
	PayProfileName         string                        `json:"payProfileName"         bun:"pay_profile_name,type:VARCHAR(100),nullzero"                                          keel:"pay_profile_name,nullzero,label=Pay Profile"`
	PeriodStart            int64                         `json:"periodStart"            bun:"period_start,type:BIGINT,notnull"                                                     keel:"period_start,type=epoch"`
	PeriodEnd              int64                         `json:"periodEnd"              bun:"period_end,type:BIGINT,notnull"                                                       keel:"period_end,type=epoch"`
	PayDate                int64                         `json:"payDate"                bun:"pay_date,type:BIGINT,notnull"                                                         keel:"pay_date"`
	GrossEarningsMinor     int64                         `json:"grossEarningsMinor"     bun:"gross_earnings_minor,type:BIGINT,notnull"                                             keel:"gross_earnings_minor,label=Gross Earnings (Minor)"`
	ReimbursementsMinor    int64                         `json:"reimbursementsMinor"    bun:"reimbursements_minor,type:BIGINT,notnull"                                             keel:"reimbursements_minor,label=Reimbursements (Minor)"`
	DeductionsMinor        int64                         `json:"deductionsMinor"        bun:"deductions_minor,type:BIGINT,notnull"                                                 keel:"deductions_minor,label=Deductions (Minor)"`
	CarryForwardInMinor    int64                         `json:"carryForwardInMinor"    bun:"carry_forward_in_minor,type:BIGINT,notnull"                                           keel:"carry_forward_in_minor,label=Carry Forward In (Minor)"`
	CarryForwardOutMinor   int64                         `json:"carryForwardOutMinor"   bun:"carry_forward_out_minor,type:BIGINT,notnull"                                          keel:"carry_forward_out_minor,label=Carry Forward Out (Minor)"`
	NetPayMinor            int64                         `json:"netPayMinor"            bun:"net_pay_minor,type:BIGINT,notnull"                                                    keel:"net_pay_minor,label=Net Pay (Minor)"`
	TotalMiles             decimal.Decimal               `json:"totalMiles"             bun:"total_miles,type:NUMERIC(19,4),notnull,default:0"                                     keel:"total_miles,format=distance"`
	ShipmentCount          int                           `json:"shipmentCount"          bun:"shipment_count,type:INTEGER,notnull"                                                  keel:"shipment_count,label=Shipments,format=count"`
	CurrencyCode           string                        `json:"currencyCode"           bun:"currency_code,type:VARCHAR(3),notnull,default:'USD'"                                  keel:"currency_code,label=Currency"`
	ExchangeRate           decimal.NullDecimal           `json:"exchangeRate"           bun:"exchange_rate,type:NUMERIC(24,12),nullzero"                                           keel:"exchange_rate"`
	ExchangeRateDate       *int64                        `json:"exchangeRateDate"       bun:"exchange_rate_date,type:BIGINT,nullzero"                                              keel:"exchange_rate_date"`
	HasExceptions          bool                          `json:"hasExceptions"          bun:"has_exceptions,type:BOOLEAN,notnull"                                                  keel:"has_exceptions"`
	Exceptions             []Exception                   `json:"exceptions"             bun:"exceptions,type:JSONB,nullzero"                                                       keel:"exceptions,jsonb,nullzero,noreport"`
	Notes                  string                        `json:"notes"                  bun:"notes,type:TEXT,nullzero"                                                             keel:"notes,nullzero"`
	SubmittedByID          pulid.ID                      `json:"submittedById"          bun:"submitted_by_id,type:VARCHAR(100),nullzero"                                           keel:"submitted_by_id,nullzero"`
	SubmittedAt            *int64                        `json:"submittedAt"            bun:"submitted_at,type:BIGINT,nullzero"                                                    keel:"submitted_at"`
	ApprovedByID           pulid.ID                      `json:"approvedById"           bun:"approved_by_id,type:VARCHAR(100),nullzero"                                            keel:"approved_by_id,nullzero"`
	ApprovedAt             *int64                        `json:"approvedAt"             bun:"approved_at,type:BIGINT,nullzero"                                                     keel:"approved_at"`
	PostedByID             pulid.ID                      `json:"postedById"             bun:"posted_by_id,type:VARCHAR(100),nullzero"                                              keel:"posted_by_id,nullzero"`
	PostedAt               *int64                        `json:"postedAt"               bun:"posted_at,type:BIGINT,nullzero"                                                       keel:"posted_at"`
	PostedJournalBatchID   *pulid.ID                     `json:"postedJournalBatchId"   bun:"posted_journal_batch_id,type:VARCHAR(100),nullzero"                                   keel:"posted_journal_batch_id"`
	PostedPayableAccountID *pulid.ID                     `json:"postedPayableAccountId" bun:"posted_payable_account_id,type:VARCHAR(100),nullzero"                                 keel:"posted_payable_account_id"`
	PaidAt                 *int64                        `json:"paidAt"                 bun:"paid_at,type:BIGINT,nullzero"                                                         keel:"paid_at"`
	PaidByID               pulid.ID                      `json:"paidById"               bun:"paid_by_id,type:VARCHAR(100),nullzero"                                                keel:"paid_by_id,nullzero"`
	PaymentMethod          string                        `json:"paymentMethod"          bun:"payment_method,type:VARCHAR(50),nullzero"                                             keel:"payment_method,nullzero"`
	PaymentReference       string                        `json:"paymentReference"       bun:"payment_reference,type:VARCHAR(100),nullzero"                                         keel:"payment_reference,nullzero"`
	PaidJournalBatchID     *pulid.ID                     `json:"paidJournalBatchId"     bun:"paid_journal_batch_id,type:VARCHAR(100),nullzero"                                     keel:"paid_journal_batch_id"`
	PaidExchangeRate       decimal.NullDecimal           `json:"paidExchangeRate"       bun:"paid_exchange_rate,type:NUMERIC(24,12),nullzero"                                      keel:"paid_exchange_rate"`
	PaidExchangeRateDate   *int64                        `json:"paidExchangeRateDate"   bun:"paid_exchange_rate_date,type:BIGINT,nullzero"                                         keel:"paid_exchange_rate_date"`
	VoidedByID             pulid.ID                      `json:"voidedById"             bun:"voided_by_id,type:VARCHAR(100),nullzero"                                              keel:"voided_by_id,nullzero"`
	VoidedAt               *int64                        `json:"voidedAt"               bun:"voided_at,type:BIGINT,nullzero"                                                       keel:"voided_at"`
	VoidReason             string                        `json:"voidReason"             bun:"void_reason,type:TEXT,nullzero"                                                       keel:"void_reason,nullzero"`
	VoidJournalBatchID     *pulid.ID                     `json:"voidJournalBatchId"     bun:"void_journal_batch_id,type:VARCHAR(100),nullzero"                                     keel:"void_journal_batch_id"`
	Version                int64                         `json:"version"                bun:"version,type:BIGINT,notnull"                                                          keel:"version,version"`
	CreatedAt              int64                         `json:"createdAt"              bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt              int64                         `json:"updatedAt"              bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit  `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization  `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	Worker       *worker.Worker        `json:"worker,omitempty"       bun:"rel:belongs-to,join:worker_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"      keel:"belongs:worker_id,label=Driver"`
	Batch        *SettlementBatch      `json:"batch,omitempty"        bun:"rel:belongs-to,join:batch_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"       keel:"belongs:batch_id,label=Settlement Batch"`
	PayProfile   *driverpay.PayProfile `json:"payProfile,omitempty"   bun:"rel:belongs-to,join:pay_profile_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id" keel:"belongs:pay_profile_id,noreport"`
	Lines        []*SettlementLine     `json:"lines,omitempty"        bun:"rel:has-many,join:id=settlement_id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"    keel:"many:settlement_id,label=Settlement Lines"`
}

// Each earning, deduction and reimbursement on a settlement
type SettlementLine struct {
	bun.BaseModel `bun:"table:driver_settlement_lines,alias:dstll" json:"-"`
	keel.Table    `                                                         keel:"driver_settlement_lines,report=Payroll"`

	ID                   pulid.ID                `json:"id"                   bun:"id,pk,type:VARCHAR(100),notnull"                                                      keel:"id,pk,autoid:dstll_"`
	BusinessUnitID       pulid.ID                `json:"businessUnitId"       bun:"business_unit_id,pk,type:VARCHAR(100),notnull"                                        keel:"business_unit_id,pk,tenant"`
	OrganizationID       pulid.ID                `json:"organizationId"       bun:"organization_id,pk,type:VARCHAR(100),notnull"                                         keel:"organization_id,pk,tenant"`
	SettlementID         pulid.ID                `json:"settlementId"         bun:"settlement_id,type:VARCHAR(100),notnull"                                              keel:"settlement_id"`
	LineNumber           int                     `json:"lineNumber"           bun:"line_number,type:INTEGER,notnull"                                                     keel:"line_number,label=Line,dimension"`
	Category             LineCategory            `json:"category"             bun:"category,type:VARCHAR(50),notnull"                                                    keel:"category"`
	ComponentKind        driverpay.ComponentKind `json:"componentKind"        bun:"component_kind,type:VARCHAR(50),nullzero"                                             keel:"component_kind,nullzero,label=Component"`
	Method               driverpay.CalcMethod    `json:"method"               bun:"method,type:VARCHAR(50),nullzero"                                                     keel:"method,nullzero,label=Calculation Method"`
	Description          string                  `json:"description"          bun:"description,type:VARCHAR(255),notnull"                                                keel:"description"`
	Quantity             decimal.Decimal         `json:"quantity"             bun:"quantity,type:NUMERIC(19,4),notnull,default:0"                                        keel:"quantity"`
	Rate                 decimal.Decimal         `json:"rate"                 bun:"rate,type:NUMERIC(19,4),notnull,default:0"                                            keel:"rate"`
	AmountMinor          int64                   `json:"amountMinor"          bun:"amount_minor,type:BIGINT,notnull"                                                     keel:"amount_minor,label=Amount (Minor)"`
	ShipmentID           *pulid.ID               `json:"shipmentId"           bun:"shipment_id,type:VARCHAR(100),nullzero"                                               keel:"shipment_id"`
	MoveID               *pulid.ID               `json:"moveId"               bun:"move_id,type:VARCHAR(100),nullzero"                                                   keel:"move_id"`
	PayEventID           *pulid.ID               `json:"payEventId"           bun:"pay_event_id,type:VARCHAR(100),nullzero"                                              keel:"pay_event_id"`
	RecurringDeductionID *pulid.ID               `json:"recurringDeductionId" bun:"recurring_deduction_id,type:VARCHAR(100),nullzero"                                    keel:"recurring_deduction_id"`
	RecurringEarningID   *pulid.ID               `json:"recurringEarningId"   bun:"recurring_earning_id,type:VARCHAR(100),nullzero"                                      keel:"recurring_earning_id"`
	PayCodeID            *pulid.ID               `json:"payCodeId"            bun:"pay_code_id,type:VARCHAR(100),nullzero"                                               keel:"pay_code_id"`
	AdvanceID            *pulid.ID               `json:"advanceId"            bun:"advance_id,type:VARCHAR(100),nullzero"                                                keel:"advance_id"`
	EscrowAccountID      *pulid.ID               `json:"escrowAccountId"      bun:"escrow_account_id,type:VARCHAR(100),nullzero"                                         keel:"escrow_account_id"`
	ProNumber            string                  `json:"proNumber"            bun:"pro_number,type:VARCHAR(100),nullzero"                                                keel:"pro_number,nullzero"`
	AddedByAgent         bool                    `json:"addedByAgent"         bun:"added_by_agent,type:BOOLEAN,notnull,default:false"                                    keel:"added_by_agent"`
	CreatedAt            int64                   `json:"createdAt"            bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt            int64                   `json:"updatedAt"            bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`

	Settlement *Settlement        `json:"settlement,omitempty" bun:"rel:belongs-to,join:settlement_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id" keel:"belongs:settlement_id"`
	Shipment   *shipment.Shipment `json:"shipment,omitempty"   bun:"rel:belongs-to,join:shipment_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"   keel:"belongs:shipment_id"`
	PayCode    *driverpay.PayCode `json:"payCode,omitempty"    bun:"rel:belongs-to,join:pay_code_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"   keel:"belongs:pay_code_id"`
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

func (s *Settlement) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if s.ID.IsNil() {
			s.ID = pulid.MustNew("dstl_")
		}
		s.CreatedAt = now
	case *bun.UpdateQuery:
		s.UpdatedAt = now
	}
	return nil
}

func (s *Settlement) BeforeCreate(ctx context.Context) error {
	return s.BeforeAppendModel(ctx, (*bun.InsertQuery)(nil))
}

func (l *SettlementLine) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if l.ID.IsNil() {
			l.ID = pulid.MustNew("dstll_")
		}
		l.CreatedAt = now
	case *bun.UpdateQuery:
		l.UpdatedAt = now
	}
	return nil
}

func (l *SettlementLine) BeforeCreate(ctx context.Context) error {
	return l.BeforeAppendModel(ctx, (*bun.InsertQuery)(nil))
}
