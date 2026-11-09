package organizationservice

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
	"github.com/emoss08/trenova/pkg/validationframework"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	Integrity validationframework.IntegrityChecker

	DB *postgres.Connection
}

type Validator struct {
	validator tenantedvalidation.EntityValidator[*tenant.Organization]
}

func NewValidator(p ValidatorParams) *Validator {
	builder := tenantedvalidation.NewBuilder[*tenant.Organization]().
		WithModelName("Organization").
		WithIntegrityChecker(p.Integrity).
		WithIntegrityScope(businessUnitWideScope).
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
			validationframework.NewRule[*tenant.Organization]("operating_capabilities").
				OnUpdate().
				WithValidation(validateOperatingCapabilities),
		)

	return &Validator{
		validator: builder.Build(),
	}
}

func businessUnitWideScope(ctx context.Context) context.Context {
	return dbscope.WithSystem(
		ctx,
		"check organization names, SCAC and DOT numbers and login slugs across the business unit",
	)
}

func validateOperatingCapabilities(
	_ context.Context,
	entity *tenant.Organization,
	_ *validationframework.Context,
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
) error {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *tenant.Organization,
) error {
	return v.validator.ValidateUpdate(ctx, entity)
}
