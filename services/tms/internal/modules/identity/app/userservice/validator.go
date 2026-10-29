package userservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
	"github.com/emoss08/trenova/pkg/validationframework"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	Repo tenantports.UserRepository
}

type Validator struct {
	validator tenantedvalidation.EntityValidator[*tenant.User]
}

func NewValidator(p ValidatorParams) *Validator {
	return newValidator(p.Repo)
}

func newValidator(users tenantports.UserRepository) *Validator {
	builder := tenantedvalidation.NewBuilder[*tenant.User]().
		WithModelName("User")

	if users != nil {
		builder = builder.WithCustomRule(identityUniquenessRule(users))
	}

	return &Validator{validator: builder.Build()}
}

func identityUniquenessRule(
	users tenantports.UserRepository,
) validationframework.TenantedRule[*tenant.User] {
	return validationframework.NewTenantedRule[*tenant.User]("identity_uniqueness").
		WithStage(validationframework.ValidationStageDataIntegrity).
		WithPriority(validationframework.ValidationPriorityHigh).
		WithValidation(func(
			ctx context.Context,
			entity *tenant.User,
			valCtx *validationframework.TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			conflicts, err := users.FindIdentityConflicts(
				ctx,
				tenantports.UserIdentityConflictsRequest{
					EmailAddress:  entity.EmailAddress,
					Username:      entity.Username,
					ExcludeUserID: valCtx.EntityID,
				},
			)
			if err != nil {
				return err
			}

			if conflicts.EmailAddress {
				multiErr.AddError(tenant.NewDuplicateEmailAddressError())
			}
			if conflicts.Username {
				multiErr.AddError(tenant.NewDuplicateUsernameError())
			}

			return nil
		})
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *tenant.User,
) *errortypes.MultiError {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *tenant.User,
) *errortypes.MultiError {
	return v.validator.ValidateUpdate(ctx, entity)
}
