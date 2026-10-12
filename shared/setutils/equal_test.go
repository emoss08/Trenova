package setutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSameMembersIgnoresOrderAndRepeats(t *testing.T) {
	t.Parallel()

	assert.True(t, SameMembers([]string{"a", "b"}, []string{"b", "a", "a"}))
	assert.True(t, SameMembers[string](nil, []string{}))
	assert.False(t, SameMembers([]string{"a"}, []string{"a", "b"}))
	assert.False(t, SameMembers([]string{"a", "b"}, []string{"a"}))
	assert.False(t, SameMembers([]string{"a"}, nil))
}
