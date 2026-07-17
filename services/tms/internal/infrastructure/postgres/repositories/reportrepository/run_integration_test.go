//go:build integration

package reportrepository

import (
	"context"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
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
		counts, countErr := repo.CountActive(ctx, &repositories.CountActiveReportRunsRequest{
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

	counts, err := repo.CountActive(ctx, &repositories.CountActiveReportRunsRequest{
		TenantInfo: tenant,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, counts.Queued)
	assert.Equal(t, 1, counts.Running)
}
