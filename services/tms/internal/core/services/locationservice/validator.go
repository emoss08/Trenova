package locationservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	DB *postgres.Connection
}

type Validator struct {
	validator *validationframework.TenantedValidator[*location.Location]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		validator: validationframework.
			NewTenantedValidatorBuilder[*location.Location]().
			WithModelName("Location").
			WithUniquenessChecker(validationframework.NewBunUniquenessCheckerLazy(func() bun.IDB { return p.DB.DB() })).
			WithReferenceChecker(validationframework.NewBunReferenceCheckerLazy(func() bun.IDB { return p.DB.DB() })).
			WithUniqueField(
				"code",
				"code",
				"Location with this code already exists in your organization",
				func(l *location.Location) any { return l.Code },
			).
			WithCustomReferenceCheck(
				"stateId",
				"State does not exist",
				func(l *location.Location) pulid.ID { return l.StateID },
				createStateCheck(p.DB),
			).
			WithCustomReferenceCheck(
				"locationCategoryId",
				"Location category does not exist",
				func(l *location.Location) pulid.ID { return l.LocationCategoryID },
				createLocationCategoryCheck(p.DB),
			).
			Build(),
	}
}

func createStateCheck(
	db *postgres.Connection,
) validationframework.CustomReferenceCheckFunc {
	return func(ctx context.Context, _, _ pulid.ID, refID pulid.ID) (bool, error) {
		if refID.IsNil() {
			return true, nil
		}

		exists, err := db.DB().NewSelect().
			TableExpr("us_states").
			ColumnExpr("1").
			Where("id = ?", refID).
			Exists(ctx)
		if err != nil {
			return false, err
		}
		return exists, nil
	}
}

func createLocationCategoryCheck(
	db *postgres.Connection,
) validationframework.CustomReferenceCheckFunc {
	return func(ctx context.Context, orgID, buID pulid.ID, refID pulid.ID) (bool, error) {
		if refID.IsNil() {
			return true, nil
		}

		exists, err := db.DB().NewSelect().
			TableExpr("location_categories").
			ColumnExpr("1").
			Where("id = ?", refID).
			Where("organization_id = ?", orgID).
			Where("business_unit_id = ?", buID).
			Exists(ctx)
		if err != nil {
			return false, err
		}
		return exists, nil
	}
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *location.Location,
) *errortypes.MultiError {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *location.Location,
) *errortypes.MultiError {
	return v.validator.ValidateUpdate(ctx, entity)
}
