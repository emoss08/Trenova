package fleetcodeservice

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/fleetcode"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/fleetcode/fleetcodedb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/keelcheck"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	DB *postgres.Connection
}

type Validator struct {
	validator tenantedvalidation.EntityValidator[*fleetcode.FleetCode]
}

func NewValidator(p ValidatorParams) *Validator {
	checker := keelcheck.NewChecker(p.DB, fleetcodedb.FleetCodesSchema())

	return &Validator{
		validator: tenantedvalidation.NewBuilder[*fleetcode.FleetCode]().
			WithModelName("FleetCode").
			WithUniquenessChecker(checker).
			WithReferenceChecker(checker).
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
	return func(ctx context.Context, _, _ pulid.ID, refID pulid.ID) (bool, error) {
		return dbtx.Read(ctx, db, func(ctx context.Context) (bool, error) {
			m := tenantdb.OrganizationMemberships
			return m.Where(m.UserID.Eq(refID)).Exists(ctx, db.Keel(ctx))
		})
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
