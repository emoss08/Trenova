package driverportalservice

import (
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/workerportsmock"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/testutil/capabilitytest"
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
) *tenantportsmock.MockOrganizationRepository {
	t.Helper()

	return capabilitytest.OrgRepo(t, capabilitytest.OrgRepoParams{
		TenantInfo:             tenantInfo,
		Lock:                   tenantports.CapabilityLockNone,
		BrokerageEnabled:       true,
		AssetOperationsEnabled: enabled,
	})
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
		portalRepo: workerportsmock.NewMockPortalAccessRepository(t),
	}

	result, err := svc.InviteWorker(
		t.Context(),
		&InviteWorkerRequest{TenantInfo: tenantInfo, WorkerID: pulid.MustNew("wrk_")},
		&permissionports.RequestActor{UserID: pulid.MustNew("usr_")},
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

	portalRepo := workerportsmock.NewMockPortalAccessRepository(t)
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
		&permissionports.RequestActor{UserID: pulid.MustNew("usr_")},
	)

	assert.Nil(t, result)
	require.ErrorIs(t, err, lookupErr)
}
