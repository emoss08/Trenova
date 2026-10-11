package briefingjobs

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// The scheduler validates every schedule when the worker starts and refuses
// to start on one whose arguments do not match its workflow, so a schedule
// that drifts from its workflow's signature stops every worker.
func TestSchedules_MatchTheirWorkflowSignatures(t *testing.T) {
	t.Parallel()

	for _, s := range NewScheduleProvider().GetSchedules() {
		require.NoError(t, s.Validate(), s.ID)
	}
}
