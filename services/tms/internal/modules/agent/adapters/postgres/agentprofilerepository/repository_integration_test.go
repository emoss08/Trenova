//go:build integration

package agentprofilerepository

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentprofile"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentprofileports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/hashutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type profileFixture struct {
	profiles agentprofileports.ProfileRepository
	guidance agentprofileports.GuidanceRepository
	facts    agentprofileports.FactsReader
	tenant   pagination.TenantInfo
	userID   pulid.ID
}

func newProfileFixture(t *testing.T) (context.Context, profileFixture) {
	t.Helper()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)
	data := seedtest.SeedFullTestData(t, ctx, db)

	params := Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()}

	return ctx, profileFixture{
		profiles: NewProfiles(params),
		guidance: NewGuidance(params),
		facts:    NewFacts(params),
		tenant:   pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID},
		userID:   data.User.ID,
	}
}

func (f profileFixture) version(ref agentprofileports.SubjectRef, content string) *agentprofile.Profile {
	return &agentprofile.Profile{
		OrganizationID: f.tenant.OrgID,
		BusinessUnitID: f.tenant.BuID,
		Subject:        ref.Subject,
		UserID:         ref.PersonPtr(),
		Trigger:        agentprofile.TriggerScheduled,
		Content:        content,
		Facts:          agentprofile.Facts{{Key: "shipments.volume", Text: "Entered 40 shipments."}},
		Digest:         hashutils.SHA256Hex(content),
		CheckedAt:      timeutils.NowUnix(),
	}
}

func TestPublish_KeepsOneCurrentVersionPerProfile(t *testing.T) {
	ctx, f := newProfileFixture(t)
	org := agentprofileports.OrganizationSubject(f.tenant)
	person := agentprofileports.UserSubject(f.tenant, f.userID)

	none, err := f.profiles.GetCurrent(ctx, org)
	require.NoError(t, err)
	assert.Nil(t, none, "before the first version there is nothing to tell agents")

	first, err := f.profiles.Publish(ctx, f.version(org, "Runs a regional fleet."))
	require.NoError(t, err)
	mine, err := f.profiles.Publish(ctx, f.version(person, "Dispatches regional loads."))
	require.NoError(t, err)
	second, err := f.profiles.Publish(ctx, f.version(org, "Runs a regional fleet and brokers."))
	require.NoError(t, err)

	current, err := f.profiles.GetCurrent(ctx, org)
	require.NoError(t, err)
	require.NotNil(t, current)
	assert.Equal(t, second.ID, current.ID)

	personal, err := f.profiles.GetCurrent(ctx, person)
	require.NoError(t, err)
	require.NotNil(t, personal)
	assert.Equal(t, mine.ID, personal.ID, "the organization's versions never touch a person's")

	history, err := f.profiles.ListHistory(ctx, agentprofileports.ListProfileHistoryRequest{Ref: org})
	require.NoError(t, err)
	require.Len(t, history, 2)
	assert.Equal(t, second.ID, history[0].ID)
	assert.Equal(t, agentprofile.StatusSuperseded, history[1].Status)
	assert.Equal(t, first.ID, history[1].ID)
}

func TestRecordFailure_LeavesTheCurrentVersionInPlace(t *testing.T) {
	ctx, f := newProfileFixture(t)
	org := agentprofileports.OrganizationSubject(f.tenant)

	current, err := f.profiles.Publish(ctx, f.version(org, "Runs a regional fleet."))
	require.NoError(t, err)

	failed := f.version(org, "")
	failed.Fail("the profile stated figures the facts do not hold: 4,000")
	_, err = f.profiles.RecordFailure(ctx, failed)
	require.NoError(t, err)

	still, err := f.profiles.GetCurrent(ctx, org)
	require.NoError(t, err)
	assert.Equal(t, current.ID, still.ID)

	history, err := f.profiles.ListHistory(ctx, agentprofileports.ListProfileHistoryRequest{
		Ref:   org,
		Limit: 1,
	})
	require.NoError(t, err)
	require.Len(t, history, 1)
	assert.Equal(t, agentprofile.StatusFailed, history[0].Status)
}

func TestMarkChecked_TouchesOnlyTheCurrentVersion(t *testing.T) {
	ctx, f := newProfileFixture(t)
	org := agentprofileports.OrganizationSubject(f.tenant)

	current, err := f.profiles.Publish(ctx, f.version(org, "Runs a regional fleet."))
	require.NoError(t, err)

	at := current.CheckedAt + 3600
	require.NoError(t, f.profiles.MarkChecked(ctx, org, current.ID, at))

	read, err := f.profiles.GetCurrent(ctx, org)
	require.NoError(t, err)
	assert.Equal(t, at, read.CheckedAt)

	err = f.profiles.MarkChecked(ctx, org, pulid.MustNew("aprf_"), at)
	require.Error(t, err, "a version that is not current is not marked")
}

func TestPrune_KeepsTheNewestVersionsAndTheCurrentOne(t *testing.T) {
	ctx, f := newProfileFixture(t)
	org := agentprofileports.OrganizationSubject(f.tenant)

	var last *agentprofile.Profile
	for idx := range 5 {
		published, err := f.profiles.Publish(ctx, f.version(org, "Version "+string(rune('A'+idx))))
		require.NoError(t, err)
		last = published
	}

	pruned, err := f.profiles.Prune(ctx, agentprofileports.PruneProfilesRequest{Ref: org, Keep: 2})
	require.NoError(t, err)
	assert.Equal(t, 3, pruned)

	history, err := f.profiles.ListHistory(ctx, agentprofileports.ListProfileHistoryRequest{Ref: org})
	require.NoError(t, err)
	require.Len(t, history, 2)
	assert.Equal(t, last.ID, history[0].ID)
	assert.Equal(t, agentprofile.StatusCurrent, history[0].Status)
}

func TestGuidance_IsKeptPerProfileAndRetiredUnderItsVersion(t *testing.T) {
	ctx, f := newProfileFixture(t)
	org := agentprofileports.OrganizationSubject(f.tenant)
	person := agentprofileports.UserSubject(f.tenant, f.userID)

	add := func(ref agentprofileports.SubjectRef, content string) *agentprofile.Guidance {
		created, err := f.guidance.Create(ctx, &agentprofile.Guidance{
			OrganizationID: f.tenant.OrgID,
			BusinessUnitID: f.tenant.BuID,
			Subject:        ref.Subject,
			UserID:         ref.PersonPtr(),
			Content:        content,
			Status:         agentprofile.GuidanceStatusActive,
			Source:         agentprofile.GuidanceSourcePerson,
			CreatedByID:    f.userID,
		})
		require.NoError(t, err)

		return created
	}

	seasonal := add(org, "The Laredo lane is seasonal.")
	add(org, "We stopped running reefers in June.")
	add(person, "I only handle the Midwest accounts.")

	active, err := f.guidance.ListActive(ctx, org)
	require.NoError(t, err)
	require.Len(t, active, 2)
	assert.Equal(t, seasonal.ID, active[0].ID, "oldest first, so later lines read as refining")

	count, err := f.guidance.CountActive(ctx, person)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	seasonal.Retire(f.userID, timeutils.NowUnix())
	retired, err := f.guidance.Update(ctx, seasonal)
	require.NoError(t, err)
	assert.Equal(t, agentprofile.GuidanceStatusRetired, retired.Status)

	stale := *retired
	stale.Version--
	_, err = f.guidance.Update(ctx, &stale)
	require.Error(t, err, "a change under an old version is refused")

	listed, err := f.guidance.List(ctx, &agentprofileports.ListGuidanceRequest{
		Ref:            org,
		IncludeRetired: true,
	})
	require.NoError(t, err)
	require.Len(t, listed, 2)
	assert.Equal(t, agentprofile.GuidanceStatusActive, listed[0].Status, "active lines lead")
	require.NotNil(t, listed[0].CreatedBy, "who gave each line is read with it")

	_, err = f.guidance.GetByID(ctx, agentprofileports.GetGuidanceRequest{
		ID:         pulid.MustNew("aprg_"),
		TenantInfo: f.tenant,
	})
	assert.True(t, errortypes.IsNotFoundError(err))
}

func TestFacts_ReadEveryAggregateWithoutError(t *testing.T) {
	ctx, f := newProfileFixture(t)
	now := timeutils.NowUnix()
	window := agentprofileports.FactsWindow{
		TenantInfo: f.tenant,
		Start:      now - agentprofileports.OrganizationWindowDays*24*60*60,
		End:        now,
	}

	organization, err := f.facts.OrganizationActivity(ctx, window)
	require.NoError(t, err)
	require.NotNil(t, organization)
	assert.NotNil(t, organization.TopCustomers)
	assert.NotNil(t, organization.TopLanes)
	assert.NotNil(t, organization.Integrations)

	person, err := f.facts.UserActivity(ctx, &agentprofileports.UserFactsRequest{
		Window: window,
		UserID: f.userID,
	})
	require.NoError(t, err)
	require.NotNil(t, person)
	assert.False(t, person.Driver)

	users, err := f.facts.ActiveUsers(ctx, agentprofileports.ActiveUsersRequest{
		TenantInfo: f.tenant,
		Since:      window.Start,
	})
	require.NoError(t, err)
	assert.Empty(t, users, "nobody has talked to an agent in a fresh organization")
}

func TestPublish_TheSameVersionWrittenTwiceIsKeptOnce(t *testing.T) {
	ctx, f := newProfileFixture(t)
	org := agentprofileports.OrganizationSubject(f.tenant)

	entity := f.version(org, "Runs a regional fleet.")
	entity.ID = pulid.MustNew("aprf_")
	first, err := f.profiles.Publish(ctx, entity)
	require.NoError(t, err)

	again := f.version(org, "Runs a regional fleet.")
	again.ID = entity.ID
	second, err := f.profiles.Publish(ctx, again)
	require.NoError(t, err)
	assert.Equal(t, first.ID, second.ID)

	history, err := f.profiles.ListHistory(ctx, agentprofileports.ListProfileHistoryRequest{Ref: org})
	require.NoError(t, err)
	require.Len(t, history, 1)
	assert.Equal(t, agentprofile.StatusCurrent, history[0].Status)
}
