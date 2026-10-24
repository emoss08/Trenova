package costingcontrol

import (
	"context"
	"errors"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/rating/domain/fuelsurcharge"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

var (
	_ validationframework.TenantedEntity = (*CostingControl)(nil)
)

type CostingControl struct {
	keel.Table `keel:"costing_controls"`

	ID                   pulid.ID            `json:"id" keel:"id,pk,autoid:cstc_"`
	BusinessUnitID       pulid.ID            `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID       pulid.ID            `json:"organizationId" keel:"organization_id,pk,tenant"`
	FuelIndexID          *pulid.ID           `json:"fuelIndexId" keel:"fuel_index_id"`
	UseLiveFuelPrice     bool                `json:"useLiveFuelPrice" keel:"use_live_fuel_price"`
	MilesPerGallon       decimal.Decimal     `json:"milesPerGallon" keel:"miles_per_gallon"`
	IncludeDeadheadMiles bool                `json:"includeDeadheadMiles" keel:"include_deadhead_miles"`
	GLActualsEnabled     bool                `json:"glActualsEnabled" keel:"gl_actuals_enabled"`
	GLRollingMonths      int16               `json:"glRollingMonths" keel:"gl_rolling_months"`
	PlannedMonthlyMiles  *int64              `json:"plannedMonthlyMiles" keel:"planned_monthly_miles"`
	TargetMarginPercent  decimal.NullDecimal `json:"targetMarginPercent" keel:"target_margin_percent"`
	Version              int64               `json:"version" keel:"version,version"`
	CreatedAt            int64               `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt            int64               `json:"updatedAt" keel:"updated_at,autoupdate"`

	// Relationships
	FuelIndex    *fuelsurcharge.FuelIndex `json:"fuelIndex,omitempty" keel:"belongs:fuel_index_id"`
	BusinessUnit *tenant.BusinessUnit     `json:"businessUnit,omitempty"`
	Organization *tenant.Organization     `json:"organization,omitempty"`
	Categories   []*CostCategory          `json:"categories,omitempty" keel:"many:costing_control_id"`
}

func (cc *CostingControl) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
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
	))

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

func (cc *CostingControl) BeforeCreate(ctx context.Context) error {
	if cc.MilesPerGallon.IsZero() {
		cc.MilesPerGallon = decimal.RequireFromString("6.5")
	}
	if cc.GLRollingMonths == 0 {
		cc.GLRollingMonths = 3
	}

	return nil
}
