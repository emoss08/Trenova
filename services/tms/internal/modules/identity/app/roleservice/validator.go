package roleservice

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/identity/domain/rbac"
	"github.com/emoss08/trenova/internal/modules/identity/domain/rbac/rbacdb"
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
	validator tenantedvalidation.EntityValidator[*rbac.Role]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		validator: tenantedvalidation.NewBuilder[*rbac.Role]().
			WithModelName("Role").
			WithUniquenessChecker(
				keelcheck.NewChecker(p.DB, rbacdb.RolesSchema()),
			).
			WithUniqueField(
				"name",
				"name",
				"Role with this name already exists in your organization",
				func(e *rbac.Role) any { return e.Name },
			).
			Build(),
	}
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *rbac.Role,
) *errortypes.MultiError {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *rbac.Role,
) *errortypes.MultiError {
	return v.validator.ValidateUpdate(ctx, entity)
}
