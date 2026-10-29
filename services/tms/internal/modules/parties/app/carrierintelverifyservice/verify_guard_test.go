package carrierintelverifyservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/platformplan"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/require"
)

func TestPaidEquipmentVerificationRefusesWhenThePlanRestrictsIt(t *testing.T) {
	t.Parallel()

	svc := &Service{
		plans: plantest.Restricting(t, platformplan.CapabilityCarrierIntelligencePaid),
	}
	tenant := plantest.Tenant()

	verification, err := svc.VerifyEquipment(
		t.Context(),
		&VerifyEquipmentRequest{TenantInfo: tenant},
	)
	require.Nil(t, verification)
	require.True(t, errortypes.IsPlanRestrictionError(err))
}
