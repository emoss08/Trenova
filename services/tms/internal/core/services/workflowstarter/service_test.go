package workflowstarter

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
)

// With the client connecting lazily, an outage surfaces on the call that hits
// it. A person has to get a message they can act on, not a gRPC dial error.
func TestStartWorkflow_SaysSoPlainlyWhenTemporalIsUnreachable(t *testing.T) {
	t.Parallel()

	c, err := client.NewLazyClient(client.Options{HostPort: "127.0.0.1:1"})
	require.NoError(t, err)
	t.Cleanup(c.Close)

	starter := &Service{client: c}

	_, err = starter.StartWorkflow(t.Context(), client.StartWorkflowOptions{
		ID:        "probe",
		TaskQueue: "probe-queue",
	}, "ProbeWorkflow")
	require.Error(t, err)

	var business *errortypes.BusinessError
	require.True(t, errors.As(err, &business), "got %T: %v", err, err)
	assert.Contains(t, business.Error(), "temporarily unavailable")
}

func TestUnreachable_LeavesEveryOtherErrorAlone(t *testing.T) {
	t.Parallel()

	other := errors.New("workflow execution already started")
	assert.Same(t, other, unreachable(other))
	assert.NoError(t, unreachable(nil))
}
