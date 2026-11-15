package agentrunservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/testutil/mocks/kernelportsmock"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/require"
)

func TestStartForDefinitionRefusesBackgroundRunsThePlanRestricts(t *testing.T) {
	t.Parallel()

	workflows := kernelportsmock.NewMockWorkflowStarter(t)
	svc := &Service{
		workflows: workflows,
		features:  plantest.Without(platformcatalog.FeatureAgentAutomation),
	}

	run, err := svc.StartForDefinition(
		t.Context(),
		&agentruntimeports.StartAgentRunForDefinitionRequest{
			TenantInfo: plantest.Tenant(),
			SystemKey:  "billing_exception",
		},
		nil,
	)

	require.Nil(t, run)
	require.True(t, errortypes.IsPlanRestrictionError(err))
}
