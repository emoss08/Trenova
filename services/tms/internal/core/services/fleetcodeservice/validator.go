package fleetcodeservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/fleetcode"
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
	validator *validationframework.TenantedValidator[*fleetcode.FleetCode]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		validator: validationframework.
			NewTenantedValidatorBuilder[*fleetcode.FleetCode]().
			WithModelName("FleetCode").
			WithUniquenessChecker(validationframework.NewBunUniquenessCheckerLazy(func() bun.IDB { return p.DB.DB() })).
			WithReferenceChecker(validationframework.NewBunReferenceCheckerLazy(func() bun.IDB { return p.DB.DB() })).
			WithUniqueField(
				"code",
				"code",
				"Fleet code with this code already exists in your organization",
				func(fc *fleetcode.FleetCode) any { return fc.Code },
			).
			WithCustomReferenceCheck(
				"managerId",
				"Manager does not exist or is not a member of your organization",
				func(fc *fleetcode.FleetCode) pulid.ID { return fc.ManagerID },
				createUserOrganizationCheck(p.DB),
			).
			Build(),
	}
}

func createUserOrganizationCheck(
	db *postgres.Connection,
) validationframework.CustomReferenceCheckFunc {
	return func(ctx context.Context, orgID, _ pulid.ID, refID pulid.ID) (bool, error) {
		exists, err := db.DB().NewSelect().
			TableExpr("user_organization_memberships").
			ColumnExpr("1").
			Where("user_id = ?", refID).
			Where("organization_id = ?", orgID).
			Exists(ctx)
		if err != nil {
			return false, err
		}
		return exists, nil
	}
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *fleetcode.FleetCode,
) *errortypes.MultiError {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *fleetcode.FleetCode,
) *errortypes.MultiError {
	return v.validator.ValidateUpdate(ctx, entity)
}
