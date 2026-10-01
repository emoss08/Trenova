package resolver

import (
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func requireRequiredPatchError(t *testing.T, err error, field string) {
	t.Helper()

	var valErr *errortypes.Error
	require.ErrorAs(t, err, &valErr)
	assert.Equal(t, field, valErr.Field)
	assert.Equal(t, errortypes.ErrRequired, valErr.Code)
	assert.Contains(t, valErr.Message, "cannot be cleared")
}

func TestRequiredPatchValue(t *testing.T) {
	t.Parallel()

	value := "TRC-100"
	got, err := requiredPatchValue("code", "Code", &value)
	require.NoError(t, err)
	assert.Equal(t, value, got)

	got, err = requiredPatchValue[string]("code", "Code", nil)
	requireRequiredPatchError(t, err, "code")
	assert.Empty(t, got)
}
