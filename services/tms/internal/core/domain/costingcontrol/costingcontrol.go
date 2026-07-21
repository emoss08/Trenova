package costingcontrol

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/fuelsurcharge"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*CostingControl)(nil)
	_ validationframework.TenantedEntity = (*CostingControl)(nil)
)

type CostingControl struct {
	bun.BaseModel `bun:"table:costing_controls,alias:cstc" json:"-"`

	ID                   pulid.ID            `json:"id"                   bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID       pulid.ID            `json:"businessUnitId"       bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID       pulid.ID            `json:"organizationId"       bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	FuelIndexID          *pulid.ID           `json:"fuelIndexId"          bun:"fuel_index_id,type:VARCHAR(100),nullzero"`
	UseLiveFuelPrice     bool                `json:"useLiveFuelPrice"     bun:"use_live_fuel_price,type:BOOLEAN,notnull,default:true"`
	MilesPerGallon       decimal.Decimal     `json:"milesPerGallon"       bun:"miles_per_gallon,type:NUMERIC(6,2),notnull,default:6.5"`
	IncludeDeadheadMiles bool                `json:"includeDeadheadMiles" bun:"include_deadhead_miles,type:BOOLEAN,notnull,default:true"`
	GLActualsEnabled     bool                `json:"glActualsEnabled"     bun:"gl_actuals_enabled,type:BOOLEAN,notnull,default:false"`
	GLRollingMonths      int16               `json:"glRollingMonths"      bun:"gl_rolling_months,type:SMALLINT,notnull,default:3"`
	PlannedMonthlyMiles  *int64              `json:"plannedMonthlyMiles"  bun:"planned_monthly_miles,type:BIGINT,nullzero"`
	TargetMarginPercent  decimal.NullDecimal `json:"targetMarginPercent"  bun:"target_margin_percent,type:NUMERIC(6,3),nullzero"`
	Version              int64               `json:"version"              bun:"version,type:BIGINT"`
	CreatedAt            int64               `json:"createdAt"            bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt            int64               `json:"updatedAt"            bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	FuelIndex    *fuelsurcharge.FuelIndex `json:"fuelIndex,omitempty"    bun:"rel:belongs-to,join:fuel_index_id=id"`
	BusinessUnit *tenant.BusinessUnit     `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization     `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	Categories   []*CostCategory          `json:"categories,omitempty"   bun:"rel:has-many,join:id=costing_control_id"`
}

func (cc *CostingControl) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(
		cc,
		validation.Field(&cc.MilesPerGallon,
			validation.Required.Error("Miles per gallon is required"),
			validation.By(func(any) error {
				if cc.MilesPerGallon.LessThanOrEqual(decimal.Zero) {
					return errors.New("Miles per gallon must be greater than 0")
				}
				if cc.MilesPerGallon.GreaterThan(decimal.NewFromInt(20)) {
					return errors.New("Miles per gallon must be 20 or less")
				}
				return nil
			}),
		),
		validation.Field(&cc.FuelIndexID,
			validation.When(
				cc.UseLiveFuelPrice,
				validation.Required.Error(
					"Fuel index is required when live fuel pricing is enabled",
				),
			),
		),
		validation.Field(&cc.GLRollingMonths,
			validation.Required.Error("GL rolling months is required"),
			validation.Min(1).Error("GL rolling months must be at least 1"),
			validation.Max(12).Error("GL rolling months must be 12 or less"),
		),
		validation.Field(&cc.PlannedMonthlyMiles,
			validation.By(func(any) error {
				if cc.PlannedMonthlyMiles != nil && *cc.PlannedMonthlyMiles < 1 {
					return errors.New("Planned monthly miles must be greater than 0")
				}
				return nil
			}),
		),
		validation.Field(&cc.TargetMarginPercent,
			validation.By(func(any) error {
				if !cc.TargetMarginPercent.Valid {
					return nil
				}
				if cc.TargetMarginPercent.Decimal.LessThan(decimal.Zero) ||
					cc.TargetMarginPercent.Decimal.GreaterThan(decimal.NewFromInt(100)) {
					return errors.New("Target margin percent must be between 0 and 100")
				}
				return nil
			}),
		),
	)
	if err != nil {
		if validationErrs, ok := errors.AsType[validation.Errors](err); ok {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}

	for i, cat := range cc.Categories {
		if cat == nil {
			continue
		}
		cat.Validate(multiErr.WithIndex("categories", i))
	}
}

func (cc *CostingControl) GetID() pulid.ID {
	return cc.ID
}

func (cc *CostingControl) GetTableName() string {
	return "costing_controls"
}

func (cc *CostingControl) GetOrganizationID() pulid.ID {
	return cc.OrganizationID
}

func (cc *CostingControl) GetBusinessUnitID() pulid.ID {
	return cc.BusinessUnitID
}

func (cc *CostingControl) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if cc.ID.IsNil() {
			cc.ID = pulid.MustNew("cstc_")
		}

		cc.CreatedAt = now
	case *bun.UpdateQuery:
		cc.UpdatedAt = now
	}

	return nil
}
