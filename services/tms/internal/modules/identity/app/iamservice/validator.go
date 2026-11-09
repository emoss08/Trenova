package iamservice

import (
	"context"
	"maps"
	"strings"

	"github.com/emoss08/trenova/internal/modules/identity/domain/iam"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/iamports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
	"github.com/emoss08/trenova/pkg/validationframework"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	Integrity validationframework.IntegrityChecker
	Repo      iamports.IAMRepository
}

type Validator struct {
	validator tenantedvalidation.EntityValidator[*iam.AccessPolicy]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		validator: tenantedvalidation.NewBuilder[*iam.AccessPolicy]().
			WithModelName("Access Policy").
			WithIntegrityChecker(p.Integrity).
			WithUniqueField(
				"name",
				"name",
				"Access policy with this name already exists",
				func(policy *iam.AccessPolicy) any { return policy.Name },
			).
			WithCustomRule(newAccessPolicyDecisionScopeRule(p.Repo)).
			Build(),
	}
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *iam.AccessPolicy,
) error {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *iam.AccessPolicy,
) error {
	return v.validator.ValidateUpdate(ctx, entity)
}

func newAccessPolicyDecisionScopeRule(
	repo iamports.IAMRepository,
) validationframework.Rule[*iam.AccessPolicy] {
	return validationframework.NewRule[*iam.AccessPolicy]("access_policy_decision_scope").
		WithStage(validationframework.StageDataIntegrity).
		OnBoth().
		WithValidation(func(
			ctx context.Context,
			entity *iam.AccessPolicy,
			_ *validationframework.Context,
			multiErr *errortypes.MultiError,
		) error {
			if !entity.Enabled {
				return nil
			}

			policies, err := repo.ListEnabledAccessPolicies(
				ctx,
				iamports.IAMPolicyLookupRequest{
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
