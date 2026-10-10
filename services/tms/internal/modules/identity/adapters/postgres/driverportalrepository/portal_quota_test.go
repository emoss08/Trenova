package driverportalrepository

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/testutil/mocks/platformplanportsmock"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
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
	guard := platformplanportsmock.NewMockQuotaGuard(t)
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
	guard.EXPECT().Enforce(mock.Anything, &platformplanports.QuotaRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: invitation.OrganizationID,
			BuID:  invitation.BusinessUnitID,
		},
		Meter:    platformcatalog.MeterUserSeats,
		Quantity: 1,
	}).Return(errortypes.NewQuotaExceededError(string(platformcatalog.MeterUserSeats), 1, 1, "free_demo"))
	sqlMock.ExpectRollback()

	user, err := repo.ActivatePortalAccess(t.Context(), &workerports.ActivatePortalAccessRequest{
		Invitation: invitation,
		User:       &tenant.User{ID: pulid.MustNew("usr_")},
	})

	require.Nil(t, user)
	require.True(t, errortypes.IsQuotaExceededError(err))
	require.NoError(t, sqlMock.ExpectationsWereMet())
}
