package briefingservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/platformplan"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/require"
)

func TestRegenerateRefusesWhenThePlanRestrictsAutomation(t *testing.T) {
	t.Parallel()

	svc := &Service{plans: plantest.Restricting(t, platformplan.CapabilityAgentAutomation)}

	page, err := svc.Regenerate(t.Context(), services.GetBriefingRequest{TenantInfo: plantest.Tenant()}, nil)

	require.Nil(t, page)
	require.True(t, errortypes.IsPlanRestrictionError(err))
}
