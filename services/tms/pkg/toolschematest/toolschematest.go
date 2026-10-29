package toolschematest

import (
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/require"
)

func FieldErrors(t *testing.T, err error) map[string]string {
	t.Helper()

	require.Error(t, err)
	multiErr, ok := err.(*errortypes.MultiError)
	require.True(t, ok, "expected field-level errors, got %T: %v", err, err)

	out := make(map[string]string, len(multiErr.Errors))
	for _, entry := range multiErr.Errors {
		out[entry.Field] = entry.Message
	}

	return out
}
