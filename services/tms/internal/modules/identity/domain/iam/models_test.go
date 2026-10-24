package iam

import (
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIdentityProviderValidate(t *testing.T) {
	t.Parallel()

	provider := &IdentityProvider{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Name:           "Corporate IdP",
		Slug:           "corporate",
		Protocol:       IdentityProviderProtocolOIDC,
	}
	multiErr := errortypes.NewMultiError()
	provider.Validate(multiErr)
	assert.False(t, multiErr.HasErrors())

	provider.Protocol = IdentityProviderProtocol("LDAP")
	multiErr = errortypes.NewMultiError()
	provider.Validate(multiErr)
	assert.True(t, multiErr.HasErrors())
}

func TestIdentityProviderBeforeCreate(t *testing.T) {
	t.Parallel()

	provider := &IdentityProvider{}
	err := provider.BeforeCreate(t.Context())
	require.NoError(t, err)
	assert.True(t, provider.ID.IsNotNil())
	assert.NotZero(t, provider.CreatedAt)
	assert.NotNil(t, provider.AllowedDomains)
	assert.NotNil(t, provider.AttributeMap)
	assert.Equal(t, []string{"openid", "email", "profile"}, provider.OIDCScopes)

	err = provider.BeforeUpdate(t.Context())
	require.NoError(t, err)
	assert.NotZero(t, provider.UpdatedAt)
}

func TestAccessPolicyValidate(t *testing.T) {
	t.Parallel()

	policy := &AccessPolicy{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Name:           "Terminal scope",
		Resource:       "shipment",
		Operation:      "read",
		Effect:         PolicyEffectDeny,
	}
	multiErr := errortypes.NewMultiError()
	policy.Validate(multiErr)
	assert.False(t, multiErr.HasErrors())

	policy.Effect = PolicyEffect("audit")
	multiErr = errortypes.NewMultiError()
	policy.Validate(multiErr)
	assert.True(t, multiErr.HasErrors())
}

func TestIAMBeforeCreateInitializesCollections(t *testing.T) {
	t.Parallel()

	externalIdentity := &ExternalIdentity{}
	err := externalIdentity.BeforeCreate(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, externalIdentity.RawClaims)

	authEvent := &AuthEvent{}
	err = authEvent.BeforeCreate(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, authEvent.RiskSignals)

	riskDecision := &RiskDecision{}
	err = riskDecision.BeforeCreate(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, riskDecision.Signals)

	accessPolicy := &AccessPolicy{}
	err = accessPolicy.BeforeCreate(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, accessPolicy.Conditions)
}
