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
		).
		WithCaseSensitiveUniqueField(
			"loginSlug",
			"login_slug",
			"Organization with this login slug already exists",
			func(o *tenant.Organization) any { return o.LoginSlug },
		).
		WithCustomRule(
			validationframework.NewScopedRule[*tenant.Organization]("operating_capabilities").
				OnUpdate().
				WithParallelSafeExecution().
				WithValidation(validateOperatingCapabilities),
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

func validateOperatingCapabilities(
	_ context.Context,
	entity *tenant.Organization,
	_ *validationframework.ScopedValidationContext,
	multiErr *errortypes.MultiError,
) error {
	if entity.BrokerageEnabled || entity.AssetOperationsEnabled {
		return nil
	}

	const message = "At least one operating capability must be enabled. " +
		"An organization with neither brokerage nor asset operations cannot move freight"

	multiErr.Add(brokerageEnabledField, errortypes.ErrInvalid, message)
	multiErr.Add(assetOperationsEnabledField, errortypes.ErrInvalid, message)

	return nil
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
