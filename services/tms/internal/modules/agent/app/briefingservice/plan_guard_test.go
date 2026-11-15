package briefingservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/briefingports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/require"
)

func TestRegenerateRefusesWhenThePlanLeavesOutTheDesk(t *testing.T) {
	t.Parallel()

	svc := &Service{features: plantest.Without(platformcatalog.FeatureAIDesk)}

	page, err := svc.Regenerate(t.Context(), briefingports.GetBriefingRequest{TenantInfo: plantest.Tenant()}, nil)

	require.Nil(t, page)
	require.True(t, errortypes.IsPlanRestrictionError(err))
}
