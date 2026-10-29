//go:build integration

package emailrepository_test

import (
	"context"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/workspace/adapters/postgres/emailrepository"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func seedEmailProfile(
	t *testing.T,
	ctx context.Context,
	repo emailports.EmailRepository,
	tenant *seedtest.TestData,
	name string,
) *email.Profile {
	t.Helper()

	draft := &email.Profile{
		BusinessUnitID: tenant.BusinessUnit.ID,
		OrganizationID: tenant.Organization.ID,
		Name:           name,
		Provider:       email.ProviderResend,
		SenderName:     "Dispatch",
		SenderEmail:    strings.ToLower(name) + "@example.com",
		Status:         email.ProfileStatusActive,
	}
	multiErr := errortypes.NewMultiError()
	draft.Validate(multiErr)
	require.False(t, multiErr.HasErrors(), multiErr.Error())

	profile, err := repo.CreateProfile(tenant.Ctx(ctx), draft)
	require.NoError(t, err)

	return profile
}

func TestGetProfilesByIDsReturnsOnlyTheTenantsProfiles(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	tenantA := seedtest.SeedFullTestData(t, ctx, db)
	tenantB := seedtest.SeedAdditionalTenant(t, ctx, db, "EQ")
	repo := emailrepository.New(emailrepository.Params{
		DB:     postgres.NewTestConnection(db),
		Logger: zap.NewNop(),
	})

	first := seedEmailProfile(t, ctx, repo, tenantA, "Billing")
	second := seedEmailProfile(t, ctx, repo, tenantA, "Dispatch")
	foreign := seedEmailProfile(t, ctx, repo, tenantB, "Foreign")

	profiles, err := repo.GetProfilesByIDs(
		tenantA.Ctx(ctx),
		emailports.GetEmailProfilesByIDsRequest{
			TenantInfo: tenantA.Info(),
			ProfileIDs: []pulid.ID{second.ID, foreign.ID, first.ID, pulid.MustNew("emlprof_")},
		},
	)
	require.NoError(t, err)

	names := make(map[pulid.ID]string, len(profiles))
	for _, profile := range profiles {
		names[profile.ID] = profile.Name
	}
	assert.Equal(t, map[pulid.ID]string{first.ID: "Billing", second.ID: "Dispatch"}, names)

	none, err := repo.GetProfilesByIDs(
		tenantA.Ctx(ctx),
		emailports.GetEmailProfilesByIDsRequest{TenantInfo: tenantA.Info()},
	)
	require.NoError(t, err)
	assert.Empty(t, none)
}
