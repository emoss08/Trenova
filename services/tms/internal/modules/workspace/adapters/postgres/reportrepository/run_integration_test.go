//go:build integration

package reportrepository

import (
	"context"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/reportrows"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestCountActiveUnderConcurrentEnqueue smokes the enqueue-gate's counting
// query on live Postgres: concurrent run creation across two tenants must
// yield exact, tenant-isolated queued/running counts.
func TestCountActiveUnderConcurrentEnqueue(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()

	dataA := seedtest.SeedFullTestData(t, ctx, db)

	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	stateB := seedtest.NewState().WithName("Texas").WithAbbreviation("TX").Build(t, ctx, tx)
	buB := seedtest.NewBusinessUnit().WithName("Gate B BU").WithCode("GATEB").Build(t, ctx, tx)
	orgB := seedtest.NewOrganization(buB.ID, stateB.ID).
		WithName("Gate Org B").
		WithScacCode("GATB").
		WithBucketName("gate-bucket-b").
		Build(t, ctx, tx)
	userB := seedtest.NewUser(orgB.ID, buB.ID).
		WithUsername("gate_b_user").
		WithEmail("gate_b@example.com").
		WithPassword("password123").
		Build(t, ctx, tx)
	require.NoError(t, tx.Commit())

	repo := NewRunRepository(RunParams{
		DB:     postgres.NewTestConnection(db),
		Logger: zap.NewNop(),
	})

	tenantA := pagination.TenantInfo{
		OrgID:  dataA.Organization.ID,
		BuID:   dataA.BusinessUnit.ID,
		UserID: dataA.User.ID,
	}
	tenantB := pagination.TenantInfo{
		OrgID:  orgB.ID,
		BuID:   buB.ID,
		UserID: userB.ID,
	}

	const (
		queuedPerOrg  = 10
		runningPerOrg = 2
	)

	createRuns := func(tenant pagination.TenantInfo, status report.RunStatus, count int) {
		var wg sync.WaitGroup
		errs := make(chan error, count)
		for range count {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, createErr := repo.Create(ctx, &report.ReportRun{
					BusinessUnitID: tenant.BuID,
					OrganizationID: tenant.OrgID,
					RequestedByID:  tenant.UserID,
					CannedKey:      "shipment-volume-by-status",
					CannedVersion:  "1.0.0",
					Trigger:        report.RunTriggerManual,
					Format:         report.FormatCSV,
					Status:         status,
				})
				errs <- createErr
			}()
		}
		wg.Wait()
		close(errs)
		for createErr := range errs {
			require.NoError(t, createErr)
		}
	}

	var seedWG sync.WaitGroup
	for _, tenant := range []pagination.TenantInfo{tenantA, tenantB} {
		seedWG.Add(1)
		go func() {
			defer seedWG.Done()
			createRuns(tenant, report.RunStatusQueued, queuedPerOrg)
			createRuns(tenant, report.RunStatusRunning, runningPerOrg)
		}()
	}
	seedWG.Wait()

	for _, tenant := range []pagination.TenantInfo{tenantA, tenantB} {
		counts, countErr := repo.CountActive(ctx, &reportports.CountActiveReportRunsRequest{
			TenantInfo: tenant,
		})
		require.NoError(t, countErr)
		assert.Equal(t, queuedPerOrg, counts.Queued)
		assert.Equal(t, runningPerOrg, counts.Running)
	}
}

// TestCountActiveIgnoresTerminalRuns asserts the gate only counts queued and
// running runs — terminal states never consume enqueue capacity.
func TestCountActiveIgnoresTerminalRuns(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()

	data := seedtest.SeedFullTestData(t, ctx, db)
	tenant := pagination.TenantInfo{
		OrgID:  data.Organization.ID,
		BuID:   data.BusinessUnit.ID,
		UserID: data.User.ID,
	}

	repo := NewRunRepository(RunParams{
		DB:     postgres.NewTestConnection(db),
		Logger: zap.NewNop(),
	})

	statuses := []report.RunStatus{
		report.RunStatusQueued,
		report.RunStatusRunning,
		report.RunStatusSucceeded,
		report.RunStatusFailed,
		report.RunStatusCanceled,
		report.RunStatusExpired,
	}
	for _, status := range statuses {
		_, err := repo.Create(context.Background(), &report.ReportRun{
			BusinessUnitID: tenant.BuID,
			OrganizationID: tenant.OrgID,
			RequestedByID:  tenant.UserID,
			CannedKey:      "shipment-volume-by-status",
			CannedVersion:  "1.0.0",
			Trigger:        report.RunTriggerManual,
			Format:         report.FormatCSV,
			Status:         status,
		})
		require.NoError(t, err)
	}

	counts, err := repo.CountActive(ctx, &reportports.CountActiveReportRunsRequest{
		TenantInfo: tenant,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, counts.Queued)
	assert.Equal(t, 1, counts.Running)
}

func TestFinishedRunKeepsItsSampleOutOfLists(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()

	data := seedtest.SeedFullTestData(t, ctx, db)
	tenant := pagination.TenantInfo{
		OrgID:  data.Organization.ID,
		BuID:   data.BusinessUnit.ID,
		UserID: data.User.ID,
	}
	repo := NewRunRepository(RunParams{
		DB:     postgres.NewTestConnection(db),
		Logger: zap.NewNop(),
	})

	created, err := repo.Create(ctx, finishedRun(tenant, "revenue-by-customer", nil, 100))
	require.NoError(t, err)

	got, err := repo.GetByID(ctx, &reportports.GetReportRunRequest{
		TenantInfo: tenant,
		RunID:      created.ID,
	})
	require.NoError(t, err)
	require.NotNil(t, got.Sample)
	assert.Equal(t, "customer", got.Sample.Schema[0].ID)
	assert.Equal(t, []any{"ACME", "1180.5"}, got.Sample.Rows[0])
	assert.Equal(t, []any{nil, "1180.5"}, got.Sample.Totals)
	assert.True(t, got.Sample.More)

	listed, err := repo.List(ctx, &reportports.ListReportRunsRequest{TenantInfo: tenant})
	require.NoError(t, err)
	require.Len(t, listed, 1)
	assert.Nil(t, listed[0].Sample)
}

func TestLatestComparableFindsTheRequestersPreviousRunOfTheSameReportAndParameters(
	t *testing.T,
) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()

	data := seedtest.SeedFullTestData(t, ctx, db)
	tenant := pagination.TenantInfo{
		OrgID:  data.Organization.ID,
		BuID:   data.BusinessUnit.ID,
		UserID: data.User.ID,
	}

	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	colleague := seedtest.NewUser(data.Organization.ID, data.BusinessUnit.ID).
		WithUsername("comparable_colleague").
		WithEmail("comparable_colleague@example.com").
		WithPassword("password123").
		Build(t, ctx, tx)
	require.NoError(t, tx.Commit())
	colleagueTenant := tenant
	colleagueTenant.UserID = colleague.ID

	repo := NewRunRepository(RunParams{
		DB:     postgres.NewTestConnection(db),
		Logger: zap.NewNop(),
	})
	create := func(run *report.ReportRun) *report.ReportRun {
		created, createErr := repo.Create(ctx, run)
		require.NoError(t, createErr)
		return created
	}

	window := map[string]any{"window": float64(30)}
	older := create(finishedRun(tenant, "revenue-by-customer", window, 100))
	previous := create(finishedRun(tenant, "revenue-by-customer", window, 200))
	create(finishedRun(tenant, "revenue-by-customer", map[string]any{"window": float64(7)}, 250))
	create(finishedRun(tenant, "shipment-volume-by-status", window, 260))
	create(finishedRun(colleagueTenant, "revenue-by-customer", window, 270))
	withoutSample := finishedRun(tenant, "revenue-by-customer", window, 280)
	withoutSample.Sample = nil
	create(withoutSample)
	failed := finishedRun(tenant, "revenue-by-customer", window, 290)
	failed.Status = report.RunStatusFailed
	create(failed)
	current := create(finishedRun(tenant, "revenue-by-customer", window, 300))
	create(finishedRun(tenant, "revenue-by-customer", window, 400))

	got, err := repo.LatestComparable(ctx, &reportports.GetComparableReportRunRequest{
		TenantInfo:      tenant,
		RequestedBy:     tenant.UserID,
		CannedKey:       current.CannedKey,
		Params:          current.Params,
		ExcludeRunID:    current.ID,
		CompletedBefore: current.CompletedAt,
	})
	require.NoError(t, err)
	assert.Equal(t, previous.ID, got.ID)
	assert.Equal(t, int64(200), got.CompletedAt)
	require.NotNil(t, got.Sample)

	_, err = repo.LatestComparable(ctx, &reportports.GetComparableReportRunRequest{
		TenantInfo:      tenant,
		RequestedBy:     tenant.UserID,
		CannedKey:       older.CannedKey,
		Params:          older.Params,
		ExcludeRunID:    older.ID,
		CompletedBefore: older.CompletedAt,
	})
	require.Error(t, err)
}

func TestLatestComparableTreatsMissingAndEmptyParametersAlike(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()

	data := seedtest.SeedFullTestData(t, ctx, db)
	tenant := pagination.TenantInfo{
		OrgID:  data.Organization.ID,
		BuID:   data.BusinessUnit.ID,
		UserID: data.User.ID,
	}
	repo := NewRunRepository(RunParams{
		DB:     postgres.NewTestConnection(db),
		Logger: zap.NewNop(),
	})

	previous, err := repo.Create(ctx, finishedRun(tenant, "revenue-by-customer", nil, 100))
	require.NoError(t, err)

	got, err := repo.LatestComparable(ctx, &reportports.GetComparableReportRunRequest{
		TenantInfo:      tenant,
		RequestedBy:     tenant.UserID,
		CannedKey:       "revenue-by-customer",
		Params:          map[string]any{},
		ExcludeRunID:    pulid.MustNew("rrun_"),
		CompletedBefore: 200,
	})
	require.NoError(t, err)
	assert.Equal(t, previous.ID, got.ID)
}

func finishedRun(
	tenant pagination.TenantInfo,
	cannedKey string,
	params map[string]any,
	completedAt int64,
) *report.ReportRun {
	return &report.ReportRun{
		BusinessUnitID: tenant.BuID,
		OrganizationID: tenant.OrgID,
		RequestedByID:  tenant.UserID,
		CannedKey:      cannedKey,
		CannedVersion:  "1.0.0",
		Trigger:        report.RunTriggerManual,
		Format:         report.FormatXLSX,
		Status:         report.RunStatusSucceeded,
		Params:         params,
		CompletedAt:    completedAt,
		Sample: &reportrows.Sample{
			Schema: []reportrows.Column{
				{ID: "customer", Label: "Customer"},
				{ID: "revenue", Label: "Revenue"},
			},
			Rows:   [][]any{{"ACME", "1180.5"}},
			Totals: []any{nil, "1180.5"},
			More:   true,
		},
	}
}
