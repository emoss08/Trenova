//go:build integration

package auditoverviewrepository_test

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/analytics/rollup"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/adapters/postgres/auditoverviewrepository"
	"github.com/emoss08/trenova/internal/modules/workspace/adapters/postgres/auditrollup"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var chicago = func() *time.Location {
	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		panic(err)
	}
	return loc
}()

func at(day, hour, minute int) int64 {
	return time.Date(2026, time.November, day, hour, minute, 0, 0, chicago).Unix()
}

type entry struct {
	tenant        pagination.TenantInfo
	timestamp     int64
	resource      string
	operation     string
	category      string
	channel       string
	principalType string
	principalID   string
	actorName     string
}

func insert(ctx context.Context, t *testing.T, db *keel.DB, e entry) {
	t.Helper()
	_, err := keel.Exec(dbscope.WithSystem(ctx, "test"), db, `
		INSERT INTO audit_entries (
			id, business_unit_id, organization_id, "timestamp", resource, resource_id, operation,
			action, category, retention_class, channel, principal_type, principal_id, actor_name
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'Operational', $10, $11, $12, $13)`,
		pulid.MustNew("ae_").String(),
		e.tenant.BuID.String(),
		e.tenant.OrgID.String(),
		e.timestamp,
		e.resource,
		pulid.MustNew("rec_").String(),
		e.operation,
		e.resource+"."+e.operation,
		e.category,
		e.channel,
		e.principalType,
		e.principalID,
		e.actorName,
	)
	require.NoError(t, err)
}

type fixture struct {
	ctx      context.Context
	repo     auditports.OverviewRepository
	tenant   pagination.TenantInfo
	other    pagination.TenantInfo
	otherCtx context.Context
}

// writeTheDay writes Thursday, November 5, 2026 in Chicago for one tenant, one entry the day
// before, and one entry for another tenant the same day, then refreshes the rollups.
func writeTheDay(t *testing.T) *fixture {
	t.Helper()
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	other := seedtest.SeedAdditionalTenant(t, ctx, db, "AO")
	tenant := data.Info()
	system := dbscope.WithSystem(ctx, "test")

	_, err := keel.Exec(system, db, `SELECT audit_partitions.ensure_partitions($1, 1)`, at(1, 0, 0))
	require.NoError(t, err)
	for _, info := range []pagination.TenantInfo{tenant, other.Info()} {
		_, err = keel.Exec(
			system,
			db,
			`UPDATE organizations SET timezone = 'America/Chicago' WHERE id = $1 AND business_unit_id = $2`,
			info.OrgID.String(),
			info.BuID.String(),
		)
		require.NoError(t, err)
	}

	priya := entry{tenant: tenant, resource: "shipment", operation: "update", category: "User",
		channel: "web", principalType: "system", principalID: "sys_priya", actorName: "Priya Raman"}
	dana := priya
	dana.principalID, dana.actorName = "sys_dana", "Dana Whitfield"
	billing := priya
	billing.channel, billing.principalType, billing.principalID, billing.actorName =
		"agent_run", "agent", "agd_billing", "Billing agent"

	e := priya
	e.timestamp = at(5, 9, 0)
	insert(ctx, t, db, e)
	e.timestamp = at(5, 9, 30)
	insert(ctx, t, db, e)
	e.timestamp, e.channel = at(5, 10, 15), "agent_conversation"
	insert(ctx, t, db, e)

	e = billing
	e.timestamp, e.resource = at(5, 10, 0), "invoice"
	insert(ctx, t, db, e)

	e = dana
	e.timestamp, e.resource, e.operation = at(5, 22, 0), "customer", "delete"
	insert(ctx, t, db, e)
	e.timestamp, e.resource, e.operation, e.category = at(5, 11, 0), "role", "update", "Security"
	insert(ctx, t, db, e)

	e = priya
	e.timestamp = at(4, 12, 0)
	insert(ctx, t, db, e)

	e = priya
	e.tenant, e.timestamp = other.Info(), at(5, 9, 0)
	insert(ctx, t, db, e)

	_, err = rollup.Refresh(system, db, auditrollup.All()...)
	require.NoError(t, err)

	return &fixture{
		ctx: dbscope.WithTenant(ctx, tenant.DBTenant()),
		repo: auditoverviewrepository.New(
			auditoverviewrepository.Params{DB: postgres.WrapDB(db), Logger: zap.NewNop()},
		),
		tenant:   tenant,
		other:    other.Info(),
		otherCtx: other.Ctx(ctx),
	}
}

func thursday(
	info pagination.TenantInfo,
	access auditports.EntryAccess,
) *auditports.OverviewRequest {
	return &auditports.OverviewRequest{
		TenantInfo: info,
		Access:     access,
		Current:    auditports.OverviewWindow{From: at(5, 0, 0), To: at(6, 0, 0) - 1},
		Previous:   auditports.OverviewWindow{From: at(4, 0, 0), To: at(5, 0, 0) - 1},
		Hourly:     true,
		ActorLimit: 5,
	}
}

var everything = auditports.EntryAccess{AllResources: true, IncludeSecurity: true}

func TestOverview_CountsTheDayByWhoWroteItAndWhatItWas(t *testing.T) {
	f := writeTheDay(t)

	got, err := f.repo.Overview(f.ctx, thursday(f.tenant, everything))
	require.NoError(t, err)

	assert.Equal(t, auditports.OverviewFigures{
		Entries:        6,
		People:         4,
		AgentsAsked:    1,
		AgentsOwn:      1,
		Sensitive:      2,
		Security:       1,
		DeletesExports: 1,
		PeopleOffHours: 1,
	}, got.Totals)
	assert.Equal(t, int64(1), got.PreviousEntries)

	byHour := map[int]auditports.ActivityBucket{}
	for _, bucket := range got.Activity {
		byHour[bucket.At.In(chicago).Hour()] = bucket
	}
	assert.Equal(t, int64(2), byHour[9].People, "hours are the organization's")
	assert.Equal(t, int64(2), byHour[10].Agents)
	assert.Equal(t, int64(1), byHour[22].People)

	assert.Equal(t, auditports.ActorCounts{Agents: 1, Jobs: 2}, got.Actors)
	require.Len(t, got.TopActors, 3)
	assert.Equal(t, auditports.ActorStat{
		PrincipalID: "sys_priya", PrincipalType: "system", Name: "Priya Raman", Entries: 3,
	}, got.TopActors[0])
	assert.Equal(t, "Dana Whitfield", got.TopActors[1].Name)
	assert.Equal(t, int64(2), got.TopActors[1].Sensitive)
}

func TestOverview_LeavesOutWhatTheReaderCannotRead(t *testing.T) {
	f := writeTheDay(t)

	got, err := f.repo.Overview(f.ctx, thursday(f.tenant, auditports.EntryAccess{
		Resources: []permission.Resource{"shipment", "role"},
	}))
	require.NoError(t, err)

	assert.Equal(
		t,
		int64(3),
		got.Totals.Entries,
		"the shipments; the role entry is a security entry",
	)
	assert.Zero(t, got.Totals.Security)
	require.Len(t, got.TopActors, 1)
	assert.Equal(t, "Priya Raman", got.TopActors[0].Name)
}

func TestOverview_AnotherTenantSeesOnlyItsOwn(t *testing.T) {
	f := writeTheDay(t)

	got, err := f.repo.Overview(f.otherCtx, thursday(f.other, everything))
	require.NoError(t, err)

	assert.Equal(t, int64(1), got.Totals.Entries)
	assert.Zero(t, got.PreviousEntries)
}

func TestOrganizationTimezone_ReadsTheOrganizationsZone(t *testing.T) {
	f := writeTheDay(t)

	zone, err := f.repo.OrganizationTimezone(f.ctx, f.tenant)
	require.NoError(t, err)
	assert.Equal(t, "America/Chicago", zone)
}
