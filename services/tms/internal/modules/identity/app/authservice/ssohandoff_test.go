package authservice

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
)

const (
	firstAddress  = "https://abc.trenova.cloud"
	customAddress = "https://tms.acme.com"
	startPath     = "/api/v1/auth/sso/start/idp_01J9Z3/acme"
)

var servedOrigins = []string{firstAddress, customAddress}

func startOn(returnTo string) *tenantports.StartSSOLoginRequest {
	return &tenantports.StartSSOLoginRequest{
		OrganizationSlug: "acme",
		ReturnTo:         returnTo,
		StartPath:        startPath,
	}
}

func TestASignInStartedOnTheRedirectAddressIsNotHandedOff(t *testing.T) {
	t.Parallel()

	_, ok, err := ssoStartHandoff(
		firstAddress+"/api/v1/auth/sso/callback/oidc",
		startOn(firstAddress+"/shipments"),
		servedOrigins,
	)

	require.NoError(t, err)
	assert.False(t, ok)
}

func TestASignInStartedOnTheOtherAddressIsHandedOffOnceWithItsReturnURLMoved(t *testing.T) {
	t.Parallel()
	redirect := firstAddress + "/api/v1/auth/sso/callback/oidc"

	handoff, ok, err := ssoStartHandoff(
		redirect,
		startOn(customAddress+"/shipments/open?view=board"),
		servedOrigins,
	)

	require.NoError(t, err)
	require.True(t, ok)
	parsed, err := url.Parse(handoff)
	require.NoError(t, err)
	assert.Equal(t, firstAddress, parsed.Scheme+"://"+parsed.Host)
	assert.Equal(t, startPath, parsed.Path, "the same start endpoint, the organization and provider kept")
	assert.Equal(t, url.Values{"returnTo": {firstAddress + "/shipments/open?view=board"}}, parsed.Query(),
		"only the return URL is carried, moved to the address the person signs in on")

	_, again, err := ssoStartHandoff(redirect, startOn(parsed.Query().Get("returnTo")), servedOrigins)
	require.NoError(t, err)
	assert.False(t, again, "the handed-off request starts on the redirect address and goes on to the provider")
}

func TestARedirectURLOutsideTheInstancesAddressesIsLeftAlone(t *testing.T) {
	t.Parallel()

	for _, redirect := range []string{
		"https://elsewhere.example.com/api/v1/auth/sso/callback/oidc",
		"not a url",
		"",
	} {
		_, ok, err := ssoStartHandoff(redirect, startOn(customAddress+"/"), servedOrigins)
		require.NoError(t, err)
		assert.False(t, ok, redirect)
	}
}

func TestAHostileReturnURLIsRefusedAsBefore(t *testing.T) {
	t.Parallel()
	redirect := firstAddress + "/api/v1/auth/sso/callback/oidc"

	for _, returnTo := range []string{
		"https://evil.example.com/",
		"//evil.example.com/",
		"/shipments",
		"javascript:alert(1)",
		"",
	} {
		handoff, ok, err := ssoStartHandoff(redirect, startOn(returnTo), servedOrigins)
		require.Error(t, err, returnTo)
		assert.False(t, ok)
		assert.Empty(t, handoff)
	}
}

func TestAReturnURLThatIsNotAPlainPathIsDroppedToTheHome(t *testing.T) {
	t.Parallel()

	assert.Equal(t, firstAddress+"/", returnToOn(firstAddress, customAddress))
	assert.Equal(t, firstAddress+"/", returnToOn(firstAddress, customAddress+"//evil.example.com/x"))
	assert.Equal(t, firstAddress+"/a", returnToOn(firstAddress, customAddress+"/a"))
	_, ok := ssoHandoffURL(firstAddress+"/cb", customAddress+"/", "//evil.example.com/start", servedOrigins)
	assert.False(t, ok, "the start path is the router's, never anything that leaves the address")
}
