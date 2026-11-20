//go:build integration

package driverpayrepository

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay/driverpaydb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/payassignment"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/payassignment/payassignmentdb"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driverpayports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func seedWorker(
	t *testing.T,
	ctx context.Context,
	db *keel.DB,
	tenant *seedtest.TestData,
	firstName string,
) *worker.Worker {
	t.Helper()

	entity := &worker.Worker{
		OrganizationID: tenant.Organization.ID,
		BusinessUnitID: tenant.BusinessUnit.ID,
		StateID:        tenant.State.ID,
		Status:         domaintypes.StatusActive,
		Type:           worker.WorkerTypeEmployee,
		DriverType:     worker.DriverTypeOTR,
		FirstName:      firstName,
		LastName:       "Driver",
		AddressLine1:   "1 Main St",
		City:           "Springfield",
		PostalCode:     "62701",
		Gender:         worker.GenderFemale,
	}
	require.NoError(t, workerdb.Workers.Create(tenant.Ctx(ctx), db, entity))

	return entity
}

func seedAssignment(
	t *testing.T,
	ctx context.Context,
	db *keel.DB,
	tenant *seedtest.TestData,
	entity *payassignment.WorkerPayAssignment,
) *payassignment.WorkerPayAssignment {
	t.Helper()

	require.NoError(t, payassignmentdb.WorkerPayAssignments.Create(tenant.Ctx(ctx), db, entity))

	return entity
}

func TestGetEffectiveForWorkersReadsEachWorkersCurrentAssignment(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	tenantA := seedtest.SeedFullTestData(t, ctx, db)
	tenantB := seedtest.SeedAdditionalTenant(t, ctx, db, "PA")
	scoped := tenantA.Ctx(ctx)

	profile := seedPayProfile(t, ctx, db, tenantA, "Linehaul")
	older := seedPayProfile(t, ctx, db, tenantA, "Older")
	require.NoError(t, driverpaydb.PayProfileComponents.CreateMany(scoped, db,
		[]*driverpay.PayProfileComponent{
			{
				PayProfileID: profile.ID,
				Kind:         driverpay.ComponentKindDetention,
				Method:       driverpay.CalcMethodPerHour,
				Rate:         decimal.NewFromInt(30),
				Sequence:     2,
				IsActive:     true,
			},
			{
				PayProfileID: profile.ID,
				Kind:         driverpay.ComponentKindStopPay,
				Method:       driverpay.CalcMethodPerStop,
				Rate:         decimal.NewFromInt(25),
				Sequence:     1,
				IsActive:     true,
			},
		},
	))

	current := seedWorker(t, ctx, db, tenantA, "Current")
	notYet := seedWorker(t, ctx, db, tenantA, "NotYet")
	ended := seedWorker(t, ctx, db, tenantA, "Ended")
	foreignWorker := seedWorker(t, ctx, db, tenantB, "Foreign")
	foreignProfile := seedPayProfile(t, ctx, db, tenantB, "Foreign")

	endedAt := int64(400)
	replacedAt := int64(300)
	seedAssignment(t, ctx, db, tenantA, &payassignment.WorkerPayAssignment{
		WorkerID: current.ID, PayProfileID: older.ID, EffectiveFrom: 100, EffectiveTo: &replacedAt,
	})
	want := seedAssignment(t, ctx, db, tenantA, &payassignment.WorkerPayAssignment{
		WorkerID: current.ID, PayProfileID: profile.ID, EffectiveFrom: 300,
	})
	seedAssignment(t, ctx, db, tenantA, &payassignment.WorkerPayAssignment{
		WorkerID: notYet.ID, PayProfileID: profile.ID, EffectiveFrom: 1_000,
	})
	seedAssignment(t, ctx, db, tenantA, &payassignment.WorkerPayAssignment{
		WorkerID: ended.ID, PayProfileID: profile.ID, EffectiveFrom: 100, EffectiveTo: &endedAt,
	})
	seedAssignment(t, ctx, db, tenantB, &payassignment.WorkerPayAssignment{
		WorkerID: foreignWorker.ID, PayProfileID: foreignProfile.ID, EffectiveFrom: 100,
	})

	repo := NewWorkerPayAssignment(Params{DB: postgres.WrapDB(db), Logger: zap.NewNop()})

	got, err := repo.GetEffectiveForWorkers(scoped, driverpayports.GetWorkerPayAssignmentsRequest{
		TenantInfo: tenantA.Info(),
		WorkerIDs: []pulid.ID{
			current.ID,
			notYet.ID,
			ended.ID,
			foreignWorker.ID,
			pulid.MustNew("wrk_"),
		},
		AsOf: 500,
	})
	require.NoError(t, err)
	require.Len(t, got, 1, "only the worker with an assignment in effect, in this tenant")

	assignment := got[current.ID]
	require.NotNil(t, assignment)
	assert.Equal(t, want.ID, assignment.ID)
	require.NotNil(t, assignment.PayProfile)
	assert.Equal(t, profile.ID, assignment.PayProfile.ID)
	require.Len(t, assignment.PayProfile.Components, 2)
	assert.Equal(t, driverpay.ComponentKindStopPay, assignment.PayProfile.Components[0].Kind,
		"components in sequence order")
	assert.Equal(t, driverpay.ComponentKindDetention, assignment.PayProfile.Components[1].Kind)

	single, err := repo.GetEffectiveForWorker(scoped, driverpayports.GetWorkerPayAssignmentRequest{
		TenantInfo: tenantA.Info(),
		WorkerID:   current.ID,
		AsOf:       500,
	})
	require.NoError(t, err)
	assert.Equal(t, single.ID, assignment.ID, "the batch read agrees with the single read")

	none, err := repo.GetEffectiveForWorkers(scoped, driverpayports.GetWorkerPayAssignmentsRequest{
		TenantInfo: tenantA.Info(),
		AsOf:       500,
	})
	require.NoError(t, err)
	assert.Empty(t, none)
}
