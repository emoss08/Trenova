package temporaltype

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestHeartbeatEveryOutsideActivityIsNoop(t *testing.T) {
	t.Parallel()

	stop := HeartbeatEvery(t.Context(), time.Millisecond)
	require.NotNil(t, stop)
	stop()
	stop()
}
