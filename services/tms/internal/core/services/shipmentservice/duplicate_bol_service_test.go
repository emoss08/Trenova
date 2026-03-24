package shipmentservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestServiceCheckForDuplicateBOLs_SkipsLookupWhenDisabled(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockShipmentRepository(t)
	controlRepo := mocks.NewMockShipmentControlRepository(t)
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	controlRepo.EXPECT().Get(mock.Anything, repositories.GetShipmentControlRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: orgID,
			BuID:  buID,
		},
	}).Return(&tenant.ShipmentControl{
		CheckForDuplicateBOLs: false,
	}, nil).Once()

	svc := &service{
		l:           zap.NewNop(),
		repo:        repo,
		controlRepo: controlRepo,
		validator:   NewTestValidator(t),
		coordinator: newStateCoordinator(),
	}

	err := svc.CheckForDuplicateBOLs(t.Context(), &repositories.DuplicateBOLCheckRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: orgID,
			BuID:  buID,
		},
		BOL: "BOL-123",
	})

	require.NoError(t, err)
}

func TestServiceCheckForDuplicateBOLs_ReturnsDuplicateError(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockShipmentRepository(t)
	controlRepo := mocks.NewMockShipmentControlRepository(t)
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	shipmentID := pulid.MustNew("shp_")

	controlRepo.EXPECT().Get(mock.Anything, repositories.GetShipmentControlRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: orgID,
			BuID:  buID,
		},
	}).Return(&tenant.ShipmentControl{
		CheckForDuplicateBOLs: true,
	}, nil).Once()
	repo.EXPECT().
		CheckForDuplicateBOLs(mock.Anything, mock.MatchedBy(func(req *repositories.DuplicateBOLCheckRequest) bool {
			return req.BOL == "BOL-123" &&
				req.TenantInfo.OrgID == orgID &&
				req.TenantInfo.BuID == buID &&
				req.ShipmentID != nil &&
				*req.ShipmentID == shipmentID
		})).
		Return([]*repositories.DuplicateBOLResult{
			{ID: pulid.MustNew("shp_"), ProNumber: "PRO-101"},
		}, nil).
		Once()

	svc := &service{
		l:           zap.NewNop(),
		repo:        repo,
		controlRepo: controlRepo,
		validator:   NewTestValidator(t),
		coordinator: newStateCoordinator(),
	}

	err := svc.CheckForDuplicateBOLs(t.Context(), &repositories.DuplicateBOLCheckRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: orgID,
			BuID:  buID,
		},
		BOL:        "BOL-123",
		ShipmentID: &shipmentID,
	})

	require.Error(t, err)
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	assertErrorField(t, multiErr, "bol")
	assert.Contains(t, multiErr.Error(), "PRO-101")
}
