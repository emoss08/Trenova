package toolaccess

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSearchesLeft_SpeaksUpOnlyNearTheCap(t *testing.T) {
	t.Parallel()

	assert.Empty(t, SearchesLeft(1))
	assert.Empty(t, SearchesLeft(2))
	assert.Contains(t, SearchesLeft(3), "search 3 of 4")
	assert.Contains(t, SearchesLeft(4), "last tool search")
}
