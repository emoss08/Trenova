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

func TestWorkerLeaveEntriesBatchFunc_GroupsByCaseAndDefaultsToEmpty(t *testing.T) {
	t.Parallel()

	firstID := pulid.MustNew("wlc_")
	secondID := pulid.MustNew("wlc_")
	lister := &stubLeaveEntriesByCaseIDsLister{
		list: func(_ context.Context, req *repositories.ListLeaveEntriesByCaseIDsRequest) (map[pulid.ID][]*worker.WorkerLeaveEntry, error) {
			assert.Equal(t, []pulid.ID{firstID, secondID}, req.LeaveCaseIDs)
			return map[pulid.ID][]*worker.WorkerLeaveEntry{
				firstID: {
					{LeaveCaseID: firstID, UsedOn: 200},
					{LeaveCaseID: firstID, UsedOn: 100},
				},
			}, nil
		},
	}
	factory := &WorkerLeaveEntriesByCaseIDLoaderFactory{entries: lister}

	values, errs := factory.batchFunc(pagination.TenantInfo{})(t.Context(), []string{
		firstID.String(),
		"bad",
		secondID.String(),
		firstID.String(),
	})

	require.Len(t, values, 4)
	require.NoError(t, errs[0])
	require.Len(t, values[0], 2)
	assert.Equal(t, int64(200), values[0][0].UsedOn)
	assert.Equal(t, int64(100), values[0][1].UsedOn)
	require.Error(t, errs[1])
	require.NoError(t, errs[2])
	assert.Empty(t, values[2])
	assert.NotNil(t, values[2])
	require.NoError(t, errs[3])
	assert.Len(t, values[3], 2)
}

func TestWorkerLeaveEntriesBatchFunc_RepositoryErrorFillsValidResults(t *testing.T) {
	t.Parallel()

	caseID := pulid.MustNew("wlc_")
	repoErr := errors.New("repository failed")
	lister := &stubLeaveEntriesByCaseIDsLister{
		list: func(context.Context, *repositories.ListLeaveEntriesByCaseIDsRequest) (map[pulid.ID][]*worker.WorkerLeaveEntry, error) {
			return nil, repoErr
		},
	}
	factory := &WorkerLeaveEntriesByCaseIDLoaderFactory{entries: lister}

	values, errs := factory.batchFunc(pagination.TenantInfo{})(t.Context(), []string{
		"bad",
		caseID.String(),
	})

	require.Len(t, values, 2)
	require.Error(t, errs[0])
	require.ErrorIs(t, errs[1], repoErr)
}

type stubLeaveEntriesByCaseIDsLister struct {
	list func(context.Context, *repositories.ListLeaveEntriesByCaseIDsRequest) (map[pulid.ID][]*worker.WorkerLeaveEntry, error)
}

func (s *stubLeaveEntriesByCaseIDsLister) ListEntriesByCaseIDs(
	ctx context.Context,
	req *repositories.ListLeaveEntriesByCaseIDsRequest,
) (map[pulid.ID][]*worker.WorkerLeaveEntry, error) {
	return s.list(ctx, req)
}
