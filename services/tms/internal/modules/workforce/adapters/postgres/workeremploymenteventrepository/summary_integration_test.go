//go:build integration

package workeremploymenteventrepository_test

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/workeremploymenteventrepository"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/employment"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/employment/employmentdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workeremploymentports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestSummarize_CountsEveryEventAndFindsTheLatestHire(t *testing.T) {
	ctx, admin, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, admin)
	tenant := data.Info()
	newWorker := func(name string) *worker.Worker {
		wrk := &worker.Worker{
			ID:             pulid.MustNew("wrk_"),
			BusinessUnitID: tenant.BuID,
			OrganizationID: tenant.OrgID,
			StateID:        data.State.ID,
			Status:         domaintypes.StatusActive,
			Type:           worker.WorkerTypeEmployee,
			DriverType:     worker.DriverTypeOTR,
			FirstName:      name,
			LastName:       "Driver",
			AddressLine1:   "1 Depot Road",
			City:           "Springfield",
			PostalCode:     "62701",
			Email:          strings.ToLower(name) + "-" + strings.ToLower(pulid.MustNew("e").String()) + "@example.com",
			PhoneNumber:    "555-0100",
			Gender:         worker.GenderMale,
			CanBeAssigned:  true,
		}
		require.NoError(t, workerdb.Workers.Create(data.Ctx(ctx), admin, wrk))
		return wrk
	}
	driver := newWorker("Summary")
	other := newWorker("Other")
	record := func(wrk *worker.Worker, kind employment.EmploymentEventKind, at int64) {
		require.NoError(t, employmentdb.WorkerEmploymentEvents.Create(
			data.Ctx(ctx),
			admin,
			&employment.WorkerEmploymentEvent{
				OrganizationID: tenant.OrgID,
				BusinessUnitID: tenant.BuID,
				WorkerID:       wrk.ID,
				Kind:           kind,
				EffectiveAt:    at,
				FromValues:     map[string]string{},
				ToValues:       map[string]string{},
			},
		))
	}
	record(driver, employment.EmploymentEventHired, 1_000)
	record(driver, employment.EmploymentEventTerminated, 2_000)
	record(driver, employment.EmploymentEventRehired, 4_000)
	record(driver, employment.EmploymentEventTransferred, 5_000)
	record(other, employment.EmploymentEventHired, 9_000)

	repo := workeremploymenteventrepository.New(workeremploymenteventrepository.Params{
		DB:     postgres.WrapDB(admin),
		Logger: zap.NewNop(),
	})

	summary, err := repo.Summarize(
		data.Ctx(ctx),
		&workeremploymentports.SummarizeWorkerEmploymentEventsRequest{
			TenantInfo: tenant,
			WorkerID:   driver.ID,
		},
	)
	require.NoError(t, err)
	assert.Equal(t, 4, summary.EventCount)
	require.NotNil(t, summary.LastHiredAt)
	assert.Equal(t, int64(4_000), *summary.LastHiredAt)

	empty, err := repo.Summarize(
		data.Ctx(ctx),
		&workeremploymentports.SummarizeWorkerEmploymentEventsRequest{
			TenantInfo: tenant,
			WorkerID:   pulid.MustNew("wrk_"),
		},
	)
	require.NoError(t, err)
	assert.Zero(t, empty.EventCount)
	assert.Nil(t, empty.LastHiredAt)
}
