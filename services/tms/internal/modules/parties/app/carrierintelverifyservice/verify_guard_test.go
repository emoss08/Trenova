package carrierintelverifyservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/require"
)

func TestPaidEquipmentVerificationRefusesWhenThePlanRestrictsIt(t *testing.T) {
	t.Parallel()

	svc := &Service{
		features: plantest.Without(platformcatalog.FeatureCarrierBrokerage),
	}
	tenant := plantest.Tenant()

	verification, err := svc.VerifyEquipment(
		t.Context(),
		&VerifyEquipmentRequest{TenantInfo: tenant},
	)
	require.Nil(t, verification)
	require.True(t, errortypes.IsPlanRestrictionError(err))
}
