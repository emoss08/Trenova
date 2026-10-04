package apikeyservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/platformplan"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
)

func TestCreateAPIKeyRefusesWhenThePlanRestrictsAPIKeys(t *testing.T) {
	t.Parallel()

	svc := &Service{plans: plantest.Restricting(t, platformplan.CapabilityAPIKeys)}

	resp, err := svc.CreateAPIKey(t.Context(), plantest.Tenant(), &services.CreateAPIKeyRequest{
		Name: "integration",
		Permissions: []services.APIKeyPermissionInput{{
			Resource:   "shipment",
			Operations: []permission.Operation{permission.OpRead},
		}},
	}, pulid.MustNew("usr_"))

	require.Nil(t, resp)
	require.True(t, errortypes.IsPlanRestrictionError(err))
}

func TestRotateAPIKeyRefusesWhenThePlanRestrictsAPIKeys(t *testing.T) {
	t.Parallel()

	svc := &Service{plans: plantest.Restricting(t, platformplan.CapabilityAPIKeys)}

	resp, err := svc.RotateAPIKey(t.Context(), plantest.Tenant(), pulid.MustNew("ak_"))

	require.Nil(t, resp)
	require.True(t, errortypes.IsPlanRestrictionError(err))
}
