package tableconfigurationresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequiredPatchValue(t *testing.T) {
	t.Parallel()

	value := "TRC-100"
	got, err := requiredPatchValue("code", "Code", &value)
	require.NoError(t, err)
	assert.Equal(t, value, got)

	got, err = requiredPatchValue[string]("code", "Code", nil)
	resolvertest.RequireRequiredPatchError(t, err, "code")
	assert.Empty(t, got)
}
