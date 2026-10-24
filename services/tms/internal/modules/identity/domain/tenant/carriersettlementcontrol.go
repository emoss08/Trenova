package tenant

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ validationframework.TenantedEntity = (*CarrierSettlementControl)(nil)
)

type CarrierSettlementControl struct {
	keel.Table `keel:"carrier_settlement_controls"`

	ID             pulid.ID `json:"id" keel:"id,pk,autoid:carstlc_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	PayTrigger                              PayTrigger         `json:"payTrigger" keel:"pay_trigger"`
	PayPeriodFrequency                      PayPeriodFrequency `json:"payPeriodFrequency" keel:"pay_period_frequency"`
	PeriodEndDayOfWeek                      int                `json:"periodEndDayOfWeek" keel:"period_end_day_of_week"`
	PayDelayDays                            int                `json:"payDelayDays" keel:"pay_delay_days"`
	AutoGenerateBatches                     bool               `json:"autoGenerateBatches" keel:"auto_generate_batches"`
	AutoPostOnApprove                       bool               `json:"autoPostOnApprove" keel:"auto_post_on_approve"`
	VarianceToleranceMinor                  int64              `json:"varianceToleranceMinor" keel:"variance_tolerance_minor"`
	AutoMatchInboundInvoices                bool               `json:"autoMatchInboundInvoices" keel:"auto_match_inbound_invoices"`
	AutoAcceptWithinTolerance               bool               `json:"autoAcceptWithinTolerance" keel:"auto_accept_within_tolerance"`
	HoldUntilInvoiceMatched                 bool               `json:"holdUntilInvoiceMatched" keel:"hold_until_invoice_matched"`
	DefaultAPAccountID                      *pulid.ID          `json:"defaultApAccountId" keel:"default_ap_account_id"`
	DefaultPurchasedTransportationAccountID *pulid.ID          `json:"defaultPurchasedTransportationAccountId" keel:"default_purchased_transportation_account_id"`

	Version   int64 `json:"version" keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	BusinessUnit *BusinessUnit `json:"businessUnit,omitempty"`
	Organization *Organization `json:"organization,omitempty"`
}

func (sc *CarrierSettlementControl) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(sc,
		validation.Field(&sc.PayPeriodFrequency, validation.Required),
		validation.Field(&sc.PayTrigger, validation.Required),
	))

	if !sc.PayPeriodFrequency.IsValid() {
		multiErr.Add(
			"payPeriodFrequency",
			errortypes.ErrInvalid,
			"Pay period frequency is invalid",
		)
	}
	if !sc.PayTrigger.IsValid() {
		multiErr.Add("payTrigger", errortypes.ErrInvalid, "Pay trigger is invalid")
	}
	if sc.PeriodEndDayOfWeek < 0 || sc.PeriodEndDayOfWeek > 6 {
		multiErr.Add(
			"periodEndDayOfWeek",
			errortypes.ErrInvalid,
			"Period end day of week must be between 0 (Sunday) and 6 (Saturday)",
		)
	}
	if sc.PayDelayDays < 0 || sc.PayDelayDays > 30 {
		multiErr.Add(
			"payDelayDays",
			errortypes.ErrInvalid,
			"Pay delay days must be between 0 and 30",
		)
	}
	if sc.VarianceToleranceMinor < 0 {
		multiErr.Add(
			"varianceToleranceMinor",
			errortypes.ErrInvalid,
			"Variance tolerance cannot be negative",
		)
	}
}

func (sc *CarrierSettlementControl) GetID() pulid.ID { return sc.ID }

func (sc *CarrierSettlementControl) GetTableName() string { return "carrier_settlement_controls" }

func (sc *CarrierSettlementControl) GetOrganizationID() pulid.ID { return sc.OrganizationID }

func (sc *CarrierSettlementControl) GetBusinessUnitID() pulid.ID { return sc.BusinessUnitID }

func (sc *CarrierSettlementControl) BeforeCreate(ctx context.Context) error {
	if sc.PayTrigger == "" {
		sc.PayTrigger = PayTriggerShipmentDelivered
	}
	if sc.PayPeriodFrequency == "" {
		sc.PayPeriodFrequency = PayPeriodFrequencyWeekly
	}
	if sc.PeriodEndDayOfWeek == 0 {
		sc.PeriodEndDayOfWeek = 6
	}
	if sc.PayDelayDays == 0 {
		sc.PayDelayDays = 5
	}

	return nil
}
