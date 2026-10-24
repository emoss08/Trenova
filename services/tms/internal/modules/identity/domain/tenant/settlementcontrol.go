package tenant

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*SettlementControl)(nil)
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
	bun.BaseModel `bun:"table:settlement_controls,alias:stlc" json:"-"`
	keel.Table    `keel:"settlement_controls"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull" keel:"id,pk,autoid:stlc_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`

	PayPeriodFrequency            PayPeriodFrequency `json:"payPeriodFrequency"            bun:"pay_period_frequency,type:VARCHAR(50),notnull,default:'Weekly'" keel:"pay_period_frequency"`
	PeriodEndDayOfWeek            int                `json:"periodEndDayOfWeek"            bun:"period_end_day_of_week,type:INTEGER,notnull,default:6" keel:"period_end_day_of_week"`
	PayDelayDays                  int                `json:"payDelayDays"                  bun:"pay_delay_days,type:INTEGER,notnull,default:5" keel:"pay_delay_days"`
	PayTrigger                    PayTrigger         `json:"payTrigger"                    bun:"pay_trigger,type:VARCHAR(50),notnull,default:'ShipmentDelivered'" keel:"pay_trigger"`
	AutoGenerateBatches           bool               `json:"autoGenerateBatches"           bun:"auto_generate_batches,type:BOOLEAN,notnull" keel:"auto_generate_batches"`
	AutoApproveClean              bool               `json:"autoApproveClean"              bun:"auto_approve_clean,type:BOOLEAN,notnull" keel:"auto_approve_clean"`
	AutoAttachAccruals            bool               `json:"autoAttachAccruals"            bun:"auto_attach_accruals,type:BOOLEAN,notnull" keel:"auto_attach_accruals"`
	AutoPostOnApprove             bool               `json:"autoPostOnApprove"             bun:"auto_post_on_approve,type:BOOLEAN,notnull" keel:"auto_post_on_approve"`
	AllowNegativeNet              bool               `json:"allowNegativeNet"              bun:"allow_negative_net,type:BOOLEAN,notnull" keel:"allow_negative_net"`
	VarianceThresholdPct          decimal.Decimal    `json:"varianceThresholdPct"          bun:"variance_threshold_pct,type:NUMERIC(7,4),notnull,default:25" keel:"variance_threshold_pct"`
	VarianceLookbackWeeks         int                `json:"varianceLookbackWeeks"         bun:"variance_lookback_weeks,type:INTEGER,notnull,default:8" keel:"variance_lookback_weeks"`
	DefaultEscrowInterestRate     decimal.Decimal    `json:"defaultEscrowInterestRate"     bun:"default_escrow_interest_rate,type:NUMERIC(7,4),notnull,default:0" keel:"default_escrow_interest_rate"`
	EscrowInterestFrequencyMonths int                `json:"escrowInterestFrequencyMonths" bun:"escrow_interest_frequency_months,type:INTEGER,notnull,default:3" keel:"escrow_interest_frequency_months"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT,notnull" keel:"version,version"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`

	BusinessUnit *BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
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

func (sc *SettlementControl) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if sc.ID.IsNil() {
			sc.ID = pulid.MustNew("stlc_")
		}
		sc.CreatedAt = now
	case *bun.UpdateQuery:
		sc.UpdatedAt = now
	}
	return nil
}
