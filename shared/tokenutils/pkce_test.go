package tokenutils

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPKCEChallengeMatchesRFC7636(t *testing.T) {
	require.Equal(
		t,
		"E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
		PKCEChallenge("dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"),
	)
}

func TestPKCEVerifierIsAnUnreservedStringOfFortyThreeCharacters(t *testing.T) {
	first, err := PKCEVerifier()
	require.NoError(t, err)
	second, err := PKCEVerifier()
	require.NoError(t, err)

	require.Regexp(t, regexp.MustCompile(`^[A-Za-z0-9_-]{43}$`), first)
	require.NotEqual(t, first, second)
}
