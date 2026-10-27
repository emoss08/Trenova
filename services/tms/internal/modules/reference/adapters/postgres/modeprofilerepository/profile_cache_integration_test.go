//go:build integration

package modeprofilerepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/reference/domain/modeprofile"
	"github.com/emoss08/trenova/internal/testutil/cachetest"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newActiveProfile(code string, enforcement tenant.EnforcementLevel) *modeprofile.Profile {
	return &modeprofile.Profile{
		Name:           code,
		Code:           code,
		Status:         modeprofile.ProfileStatusActive,
		ServiceModel:   modeprofile.ServiceModelTruckload,
		EquipmentClass: modeprofile.EquipmentClassDryVan,
		Rules: []*modeprofile.CapabilityRule{{
			RuleKey:     modeprofile.RuleKeyDuplicateBOL,
			Capability:  modeprofile.CapabilityCore,
			Enforcement: enforcement,
			Enabled:     true,
		}},
	}
}

func profileByID(profiles []*modeprofile.Profile, id pulid.ID) *modeprofile.Profile {
	for _, profile := range profiles {
		if profile.ID == id {
			return profile
		}
	}

	return nil
}

func TestActiveProfilesAreServedFromTheResultCacheUntilAProfileChanges(t *testing.T) {
	ctx, db, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	tenantA := seedtest.SeedFullTestData(t, ctx, db)
	tenantB := seedtest.SeedAdditionalTenant(t, ctx, db, "MP")
	cached := cachetest.Open(t, ctx, dsn)
	repo := NewProfileRepository(ProfileParams{
		DB:     postgres.NewTestConnection(cached),
		Logger: zap.NewNop(),
	})
	actx := tenantA.Ctx(ctx)

	created, err := repo.Create(actx, newActiveProfile("CACHED-OTR", tenant.EnforcementLevelBlock))
	require.NoError(t, err)

	var profiles []*modeprofile.Profile
	readFor := func(data *seedtest.TestData) func() error {
		return func() error {
			var readErr error
			profiles, readErr = repo.GetActiveProfiles(data.Ctx(ctx), data.Info())
			return readErr
		}
	}

	t.Run("a second read is a hit with the rules loaded", func(t *testing.T) {
		cachetest.RequireMiss(t, cached, readFor(tenantA))
		cachetest.RequireHit(t, cached, readFor(tenantA))

		profile := profileByID(profiles, created.ID)
		require.NotNil(t, profile, "the created profile is active")
		require.Len(t, profile.Rules, 1, "the rule load is replayed from the entry")
		assert.Equal(t, tenant.EnforcementLevelBlock, profile.Rules[0].Enforcement)
	})

	t.Run("another tenant never reads the first tenant's entry", func(t *testing.T) {
		cachetest.RequireMiss(t, cached, readFor(tenantB))
		assert.Nil(t, profileByID(profiles, created.ID))
	})

	t.Run("changing a rule invalidates the active profiles", func(t *testing.T) {
		created.Rules = []*modeprofile.CapabilityRule{{
			RuleKey:     modeprofile.RuleKeyDuplicateBOL,
			Capability:  modeprofile.CapabilityCore,
			Enforcement: tenant.EnforcementLevelWarn,
			Enabled:     true,
		}}
		_, err := repo.Update(actx, created)
		require.NoError(t, err)

		cachetest.RequireMiss(t, cached, readFor(tenantA))
		profile := profileByID(profiles, created.ID)
		require.NotNil(t, profile)
		require.Len(t, profile.Rules, 1)
		assert.Equal(t, tenant.EnforcementLevelWarn, profile.Rules[0].Enforcement)
		cachetest.RequireHit(t, cached, readFor(tenantA))
	})

	t.Run("a new profile invalidates the active profiles", func(t *testing.T) {
		second, err := repo.Create(
			actx,
			newActiveProfile("CACHED-DEDICATED", tenant.EnforcementLevelBlock),
		)
		require.NoError(t, err)

		cachetest.RequireMiss(t, cached, readFor(tenantA))
		assert.NotNil(t, profileByID(profiles, second.ID))
		assert.NotNil(t, profileByID(profiles, created.ID))
	})
}
