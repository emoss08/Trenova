package stringutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUpperFirst(t *testing.T) {
	assert.Equal(t, "Workers", UpperFirst("workers"))
	assert.Equal(t, "Éclair", UpperFirst("éclair"))
	assert.Empty(t, UpperFirst(""))
}

func TestWholeWordsWithin(t *testing.T) {
	assert.Equal(t, "short", WholeWordsWithin("short", 10))
	assert.Equal(t, "Waiting on one", WholeWordsWithin("Waiting on one driver's hours", 16))
	assert.Equal(t, "Late, at", WholeWordsWithin("Late, at the dock", 9))
}
