package reconcile

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/emoss08/trenova/operator/internal/instanceapi"
)

func TestAReportThatIsMadeNotReadyLeavesTheCallersReportAlone(t *testing.T) {
	t.Parallel()
	l := &Loop{applied: 4}
	given := instanceapi.StatusReport{Ready: true, Components: []instanceapi.ComponentStatus{}}

	out := l.notReady(&given, "storage is down")

	require.False(t, out.Ready)
	require.NotNil(t, out.Message)
	require.NotNil(t, out.AppliedRevision)
	require.True(t, given.Ready)
	require.Nil(t, given.Message)
	require.Nil(t, given.AppliedRevision)
}
