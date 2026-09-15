package latecharge

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*LateChargeAssessment)(nil)

// LateChargeAssessment records one late charge on one invoice for one period.
// The unique key on (invoice, period) is what makes a run idempotent: a second
// run on the same day inserts nothing and raises no memo.
type LateChargeAssessment struct {
	bun.BaseModel `bun:"table:late_charge_assessments,alias:lca" json:"-"`

	ID                    pulid.ID        `json:"id"                    bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID        pulid.ID        `json:"organizationId"        bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID        pulid.ID        `json:"businessUnitId"        bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	CustomerID            pulid.ID        `json:"customerId"            bun:"customer_id,type:VARCHAR(100),notnull"`
	SourceInvoiceID       pulid.ID        `json:"sourceInvoiceId"       bun:"source_invoice_id,type:VARCHAR(100),notnull"`
	PeriodIndex           int             `json:"periodIndex"           bun:"period_index,type:INTEGER,notnull"`
	PeriodStart           int64           `json:"periodStart"           bun:"period_start,type:BIGINT,notnull"`
	PeriodEnd             int64           `json:"periodEnd"             bun:"period_end,type:BIGINT,notnull"`
	AsOfDate              int64           `json:"asOfDate"              bun:"as_of_date,type:BIGINT,notnull"`
	BasisOpenBalanceMinor int64           `json:"basisOpenBalanceMinor" bun:"basis_open_balance_minor,type:BIGINT,notnull"`
	RatePercent           decimal.Decimal `json:"ratePercent"           bun:"rate_percent,type:NUMERIC(9,4),notnull"`
	ChargeMinor           int64           `json:"chargeMinor"           bun:"charge_minor,type:BIGINT,notnull"`
	DebitMemoInvoiceID    pulid.ID        `json:"debitMemoInvoiceId"    bun:"debit_memo_invoice_id,type:VARCHAR(100),nullzero"`
	DebitMemoLineID       pulid.ID        `json:"debitMemoLineId"       bun:"debit_memo_line_id,type:VARCHAR(100),nullzero"`
	RunKey                string          `json:"runKey"                bun:"run_key,type:VARCHAR(100),notnull"`
	CreatedByID           pulid.ID        `json:"createdById"           bun:"created_by_id,type:VARCHAR(100),nullzero"`
	CreatedAt             int64           `json:"createdAt"             bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt             int64           `json:"updatedAt"             bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	SourceInvoice *invoice.Invoice `json:"sourceInvoice,omitempty" bun:"rel:belongs-to,join:source_invoice_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	DebitMemo     *invoice.Invoice `json:"debitMemo,omitempty"     bun:"rel:belongs-to,join:debit_memo_invoice_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
}

func (a *LateChargeAssessment) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		a,
		validation.Field(&a.OrganizationID, validation.Required),
		validation.Field(&a.BusinessUnitID, validation.Required),
		validation.Field(&a.CustomerID, validation.Required.Error("Customer is required")),
		validation.Field(&a.SourceInvoiceID, validation.Required.Error("Source invoice is required")),
		validation.Field(&a.PeriodIndex, validation.Min(1).Error("Period index starts at 1")),
		validation.Field(&a.AsOfDate, validation.Required.Error("As-of date is required")),
		validation.Field(&a.ChargeMinor, validation.Min(int64(1)).Error("Charge must be greater than zero")),
		validation.Field(&a.RunKey, validation.Required.Error("Run key is required")),
	))
	if a.PeriodEnd < a.PeriodStart {
		multiErr.Add("periodEnd", errortypes.ErrInvalid, "Period end must not precede its start")
	}
}

func (a *LateChargeAssessment) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if a.ID.IsNil() {
			a.ID = pulid.MustNew("lca_")
		}
		a.CreatedAt = now
	case *bun.UpdateQuery:
		a.UpdatedAt = now
	}

	return nil
}

func (a *LateChargeAssessment) GetID() pulid.ID {
	return a.ID
}

func (a *LateChargeAssessment) GetOrganizationID() pulid.ID {
	return a.OrganizationID
}

func (a *LateChargeAssessment) GetBusinessUnitID() pulid.ID {
	return a.BusinessUnitID
}
