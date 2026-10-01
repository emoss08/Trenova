package resolvertest

import (
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func RequireRequiredPatchError(t *testing.T, err error, field string) {
	t.Helper()

	var valErr *errortypes.Error
	require.ErrorAs(t, err, &valErr)
	assert.Equal(t, field, valErr.Field)
	assert.Equal(t, errortypes.ErrRequired, valErr.Code)
	assert.Contains(t, valErr.Message, "cannot be cleared")
}
