package driversettlement

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/driverpay"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/worker"
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

type Settlement struct {
	bun.BaseModel             `bun:"table:driver_settlements,alias:dstl" json:"-"`
	pagination.CursorValueSet `bun:",embed"                              json:"-"`

	ID                   pulid.ID                      `json:"id"                   bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID       pulid.ID                      `json:"businessUnitId"       bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID       pulid.ID                      `json:"organizationId"       bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	WorkerID             pulid.ID                      `json:"workerId"             bun:"worker_id,type:VARCHAR(100),notnull"`
	BatchID              *pulid.ID                     `json:"batchId"              bun:"batch_id,type:VARCHAR(100),nullzero"`
	PayProfileID         *pulid.ID                     `json:"payProfileId"         bun:"pay_profile_id,type:VARCHAR(100),nullzero"`
	SettlementNumber     string                        `json:"settlementNumber"     bun:"settlement_number,type:VARCHAR(100),notnull"`
	Status               Status                        `json:"status"               bun:"status,type:VARCHAR(50),notnull,default:'Draft'"`
	Classification       driverpay.PayeeClassification `json:"classification"       bun:"classification,type:VARCHAR(50),notnull"`
	PayProfileName       string                        `json:"payProfileName"       bun:"pay_profile_name,type:VARCHAR(100),nullzero"`
	PeriodStart          int64                         `json:"periodStart"          bun:"period_start,type:BIGINT,notnull"`
	PeriodEnd            int64                         `json:"periodEnd"            bun:"period_end,type:BIGINT,notnull"`
	PayDate              int64                         `json:"payDate"              bun:"pay_date,type:BIGINT,notnull"`
	GrossEarningsMinor   int64                         `json:"grossEarningsMinor"   bun:"gross_earnings_minor,type:BIGINT,notnull,default:0"`
	ReimbursementsMinor  int64                         `json:"reimbursementsMinor"  bun:"reimbursements_minor,type:BIGINT,notnull,default:0"`
	DeductionsMinor      int64                         `json:"deductionsMinor"      bun:"deductions_minor,type:BIGINT,notnull,default:0"`
	CarryForwardInMinor  int64                         `json:"carryForwardInMinor"  bun:"carry_forward_in_minor,type:BIGINT,notnull,default:0"`
	CarryForwardOutMinor int64                         `json:"carryForwardOutMinor" bun:"carry_forward_out_minor,type:BIGINT,notnull,default:0"`
	NetPayMinor          int64                         `json:"netPayMinor"          bun:"net_pay_minor,type:BIGINT,notnull,default:0"`
	TotalMiles           decimal.Decimal               `json:"totalMiles"           bun:"total_miles,type:NUMERIC(19,4),notnull,default:0"`
	ShipmentCount        int                           `json:"shipmentCount"        bun:"shipment_count,type:INTEGER,notnull,default:0"`
	CurrencyCode         string                        `json:"currencyCode"         bun:"currency_code,type:VARCHAR(3),notnull,default:'USD'"`
	HasExceptions        bool                          `json:"hasExceptions"        bun:"has_exceptions,type:BOOLEAN,notnull,default:false"`
	Exceptions           []Exception                   `json:"exceptions"           bun:"exceptions,type:JSONB,nullzero"`
	Notes                string                        `json:"notes"                bun:"notes,type:TEXT,nullzero"`
	SubmittedByID        pulid.ID                      `json:"submittedById"        bun:"submitted_by_id,type:VARCHAR(100),nullzero"`
	SubmittedAt          *int64                        `json:"submittedAt"          bun:"submitted_at,type:BIGINT,nullzero"`
	ApprovedByID         pulid.ID                      `json:"approvedById"         bun:"approved_by_id,type:VARCHAR(100),nullzero"`
	ApprovedAt           *int64                        `json:"approvedAt"           bun:"approved_at,type:BIGINT,nullzero"`
	PostedByID           pulid.ID                      `json:"postedById"           bun:"posted_by_id,type:VARCHAR(100),nullzero"`
	PostedAt             *int64                        `json:"postedAt"             bun:"posted_at,type:BIGINT,nullzero"`
	PostedJournalBatchID *pulid.ID                     `json:"postedJournalBatchId" bun:"posted_journal_batch_id,type:VARCHAR(100),nullzero"`
	PaidAt               *int64                        `json:"paidAt"               bun:"paid_at,type:BIGINT,nullzero"`
	PaidByID             pulid.ID                      `json:"paidById"             bun:"paid_by_id,type:VARCHAR(100),nullzero"`
	PaymentMethod        string                        `json:"paymentMethod"        bun:"payment_method,type:VARCHAR(50),nullzero"`
	PaymentReference     string                        `json:"paymentReference"     bun:"payment_reference,type:VARCHAR(100),nullzero"`
	VoidedByID           pulid.ID                      `json:"voidedById"           bun:"voided_by_id,type:VARCHAR(100),nullzero"`
	VoidedAt             *int64                        `json:"voidedAt"             bun:"voided_at,type:BIGINT,nullzero"`
	VoidReason           string                        `json:"voidReason"           bun:"void_reason,type:TEXT,nullzero"`
	VoidJournalBatchID   *pulid.ID                     `json:"voidJournalBatchId"   bun:"void_journal_batch_id,type:VARCHAR(100),nullzero"`
	Version              int64                         `json:"version"              bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt            int64                         `json:"createdAt"            bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt            int64                         `json:"updatedAt"            bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit *tenant.BusinessUnit  `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization  `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	Worker       *worker.Worker        `json:"worker,omitempty"       bun:"rel:belongs-to,join:worker_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	Batch        *SettlementBatch      `json:"batch,omitempty"        bun:"rel:belongs-to,join:batch_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	PayProfile   *driverpay.PayProfile `json:"payProfile,omitempty"   bun:"rel:belongs-to,join:pay_profile_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	Lines        []*SettlementLine     `json:"lines,omitempty"        bun:"rel:has-many,join:id=settlement_id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
}

type SettlementLine struct {
	bun.BaseModel `bun:"table:driver_settlement_lines,alias:dstll" json:"-"`

	ID                   pulid.ID                `json:"id"                   bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID       pulid.ID                `json:"businessUnitId"       bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID       pulid.ID                `json:"organizationId"       bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	SettlementID         pulid.ID                `json:"settlementId"         bun:"settlement_id,type:VARCHAR(100),notnull"`
	LineNumber           int                     `json:"lineNumber"           bun:"line_number,type:INTEGER,notnull"`
	Category             LineCategory            `json:"category"             bun:"category,type:VARCHAR(50),notnull"`
	ComponentKind        driverpay.ComponentKind `json:"componentKind"        bun:"component_kind,type:VARCHAR(50),nullzero"`
	Method               driverpay.CalcMethod    `json:"method"               bun:"method,type:VARCHAR(50),nullzero"`
	Description          string                  `json:"description"          bun:"description,type:VARCHAR(255),notnull"`
	Quantity             decimal.Decimal         `json:"quantity"             bun:"quantity,type:NUMERIC(19,4),notnull,default:0"`
	Rate                 decimal.Decimal         `json:"rate"                 bun:"rate,type:NUMERIC(19,4),notnull,default:0"`
	AmountMinor          int64                   `json:"amountMinor"          bun:"amount_minor,type:BIGINT,notnull"`
	ShipmentID           *pulid.ID               `json:"shipmentId"           bun:"shipment_id,type:VARCHAR(100),nullzero"`
	MoveID               *pulid.ID               `json:"moveId"               bun:"move_id,type:VARCHAR(100),nullzero"`
	PayEventID           *pulid.ID               `json:"payEventId"           bun:"pay_event_id,type:VARCHAR(100),nullzero"`
	RecurringDeductionID *pulid.ID               `json:"recurringDeductionId" bun:"recurring_deduction_id,type:VARCHAR(100),nullzero"`
	RecurringEarningID   *pulid.ID               `json:"recurringEarningId"   bun:"recurring_earning_id,type:VARCHAR(100),nullzero"`
	PayCodeID            *pulid.ID               `json:"payCodeId"            bun:"pay_code_id,type:VARCHAR(100),nullzero"`
	AdvanceID            *pulid.ID               `json:"advanceId"            bun:"advance_id,type:VARCHAR(100),nullzero"`
	EscrowAccountID      *pulid.ID               `json:"escrowAccountId"      bun:"escrow_account_id,type:VARCHAR(100),nullzero"`
	ProNumber            string                  `json:"proNumber"            bun:"pro_number,type:VARCHAR(100),nullzero"`
	CreatedAt            int64                   `json:"createdAt"            bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt            int64                   `json:"updatedAt"            bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (s *Settlement) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(s,
		validation.Field(&s.WorkerID, validation.Required.Error("Worker is required")),
		validation.Field(&s.PeriodStart, validation.Required.Error("Period start is required")),
		validation.Field(&s.PeriodEnd, validation.Required.Error("Period end is required")),
		validation.Field(&s.PayDate, validation.Required.Error("Pay date is required")),
		validation.Field(&s.CurrencyCode,
			validation.Required.Error("Currency code is required"),
			validation.Length(3, 3).Error("Currency code must be 3 characters"),
		),
	)
	if err != nil {
		if validationErrs, ok := errors.AsType[validation.Errors](err); ok {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}

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
