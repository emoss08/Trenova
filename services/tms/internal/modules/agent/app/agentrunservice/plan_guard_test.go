package agentrunservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/testutil/mocks/kernelportsmock"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/platformplan"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/require"
)

func TestStartForDefinitionRefusesBackgroundRunsThePlanRestricts(t *testing.T) {
	t.Parallel()

	workflows := kernelportsmock.NewMockWorkflowStarter(t)
	workflows.EXPECT().Enabled().Return(true)
	svc := &Service{
		workflows: workflows,
		plans:     plantest.Restricting(t, platformplan.CapabilityAgentAutomation),
	}

	run, err := svc.StartForDefinition(t.Context(), &agentports.StartAgentRunForDefinitionRequest{
		TenantInfo: plantest.Tenant(),
		SystemKey:  "billing_exception",
	}, nil)

	require.Nil(t, run)
	require.True(t, errortypes.IsPlanRestrictionError(err))
}
