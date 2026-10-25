package userrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/internal/testutil/mocks/platformplanportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestCreateRefusesASeatPastTheQuotaBeforeInsertingAnything(t *testing.T) {
	t.Parallel()

	guard := platformplanportsmock.NewMockQuotaGuard(t)
	repo := &repository{
		db:    postgres.NewTestConnection(keeltest.RenderDB(t)),
		l:     zap.NewNop(),
		quota: guard,
	}
	buID := pulid.MustNew("bu_")
	orgID := pulid.MustNew("org_")

	guard.EXPECT().Enforce(mock.Anything, &platformplanports.QuotaRequest{
		TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
		Meter:      platformcatalog.MeterUserSeats,
		Quantity:   1,
	}).Return(errortypes.NewQuotaExceededError(string(platformcatalog.MeterUserSeats), 3, 3, "starter"))

	created, err := repo.Create(keeltest.TenantContext(t), tenantports.CreateUserRequest{
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
}
