package shipmentservice

import (
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestServiceGetUIPolicy_MapsShipmentControl(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}

	controlRepo := tenantportsmock.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, tenantports.GetShipmentControlRequest{TenantInfo: tenantInfo}).
		Return(&tenant.ShipmentControl{
			AllowMoveRemovals:      false,
			CheckForDuplicateBOLs:  true,
			MaxShipmentWeightLimit: 80000,
		}, nil).
		Once()

	svc := &service{
		l:           zap.NewNop(),
		controlRepo: controlRepo,
	}

	policy, err := svc.GetUIPolicy(t.Context(), tenantInfo)

	require.NoError(t, err)
	require.NotNil(t, policy)
	assert.False(t, policy.AllowMoveRemovals)
	assert.True(t, policy.CheckForDuplicateBOLs)
	assert.Equal(t, int32(80000), policy.MaxShipmentWeightLimit)
}
