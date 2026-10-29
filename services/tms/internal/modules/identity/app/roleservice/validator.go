package roleservice

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission/permissiondb"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/keelcheck"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	DB *postgres.Connection
}

type Validator struct {
	validator tenantedvalidation.EntityValidator[*permission.Role]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		validator: tenantedvalidation.NewBuilder[*permission.Role]().
			WithModelName("Role").
			WithUniquenessChecker(
				keelcheck.NewChecker(p.DB, permissiondb.RolesSchema()),
			).
			WithUniqueField(
				"name",
				"name",
				"Role with this name already exists in your organization",
				func(e *permission.Role) any { return e.Name },
			).
			Build(),
	}
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *permission.Role,
) *errortypes.MultiError {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *permission.Role,
) *errortypes.MultiError {
	return v.validator.ValidateUpdate(ctx, entity)
}
