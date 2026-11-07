package tenant

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUsernameBase(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "dana.whitfield", UsernameBase("Dana.Whitfield@acme.example"))
	assert.Equal(t, "ops", UsernameBase("+ops+@acme.example"))
	assert.Equal(t, "abcdefghijklmnopqrst", UsernameBase("abcdefghijklmnopqrstuvwxyz@acme.example"))
	assert.Empty(t, UsernameBase("+++@acme.example"))
}

func TestUsernameCandidatesStayWithinTheUsernameLength(t *testing.T) {
	t.Parallel()

	candidates := UsernameCandidates("abcdefghijklmnopqrst", "a1b2c3")

	assert.Equal(t, "abcdefghijklmnopqrst", candidates[0])
	assert.Equal(t, "abcdefghijklmnopqr-2", candidates[1])
	assert.Equal(t, "abcdefghijklm-a1b2c3", candidates[len(candidates)-1])
	for _, candidate := range candidates {
		assert.LessOrEqual(t, len(candidate), MaxUsernameLength)
	}
}

func TestUsernameCandidatesFallBackWhenTheBaseIsEmpty(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "user", UsernameCandidates("", "")[0])
}
