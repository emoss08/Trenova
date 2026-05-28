package iamservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/iam"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type accessPolicyValidatorRepo struct {
	repositories.IAMRepository
	policies []*iam.AccessPolicy
}

func (r *accessPolicyValidatorRepo) ListEnabledAccessPolicies(
	context.Context,
	repositories.IAMPolicyLookupRequest,
) ([]*iam.AccessPolicy, error) {
	return r.policies, nil
}

type accessPolicyUniquenessChecker struct {
	exists bool
	req    *validationframework.UniquenessRequest
}

func (c *accessPolicyUniquenessChecker) CheckUniqueness(
	_ context.Context,
	req *validationframework.UniquenessRequest,
) (bool, error) {
	c.req = req
	return c.exists, nil
}

func TestValidatorValidateCreateRejectsOppositeEffectSamePolicyScope(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	validator := newValidator(
		&accessPolicyValidatorRepo{
			policies: []*iam.AccessPolicy{
				{
					ID:             pulid.MustNew("pol_"),
					OrganizationID: orgID,
					BusinessUnitID: buID,
					Name:           "Deny shipment read",
					Resource:       "shipment",
					Operation:      "read",
					Effect:         iam.PolicyEffectDeny,
					Enabled:        true,
				},
			},
		},
		&accessPolicyUniquenessChecker{},
	)

	multiErr := validator.ValidateCreate(t.Context(), &iam.AccessPolicy{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Name:           "Allow shipment read",
		Resource:       "shipment",
		Operation:      "read",
		Effect:         iam.PolicyEffectAllow,
		Enabled:        true,
	})

	require.NotNil(t, multiErr)
	require.Len(t, multiErr.Errors, 1)
	assert.Equal(t, "effect", multiErr.Errors[0].Field)
	assert.Equal(t, errortypes.ErrDuplicate, multiErr.Errors[0].Code)
}

func TestValidatorValidateCreateRejectsDuplicatePolicyScope(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	validator := newValidator(
		&accessPolicyValidatorRepo{
			policies: []*iam.AccessPolicy{
				{
					ID:             pulid.MustNew("pol_"),
					OrganizationID: orgID,
					BusinessUnitID: buID,
					Name:           "Allow shipment read",
					Resource:       "shipment",
					Operation:      "read",
					Effect:         iam.PolicyEffectAllow,
					Conditions:     map[string]string{"principalType": "user"},
					Enabled:        true,
				},
			},
		},
		&accessPolicyUniquenessChecker{},
	)

	multiErr := validator.ValidateCreate(t.Context(), &iam.AccessPolicy{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Name:           "Allow shipment read duplicate",
		Resource:       "shipment",
		Operation:      "read",
		Effect:         iam.PolicyEffectAllow,
		Conditions:     map[string]string{"principalType": "user"},
		Enabled:        true,
	})

	require.NotNil(t, multiErr)
	require.Len(t, multiErr.Errors, 1)
	assert.Equal(t, "resource", multiErr.Errors[0].Field)
	assert.Equal(t, errortypes.ErrDuplicate, multiErr.Errors[0].Code)
}

func TestValidatorValidateCreateRejectsDuplicateName(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	uniquenessChecker := &accessPolicyUniquenessChecker{exists: true}
	validator := newValidator(&accessPolicyValidatorRepo{}, uniquenessChecker)

	multiErr := validator.ValidateCreate(t.Context(), &iam.AccessPolicy{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Name:           "Allow shipment read",
		Resource:       "shipment",
		Operation:      "read",
		Effect:         iam.PolicyEffectAllow,
		Enabled:        true,
	})

	require.NotNil(t, multiErr)
	require.Len(t, multiErr.Errors, 1)
	assert.Equal(t, "name", multiErr.Errors[0].Field)
	assert.Equal(t, errortypes.ErrDuplicate, multiErr.Errors[0].Code)
	require.NotNil(t, uniquenessChecker.req)
	assert.Equal(t, "access_policies", uniquenessChecker.req.TableName)
	assert.Equal(t, orgID, uniquenessChecker.req.OrganizationID)
	assert.Equal(t, buID, uniquenessChecker.req.BusinessUnitID)
	require.Len(t, uniquenessChecker.req.Fields, 1)
	assert.Equal(t, "name", uniquenessChecker.req.Fields[0].Column)
	assert.False(t, uniquenessChecker.req.Fields[0].CaseSensitive)
}

func TestValidatorValidateCreateAllowsConditionSpecificOppositeEffect(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	validator := newValidator(
		&accessPolicyValidatorRepo{
			policies: []*iam.AccessPolicy{
				{
					ID:             pulid.MustNew("pol_"),
					OrganizationID: orgID,
					BusinessUnitID: buID,
					Name:           "Deny user shipment read",
					Resource:       "shipment",
					Operation:      "read",
					Effect:         iam.PolicyEffectDeny,
					Conditions:     map[string]string{"principalType": "user"},
					Enabled:        true,
				},
			},
		},
		&accessPolicyUniquenessChecker{},
	)

	multiErr := validator.ValidateCreate(t.Context(), &iam.AccessPolicy{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Name:           "Allow API shipment read",
		Resource:       "shipment",
		Operation:      "read",
		Effect:         iam.PolicyEffectAllow,
		Conditions:     map[string]string{"principalType": "apiKey"},
		Enabled:        true,
	})

	require.Nil(t, multiErr)
}

func TestAccessPoliciesConflict(t *testing.T) {
	t.Parallel()

	policyID := pulid.MustNew("pol_")
	basePolicy := &iam.AccessPolicy{
		ID:        pulid.MustNew("pol_"),
		Resource:  "shipment",
		Operation: "read",
		Effect:    iam.PolicyEffectAllow,
		Enabled:   true,
	}

	tests := []struct {
		name     string
		left     *iam.AccessPolicy
		right    *iam.AccessPolicy
		expected bool
	}{
		{
			name: "opposite effect same unconditional scope conflicts",
			left: basePolicy,
			right: &iam.AccessPolicy{
				ID:        pulid.MustNew("pol_"),
				Resource:  "shipment",
				Operation: "read",
				Effect:    iam.PolicyEffectDeny,
				Enabled:   true,
			},
			expected: true,
		},
		{
			name: "same effect duplicate does not conflict",
			left: basePolicy,
			right: &iam.AccessPolicy{
				ID:        pulid.MustNew("pol_"),
				Resource:  "shipment",
				Operation: "read",
				Effect:    iam.PolicyEffectAllow,
				Enabled:   true,
			},
		},
		{
			name: "different operation does not conflict",
			left: basePolicy,
			right: &iam.AccessPolicy{
				ID:        pulid.MustNew("pol_"),
				Resource:  "shipment",
				Operation: "update",
				Effect:    iam.PolicyEffectDeny,
				Enabled:   true,
			},
		},
		{
			name: "disabled policy does not conflict",
			left: basePolicy,
			right: &iam.AccessPolicy{
				ID:        pulid.MustNew("pol_"),
				Resource:  "shipment",
				Operation: "read",
				Effect:    iam.PolicyEffectDeny,
				Enabled:   false,
			},
		},
		{
			name: "same policy id does not conflict",
			left: &iam.AccessPolicy{
				ID:        policyID,
				Resource:  "shipment",
				Operation: "read",
				Effect:    iam.PolicyEffectAllow,
				Enabled:   true,
			},
			right: &iam.AccessPolicy{
				ID:        policyID,
				Resource:  "shipment",
				Operation: "read",
				Effect:    iam.PolicyEffectDeny,
				Enabled:   true,
			},
		},
		{
			name: "equivalent trimmed conditions conflict",
			left: &iam.AccessPolicy{
				ID:         pulid.MustNew("pol_"),
				Resource:   "shipment",
				Operation:  "read",
				Effect:     iam.PolicyEffectAllow,
				Conditions: map[string]string{"principalType": " user "},
				Enabled:    true,
			},
			right: &iam.AccessPolicy{
				ID:         pulid.MustNew("pol_"),
				Resource:   "shipment",
				Operation:  "read",
				Effect:     iam.PolicyEffectDeny,
				Conditions: map[string]string{" principalType ": "user"},
				Enabled:    true,
			},
			expected: true,
		},
		{
			name: "different conditions do not conflict",
			left: &iam.AccessPolicy{
				ID:         pulid.MustNew("pol_"),
				Resource:   "shipment",
				Operation:  "read",
				Effect:     iam.PolicyEffectAllow,
				Conditions: map[string]string{"principalType": "user"},
				Enabled:    true,
			},
			right: &iam.AccessPolicy{
				ID:         pulid.MustNew("pol_"),
				Resource:   "shipment",
				Operation:  "read",
				Effect:     iam.PolicyEffectDeny,
				Conditions: map[string]string{"principalType": "apiKey"},
				Enabled:    true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, accessPoliciesConflict(tt.left, tt.right))
		})
	}
}

func TestAccessPoliciesDuplicate(t *testing.T) {
	t.Parallel()

	basePolicy := &iam.AccessPolicy{
		ID:        pulid.MustNew("pol_"),
		Resource:  "shipment",
		Operation: "read",
		Effect:    iam.PolicyEffectAllow,
		Enabled:   true,
	}

	tests := []struct {
		name     string
		left     *iam.AccessPolicy
		right    *iam.AccessPolicy
		expected bool
	}{
		{
			name: "same effect same unconditional scope duplicates",
			left: basePolicy,
			right: &iam.AccessPolicy{
				ID:        pulid.MustNew("pol_"),
				Resource:  "shipment",
				Operation: "read",
				Effect:    iam.PolicyEffectAllow,
				Enabled:   true,
			},
			expected: true,
		},
		{
			name: "opposite effect conflicts but does not duplicate",
			left: basePolicy,
			right: &iam.AccessPolicy{
				ID:        pulid.MustNew("pol_"),
				Resource:  "shipment",
				Operation: "read",
				Effect:    iam.PolicyEffectDeny,
				Enabled:   true,
			},
		},
		{
			name: "different conditions do not duplicate",
			left: &iam.AccessPolicy{
				ID:         pulid.MustNew("pol_"),
				Resource:   "shipment",
				Operation:  "read",
				Effect:     iam.PolicyEffectAllow,
				Conditions: map[string]string{"principalType": "user"},
				Enabled:    true,
			},
			right: &iam.AccessPolicy{
				ID:         pulid.MustNew("pol_"),
				Resource:   "shipment",
				Operation:  "read",
				Effect:     iam.PolicyEffectAllow,
				Conditions: map[string]string{"principalType": "apiKey"},
				Enabled:    true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, accessPoliciesDuplicate(tt.left, tt.right))
		})
	}
}
