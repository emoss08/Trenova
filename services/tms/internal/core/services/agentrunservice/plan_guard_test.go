package agentrunservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/platformplan"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/require"
)

func TestStartForDefinitionRefusesBackgroundRunsThePlanRestricts(t *testing.T) {
	t.Parallel()

	workflows := mocks.NewMockWorkflowStarter(t)
	workflows.EXPECT().Enabled().Return(true)
	svc := &Service{
		workflows: workflows,
		plans:     plantest.Restricting(t, platformplan.CapabilityAgentAutomation),
	}

	run, err := svc.StartForDefinition(t.Context(), &services.StartAgentRunForDefinitionRequest{
		TenantInfo: plantest.Tenant(),
		SystemKey:  "billing_exception",
	}, nil)

	require.Nil(t, run)
	require.True(t, errortypes.IsPlanRestrictionError(err))
}
