package userrepository

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/emoss08/trenova/internal/core/domain/platformcatalog"
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

func TestReplaceOrganizationMembershipsRefusesASeatPastTheQuota(t *testing.T) {
	t.Parallel()

	conn, sqlMock := dbtest.NewSQLMock(t)
	guard := mocks.NewMockQuotaGuard(t)
	repo := &repository{db: conn, l: zap.NewNop(), quota: guard}
	buID := pulid.MustNew("bu_")
	orgID := pulid.MustNew("org_")

	sqlMock.ExpectQuery(`SELECT count\(\*\) FROM "organizations"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	sqlMock.ExpectQuery(`FROM "user_organization_memberships"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	sqlMock.ExpectBegin()
	sqlMock.ExpectExec(`SET LOCAL lock_timeout`).WillReturnResult(sqlmock.NewResult(0, 0))
	sqlMock.ExpectQuery(`FROM "user_organization_memberships"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	guard.EXPECT().Enforce(mock.Anything, &services.QuotaRequest{
		TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
		Meter:      platformcatalog.MeterUserSeats,
		Quantity:   1,
	}).Return(errortypes.NewQuotaExceededError(string(platformcatalog.MeterUserSeats), 1, 1, "free_demo"))
	sqlMock.ExpectRollback()

	memberships, err := repo.ReplaceOrganizationMemberships(
		t.Context(),
		repositories.ReplaceOrganizationMembershipsRequest{
			ActorID:         pulid.MustNew("usr_"),
			UserID:          pulid.MustNew("usr_"),
			BusinessUnitID:  buID,
			OrganizationIDs: []pulid.ID{orgID},
		},
	)

	require.Nil(t, memberships)
	require.True(t, errortypes.IsQuotaExceededError(err))
	require.NoError(t, sqlMock.ExpectationsWereMet())
}
