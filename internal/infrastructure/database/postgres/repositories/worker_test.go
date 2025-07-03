package repositories_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/stretchr/testify/require"

	repoports "github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/database/postgres/repositories"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/emoss08/trenova/test/testutils"
)

func TestWorkerRepository(t *testing.T) {
	org := ts.Fixture.MustRow("Organization.trenova").(*organization.Organization)
	bu := ts.Fixture.MustRow("BusinessUnit.trenova").(*businessunit.BusinessUnit)
	wrk := ts.Fixture.MustRow("Worker.worker_1").(*worker.Worker)
	usState := ts.Fixture.MustRow("UsState.ca").(*usstate.UsState)

	repo := repositories.NewWorkerRepository(repositories.WorkerRepositoryParams{
		Logger: logger.NewLogger(testutils.NewTestConfig()),
		DB:     ts.DB,
	})

	t.Run("list workers", func(t *testing.T) {
		opts := &repoports.ListWorkerRequest{
			Filter: &ports.QueryOptions{
				Limit:  10,
				Offset: 0,
				TenantOpts: &ports.TenantOptions{
					OrgID: org.ID,
					BuID:  bu.ID,
				},
			},
		}

		testutils.TestRepoList(ctx, t, repo, opts)
	})

	t.Run("list workers with query", func(t *testing.T) {
		opts := &repoports.ListWorkerRequest{
			Filter: &ports.QueryOptions{
				Limit:  10,
				Offset: 0,
				Query:  "John",
				TenantOpts: &ports.TenantOptions{
					OrgID: org.ID,
					BuID:  bu.ID,
				},
			},
		}

		testutils.TestRepoList(ctx, t, repo, opts)
	})

	t.Run("list workers with filter id", func(t *testing.T) {
		opts := &repoports.ListWorkerRequest{
			Filter: &ports.QueryOptions{
				Limit:  10,
				Offset: 0,
				ID:     pulid.Must(wrk.ID.String()),
				TenantOpts: &ports.TenantOptions{
					OrgID: org.ID,
					BuID:  bu.ID,
				},
			},
		}

		testutils.TestRepoList(ctx, t, repo, opts)
	})

	t.Run("list workers with profiles", func(t *testing.T) {
		opts := &repoports.ListWorkerRequest{
			Filter: &ports.QueryOptions{
				Limit:  10,
				Offset: 0,
				TenantOpts: &ports.TenantOptions{
					OrgID: org.ID,
					BuID:  bu.ID,
				},
			},
			WorkerFilterOptions: repoports.WorkerFilterOptions{
				IncludeProfile: true,
			},
		}

		result, err := repo.List(ctx, opts)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotEmpty(t, result.Items)

		// Check if any worker has a profile
		hasWorkerWithProfile := false
		for _, worker := range result.Items {
			if worker.Profile != nil {
				hasWorkerWithProfile = true
				break
			}
		}
		require.True(t, hasWorkerWithProfile, "At least one worker should have a profile")
	})

	t.Run("list workers with pto", func(t *testing.T) {
		opts := &repoports.ListWorkerRequest{
			Filter: &ports.QueryOptions{
				Limit:  10,
				Offset: 0,
				TenantOpts: &ports.TenantOptions{
					OrgID: org.ID,
					BuID:  bu.ID,
				},
			},
			WorkerFilterOptions: repoports.WorkerFilterOptions{
				IncludePTO: true,
			},
		}

		result, err := repo.List(ctx, opts)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotEmpty(t, result.Items)
		// Check if any worker has PTO - worker_1 has PTO in fixtures
		hasWorkerWithPTO := false
		for _, worker := range result.Items {
			if len(worker.PTO) > 0 {
				hasWorkerWithPTO = true
				break
			}
		}
		require.True(t, hasWorkerWithPTO, "At least one worker should have PTO")
	})

	t.Run("get by id", func(t *testing.T) {
		testutils.TestRepoGetByID(ctx, t, repo, &repoports.GetWorkerByIDRequest{
			WorkerID: wrk.ID,
			OrgID:    org.ID,
			BuID:     bu.ID,
		})
	})

	t.Run("get with invalid id", func(t *testing.T) {
		entity, err := repo.GetByID(ctx, &repoports.GetWorkerByIDRequest{
			WorkerID: "invalid-id",
			OrgID:    org.ID,
			BuID:     bu.ID,
		})

		require.Error(t, err, "entity not found")
		require.Nil(t, entity)
	})

	t.Run("create", func(t *testing.T) {
		newEntity := &worker.Worker{
			FirstName:      "John",
			LastName:       "Doe",
			Gender:         "Male",
			AddressLine1:   "123 Main St",
			City:           "Los Angeles",
			StateID:        usState.ID,
			PostalCode:     "90001",
			BusinessUnitID: bu.ID,
			OrganizationID: org.ID,
			Profile: &worker.WorkerProfile{
				DOB:             timeutils.NowUnix(),
				LicenseNumber:   "1234567890",
				LicenseStateID:  usState.ID,
				BusinessUnitID:  bu.ID,
				OrganizationID:  org.ID,
				HazmatExpiry:    timeutils.NowUnix(),
				LicenseExpiry:   timeutils.NowUnix(),
				HireDate:        timeutils.NowUnix(),
				MVRDueDate:      timeutils.NowUnixPointer(),
				PhysicalDueDate: timeutils.NowUnixPointer(),
			},
		}

		testutils.TestRepoCreate(ctx, t, repo, newEntity)
	})

	t.Run("update worker", func(t *testing.T) {
		wrk.FirstName = "Jane"

		result, err := repo.Update(ctx, wrk)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, "Jane", result.FirstName)
	})

	t.Run("update worker with invalid id", func(t *testing.T) {
		wrk.ID = "invalid-id"

		result, err := repo.Update(ctx, wrk)
		require.Error(t, err)
		require.Nil(t, result)
	})
}
