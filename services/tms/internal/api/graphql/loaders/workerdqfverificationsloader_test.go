package loaders

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkerDQFVerificationsBatchFunc_GroupsByWorkerAndDefaultsToEmpty(t *testing.T) {
	t.Parallel()

	firstID := pulid.MustNew("wrk_")
	secondID := pulid.MustNew("wrk_")
	lister := &stubEmploymentVerificationsByWorkerIDsLister{
		list: func(_ context.Context, req *repositories.ListEmploymentVerificationsByWorkerIDsRequest) (map[pulid.ID][]*worker.WorkerEmploymentVerification, error) {
			assert.Equal(t, []pulid.ID{firstID, secondID}, req.WorkerIDs)
			assert.True(t, req.IncludeDocument)
			return map[pulid.ID][]*worker.WorkerEmploymentVerification{
				firstID: {
					{WorkerID: firstID, EmployerName: "Acme"},
					{WorkerID: firstID, EmployerName: "Globex"},
				},
			}, nil
		},
	}
	factory := &WorkerDQFVerificationsByWorkerIDLoaderFactory{verifications: lister}

	values, errs := factory.batchFunc(pagination.TenantInfo{})(t.Context(), []string{
		firstID.String(),
		"bad",
		secondID.String(),
		firstID.String(),
	})

	require.Len(t, values, 4)
	require.NoError(t, errs[0])
	require.Len(t, values[0], 2)
	assert.Equal(t, "Acme", values[0][0].EmployerName)
	assert.Equal(t, "Globex", values[0][1].EmployerName)
	require.Error(t, errs[1])
	require.NoError(t, errs[2])
	assert.Empty(t, values[2])
	assert.NotNil(t, values[2])
	require.NoError(t, errs[3])
	assert.Len(t, values[3], 2)
}

func TestWorkerDQFVerificationsBatchFunc_RepositoryErrorFillsValidResults(t *testing.T) {
	t.Parallel()

	workerID := pulid.MustNew("wrk_")
	repoErr := errors.New("repository failed")
	lister := &stubEmploymentVerificationsByWorkerIDsLister{
		list: func(context.Context, *repositories.ListEmploymentVerificationsByWorkerIDsRequest) (map[pulid.ID][]*worker.WorkerEmploymentVerification, error) {
			return nil, repoErr
		},
	}
	factory := &WorkerDQFVerificationsByWorkerIDLoaderFactory{verifications: lister}

	values, errs := factory.batchFunc(pagination.TenantInfo{})(t.Context(), []string{
		"bad",
		workerID.String(),
	})

	require.Len(t, values, 2)
	require.Error(t, errs[0])
	require.ErrorIs(t, errs[1], repoErr)
}

type stubEmploymentVerificationsByWorkerIDsLister struct {
	list func(context.Context, *repositories.ListEmploymentVerificationsByWorkerIDsRequest) (map[pulid.ID][]*worker.WorkerEmploymentVerification, error)
}

func (s *stubEmploymentVerificationsByWorkerIDsLister) ListVerificationsByWorkerIDs(
	ctx context.Context,
	req *repositories.ListEmploymentVerificationsByWorkerIDsRequest,
) (map[pulid.ID][]*worker.WorkerEmploymentVerification, error) {
	return s.list(ctx, req)
}
