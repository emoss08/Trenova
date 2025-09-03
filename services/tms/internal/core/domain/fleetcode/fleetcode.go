/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package fleetcode

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/domain/user"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/rotisserie/eris"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*FleetCode)(nil)
	_ domain.Validatable        = (*FleetCode)(nil)
)

type FleetCode struct {
	bun.BaseModel `bun:"table:fleet_codes,alias:fc" json:"-"`

	// Primary identifiers
	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100)"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`

	// Relationship identifiers (Non-Primary-Keys)
	ManagerID pulid.ID `json:"managerId" bun:"manager_id,type:VARCHAR(100),notnull"`

	// Core fields
	Status       domain.Status       `json:"status"       bun:"status,type:status_enum,notnull,default:'Active'"`
	Name         string              `json:"name"         bun:"name,type:VARCHAR(100),notnull"`
	Description  string              `json:"description"  bun:"description,type:TEXT"`
	RevenueGoal  decimal.NullDecimal `json:"revenueGoal"  bun:"revenue_goal,type:NUMERIC(10,2),nullzero"`
	DeadheadGoal decimal.NullDecimal `json:"deadheadGoal" bun:"deadhead_goal,type:NUMERIC(10,2),nullzero"`
	MileageGoal  decimal.NullDecimal `json:"mileageGoal"  bun:"mileage_goal,type:NUMERIC(10,2),nullzero"`
	Color        string              `json:"color"        bun:"color,type:VARCHAR(10)"`

	// Metadata
	Version      int64  `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt    int64  `json:"createdAt" bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt    int64  `json:"updatedAt" bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	SearchVector string `json:"-"         bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank         string `json:"-"         bun:"rank,type:VARCHAR(100),scanonly"`

	// Relationships
	BusinessUnit *businessunit.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *organization.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	Manager      *user.User                 `json:"manager,omitempty"      bun:"rel:belongs-to,join:manager_id=id"`
}

func (fc *FleetCode) Validate(ctx context.Context, multiErr *errors.MultiError) {
	err := validation.ValidateStructWithContext(ctx, fc,
		// Name is required and must be between 1 and 100 characters
		validation.Field(&fc.Name,
			validation.Required.Error("Name is required. Please try again"),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
		),

		// Ensure revenue and deadhead goals are not negative
		validation.Field(&fc.RevenueGoal,
			validation.Min(0).Error("Revenue goal must be greater than or equal to 0"),
		),
		validation.Field(&fc.DeadheadGoal,
			validation.Min(0).Error("Deadhead goal must be greater than or equal to 0"),
		),

		// Manager is required
		validation.Field(&fc.ManagerID,
			validation.Required.Error("Manager is required"),
		),

		// Color must be a valid hex color
		validation.Field(&fc.Color,
			is.HexColor.Error("Color must be a valid hex color. Please try again."),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if eris.As(err, &validationErrs) {
			errors.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

// Pagination Configuration
func (fc *FleetCode) GetID() string {
	return fc.ID.String()
}

func (fc *FleetCode) GetTableName() string {
	return "fleet_codes"
}

func (fc *FleetCode) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if fc.ID.IsNil() {
			fc.ID = pulid.MustNew("fc_")
		}

		fc.CreatedAt = now
	case *bun.UpdateQuery:
		fc.UpdatedAt = now
	}

	return nil
}
