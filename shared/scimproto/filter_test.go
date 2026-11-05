package scimproto

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var userFilterAttributes = []string{"id", "username", "externalid", "emails.value", "active"}

func TestParseFilter_ReadsTheEqualityFiltersIdentityProvidersSend(t *testing.T) {
	t.Parallel()

	cases := map[string]Condition{
		`userName eq "ada@example.com"`: {
			"username",
			"ada@example.com",
		},
		`externalId eq "00u1abc"`: {"externalid", "00u1abc"},
		`emails[type eq "work"].value eq "Ada@Example.com"`: {
			"emails.value",
			"Ada@Example.com",
		},
		`urn:ietf:params:scim:schemas:core:2.0:User:userName eq "x"`: {"username", "x"},
		`active eq true`: {"active", "true"},
		`emails eq "ada@example.com"`: {
			"emails.value",
			"ada@example.com",
		},
	}

	for raw, want := range cases {
		filter, err := ParseFilter(raw, userFilterAttributes...)
		require.NoError(t, err, raw)
		require.Len(t, filter.Conditions, 1, raw)
		assert.Equal(t, want, filter.Conditions[0], raw)
	}
}

func TestParseFilter_JoinsConditionsWithAnd(t *testing.T) {
	t.Parallel()

	filter, err := ParseFilter(
		`id eq "scg_1" and members[value eq "usr_ABC"]`,
		"id", "members.value",
	)
	require.NoError(t, err)
	assert.Equal(t, []Condition{
		{Attribute: "id", Value: "scg_1"},
		{Attribute: "members.value", Value: "usr_ABC"},
	}, filter.Conditions)

	value, ok := filter.Value("members.value")
	assert.True(t, ok)
	assert.Equal(t, "usr_ABC", value, "a member id keeps its case")
}

func TestParseFilter_RefusesWhatItCannotAnswer(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{
		`userName co "ada"`,
		`userName eq "a" or userName eq "b"`,
		`not (userName eq "a")`,
		`title eq "Dispatcher"`,
		`userName eq "unterminated`,
		`userName eq`,
		`(userName eq "a"`,
	} {
		_, err := ParseFilter(raw, userFilterAttributes...)
		require.Error(t, err, raw)
		scimErr, ok := AsError(err)
		require.True(t, ok, raw)
		assert.Equal(t, http.StatusBadRequest, scimErr.Status, raw)
		assert.Equal(t, ScimTypeInvalidFilter, scimErr.ScimType, raw)
	}
}

func TestParseFilter_EmptyFilterMatchesEverything(t *testing.T) {
	t.Parallel()

	filter, err := ParseFilter("  ", userFilterAttributes...)
	require.NoError(t, err)
	assert.True(t, filter.Empty())
}
