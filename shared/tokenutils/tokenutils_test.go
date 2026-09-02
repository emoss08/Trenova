package tokenutils

import (
	"encoding/base64"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

var hexDigest = regexp.MustCompile(`^[0-9a-f]{64}$`)

func TestNewMintsURLSafeTokenWithMatchingHash(t *testing.T) {
	t.Parallel()

	token, tokenHash, err := New()
	require.NoError(t, err)

	raw, err := base64.RawURLEncoding.DecodeString(token)
	require.NoError(t, err, "token must be unpadded base64url")
	require.Len(t, raw, 32)

	require.Equal(t, Hash(token), tokenHash)
	require.Regexp(t, hexDigest, tokenHash)
}

func TestNewMintsDistinctTokens(t *testing.T) {
	t.Parallel()

	first, firstHash, err := New()
	require.NoError(t, err)
	second, secondHash, err := New()
	require.NoError(t, err)

	require.NotEqual(t, first, second)
	require.NotEqual(t, firstHash, secondHash)
}

func TestHashMatchesSHA256HexVector(t *testing.T) {
	t.Parallel()

	require.Equal(
		t,
		"9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08",
		Hash("test"),
	)
}

func TestHashIsDeterministic(t *testing.T) {
	t.Parallel()

	require.Equal(t, Hash("token-value"), Hash("token-value"))
	require.NotEqual(t, Hash("token-value"), Hash("token-value2"))
}
