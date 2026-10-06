package integrationservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/domain/platformplan"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestUpdateConfigRefusesEnablingAnIntegrationThePlanRestricts(t *testing.T) {
	t.Parallel()

	svc := &Service{l: zap.NewNop(), plans: plantest.Restricting(t, platformplan.CapabilityIntegrations)}

	resp, err := svc.UpdateConfig(
		t.Context(),
		plantest.Tenant(),
		integration.TypeGoogleMaps,
		&services.UpdateConfigRequest{Enabled: true},
		pulid.MustNew("usr_"),
	)

	require.Nil(t, resp)
	require.True(t, errortypes.IsPlanRestrictionError(err))
}

func TestTestConnectionRefusesWhenThePlanRestrictsIntegrations(t *testing.T) {
	t.Parallel()

	svc := &Service{l: zap.NewNop(), plans: plantest.Restricting(t, platformplan.CapabilityIntegrations)}

	resp, err := svc.TestConnection(
		t.Context(),
		plantest.Tenant(),
		integration.TypeSamsara,
		pulid.MustNew("usr_"),
	)

	require.Nil(t, resp)
	require.True(t, errortypes.IsPlanRestrictionError(err))
}
