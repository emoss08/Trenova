package driverportalservice

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func assetOperationsOrgRepo(
	t *testing.T,
	tenantInfo pagination.TenantInfo,
	enabled bool,
) *mocks.MockOrganizationRepository {
	t.Helper()

	orgRepo := mocks.NewMockOrganizationRepository(t)
	orgRepo.EXPECT().
		GetByID(mock.Anything, repositories.GetOrganizationByIDRequest{TenantInfo: tenantInfo}).
		Return(&tenant.Organization{
			ID:                     tenantInfo.OrgID,
			BusinessUnitID:         tenantInfo.BuID,
			BrokerageEnabled:       true,
			AssetOperationsEnabled: enabled,
		}, nil).
		Once()

	return orgRepo
}

func TestInviteWorker_RefusedWhenAssetOperationsDisabled(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}

	svc := &Service{
		l:          zap.NewNop(),
		orgRepo:    assetOperationsOrgRepo(t, tenantInfo, false),
		portalRepo: mocks.NewMockPortalAccessRepository(t),
	}

	result, err := svc.InviteWorker(
		t.Context(),
		&InviteWorkerRequest{TenantInfo: tenantInfo, WorkerID: pulid.MustNew("wrk_")},
		&serviceports.RequestActor{UserID: pulid.MustNew("usr_")},
	)

	assert.Nil(t, result)
	require.Error(t, err)

	validationErr := new(errortypes.Error)
	require.ErrorAs(t, err, &validationErr)
	assert.Equal(t, "feature", validationErr.Field)
	assert.Equal(t, errortypes.ErrInvalidOperation, validationErr.Code)
	assert.Equal(
		t,
		"Driver portal access requires asset operations. Enable asset operations "+
			"for this organization before inviting drivers",
		validationErr.Message,
	)
}

func TestInviteWorker_ProceedsWhenAssetOperationsEnabled(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}
	workerID := pulid.MustNew("wrk_")
	lookupErr := errors.New("worker lookup failed")

	portalRepo := mocks.NewMockPortalAccessRepository(t)
	portalRepo.EXPECT().
		GetWorkerForPortalManagement(mock.Anything, tenantInfo, workerID).
		Return(nil, lookupErr).
		Once()

	svc := &Service{
		l:          zap.NewNop(),
		orgRepo:    assetOperationsOrgRepo(t, tenantInfo, true),
		portalRepo: portalRepo,
	}

	result, err := svc.InviteWorker(
		t.Context(),
		&InviteWorkerRequest{TenantInfo: tenantInfo, WorkerID: workerID},
		&serviceports.RequestActor{UserID: pulid.MustNew("usr_")},
	)

	assert.Nil(t, result)
	require.ErrorIs(t, err, lookupErr)
}
