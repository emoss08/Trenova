package organizationservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	DB *postgres.Connection
}

type Validator struct {
	validator *validationframework.ScopedValidator[*tenant.Organization]
}

func NewValidator(p ValidatorParams) *Validator {
	builder := validationframework.NewScopedValidatorBuilder[*tenant.Organization]().
		WithModelName("Organization").
		WithScopeField(
			"businessUnitId",
			"business_unit_id",
			func(o *tenant.Organization) any { return o.BusinessUnitID },
		).
		WithUniqueField(
			"name",
			"name",
			"Organization with this name already exists in this business unit",
			func(o *tenant.Organization) any { return o.Name },
		).
		WithCaseSensitiveUniqueField(
			"scacCode",
			"scac_code",
			"Organization with this SCAC code already exists in this business unit",
			func(o *tenant.Organization) any { return o.ScacCode },
		).
		WithCaseSensitiveUniqueField(
			"dotNumber",
			"dot_number",
			"Organization with this DOT number already exists in this business unit",
			func(o *tenant.Organization) any { return o.DOTNumber },
		)

	if p.DB != nil {
		builder.WithUniquenessChecker(
			validationframework.NewBunUniquenessCheckerLazy(
				func() bun.IDB { return p.DB.DB() },
			),
		)
	}

	return &Validator{
		validator: builder.Build(),
	}
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *tenant.Organization,
) *errortypes.MultiError {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *tenant.Organization,
) *errortypes.MultiError {
	return v.validator.ValidateUpdate(ctx, entity)
}
