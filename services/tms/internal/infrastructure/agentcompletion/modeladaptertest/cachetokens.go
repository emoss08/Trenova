package modeladaptertest

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/stretchr/testify/require"
)

func CompleteWith(
	t *testing.T,
	adapter modeladapter.Adapter,
	call *modeladapter.Call,
) *modeladapter.Response {
	t.Helper()

	resp, err := adapter.Complete(t.Context(), call)
	require.NoError(t, err)

	return resp
}
