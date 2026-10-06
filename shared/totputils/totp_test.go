package totputils

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const rfcSecret = "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"

func TestCodeAtMatchesRFC6238Vectors(t *testing.T) {
	t.Parallel()

	vectors := map[int64]string{
		59:          "287082",
		1111111109:  "081804",
		1111111111:  "050471",
		1234567890:  "005924",
		2000000000:  "279037",
		20000000000: "353130",
	}
	for unix, want := range vectors {
		got, err := CodeAt(rfcSecret, Step(time.Unix(unix, 0)))
		require.NoError(t, err)
		assert.Equal(t, want, got, "unix %d", unix)
	}
}

func TestVerifyAcceptsTheCurrentAndAdjacentSteps(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_000_000, 0)
	for _, offset := range []int64{-1, 0, 1} {
		code, err := CodeAt(rfcSecret, Step(now)+offset)
		require.NoError(t, err)

		step, err := Verify(VerifyRequest{Secret: rfcSecret, Code: code, At: now, Skew: DefaultSkew})
		require.NoError(t, err)
		assert.Equal(t, Step(now)+offset, step)
	}

	stale, err := CodeAt(rfcSecret, Step(now)-3)
	require.NoError(t, err)
	_, err = Verify(VerifyRequest{Secret: rfcSecret, Code: stale, At: now, Skew: DefaultSkew})
	require.ErrorIs(t, err, ErrInvalidCode)
}

func TestVerifyRefusesAReplayedStep(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_000_000, 0)
	code, err := CodeAt(rfcSecret, Step(now))
	require.NoError(t, err)

	_, err = Verify(VerifyRequest{
		Secret: rfcSecret,
		Code:   code,
		At:     now,
		Skew:   DefaultSkew,
		After:  Step(now),
	})
	require.ErrorIs(t, err, ErrInvalidCode)
}

func TestVerifyRejectsMalformedInput(t *testing.T) {
	t.Parallel()

	now := time.Now()
	for _, code := range []string{"", "12345", "1234567", "12a456"} {
		_, err := Verify(VerifyRequest{Secret: rfcSecret, Code: code, At: now})
		require.ErrorIs(t, err, ErrInvalidCode, code)
	}

	_, err := Verify(VerifyRequest{Secret: "!!!", Code: "123456", At: now})
	require.ErrorIs(t, err, ErrInvalidSecret)
}

func TestVerifyIgnoresSpacingInTheCode(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_000_000, 0)
	code, err := CodeAt(rfcSecret, Step(now))
	require.NoError(t, err)

	_, err = Verify(VerifyRequest{
		Secret: rfcSecret,
		Code:   code[:3] + " " + code[3:],
		At:     now,
	})
	require.NoError(t, err)
}

func TestGenerateSecretRoundTrips(t *testing.T) {
	t.Parallel()

	secret, err := GenerateSecret()
	require.NoError(t, err)
	assert.Len(t, secret, 32)

	code, err := CodeAt(secret, Step(time.Now()))
	require.NoError(t, err)
	assert.Len(t, code, Digits)
}

func TestURIDescribesTheAuthenticator(t *testing.T) {
	t.Parallel()

	uri := URI(URIParams{Issuer: "Trenova", Account: "ada@example.com", Secret: rfcSecret})
	assert.True(t, strings.HasPrefix(uri, "otpauth://totp/Trenova:ada@example.com?"))
	assert.Contains(t, uri, "secret="+rfcSecret)
	assert.Contains(t, uri, "issuer=Trenova")
	assert.Contains(t, uri, "digits=6")
	assert.Contains(t, uri, "period=30")
}

func TestGenerateRecoveryCodesAreDistinctAndNormalized(t *testing.T) {
	t.Parallel()

	codes, err := GenerateRecoveryCodes(RecoveryCodeCount)
	require.NoError(t, err)
	require.Len(t, codes, RecoveryCodeCount)

	seen := make(map[string]struct{}, len(codes))
	for _, code := range codes {
		assert.Len(t, code, 11)
		assert.Equal(t, "-", code[5:6])
		seen[code] = struct{}{}
		assert.Equal(
			t,
			strings.ReplaceAll(code, "-", ""),
			NormalizeRecoveryCode(" "+strings.ToUpper(code)+" "),
		)
	}
	assert.Len(t, seen, RecoveryCodeCount)

	_, err = GenerateRecoveryCodes(0)
	require.Error(t, err)
}
