package auditoverview

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var tenant = pagination.TenantInfo{OrgID: pulid.ID("org_1"), BuID: pulid.ID("bu_1")}

type fakeRepo struct {
	zone   string
	rollup *auditports.OverviewRollup
	reqs   []*auditports.OverviewRequest
}

func (f *fakeRepo) OrganizationTimezone(context.Context, pagination.TenantInfo) (string, error) {
	return f.zone, nil
}

func (f *fakeRepo) Overview(
	_ context.Context,
	req *auditports.OverviewRequest,
) (*auditports.OverviewRollup, error) {
	f.reqs = append(f.reqs, req)
	if f.rollup == nil {
		return &auditports.OverviewRollup{}, nil
	}
	return f.rollup, nil
}

var chicago = mustZone("America/Chicago")

func mustZone(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

// 3:45 PM on Saturday, November 7, 2026 in Chicago.
var now = time.Date(2026, 11, 7, 15, 45, 0, 0, chicago).Unix()

func newService(repo *fakeRepo) *Service {
	return New(Params{Repo: repo, Now: func() int64 { return now }})
}

func allAccess() auditports.EntryAccess {
	return auditports.EntryAccess{AllResources: true, IncludeSecurity: true}
}

func TestTheWindowIsWholeDaysOfTheOrganizationsZoneEndingToday(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{zone: "America/Chicago"}
	result, err := newService(
		repo,
	).Build(t.Context(), &Request{TenantInfo: tenant, Days: 7, Access: allAccess()})
	require.NoError(t, err)

	require.Len(t, repo.reqs, 1)
	req := repo.reqs[0]
	assert.Equal(t, time.Date(2026, 11, 1, 0, 0, 0, 0, chicago).Unix(), req.Current.From)
	assert.Equal(t, time.Date(2026, 11, 8, 0, 0, 0, 0, chicago).Unix()-1, req.Current.To)
	assert.Equal(t, time.Date(2026, 10, 25, 0, 0, 0, 0, chicago).Unix(), req.Previous.From)
	assert.Equal(t, req.Current.From-1, req.Previous.To)
	assert.True(t, req.Hourly)
	assert.Equal(t, UnitHour, result.Unit)
	assert.Equal(t, "America/Chicago", result.Timezone)
}

func TestAMonthOrMoreReadsDays(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{zone: "America/Chicago"}
	result, err := newService(
		repo,
	).Build(t.Context(), &Request{TenantInfo: tenant, Days: 30, Access: allAccess()})
	require.NoError(t, err)

	assert.False(t, repo.reqs[0].Hourly)
	assert.Equal(t, UnitDay, result.Unit)
	assert.Len(t, result.Activity, 30)
}

func TestAWindowOutsideADayToNinetyDaysIsRefused(t *testing.T) {
	t.Parallel()

	for _, days := range []int{0, 91} {
		_, err := newService(
			&fakeRepo{},
		).Build(t.Context(), &Request{TenantInfo: tenant, Days: days, Access: allAccess()})
		var validation *errortypes.Error
		require.ErrorAs(t, err, &validation, "days %d", days)
	}
}

func TestEveryHourUpToNowIsABucketEvenAQuietOne(t *testing.T) {
	t.Parallel()

	threePm := time.Date(2026, 11, 7, 15, 0, 0, 0, chicago)
	repo := &fakeRepo{zone: "America/Chicago", rollup: &auditports.OverviewRollup{
		Activity: []auditports.ActivityBucket{{At: threePm, People: 4, Agents: 1}},
	}}
	result, err := newService(
		repo,
	).Build(t.Context(), &Request{TenantInfo: tenant, Days: 1, Access: allAccess()})
	require.NoError(t, err)

	require.Len(t, result.Activity, 16, "midnight through the 3 PM hour, and no hour after now")
	assert.Equal(
		t,
		time.Date(2026, 11, 7, 0, 0, 0, 0, chicago).Unix(),
		result.Activity[0].At.Unix(),
	)
	assert.Zero(t, result.Activity[0].People)
	last := result.Activity[15]
	assert.Equal(t, threePm.Unix(), last.At.Unix())
	assert.Equal(t, int64(4), last.People)
	assert.Equal(t, int64(1), last.Agents)
}

func TestSomeoneWhoCanReadNoKindOfRecordGetsAnEmptyOverviewWithoutARead(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{zone: "America/Chicago"}
	result, err := newService(repo).Build(t.Context(), &Request{
		TenantInfo: tenant, Days: 7, Access: auditports.EntryAccess{Resources: []permission.Resource{}},
	})
	require.NoError(t, err)

	assert.Empty(t, repo.reqs)
	assert.Zero(t, result.Totals.Entries)
	assert.NotEmpty(t, result.Activity)
	assert.Equal(
		t,
		"Nothing was written to the audit log in the last 7 days.",
		text(result.Summary),
	)
}

func TestTheOthersAreEveryoneOutsideTheTopActors(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{zone: "UTC", rollup: &auditports.OverviewRollup{
		Totals: auditports.OverviewFigures{Entries: 100, People: 100},
		Actors: auditports.ActorCounts{People: 7},
		TopActors: []auditports.ActorStat{
			{PrincipalID: "usr_1", Entries: 60},
			{PrincipalID: "usr_2", Entries: 25},
		},
	}}
	result, err := newService(
		repo,
	).Build(t.Context(), &Request{TenantInfo: tenant, Days: 7, Access: allAccess()})
	require.NoError(t, err)

	assert.Equal(t, 5, result.OtherActors)
	assert.Equal(t, int64(15), result.OtherEntries)
}

func TestTheSummaryNamesTheVolumeTheChangeTheReviewsAndTheAgents(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{zone: "UTC", rollup: &auditports.OverviewRollup{
		Totals: auditports.OverviewFigures{
			Entries: 1000, People: 700, AgentsAsked: 150, AgentsOwn: 50, Integrations: 100,
			PeopleOffHours: 140,
		},
		PreviousEntries: 800,
		Actors:          auditports.ActorCounts{People: 26, Agents: 3, Integrations: 2},
		OpenReviews: []*audit.Review{
			{
				ID:            "arv_1",
				Kind:          audit.AlertBulkDelete,
				ActorName:     "Dana Whitfield",
				Count:         50,
				WindowMinutes: 10,
			},
		},
		OpenReviewCount: 1,
	}}
	result, err := newService(repo).Build(t.Context(), &Request{
		TenantInfo: tenant, Days: 7, Access: allAccess(), IncludeReviews: true,
	})
	require.NoError(t, err)

	assert.Equal(t,
		"26 people, 3 agents and 2 API keys made 1,000 changes in the last 7 days. "+
			"That is 25% more than the 7 days before. "+
			"One thing needs a review: Dana Whitfield deleted 50 records within 10 minutes. "+
			"Agents made 20% of the changes, 75% of them asked for in a conversation. "+
			"20% of people's changes were made between 8 PM and 6 AM.",
		text(result.Summary),
	)

	var review, agents *Segment
	for i := range result.Summary {
		switch result.Summary[i].Target {
		case TargetReview:
			review = &result.Summary[i]
		case TargetView:
			agents = &result.Summary[i]
		}
	}
	require.NotNil(t, review)
	assert.Equal(t, "arv_1", review.TargetID)
	require.NotNil(t, agents)
	assert.Equal(t, ViewAgents, agents.TargetID)
}

func TestWithoutSecurityAccessTheSummaryNeverMentionsReviews(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{zone: "UTC", rollup: &auditports.OverviewRollup{
		Totals:          auditports.OverviewFigures{Entries: 10, People: 10},
		Actors:          auditports.ActorCounts{People: 2},
		OpenReviewCount: 4,
		OpenReviews:     []*audit.Review{{ID: "arv_1", Kind: audit.AlertRoleChanged}},
	}}
	result, err := newService(
		repo,
	).Build(t.Context(), &Request{TenantInfo: tenant, Days: 7, Access: allAccess()})
	require.NoError(t, err)

	assert.False(t, repo.reqs[0].IncludeReviews)
	assert.NotContains(t, text(result.Summary), "review")
}

func text(segments []Segment) string {
	var b strings.Builder
	for _, segment := range segments {
		b.WriteString(segment.Text)
	}
	return b.String()
}
