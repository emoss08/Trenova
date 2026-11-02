package sessiontoken_test

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/pkg/sessiontoken"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newIssuer() *sessiontoken.Issuer {
	return sessiontoken.NewIssuer("unit-test-session-secret-with-32-bytes")
}

func TestIssueAndParseRoundTrip(t *testing.T) {
	t.Parallel()

	issuer := newIssuer()
	sessionID := pulid.MustNew("ses_")
	userID := pulid.MustNew("usr_")
	issued, err := issuer.Issue(sessionID, userID)
	require.NoError(t, err)

	parsedID, secret, err := sessiontoken.Parse(issued.Token)
	require.NoError(t, err)
	assert.Equal(t, sessionID, parsedID)
	assert.True(t, issuer.Matches(sessionID, userID, secret, issued.SecretHash))
	assert.NotContains(t, issued.SecretHash, secret)
}

func TestIssue_SecretCarriesAtLeast256Bits(t *testing.T) {
	t.Parallel()

	issued, err := newIssuer().Issue(pulid.MustNew("ses_"), pulid.MustNew("usr_"))
	require.NoError(t, err)

	_, secret, err := sessiontoken.Parse(issued.Token)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(secret), 43)
}

func TestIssue_EveryTokenIsDistinct(t *testing.T) {
	t.Parallel()

	issuer := newIssuer()
	sessionID := pulid.MustNew("ses_")
	userID := pulid.MustNew("usr_")
	first, err := issuer.Issue(sessionID, userID)
	require.NoError(t, err)
	second, err := issuer.Issue(sessionID, userID)
	require.NoError(t, err)

	assert.NotEqual(t, first.Token, second.Token)
	assert.NotEqual(t, first.SecretHash, second.SecretHash)
}

func TestMatches_BindsTheHashToItsSessionAndUser(t *testing.T) {
	t.Parallel()

	issuer := newIssuer()
	sessionID := pulid.MustNew("ses_")
	userID := pulid.MustNew("usr_")
	issued, err := issuer.Issue(sessionID, userID)
	require.NoError(t, err)
	_, secret, err := sessiontoken.Parse(issued.Token)
	require.NoError(t, err)

	assert.False(t, issuer.Matches(pulid.MustNew("ses_"), userID, secret, issued.SecretHash))
	assert.False(t, issuer.Matches(sessionID, pulid.MustNew("usr_"), secret, issued.SecretHash))
}

func TestMatches_NeedsTheServerKey(t *testing.T) {
	t.Parallel()

	sessionID := pulid.MustNew("ses_")
	userID := pulid.MustNew("usr_")
	issued, err := newIssuer().Issue(sessionID, userID)
	require.NoError(t, err)
	_, secret, err := sessiontoken.Parse(issued.Token)
	require.NoError(t, err)

	other := sessiontoken.NewIssuer("another-session-secret-of-32-bytes!!")
	assert.False(t, other.Matches(sessionID, userID, secret, issued.SecretHash))
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

	issuer := newIssuer()
	sessionID := pulid.MustNew("ses_")
	userID := pulid.MustNew("usr_")
	issued, err := issuer.Issue(sessionID, userID)
	require.NoError(t, err)

	assert.False(t, issuer.Matches(sessionID, userID, "wrong", issued.SecretHash))
	assert.False(t, issuer.Matches(sessionID, userID, "", issued.SecretHash))
	assert.False(t, issuer.Matches(sessionID, userID, "anything", ""))
}
