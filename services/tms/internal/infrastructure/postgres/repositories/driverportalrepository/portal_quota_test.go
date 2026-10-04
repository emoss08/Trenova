package driverportalrepository

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/emoss08/trenova/internal/core/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestActivatePortalAccessRefusesASeatPastTheQuota(t *testing.T) {
	t.Parallel()

	conn, sqlMock := dbtest.NewSQLMock(t)
	guard := mocks.NewMockQuotaGuard(t)
	repo := &portalAccessRepository{db: conn, l: zap.NewNop(), quota: guard}
	invitation := &worker.PortalInvitation{
		ID:             pulid.MustNew("wpi_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}

	sqlMock.ExpectBegin()
	sqlMock.ExpectQuery(`FROM "worker_portal_invitations"`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "organization_id", "business_unit_id", "status", "expires_at",
		}).AddRow(
			invitation.ID.String(),
			invitation.OrganizationID.String(),
			invitation.BusinessUnitID.String(),
			string(worker.PortalInvitationStatusPending),
			time.Now().Add(time.Hour).Unix(),
		))
	guard.EXPECT().Enforce(mock.Anything, &services.QuotaRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: invitation.OrganizationID,
			BuID:  invitation.BusinessUnitID,
		},
		Meter:    platformcatalog.MeterUserSeats,
		Quantity: 1,
	}).Return(errortypes.NewQuotaExceededError(string(platformcatalog.MeterUserSeats), 1, 1, "free_demo"))
	sqlMock.ExpectRollback()

	user, err := repo.ActivatePortalAccess(t.Context(), &repositories.ActivatePortalAccessRequest{
		Invitation: invitation,
		User:       &tenant.User{ID: pulid.MustNew("usr_")},
	})

	require.Nil(t, user)
	require.True(t, errortypes.IsQuotaExceededError(err))
	require.NoError(t, sqlMock.ExpectationsWereMet())
}
