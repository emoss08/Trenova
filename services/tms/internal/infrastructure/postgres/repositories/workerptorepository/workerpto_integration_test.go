//go:build integration

package workerptorepository

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

var errRollbackSentinel = errors.New("rollback")

type ptoFixture struct {
	repo   *repository
	data   *seedtest.TestData
	worker *worker.Worker
	tenant pagination.TenantInfo
}

func setupPTOFixture(t *testing.T) *ptoFixture {
	t.Helper()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	wrk := &worker.Worker{
		OrganizationID: data.Organization.ID,
		BusinessUnitID: data.BusinessUnit.ID,
		StateID:        data.State.ID,
		Status:         domaintypes.StatusActive,
		Type:           worker.WorkerTypeEmployee,
		DriverType:     worker.DriverTypeOTR,
		FirstName:      "Jane",
		LastName:       "Driver",
		AddressLine1:   "1 Main St",
		City:           "Springfield",
		PostalCode:     "62701",
		Gender:         worker.GenderFemale,
	}
	_, err := db.NewInsert().Model(wrk).Exec(ctx)
	require.NoError(t, err)

	repo := New(Params{
		DB:     postgres.NewTestConnection(db),
		Logger: zap.NewNop(),
	}).(*repository)

	return &ptoFixture{
		repo:   repo,
		data:   data,
		worker: wrk,
		tenant: pagination.TenantInfo{
			OrgID:  data.Organization.ID,
			BuID:   data.BusinessUnit.ID,
			UserID: data.User.ID,
		},
	}
}

func (f *ptoFixture) newPTO(t *testing.T, start, end int64) *worker.WorkerPTO {
	t.Helper()

	created, err := f.repo.Create(t.Context(), &worker.WorkerPTO{
		OrganizationID: f.data.Organization.ID,
		BusinessUnitID: f.data.BusinessUnit.ID,
		WorkerID:       f.worker.ID,
		Status:         worker.PTOStatusRequested,
		Type:           worker.PTOTypeVacation,
		StartDate:      start,
		EndDate:        end,
		Reason:         "Integration test",
	})
	require.NoError(t, err)
	return created
}

func TestUpdateStatusApprovesRequestedAndBumpsVersion(t *testing.T) {
	f := setupPTOFixture(t)
	ctx := t.Context()
	now := timeutils.NowUnix()
	pto := f.newPTO(t, now+86400, now+86400*3)

	updated, err := f.repo.UpdateStatus(ctx, &repositories.UpdatePTOStatusRequest{
		ID:              pto.ID,
		TenantInfo:      f.tenant,
		Status:          worker.PTOStatusApproved,
		UserID:          f.data.User.ID,
		ExpectedVersion: pto.Version,
	})
	require.NoError(t, err)
	assert.Equal(t, worker.PTOStatusApproved, updated.Status)
	assert.Equal(t, f.data.User.ID, updated.ApproverID)
	assert.Equal(t, pto.Version+1, updated.Version)
	assert.GreaterOrEqual(t, updated.UpdatedAt, now)
}

func TestUpdateStatusRefusesTerminalSource(t *testing.T) {
	f := setupPTOFixture(t)
	ctx := t.Context()
	now := timeutils.NowUnix()
	pto := f.newPTO(t, now+86400, now+86400*3)

	rejected, err := f.repo.UpdateStatus(ctx, &repositories.UpdatePTOStatusRequest{
		ID:         pto.ID,
		TenantInfo: f.tenant,
		Status:     worker.PTOStatusRejected,
		UserID:     f.data.User.ID,
		Reason:     "No coverage",
	})
	require.NoError(t, err)
	assert.Equal(t, "No coverage", rejected.RejectionReason)
	assert.Equal(t, f.data.User.ID, rejected.RejectorID)

	_, err = f.repo.UpdateStatus(ctx, &repositories.UpdatePTOStatusRequest{
		ID:         pto.ID,
		TenantInfo: f.tenant,
		Status:     worker.PTOStatusApproved,
		UserID:     f.data.User.ID,
	})
	require.Error(t, err)

	var validationErr *errortypes.Error
	require.ErrorAs(t, err, &validationErr)
	assert.Equal(t, "status", validationErr.Field)

	fetched, err := f.repo.GetByID(ctx, &repositories.GetPTOByIDRequest{
		ID:         pto.ID,
		TenantInfo: f.tenant,
	})
	require.NoError(t, err)
	assert.Equal(t, worker.PTOStatusRejected, fetched.Status, "terminal row must stay untouched")
	assert.Equal(t, rejected.Version, fetched.Version)
}

func TestUpdateStatusRefusesStaleVersion(t *testing.T) {
	f := setupPTOFixture(t)
	ctx := t.Context()
	now := timeutils.NowUnix()
	pto := f.newPTO(t, now+86400, now+86400*3)

	_, err := f.repo.UpdateStatus(ctx, &repositories.UpdatePTOStatusRequest{
		ID:              pto.ID,
		TenantInfo:      f.tenant,
		Status:          worker.PTOStatusApproved,
		UserID:          f.data.User.ID,
		ExpectedVersion: pto.Version + 5,
	})
	require.Error(t, err)

	var validationErr *errortypes.Error
	require.ErrorAs(t, err, &validationErr)
	assert.Equal(t, errortypes.ErrVersionMismatch, validationErr.Code)
}

func TestUpdateStatusCancelsApprovedWithReason(t *testing.T) {
	f := setupPTOFixture(t)
	ctx := t.Context()
	now := timeutils.NowUnix()
	pto := f.newPTO(t, now+86400, now+86400*3)

	approved, err := f.repo.UpdateStatus(ctx, &repositories.UpdatePTOStatusRequest{
		ID:         pto.ID,
		TenantInfo: f.tenant,
		Status:     worker.PTOStatusApproved,
		UserID:     f.data.User.ID,
	})
	require.NoError(t, err)

	cancelled, err := f.repo.UpdateStatus(ctx, &repositories.UpdatePTOStatusRequest{
		ID:              pto.ID,
		TenantInfo:      f.tenant,
		Status:          worker.PTOStatusCancelled,
		UserID:          f.data.User.ID,
		Reason:          "Route reassigned",
		ExpectedVersion: approved.Version,
	})
	require.NoError(t, err)
	assert.Equal(t, worker.PTOStatusCancelled, cancelled.Status)
	assert.Equal(t, "Route reassigned", cancelled.CancellationReason)
	assert.Equal(t, f.data.User.ID, cancelled.CancelledByID)
	assert.Equal(t, f.data.User.ID, cancelled.ApproverID, "approver history is preserved")
}

func TestUpdateStatusIsTenantScoped(t *testing.T) {
	f := setupPTOFixture(t)
	ctx := t.Context()
	now := timeutils.NowUnix()
	pto := f.newPTO(t, now+86400, now+86400*3)

	_, err := f.repo.UpdateStatus(ctx, &repositories.UpdatePTOStatusRequest{
		ID: pto.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
		Status: worker.PTOStatusApproved,
		UserID: f.data.User.ID,
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err), "foreign tenant sees not found, got %v", err)
}

func TestHasOverlapAndGetByIDs(t *testing.T) {
	f := setupPTOFixture(t)
	ctx := t.Context()
	now := timeutils.NowUnix()
	first := f.newPTO(t, now+86400*10, now+86400*14)
	second := f.newPTO(t, now+86400*30, now+86400*32)

	overlaps, err := f.repo.HasOverlap(ctx, &repositories.PTOOverlapRequest{
		TenantInfo: f.tenant,
		WorkerID:   f.worker.ID,
		StartDate:  now + 86400*13,
		EndDate:    now + 86400*20,
	})
	require.NoError(t, err)
	assert.True(t, overlaps, "range touching an open request overlaps")

	overlaps, err = f.repo.HasOverlap(ctx, &repositories.PTOOverlapRequest{
		TenantInfo: f.tenant,
		WorkerID:   f.worker.ID,
		StartDate:  now + 86400*15,
		EndDate:    now + 86400*20,
	})
	require.NoError(t, err)
	assert.False(t, overlaps, "adjacent range does not overlap")

	overlaps, err = f.repo.HasOverlap(ctx, &repositories.PTOOverlapRequest{
		TenantInfo: f.tenant,
		WorkerID:   f.worker.ID,
		StartDate:  first.StartDate,
		EndDate:    first.EndDate,
		ExcludeID:  first.ID,
	})
	require.NoError(t, err)
	assert.False(t, overlaps, "a request never overlaps itself")

	_, err = f.repo.UpdateStatus(ctx, &repositories.UpdatePTOStatusRequest{
		ID:         first.ID,
		TenantInfo: f.tenant,
		Status:     worker.PTOStatusRejected,
		UserID:     f.data.User.ID,
		Reason:     "No coverage",
	})
	require.NoError(t, err)

	overlaps, err = f.repo.HasOverlap(ctx, &repositories.PTOOverlapRequest{
		TenantInfo: f.tenant,
		WorkerID:   f.worker.ID,
		StartDate:  first.StartDate,
		EndDate:    first.EndDate,
	})
	require.NoError(t, err)
	assert.False(t, overlaps, "rejected requests do not block new ones")

	entities, err := f.repo.GetByIDs(ctx, &repositories.GetPTOsByIDsRequest{
		IDs:        []pulid.ID{first.ID, second.ID, pulid.MustNew("wrkpto_")},
		TenantInfo: f.tenant,
	})
	require.NoError(t, err)
	assert.Len(t, entities, 2)

	entities, err = f.repo.GetByIDs(ctx, &repositories.GetPTOsByIDsRequest{
		IDs:        nil,
		TenantInfo: f.tenant,
	})
	require.NoError(t, err)
	assert.Empty(t, entities)
}

func TestRepositoryParticipatesInOuterTransaction(t *testing.T) {
	f := setupPTOFixture(t)
	ctx := t.Context()
	now := timeutils.NowUnix()
	pto := f.newPTO(t, now+86400, now+86400*3)

	err := f.repo.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		if _, txErr := f.repo.UpdateStatus(txCtx, &repositories.UpdatePTOStatusRequest{
			ID:         pto.ID,
			TenantInfo: f.tenant,
			Status:     worker.PTOStatusApproved,
			UserID:     f.data.User.ID,
		}); txErr != nil {
			return txErr
		}
		return errRollbackSentinel
	})
	require.ErrorIs(t, err, errRollbackSentinel)

	fetched, err := f.repo.GetByID(ctx, &repositories.GetPTOByIDRequest{
		ID:         pto.ID,
		TenantInfo: f.tenant,
	})
	require.NoError(t, err)
	assert.Equal(
		t,
		worker.PTOStatusRequested,
		fetched.Status,
		"rolled-back update must not persist",
	)
}
