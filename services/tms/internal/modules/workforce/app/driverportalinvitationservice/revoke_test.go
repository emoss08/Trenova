package driverportalinvitationservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/mocks/portalaccessportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/sessionportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestRevokeAccess_EndsTheDriversPortalSessions(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	workerID := pulid.MustNew("wrk_")
	portalUserID := pulid.MustNew("usr_")

	portalRepo := portalaccessportsmock.NewMockPortalAccessRepository(t)
	portalRepo.EXPECT().GetWorkerForPortalManagement(mock.Anything, tenantInfo, workerID).
		Return(&worker.Worker{ID: workerID, UserID: portalUserID}, nil).Once()
	portalRepo.EXPECT().RevokePortalAccess(mock.Anything, tenantInfo, workerID).Return(nil).Once()
	sessions := sessionportsmock.NewMockSessionRepository(t)
	sessions.EXPECT().DeleteAllForUser(mock.Anything, portalUserID).Return(nil).Once()

	svc := &Service{
		l:            zap.NewNop(),
		portalRepo:   portalRepo,
		sessions:     sessions,
		auditService: &mocks.NoopAuditService{},
	}

	require.NoError(t, svc.RevokeAccess(
		t.Context(),
		tenantInfo,
		workerID,
		&permissionports.RequestActor{UserID: pulid.MustNew("usr_")},
	))
}

func TestRevokeAccess_WithoutALinkedSignInEndsNoSessions(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	workerID := pulid.MustNew("wrk_")

	portalRepo := portalaccessportsmock.NewMockPortalAccessRepository(t)
	portalRepo.EXPECT().GetWorkerForPortalManagement(mock.Anything, tenantInfo, workerID).
		Return(&worker.Worker{ID: workerID}, nil).Once()
	portalRepo.EXPECT().RevokePortalAccess(mock.Anything, tenantInfo, workerID).Return(nil).Once()

	svc := &Service{
		l:            zap.NewNop(),
		portalRepo:   portalRepo,
		sessions:     sessionportsmock.NewMockSessionRepository(t),
		auditService: &mocks.NoopAuditService{},
	}

	require.NoError(t, svc.RevokeAccess(
		t.Context(),
		tenantInfo,
		workerID,
		&permissionports.RequestActor{UserID: pulid.MustNew("usr_")},
	))
}
