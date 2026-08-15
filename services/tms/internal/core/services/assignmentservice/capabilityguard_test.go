package assignmentservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/shipmentcommercial"
	"github.com/emoss08/trenova/internal/core/services/shipmentservice"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func brokerageOnlyService(
	t *testing.T,
	tenantInfo pagination.TenantInfo,
	moveID pulid.ID,
) (*service, *mocks.MockOrganizationRepository) {
	t.Helper()

	repo := mocks.NewMockAssignmentRepository(t)
	repo.EXPECT().
		GetMoveByID(mock.Anything, tenantInfo, moveID).
		Return(&shipment.ShipmentMove{
			ID:             moveID,
			ShipmentID:     pulid.MustNew("shp_"),
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
			Status:         shipment.MoveStatusNew,
			CoverageType:   shipment.MoveCoverageTypeUnassigned,
		}, nil).
		Once()

	orgRepo := mocks.NewMockOrganizationRepository(t)
	orgRepo.EXPECT().
		GetByID(mock.Anything, repositories.GetOrganizationByIDRequest{TenantInfo: tenantInfo}).
		Return(&tenant.Organization{
			ID:                     tenantInfo.OrgID,
			BusinessUnitID:         tenantInfo.BuID,
			BrokerageEnabled:       true,
			AssetOperationsEnabled: false,
		}, nil).
		Once()

	svc := &service{
		l:            zap.NewNop(),
		db:           testDBConnection{},
		repo:         repo,
		orgRepo:      orgRepo,
		shipmentRepo: mocks.NewMockShipmentRepository(t),
		holdRepo:     mocks.NewMockShipmentHoldRepository(t),
		commercial: shipmentcommercial.New(shipmentcommercial.Params{
			Formula:         mocks.NewMockFormulaCalculator(t),
			AccessorialRepo: mocks.NewMockAccessorialChargeRepository(t),
		}),
		shipmentValidator: shipmentservice.NewTestValidator(t),
		eventService:      noopShipmentEventService{},
		coordinator:       shipmentstate.NewCoordinator(),
	}

	return svc, orgRepo
}

func assertAssetOperationsRefusal(t *testing.T, err error, moveID pulid.ID) {
	t.Helper()

	require.Error(t, err)
	businessErr := new(errortypes.BusinessError)
	require.ErrorAs(t, err, &businessErr)
	assert.Equal(
		t,
		"Organization does not have asset operations enabled. "+
			"Cover this shipment move with a carrier instead of assigning a driver",
		businessErr.Message,
	)
	assert.Equal(t, moveID.String(), businessErr.Params["shipmentMoveId"])
}

func TestAssignToMove_RefusedWhenAssetOperationsDisabled(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("sm_")
	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}

	svc, _ := brokerageOnlyService(t, tenantInfo, moveID)

	entity, err := svc.AssignToMove(t.Context(), &repositories.AssignShipmentMoveRequest{
		TenantInfo:      tenantInfo,
		ShipmentMoveID:  moveID,
		PrimaryWorkerID: pulid.MustNew("wrk_"),
		TractorID:       pulid.MustNew("trac_"),
	})

	assert.Nil(t, entity)
	assertAssetOperationsRefusal(t, err, moveID)
}

func TestReassign_RefusedWhenAssetOperationsDisabled(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("sm_")
	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}

	svc, _ := brokerageOnlyService(t, tenantInfo, moveID)

	entity, err := svc.Reassign(t.Context(), &repositories.ReassignShipmentMoveRequest{
		TenantInfo:      tenantInfo,
		ShipmentMoveID:  moveID,
		PrimaryWorkerID: pulid.MustNew("wrk_"),
		TractorID:       pulid.MustNew("trac_"),
	})

	assert.Nil(t, entity)
	assertAssetOperationsRefusal(t, err, moveID)
}

func TestUnassign_PermittedWhenAssetOperationsDisabled(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("sm_")
	shipmentID := pulid.MustNew("shp_")
	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}

	orgRepo := assetOperationsOrgRepo(t, tenantInfo, false)
	svc := newUnassignService(t, unassignServiceParams{
		TenantInfo: tenantInfo,
		ShipmentID: shipmentID,
		MoveID:     moveID,
		OrgRepo:    orgRepo,
	})

	err := svc.Unassign(t.Context(), &repositories.UnassignShipmentMoveRequest{
		TenantInfo:     tenantInfo,
		ShipmentMoveID: moveID,
	})

	require.NoError(t, err)
	orgRepo.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything)
}
