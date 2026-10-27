package latecharge

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

// LateChargeAssessment records one late charge on one invoice for one period.
// The unique key on (invoice, period) is what makes a run idempotent: a second
// run on the same day inserts nothing and raises no memo.
type LateChargeAssessment struct {
	keel.Table `keel:"late_charge_assessments"`

	ID                    pulid.ID        `json:"id"                    keel:"id,pk,autoid:lca_"`
	OrganizationID        pulid.ID        `json:"organizationId"        keel:"organization_id,pk,tenant"`
	BusinessUnitID        pulid.ID        `json:"businessUnitId"        keel:"business_unit_id,pk,tenant"`
	CustomerID            pulid.ID        `json:"customerId"            keel:"customer_id"`
	SourceInvoiceID       pulid.ID        `json:"sourceInvoiceId"       keel:"source_invoice_id"`
	PeriodIndex           int             `json:"periodIndex"           keel:"period_index"`
	PeriodStart           int64           `json:"periodStart"           keel:"period_start"`
	PeriodEnd             int64           `json:"periodEnd"             keel:"period_end"`
	AsOfDate              int64           `json:"asOfDate"              keel:"as_of_date"`
	BasisOpenBalanceMinor int64           `json:"basisOpenBalanceMinor" keel:"basis_open_balance_minor"`
	RatePercent           decimal.Decimal `json:"ratePercent"           keel:"rate_percent"`
	ChargeMinor           int64           `json:"chargeMinor"           keel:"charge_minor"`
	DebitMemoInvoiceID    pulid.ID        `json:"debitMemoInvoiceId"    keel:"debit_memo_invoice_id,nullzero"`
	DebitMemoLineID       pulid.ID        `json:"debitMemoLineId"       keel:"debit_memo_line_id,nullzero"`
	RunKey                string          `json:"runKey"                keel:"run_key"`
	CreatedByID           pulid.ID        `json:"createdById"           keel:"created_by_id,nullzero"`
	CreatedAt             int64           `json:"createdAt"             keel:"created_at,autocreate"`
	UpdatedAt             int64           `json:"updatedAt"             keel:"updated_at,autoupdate"`

	SourceInvoice *invoice.Invoice `json:"sourceInvoice,omitempty" keel:"belongs:source_invoice_id"`
	DebitMemo     *invoice.Invoice `json:"debitMemo,omitempty"     keel:"belongs:debit_memo_invoice_id"`
}

func (a *LateChargeAssessment) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		a,
		validation.Field(&a.OrganizationID, validation.Required),
		validation.Field(&a.BusinessUnitID, validation.Required),
		validation.Field(&a.CustomerID, validation.Required.Error("Customer is required")),
		validation.Field(
			&a.SourceInvoiceID,
			validation.Required.Error("Source invoice is required"),
		),
		validation.Field(&a.PeriodIndex, validation.Min(1).Error("Period index starts at 1")),
		validation.Field(&a.AsOfDate, validation.Required.Error("As-of date is required")),
		validation.Field(
			&a.ChargeMinor,
			validation.Min(int64(1)).Error("Charge must be greater than zero"),
		),
		validation.Field(&a.RunKey, validation.Required.Error("Run key is required")),
	))
	if a.PeriodEnd < a.PeriodStart {
		multiErr.Add("periodEnd", errortypes.ErrInvalid, "Period end must not precede its start")
	}
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
