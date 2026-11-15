package telematicsservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/telematicsports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTelematicsReadsAreRefusedWithoutSamsara(t *testing.T) {
	t.Parallel()

	svc := &Service{features: plantest.Without(platformcatalog.FeatureSamsaraIntegration)}
	tenant := plantest.Tenant()

	_, err := svc.ListVehiclePositions(t.Context(), &telematicsports.ListVehiclePositionsRequest{
		TenantInfo: tenant,
	})
	assert.True(t, errortypes.IsPlanRestrictionError(err), "vehicle positions")

	_, err = svc.GetWorkerHOSState(t.Context(), tenant, pulid.MustNew("wrk_"))
	assert.True(t, errortypes.IsPlanRestrictionError(err), "a driver's hours")

	_, err = svc.ListWorkerHOSViolations(t.Context(), tenant, pulid.MustNew("wrk_"), 0, 10)
	assert.True(t, errortypes.IsPlanRestrictionError(err), "hours violations")

	_, err = svc.ListShipmentFormSubmissions(t.Context(), tenant, pulid.MustNew("shp_"))
	assert.True(t, errortypes.IsPlanRestrictionError(err), "form submissions")

	_, err = svc.EvaluateDriverFeasibility(t.Context(), &FeasibilityRequest{TenantInfo: tenant})
	assert.True(t, errortypes.IsPlanRestrictionError(err), "driver feasibility")

	err = svc.DeleteFormMapping(t.Context(), tenant, pulid.MustNew("tfm_"))
	assert.True(t, errortypes.IsPlanRestrictionError(err), "form mappings")
}

func TestTelematicsStatusOffThePlanIsAnEmptyStatusNotAnError(t *testing.T) {
	t.Parallel()

	svc := &Service{features: plantest.Without(platformcatalog.FeatureSamsaraIntegration)}

	status, err := svc.GetStatus(t.Context(), plantest.Tenant())

	require.NoError(t, err)
	assert.Equal(t, &Status{Provider: string(integration.TypeSamsara)}, status)
}
