//go:build integration

package driverportalrepository

import (
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/internal/testutil/mocks/platformplanportsmock"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestActivatePortalAccessRefusesASeatPastTheQuota(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	tenantCtx := data.Ctx(ctx)
	driver := &worker.Worker{
		StateID:       data.State.ID,
		Status:        domaintypes.StatusActive,
		Type:          worker.WorkerTypeEmployee,
		DriverType:    worker.DriverTypeOTR,
		FirstName:     "Quota",
		LastName:      "Driver",
		AddressLine1:  "1 Depot Road",
		City:          "Springfield",
		PostalCode:    "62701",
		Email:         "quota-" + strings.ToLower(pulid.MustNew("e").String()) + "@example.com",
		PhoneNumber:   "555-0100",
		Gender:        worker.GenderMale,
		CanBeAssigned: true,
	}
	require.NoError(t, workerdb.Workers.Create(tenantCtx, db, driver))

	invitation := &worker.PortalInvitation{
		WorkerID:    driver.ID,
		Email:       driver.Email,
		TokenHash:   strings.Repeat("a", 64),
		Status:      worker.PortalInvitationStatusPending,
		ExpiresAt:   time.Now().Add(time.Hour).Unix(),
		InvitedByID: data.User.ID,
	}
	require.NoError(t, workerdb.PortalInvitations.Create(tenantCtx, db, invitation))

	guard := platformplanportsmock.NewMockQuotaGuard(t)
	repo := &portalAccessRepository{db: postgres.NewTestConnection(db), l: zap.NewNop(), quota: guard}
	guard.EXPECT().Enforce(mock.Anything, &platformplanports.QuotaRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: data.Organization.ID,
			BuID:  data.BusinessUnit.ID,
		},
		Meter:    platformcatalog.MeterUserSeats,
		Quantity: 1,
	}).Return(errortypes.NewQuotaExceededError(string(platformcatalog.MeterUserSeats), 1, 1, "free_demo"))

	user, err := repo.ActivatePortalAccess(tenantCtx, &workerports.ActivatePortalAccessRequest{
		Invitation: invitation,
		User: &tenant.User{
			Name:         "Quota Driver",
			Username:     "quotadriver",
			EmailAddress: driver.Email,
		},
	})

	require.Nil(t, user)
	require.True(t, errortypes.IsQuotaExceededError(err))

	u := tenantdb.Users
	exists, err := u.Where(u.EmailAddress.Eq(driver.Email)).Exists(tenantCtx, db)
	require.NoError(t, err)
	assert.False(t, exists, "no portal user is created past the quota")
}
