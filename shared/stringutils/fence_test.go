package stringutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNeutralizeCloseTag(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "plain", NeutralizeCloseTag("plain", "</data>"))
	assert.Equal(t, "a <\\/data> b", NeutralizeCloseTag("a </data> b", "</data>"))
	assert.Equal(t, "x </data>", NeutralizeCloseTag("x </data>", ""))
	assert.Equal(t, "<\\/d><\\/d>", NeutralizeCloseTag("</d></d>", "</d>"))
}
