package tenant

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

var (
	_ validationframework.TenantedEntity = (*SettlementControl)(nil)
)

type PayPeriodFrequency string

const (
	PayPeriodFrequencyWeekly   = PayPeriodFrequency("Weekly")
	PayPeriodFrequencyBiweekly = PayPeriodFrequency("Biweekly")
	PayPeriodFrequencyMonthly  = PayPeriodFrequency("Monthly")
)

func (p PayPeriodFrequency) String() string { return string(p) }

func (p PayPeriodFrequency) IsValid() bool {
	switch p {
	case PayPeriodFrequencyWeekly, PayPeriodFrequencyBiweekly, PayPeriodFrequencyMonthly:
		return true
	default:
		return false
	}
}

type PayTrigger string

const (
	PayTriggerMoveCompleted     = PayTrigger("MoveCompleted")
	PayTriggerShipmentDelivered = PayTrigger("ShipmentDelivered")
	PayTriggerPODReceived       = PayTrigger("PODReceived")
	PayTriggerShipmentInvoiced  = PayTrigger("ShipmentInvoiced")
)

func (p PayTrigger) String() string { return string(p) }

func (p PayTrigger) IsValid() bool {
	switch p {
	case PayTriggerMoveCompleted, PayTriggerShipmentDelivered,
		PayTriggerPODReceived, PayTriggerShipmentInvoiced:
		return true
	default:
		return false
	}
}

type SettlementControl struct {
	keel.Table `keel:"settlement_controls"`

	ID             pulid.ID `json:"id" keel:"id,pk,autoid:stlc_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	PayPeriodFrequency            PayPeriodFrequency `json:"payPeriodFrequency" keel:"pay_period_frequency"`
	PeriodEndDayOfWeek            int                `json:"periodEndDayOfWeek" keel:"period_end_day_of_week"`
	PayDelayDays                  int                `json:"payDelayDays" keel:"pay_delay_days"`
	PayTrigger                    PayTrigger         `json:"payTrigger" keel:"pay_trigger"`
	AutoGenerateBatches           bool               `json:"autoGenerateBatches" keel:"auto_generate_batches"`
	AutoApproveClean              bool               `json:"autoApproveClean" keel:"auto_approve_clean"`
	AutoAttachAccruals            bool               `json:"autoAttachAccruals" keel:"auto_attach_accruals"`
	AutoPostOnApprove             bool               `json:"autoPostOnApprove" keel:"auto_post_on_approve"`
	AllowNegativeNet              bool               `json:"allowNegativeNet" keel:"allow_negative_net"`
	VarianceThresholdPct          decimal.Decimal    `json:"varianceThresholdPct" keel:"variance_threshold_pct"`
	VarianceLookbackWeeks         int                `json:"varianceLookbackWeeks" keel:"variance_lookback_weeks"`
	DefaultEscrowInterestRate     decimal.Decimal    `json:"defaultEscrowInterestRate" keel:"default_escrow_interest_rate"`
	EscrowInterestFrequencyMonths int                `json:"escrowInterestFrequencyMonths" keel:"escrow_interest_frequency_months"`

	Version   int64 `json:"version" keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	BusinessUnit *BusinessUnit `json:"businessUnit,omitempty"`
	Organization *Organization `json:"organization,omitempty"`
}

func (sc *SettlementControl) Validate(multiErr *errortypes.MultiError) {
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
	if sc.VarianceThresholdPct.IsNegative() {
		multiErr.Add(
			"varianceThresholdPct",
			errortypes.ErrInvalid,
			"Variance threshold cannot be negative",
		)
	}
	if sc.VarianceLookbackWeeks < 1 || sc.VarianceLookbackWeeks > 52 {
		multiErr.Add(
			"varianceLookbackWeeks",
			errortypes.ErrInvalid,
			"Variance lookback weeks must be between 1 and 52",
		)
	}
	if sc.DefaultEscrowInterestRate.IsNegative() ||
		sc.DefaultEscrowInterestRate.GreaterThan(decimal.NewFromInt(100)) {
		multiErr.Add(
			"defaultEscrowInterestRate",
			errortypes.ErrInvalid,
			"Default escrow interest rate must be between 0 and 100",
		)
	}
	if sc.EscrowInterestFrequencyMonths < 1 || sc.EscrowInterestFrequencyMonths > 3 {
		multiErr.Add(
			"escrowInterestFrequencyMonths",
			errortypes.ErrInvalid,
			"Escrow interest must accrue at least quarterly (1 to 3 months) per 49 CFR 376.12(k)",
		)
	}
}

func (sc *SettlementControl) GetID() pulid.ID { return sc.ID }

func (sc *SettlementControl) GetTableName() string { return "settlement_controls" }

func (sc *SettlementControl) GetOrganizationID() pulid.ID { return sc.OrganizationID }

func (sc *SettlementControl) GetBusinessUnitID() pulid.ID { return sc.BusinessUnitID }

func (sc *SettlementControl) BeforeCreate(ctx context.Context) error {
	if sc.PayPeriodFrequency == "" {
		sc.PayPeriodFrequency = PayPeriodFrequencyWeekly
	}
	if sc.PeriodEndDayOfWeek == 0 {
		sc.PeriodEndDayOfWeek = 6
	}
	if sc.PayDelayDays == 0 {
		sc.PayDelayDays = 5
	}
	if sc.PayTrigger == "" {
		sc.PayTrigger = PayTriggerShipmentDelivered
	}
	if sc.VarianceThresholdPct == 0 {
		sc.VarianceThresholdPct = 25
	}
	if sc.VarianceLookbackWeeks == 0 {
		sc.VarianceLookbackWeeks = 8
	}
	if sc.EscrowInterestFrequencyMonths == 0 {
		sc.EscrowInterestFrequencyMonths = 3
	}

	return nil
}
