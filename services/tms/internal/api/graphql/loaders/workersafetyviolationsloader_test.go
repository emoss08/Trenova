package loaders

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/workersafety"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workersafetyports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkerSafetyViolationsBatchFunc_ReadsEveryEventOnceAndGroupsByEvent(t *testing.T) {
	t.Parallel()

	firstID := pulid.MustNew("wsev_")
	secondID := pulid.MustNew("wsev_")
	calls := 0
	lister := &stubSafetyViolationsLister{
		list: func(_ context.Context, req *workersafetyports.ListWorkerSafetyViolationsRequest) ([]*workersafety.WorkerSafetyViolation, error) {
			calls++
			assert.Equal(t, []pulid.ID{firstID, secondID}, req.SafetyEventIDs)
			assert.True(t, req.WorkerID.IsNil(), "a violation is found by its event alone")
			return []*workersafety.WorkerSafetyViolation{
				{SafetyEventID: firstID, Code: "393.9"},
				{SafetyEventID: firstID, Code: "392.2"},
			}, nil
		},
	}
	factory := &WorkerSafetyViolationsByEventIDLoaderFactory{violations: lister}

	values, errs := factory.batchFunc(pagination.TenantInfo{})(t.Context(), []string{
		firstID.String(),
		"bad",
		secondID.String(),
	})

	assert.Equal(t, 1, calls)
	require.Len(t, values, 3)
	require.NoError(t, errs[0])
	require.Len(t, values[0], 2)
	assert.Equal(t, "393.9", values[0][0].Code)
	require.Error(t, errs[1])
	require.NoError(t, errs[2])
	assert.NotNil(t, values[2])
	assert.Empty(t, values[2])
}

func TestWorkerSafetyViolationsBatchFunc_RepositoryErrorReachesEveryKey(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("repository failed")
	lister := &stubSafetyViolationsLister{
		list: func(context.Context, *workersafetyports.ListWorkerSafetyViolationsRequest) ([]*workersafety.WorkerSafetyViolation, error) {
			return nil, repoErr
		},
	}
	factory := &WorkerSafetyViolationsByEventIDLoaderFactory{violations: lister}

	_, errs := factory.batchFunc(pagination.TenantInfo{})(t.Context(), []string{
		pulid.MustNew("wsev_").String(),
	})

	require.ErrorIs(t, errs[0], repoErr)
}

type stubSafetyViolationsLister struct {
	list func(context.Context, *workersafetyports.ListWorkerSafetyViolationsRequest) ([]*workersafety.WorkerSafetyViolation, error)
}

func (s *stubSafetyViolationsLister) ListViolations(
	ctx context.Context,
	req *workersafetyports.ListWorkerSafetyViolationsRequest,
) ([]*workersafety.WorkerSafetyViolation, error) {
	return s.list(ctx, req)
}
