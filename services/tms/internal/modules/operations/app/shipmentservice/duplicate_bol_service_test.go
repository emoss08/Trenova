package shipmentservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/shipmentcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentservicetest"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
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

	repo := shipmentportsmock.NewMockShipmentRepository(t)
	controlRepo := tenantportsmock.NewMockShipmentControlRepository(t)
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	controlRepo.EXPECT().Get(mock.Anything, tenantports.GetShipmentControlRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: orgID,
			BuID:  buID,
		},
	}).Return(&shipmentcontrol.ShipmentControl{
		CheckForDuplicateBOLs: false,
	}, nil).Once()

	svc := assembled(&service{
		l:            zap.NewNop(),
		repo:         repo,
		controlRepo:  controlRepo,
		validator:    newServiceValidator(t),
		eventService: noopShipmentEventService{},
		coordinator:  newStateCoordinator(),
	})

	err := svc.reader.CheckForDuplicateBOLs(t.Context(), &shipmentports.DuplicateBOLCheckRequest{
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

	repo := shipmentportsmock.NewMockShipmentRepository(t)
	controlRepo := tenantportsmock.NewMockShipmentControlRepository(t)
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	shipmentID := pulid.MustNew("shp_")

	controlRepo.EXPECT().Get(mock.Anything, tenantports.GetShipmentControlRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: orgID,
			BuID:  buID,
		},
	}).Return(&shipmentcontrol.ShipmentControl{
		CheckForDuplicateBOLs: true,
	}, nil).Once()
	repo.EXPECT().
		CheckForDuplicateBOLs(mock.Anything, mock.MatchedBy(func(req *shipmentports.DuplicateBOLCheckRequest) bool {
			return req.BOL == "BOL-123" &&
				req.TenantInfo.OrgID == orgID &&
				req.TenantInfo.BuID == buID &&
				req.ShipmentID != nil &&
				*req.ShipmentID == shipmentID
		})).
		Return([]*shipmentports.DuplicateBOLResult{
			{ID: pulid.MustNew("shp_"), ProNumber: "PRO-101"},
		}, nil).
		Once()

	svc := assembled(&service{
		l:            zap.NewNop(),
		repo:         repo,
		controlRepo:  controlRepo,
		validator:    newServiceValidator(t),
		eventService: noopShipmentEventService{},
		coordinator:  newStateCoordinator(),
	})

	err := svc.reader.CheckForDuplicateBOLs(t.Context(), &shipmentports.DuplicateBOLCheckRequest{
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
	shipmentservicetest.AssertErrorField(t, multiErr, "bol")
	assert.Contains(t, multiErr.Error(), "PRO-101")
}

func TestCheckDuplicateBOLsWithControl_NoDuplicates(t *testing.T) {
	t.Parallel()

	repo := shipmentportsmock.NewMockShipmentRepository(t)
	repo.EXPECT().
		CheckForDuplicateBOLs(mock.Anything, mock.Anything).
		Return(nil, nil).
		Once()

	svc := assembled(&service{
		l:            zap.NewNop(),
		repo:         repo,
		controlRepo:  tenantportsmock.NewMockShipmentControlRepository(t),
		validator:    newServiceValidator(t),
		eventService: noopShipmentEventService{},
		coordinator:  newStateCoordinator(),
	})

	err := svc.effects.CheckDuplicateBOLsWithControl(
		t.Context(),
		&shipmentcontrol.ShipmentControl{CheckForDuplicateBOLs: true},
		&shipmentports.DuplicateBOLCheckRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: pulid.MustNew("org_"),
				BuID:  pulid.MustNew("bu_"),
			},
			BOL: "BOL-123",
		},
	)

	require.NoError(t, err)
}

func TestServiceCheckDuplicateBOLsWithControl_SkipsABlankBOL(t *testing.T) {
	t.Parallel()

	for _, bol := range []string{"", "   "} {
		repo := shipmentportsmock.NewMockShipmentRepository(t)
		svc := assembled(&service{
			l:            zap.NewNop(),
			repo:         repo,
			validator:    newServiceValidator(t),
			eventService: noopShipmentEventService{},
			coordinator:  newStateCoordinator(),
		})

		err := svc.effects.CheckDuplicateBOLsWithControl(
			t.Context(),
			&shipmentcontrol.ShipmentControl{CheckForDuplicateBOLs: true},
			&shipmentports.DuplicateBOLCheckRequest{
				TenantInfo: pagination.TenantInfo{
					OrgID: pulid.MustNew("org_"),
					BuID:  pulid.MustNew("bu_"),
				},
				BOL: bol,
			},
		)

		require.NoError(t, err)
	}
}
