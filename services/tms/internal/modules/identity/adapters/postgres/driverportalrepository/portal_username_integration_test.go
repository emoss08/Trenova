//go:build integration

package driverportalrepository

import (
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/portalinvitation"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/portalinvitation/portalinvitationdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/portalaccessports"
	"github.com/emoss08/trenova/internal/testutil/mocks/platformplanportsmock"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// Usernames are unique across the server, so a driver whose name another
// company's user already holds gets the next free one instead of failing on
// the unique index; the activation learns only that the name was taken.
func TestActivatePortalAccessSkipsAUsernameAnotherCompanyHolds(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	other := seedtest.SeedAdditionalTenant(t, ctx, db, "UN")
	taken := "drv" + strings.ToLower(pulid.MustNew("").String()[:8])
	seedtest.NewUser(other.Organization.ID, other.BusinessUnit.ID).
		WithUsername(taken).
		WithEmail(taken+"@globex.example").
		Build(t, ctx, db)

	tenantCtx := data.Ctx(ctx)
	driver := &worker.Worker{
		StateID:       data.State.ID,
		Status:        domaintypes.StatusActive,
		Type:          worker.WorkerTypeEmployee,
		DriverType:    worker.DriverTypeOTR,
		FirstName:     "Name",
		LastName:      "Taken",
		AddressLine1:  "1 Depot Road",
		City:          "Springfield",
		PostalCode:    "62701",
		Email:         taken + "@acme.example",
		PhoneNumber:   "555-0100",
		Gender:        worker.GenderMale,
		CanBeAssigned: true,
	}
	require.NoError(t, workerdb.Workers.Create(tenantCtx, db, driver))

	invitation := &portalinvitation.PortalInvitation{
		WorkerID:    driver.ID,
		Email:       driver.Email,
		TokenHash:   strings.Repeat("b", 64),
		Status:      portalinvitation.PortalInvitationStatusPending,
		ExpiresAt:   time.Now().Add(time.Hour).Unix(),
		InvitedByID: data.User.ID,
	}
	require.NoError(t, portalinvitationdb.PortalInvitations.Create(tenantCtx, db, invitation))

	guard := platformplanportsmock.NewMockQuotaGuard(t)
	guard.EXPECT().Enforce(mock.Anything, mock.Anything).Return(nil)
	repo := &portalAccessRepository{
		db:    postgres.WrapDB(db),
		l:     zap.NewNop(),
		quota: guard,
	}

	userID := pulid.MustNew("usr_")
	user, err := repo.ActivatePortalAccess(
		dbscope.WithTenant(ctx, dbscope.Tenant{
			OrganizationID: data.Organization.ID,
			BusinessUnitID: data.BusinessUnit.ID,
			UserID:         userID,
		}),
		&portalaccessports.ActivatePortalAccessRequest{
			Invitation: invitation,
			User: &tenant.User{
				ID:                    userID,
				BusinessUnitID:        data.BusinessUnit.ID,
				CurrentOrganizationID: data.Organization.ID,
				Status:                domaintypes.StatusActive,
				Name:                  "Name Taken",
				Username:              taken,
				EmailAddress:          driver.Email,
				Timezone:              "America/Chicago",
			},
			RoleName:        "Driver",
			RoleDescription: "Driver portal access",
		},
	)

	require.NoError(t, err)
	assert.Equal(t, taken+"2", user.Username)
}
