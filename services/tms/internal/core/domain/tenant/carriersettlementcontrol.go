package tenant

import (
	"context"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*CarrierSettlementControl)(nil)
	_ validationframework.TenantedEntity = (*CarrierSettlementControl)(nil)
)

type CarrierSettlementControl struct {
	bun.BaseModel `bun:"table:carrier_settlement_controls,alias:carstlc" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`

	PayTrigger                              PayTrigger         `json:"payTrigger"                              bun:"pay_trigger,type:VARCHAR(50),notnull,default:'ShipmentDelivered'"`
	PayPeriodFrequency                      PayPeriodFrequency `json:"payPeriodFrequency"                      bun:"pay_period_frequency,type:VARCHAR(50),notnull,default:'Weekly'"`
	PeriodEndDayOfWeek                      int                `json:"periodEndDayOfWeek"                      bun:"period_end_day_of_week,type:INTEGER,notnull,default:6"`
	PayDelayDays                            int                `json:"payDelayDays"                            bun:"pay_delay_days,type:INTEGER,notnull,default:5"`
	AutoGenerateBatches                     bool               `json:"autoGenerateBatches"                     bun:"auto_generate_batches,type:BOOLEAN,notnull,default:false"`
	AutoPostOnApprove                       bool               `json:"autoPostOnApprove"                       bun:"auto_post_on_approve,type:BOOLEAN,notnull,default:false"`
	VarianceToleranceMinor                  int64              `json:"varianceToleranceMinor"                  bun:"variance_tolerance_minor,type:BIGINT,notnull,default:0"`
	DefaultAPAccountID                      *pulid.ID          `json:"defaultApAccountId"                      bun:"default_ap_account_id,type:VARCHAR(100),nullzero"`
	DefaultPurchasedTransportationAccountID *pulid.ID          `json:"defaultPurchasedTransportationAccountId" bun:"default_purchased_transportation_account_id,type:VARCHAR(100),nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit *BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
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

func (sc *CarrierSettlementControl) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if sc.ID.IsNil() {
			sc.ID = pulid.MustNew("carstlc_")
		}
		sc.CreatedAt = now
	case *bun.UpdateQuery:
		sc.UpdatedAt = now
	}
	return nil
}
