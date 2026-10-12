package agenttoolstest

import (
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/require"
)

func FieldErrors(t *testing.T, err error) map[string]string {
	t.Helper()

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	out := make(map[string]string, len(multiErr.Errors))
	for _, entry := range multiErr.Errors {
		out[entry.Field] = entry.Message
	}

	return out
}
