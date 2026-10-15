package userrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/testutil/mocks/platformplanportsmock"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
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
	guard := platformplanportsmock.NewMockQuotaGuard(t)
	repo := &repository{db: conn, l: zap.NewNop(), quota: guard}
	buID := pulid.MustNew("bu_")
	orgID := pulid.MustNew("org_")

	sqlMock.ExpectQuery(`SELECT EXISTS \(SELECT .* FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	sqlMock.ExpectQuery(`SELECT count\(\*\) FROM "organizations"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	sqlMock.ExpectQuery(`FROM "user_organization_memberships"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	sqlMock.ExpectBegin()
	sqlMock.ExpectExec(`SET LOCAL lock_timeout`).WillReturnResult(sqlmock.NewResult(0, 0))
	sqlMock.ExpectQuery(`FROM "user_organization_memberships"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	guard.EXPECT().Enforce(mock.Anything, &platformplanports.QuotaRequest{
		TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
		Meter:      platformcatalog.MeterUserSeats,
		Quantity:   1,
	}).Return(errortypes.NewQuotaExceededError(string(platformcatalog.MeterUserSeats), 1, 1, "free_demo"))
	sqlMock.ExpectRollback()

	memberships, err := repo.ReplaceOrganizationMemberships(
		t.Context(),
		tenantports.ReplaceOrganizationMembershipsRequest{
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

func TestCreateRefusesASeatPastTheQuotaBeforeInsertingAnything(t *testing.T) {
	t.Parallel()

	conn, sqlMock := dbtest.NewSQLMock(t)
	guard := platformplanportsmock.NewMockQuotaGuard(t)
	repo := &repository{db: conn, l: zap.NewNop(), quota: guard}
	buID := pulid.MustNew("bu_")
	orgID := pulid.MustNew("org_")

	guard.EXPECT().Enforce(mock.Anything, &platformplanports.QuotaRequest{
		TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
		Meter:      platformcatalog.MeterUserSeats,
		Quantity:   1,
	}).Return(errortypes.NewQuotaExceededError(string(platformcatalog.MeterUserSeats), 3, 3, "starter"))

	created, err := repo.Create(t.Context(), tenantports.CreateUserRequest{
		User: &tenant.User{
			BusinessUnitID:        buID,
			CurrentOrganizationID: orgID,
			Name:                  "Dana Whitfield",
			Username:              "dwhitfield",
			EmailAddress:          "dana@example.com",
			Timezone:              "America/Chicago",
		},
		Membership: &tenant.OrganizationMembership{
			BusinessUnitID: buID,
			OrganizationID: orgID,
			IsDefault:      true,
		},
	})

	require.Nil(t, created)
	require.True(t, errortypes.IsQuotaExceededError(err))
	require.NoError(t, sqlMock.ExpectationsWereMet())
}
