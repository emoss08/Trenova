//go:build integration

package supportlogrepository

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/identity/domain/supportlog"
	"github.com/emoss08/trenova/internal/modules/identity/ports/supportlogports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const logNow = int64(1_794_650_400)

type logRig struct {
	t      *testing.T
	tenant context.Context
	plain  context.Context
	data   *seedtest.TestData
	repo   supportlogports.Repository
}

func newLogRig(t *testing.T) *logRig {
	t.Helper()
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)
	data := seedtest.SeedFullTestData(t, ctx, db)

	return &logRig{
		t:      t,
		tenant: data.Ctx(ctx),
		plain:  ctx,
		data:   data,
		repo:   New(Params{DB: postgres.WrapDB(db), Logger: zap.NewNop()}),
	}
}

func (r *logRig) ask(period supportlog.Period, at int64) (*supportlog.LogRequest, error) {
	return r.repo.Create(r.tenant, supportlog.New(&supportlog.NewRequest{
		RequestedByID:  r.data.User.ID,
		RequesterName:  "Dana Dispatcher",
		RequesterEmail: "dana@acme.test",
		Period:         period,
		Now:            at,
	}))
}

func (r *logRig) report(req *supportlog.LogRequest, report *supportlog.Report, at int64) *supportlog.LogRequest {
	r.t.Helper()
	changed, err := req.Apply(report, at)
	require.NoError(r.t, err)
	require.True(r.t, changed)
	saved, err := r.repo.SaveReport(r.plain, req)
	require.NoError(r.t, err)

	return saved
}

func TestALogRequestBelongsToItsOrganizationAndTheOperatorReadsItWithoutOne(t *testing.T) {
	rig := newLogRig(t)

	asked, err := rig.ask(supportlog.PeriodDay, logNow)
	require.NoError(t, err)
	assert.Equal(t, rig.data.Organization.ID, asked.OrganizationID)
	assert.Equal(t, rig.data.BusinessUnit.ID, asked.BusinessUnitID)
	assert.Equal(t, supportlog.StatusRequested, asked.Status)
	assert.Equal(t, logNow-86_400, asked.Since)

	recent, err := rig.repo.Recent(rig.tenant, supportlog.RecentLimit)
	require.NoError(t, err)
	require.Len(t, recent, 1)
	assert.Equal(t, asked.ID, recent[0].ID)

	waiting, err := rig.repo.Waiting(rig.plain, supportlog.WaitingLimit)
	require.NoError(t, err)
	require.Len(t, waiting, 1, "the operator has no organization and still reads what waits")
	assert.Equal(t, "Dana Dispatcher", waiting[0].RequesterName)

	forOperator, err := rig.repo.ForOperator(rig.plain, asked.ID)
	require.NoError(t, err)
	assert.Equal(t, asked.ID, forOperator.ID)

	_, err = rig.repo.ForOperator(rig.plain, pulid.MustNew(supportlog.RequestIDPrefix))
	require.ErrorIs(t, err, supportlog.ErrRequestNotFound)
}

func TestOnlyOneLogRequestIsInFlightForTheWholeInstance(t *testing.T) {
	rig := newLogRig(t)

	first, err := rig.ask(supportlog.PeriodHour, logNow)
	require.NoError(t, err)

	_, err = rig.ask(supportlog.PeriodHour, logNow+60)
	require.Error(t, err, "a second request while one is in flight is refused by the database")
	assert.Contains(t, err.Error(), "already being sent")

	load, err := rig.repo.InstanceLoad(rig.plain, first.CreatedAt-1)
	require.NoError(t, err)
	assert.True(t, load.InFlight)
	assert.Equal(t, 1, load.Recent)

	collecting := rig.report(first, &supportlog.Report{Status: supportlog.StatusCollecting, BundleID: "bundle-1"}, logNow+120)
	assert.Equal(t, supportlog.StatusCollecting, collecting.Status)
	sent := rig.report(collecting, &supportlog.Report{Status: supportlog.StatusSent}, logNow+180)
	assert.Equal(t, supportlog.StatusSent, sent.Status)
	require.NotNil(t, sent.FinishedAt)

	load, err = rig.repo.InstanceLoad(rig.plain, first.CreatedAt)
	require.NoError(t, err)
	assert.False(t, load.InFlight, "a finished request frees the instance for the next one")
	assert.Zero(t, load.Recent, "only requests made after the given time count toward the day's limit")

	_, err = rig.ask(supportlog.PeriodHour, logNow+240)
	require.NoError(t, err)
}

func TestAReportOnARequestThatChangedMeanwhileIsRefused(t *testing.T) {
	rig := newLogRig(t)

	asked, err := rig.ask(supportlog.PeriodHour, logNow)
	require.NoError(t, err)
	stale := *asked

	rig.report(asked, &supportlog.Report{Status: supportlog.StatusCollecting, BundleID: "bundle-1"}, logNow+60)

	changed, err := stale.Apply(&supportlog.Report{Status: supportlog.StatusFailed, Message: "no disk"}, logNow+90)
	require.NoError(t, err)
	require.True(t, changed)
	_, err = rig.repo.SaveReport(rig.plain, &stale)
	require.ErrorIs(t, err, supportlog.ErrChangedMeanwhile)
}

func TestTheSweepFindsRequestsNeverPickedUpOrNeverFinished(t *testing.T) {
	rig := newLogRig(t)

	asked, err := rig.ask(supportlog.PeriodHour, logNow)
	require.NoError(t, err)

	overdue := func(requestedBefore, collectingBefore int64) []*supportlog.LogRequest {
		t.Helper()
		rows, listErr := rig.repo.Overdue(rig.plain, &supportlogports.OverdueRequest{
			RequestedBefore: requestedBefore, CollectingBefore: collectingBefore, Limit: 10,
		})
		require.NoError(t, listErr)

		return rows
	}

	made := asked.CreatedAt
	assert.Empty(t, overdue(made-1, made-1), "still within its time to be picked up")
	require.Len(t, overdue(made, made-1), 1, "never picked up")

	rig.report(asked, &supportlog.Report{Status: supportlog.StatusCollecting, BundleID: "bundle-1"}, made+60)
	assert.Empty(t, overdue(made+3_600, made+59), "collecting, and within its time to finish")
	require.Len(t, overdue(made-1, made+60), 1, "never finished")
}
