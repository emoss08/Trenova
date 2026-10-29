package carriersettlementservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/carriersettlementcontrol"
	"github.com/emoss08/trenova/internal/testutil/mocks/carriersettlementportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type approvalHoldDeps struct {
	costEvents *carriersettlementportsmock.MockCarrierCostEventRepository
	control    *tenantportsmock.MockCarrierSettlementControlRepository
	svc        *Service
}

func setupApprovalHoldTest(t *testing.T) *approvalHoldDeps {
	t.Helper()
	costEvents := carriersettlementportsmock.NewMockCarrierCostEventRepository(t)
	control := tenantportsmock.NewMockCarrierSettlementControlRepository(t)
	return &approvalHoldDeps{
		costEvents: costEvents,
		control:    control,
		svc:        &Service{costEventRepo: costEvents, settlementControl: control},
	}
}

func holdTenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}
}

func TestApprovalHold_RefusesASettlementWithUnmatchedLoads(t *testing.T) {
	deps := setupApprovalHoldTest(t)
	tenantInfo := holdTenantInfo()
	settlementID := pulid.MustNew("cs_")
	deps.control.On("GetOrCreate", mock.Anything, tenantInfo).
		Return(&carriersettlementcontrol.CarrierSettlementControl{HoldUntilInvoiceMatched: true}, nil)
	deps.costEvents.On("CountAwaitingInvoiceMatch", mock.Anything, tenantInfo, settlementID).
		Return(2, nil).Once()

	hold, err := deps.svc.approvalHoldGuard(t.Context(), tenantInfo, settlementID)

	require.NoError(t, err)
	require.True(t, hold.Refused())
	assert.Contains(t, hold.Refusal.Error(), "waiting on a resolved carrier invoice match")
}

func TestApprovalHold_IsOffByDefault(t *testing.T) {
	deps := setupApprovalHoldTest(t)
	tenantInfo := holdTenantInfo()
	deps.control.On("GetOrCreate", mock.Anything, tenantInfo).
		Return(&carriersettlementcontrol.CarrierSettlementControl{}, nil)

	hold, err := deps.svc.approvalHoldGuard(t.Context(), tenantInfo, pulid.MustNew("cs_"))

	require.NoError(t, err)
	require.False(t, hold.Refused())
	deps.costEvents.AssertNotCalled(
		t,
		"CountAwaitingInvoiceMatch",
		mock.Anything,
		mock.Anything,
		mock.Anything,
	)
}
