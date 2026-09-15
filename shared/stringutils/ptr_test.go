package stringutils_test

import (
	"testing"

	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type kind string

func TestNilIfEmpty(t *testing.T) {
	assert.Nil(t, stringutils.NilIfEmpty(kind("")))

	got := stringutils.NilIfEmpty(kind("LateCharge"))
	require.NotNil(t, got)
	assert.Equal(t, kind("LateCharge"), *got)
}
