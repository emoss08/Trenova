package authservice

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIdentityProviderResourceIDs_MatchEveryBindingTheReaderAccepts(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []string{"AzureAD", "azuread"}, identityProviderResourceIDs("azuread"))
	assert.Equal(t, []string{"Okta", "okta"}, identityProviderResourceIDs("okta"))
	assert.Equal(t, []string{"corporate-idp"}, identityProviderResourceIDs("corporate-idp"))
}
