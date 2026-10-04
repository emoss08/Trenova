package agentjobs

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/platformplan"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStartScheduledRunSkipsOrganizationsWithoutAutomation(t *testing.T) {
	t.Parallel()

	tenant := plantest.Tenant()
	a := &Activities{platformPlans: plantest.Restricting(t, platformplan.CapabilityAgentAutomation)}

	result, err := a.StartScheduledRunActivity(t.Context(), &ScheduledRunPayload{
		DefinitionID:   pulid.MustNew("agd_"),
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
	})

	require.NoError(t, err)
	assert.False(t, result.Started)
	assert.Equal(t, "plan_restricted", result.Skipped)
}
