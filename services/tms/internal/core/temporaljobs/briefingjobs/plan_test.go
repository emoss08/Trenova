package briefingjobs

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/platformplan"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsDueSkipsOrganizationsWithoutAutomation(t *testing.T) {
	t.Parallel()

	a := &Activities{plans: plantest.Restricting(t, platformplan.CapabilityAgentAutomation)}

	due, err := a.isDue(t.Context(), plantest.Tenant(), time.Now().Unix())

	require.NoError(t, err)
	assert.False(t, due)
}
