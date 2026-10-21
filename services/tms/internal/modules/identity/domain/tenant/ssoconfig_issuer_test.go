package tenant

import (
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/httpsafe"
	"github.com/stretchr/testify/require"
)

func TestValidateOIDCIssuerURL(t *testing.T) {
	t.Parallel()

	accepted := []string{
		"",
		"https://login.microsoftonline.com/11111111-2222-3333-4444-555555555555/v2.0",
		"https://trenova.okta.com",
		"https://idp.example.com/realms/trenova/",
	}
	for _, issuer := range accepted {
		multiErr := errortypes.NewMultiError()
		ValidateOIDCIssuerURL(multiErr, "issuerUrl", issuer, httpsafe.Policy{})
		require.False(t, multiErr.HasErrors(), issuer)
	}

	refused := []string{
		"http://trenova.okta.com",
		"https://user:secret@trenova.okta.com",
		"https://trenova.okta.com?x=1",
		"https://trenova.okta.com#frag",
		"https://127.0.0.1/realms/trenova",
		"https://10.0.0.8/realms/trenova",
		"https://169.254.169.254/latest",
		"https://[64:ff9b::a9fe:a9fe]/latest",
		"gopher://trenova.okta.com",
	}
	for _, issuer := range refused {
		multiErr := errortypes.NewMultiError()
		ValidateOIDCIssuerURL(multiErr, "issuerUrl", issuer, httpsafe.Policy{})
		require.True(t, multiErr.HasErrors(), issuer)
	}

	private := errortypes.NewMultiError()
	ValidateOIDCIssuerURL(
		private,
		"issuerUrl",
		"https://10.0.0.8/realms/trenova",
		httpsafe.Policy{AllowPrivateNetworks: true},
	)
	require.False(t, private.HasErrors())
}
