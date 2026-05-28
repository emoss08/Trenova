package iamservice

import (
	"context"
	"maps"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/iam"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	Repo repositories.IAMRepository
	DB   *postgres.Connection
}

type Validator struct {
	validator *validationframework.TenantedValidator[*iam.AccessPolicy]
}

func NewValidator(p ValidatorParams) *Validator {
	return newValidator(p.Repo, validationframework.NewBunUniquenessCheckerLazy(func() bun.IDB {
		return p.DB.DB()
	}))
}

func newValidator(
	repo repositories.IAMRepository,
	uniquenessChecker validationframework.UniquenessChecker,
) *Validator {
	return &Validator{
		validator: validationframework.
			NewTenantedValidatorBuilder[*iam.AccessPolicy]().
			WithModelName("Access Policy").
			WithUniquenessChecker(uniquenessChecker).
			WithUniqueField(
				"name",
				"name",
				"Access policy with this name already exists",
				func(policy *iam.AccessPolicy) any { return policy.Name },
			).
			WithCustomRule(newAccessPolicyDecisionScopeRule(repo)).
			Build(),
	}
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *iam.AccessPolicy,
) *errortypes.MultiError {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *iam.AccessPolicy,
) *errortypes.MultiError {
	return v.validator.ValidateUpdate(ctx, entity)
}

func newAccessPolicyDecisionScopeRule(
	repo repositories.IAMRepository,
) validationframework.TenantedRule[*iam.AccessPolicy] {
	return validationframework.NewTenantedRule[*iam.AccessPolicy]("access_policy_decision_scope").
		WithStage(validationframework.ValidationStageDataIntegrity).
		WithPriority(validationframework.ValidationPriorityHigh).
		OnBoth().
		WithValidation(func(
			ctx context.Context,
			entity *iam.AccessPolicy,
			_ *validationframework.TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			if !entity.Enabled {
				return nil
			}

			policies, err := repo.ListEnabledAccessPolicies(
				ctx,
				repositories.IAMPolicyLookupRequest{
					OrganizationID: entity.OrganizationID,
					BusinessUnitID: entity.BusinessUnitID,
					Resource:       entity.Resource,
					Operation:      permission.Operation(entity.Operation),
				},
			)
			if err != nil {
				return err
			}

			match := findOverlappingAccessPolicy(entity, policies)
			if match == nil {
				return nil
			}

			if match.Effect == entity.Effect {
				multiErr.Add(
					"resource",
					errortypes.ErrDuplicate,
					"An enabled access policy with the same resource, operation, effect, and conditions already exists",
				)
				return nil
			}

			multiErr.Add(
				"effect",
				errortypes.ErrDuplicate,
				"An enabled access policy with the same resource, operation, and conditions already has the opposite effect",
			)
			return nil
		})
}

func findOverlappingAccessPolicy(
	entity *iam.AccessPolicy,
	policies []*iam.AccessPolicy,
) *iam.AccessPolicy {
	for _, policy := range policies {
		if accessPoliciesOverlap(entity, policy) {
			return policy
		}
	}
	return nil
}

func accessPoliciesConflict(left *iam.AccessPolicy, right *iam.AccessPolicy) bool {
	if !accessPoliciesOverlap(left, right) {
		return false
	}
	return left.Effect != right.Effect
}

func accessPoliciesDuplicate(left *iam.AccessPolicy, right *iam.AccessPolicy) bool {
	if !accessPoliciesOverlap(left, right) {
		return false
	}
	return left.Effect == right.Effect
}

func accessPoliciesOverlap(left *iam.AccessPolicy, right *iam.AccessPolicy) bool {
	if left == nil || right == nil {
		return false
	}
	if !left.Enabled || !right.Enabled {
		return false
	}
	if left.ID.IsNotNil() && left.ID == right.ID {
		return false
	}
	if left.Resource != right.Resource || left.Operation != right.Operation {
		return false
	}
	return maps.Equal(
		normalizeAccessPolicyConditions(left.Conditions),
		normalizeAccessPolicyConditions(right.Conditions),
	)
}

func normalizeAccessPolicyConditions(conditions map[string]string) map[string]string {
	if len(conditions) == 0 {
		return map[string]string{}
	}

	normalized := make(map[string]string, len(conditions))
	for key, value := range conditions {
		normalized[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return normalized
}
