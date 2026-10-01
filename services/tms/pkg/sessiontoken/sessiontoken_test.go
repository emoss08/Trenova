package sessiontoken_test

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/pkg/sessiontoken"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIssueAndParseRoundTrip(t *testing.T) {
	t.Parallel()

	sessionID := pulid.MustNew("ses_")
	issued, err := sessiontoken.Issue(sessionID)
	require.NoError(t, err)

	parsedID, secret, err := sessiontoken.Parse(issued.Token)
	require.NoError(t, err)
	assert.Equal(t, sessionID, parsedID)
	assert.True(t, sessiontoken.Matches(secret, issued.SecretHash))
	assert.NotContains(t, issued.SecretHash, secret)
}

func TestIssue_SecretCarriesAtLeast256Bits(t *testing.T) {
	t.Parallel()

	issued, err := sessiontoken.Issue(pulid.MustNew("ses_"))
	require.NoError(t, err)

	_, secret, err := sessiontoken.Parse(issued.Token)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(secret), 43)
}

func TestIssue_EveryTokenIsDistinct(t *testing.T) {
	t.Parallel()

	sessionID := pulid.MustNew("ses_")
	first, err := sessiontoken.Issue(sessionID)
	require.NoError(t, err)
	second, err := sessiontoken.Issue(sessionID)
	require.NoError(t, err)

	assert.NotEqual(t, first.Token, second.Token)
	assert.NotEqual(t, first.SecretHash, second.SecretHash)
}

func TestParse_RejectsBareSessionIDsAndGarbage(t *testing.T) {
	t.Parallel()

	sessionID := pulid.MustNew("ses_").String()
	for _, token := range []string{
		"",
		sessionID,
		sessionID + ".",
		"." + strings.Repeat("a", 43),
		"not-an-id." + strings.Repeat("a", 43),
		sessionID + ".a.b",
	} {
		_, _, err := sessiontoken.Parse(token)
		assert.ErrorIs(t, err, sessiontoken.ErrMalformed, token)
	}
}

func TestMatches_RejectsWrongOrMissingSecrets(t *testing.T) {
	t.Parallel()

	issued, err := sessiontoken.Issue(pulid.MustNew("ses_"))
	require.NoError(t, err)

	assert.False(t, sessiontoken.Matches("wrong", issued.SecretHash))
	assert.False(t, sessiontoken.Matches("", issued.SecretHash))
	assert.False(t, sessiontoken.Matches("anything", ""))
}
