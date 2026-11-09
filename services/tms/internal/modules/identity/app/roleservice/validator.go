package roleservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/rbac"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
	"github.com/emoss08/trenova/pkg/validationframework"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	Integrity validationframework.IntegrityChecker
}

type Validator struct {
	validator tenantedvalidation.EntityValidator[*rbac.Role]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		validator: tenantedvalidation.NewBuilder[*rbac.Role]().
			WithModelName("Role").
			WithIntegrityChecker(
				p.Integrity,
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
) error {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *rbac.Role,
) error {
	return v.validator.ValidateUpdate(ctx, entity)
}
