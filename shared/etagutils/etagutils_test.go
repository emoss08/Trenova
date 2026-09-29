package etagutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStrongQuotesTheValue(t *testing.T) {
	t.Parallel()

	assert.Equal(t, `"abc"`, Strong("abc"))
}

func TestMatches(t *testing.T) {
	t.Parallel()

	etag := Strong("abc")
	for name, tc := range map[string]struct {
		header string
		want   bool
	}{
		"empty":        {"", false},
		"same":         {`"abc"`, true},
		"weak":         {`W/"abc"`, true},
		"in a list":    {`"x", "abc"`, true},
		"any":          {"*", true},
		"another":      {`"abd"`, false},
		"unquoted tag": {"abc", false},
	} {
		assert.Equal(t, tc.want, Matches(tc.header, etag), name)
	}
	assert.False(t, Matches("*", ""), "no tag matches nothing")
}
