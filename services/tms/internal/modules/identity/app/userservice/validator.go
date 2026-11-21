package userservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/pulid"

	"github.com/emoss08/trenova/internal/infrastructure/config/managedconfig"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
	"github.com/emoss08/trenova/pkg/validationframework"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	Repo       tenantports.UserRepository
	Managed    *managedconfig.Config
	RateLimits kernelports.RateLimitStore
}

var identityProbePolicy = kernelports.RateLimitPolicy{Rate: 300, Period: time.Hour, Burst: 60}

type Validator struct {
	validator tenantedvalidation.EntityValidator[*tenant.User]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		validator: tenantedvalidation.NewBuilder[*tenant.User]().
			WithModelName("User").
			WithCustomRule(identityUniquenessRule(p.Repo, p.RateLimits, p.Managed.SharedTenancy())).
			Build(),
	}
}

func identityUniquenessRule(
	users tenantports.UserRepository,
	limits kernelports.RateLimitStore,
	sharedTenancy bool,
) validationframework.Rule[*tenant.User] {
	return validationframework.NewRule[*tenant.User]("identity_uniqueness").
		WithStage(validationframework.StageDataIntegrity).
		WithValidation(func(
			ctx context.Context,
			entity *tenant.User,
			valCtx *validationframework.Context,
			multiErr *errortypes.MultiError,
		) error {
			if sharedTenancy {
				if err := admitIdentityProbe(ctx, limits, valCtx.OrganizationID); err != nil {
					return err
				}
			}
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
				multiErr.AddError(tenant.EmailAddressTakenError(sharedTenancy))
			}
			if conflicts.Username {
				multiErr.AddError(tenant.UsernameTakenError(sharedTenancy))
			}

			return nil
		})
}

func admitIdentityProbe(
	ctx context.Context,
	limits kernelports.RateLimitStore,
	organizationID pulid.ID,
) error {
	decisions, err := limits.Check(ctx, []kernelports.RateLimitRequest{{
		Key:    "identity-probe:" + organizationID.String(),
		Policy: identityProbePolicy,
		Cost:   1,
	}})
	if err != nil {
		return err
	}
	if len(decisions) == 1 && !decisions[0].Allowed {
		return errortypes.NewRateLimitError(
			"emailAddress",
			"This organization has checked as many email addresses and usernames as it may for now. Try again later.",
		).WithRetryAfter(decisions[0].RetryAfter)
	}
	return nil
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *tenant.User,
) error {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *tenant.User,
) error {
	return v.validator.ValidateUpdate(ctx, entity)
}
