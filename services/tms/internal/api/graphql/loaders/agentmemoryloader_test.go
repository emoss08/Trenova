package loaders

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubMemoryReplacements struct {
	list func(context.Context, repositories.ListAgentMemoryReplacementsRequest) ([]*agent.Memory, error)
}

func (s *stubMemoryReplacements) ListReplacements(
	ctx context.Context,
	req repositories.ListAgentMemoryReplacementsRequest,
) ([]*agent.Memory, error) {
	return s.list(ctx, req)
}

func TestReplacementBatch_GivesEachMemoryTheNewestThatTookEffect(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	replaced := pulid.MustNew("amem_")
	untouched := pulid.MustNew("amem_")
	older := &agent.Memory{
		ID:           pulid.MustNew("amem_"),
		Status:       agent.MemoryStatusRetired,
		SupersedesID: &replaced,
		CreatedAt:    100,
	}
	newer := &agent.Memory{
		ID:           pulid.MustNew("amem_"),
		Status:       agent.MemoryStatusActive,
		SupersedesID: &replaced,
		CreatedAt:    200,
	}
	waiting := &agent.Memory{
		ID:           pulid.MustNew("amem_"),
		Status:       agent.MemoryStatusSuggested,
		SupersedesID: &replaced,
		CreatedAt:    300,
	}
	stub := &stubMemoryReplacements{
		list: func(
			_ context.Context,
			req repositories.ListAgentMemoryReplacementsRequest,
		) ([]*agent.Memory, error) {
			assert.Equal(t, tenant, req.TenantInfo)
			assert.Equal(t, []pulid.ID{replaced, untouched}, req.ReplacedIDs)
			return []*agent.Memory{older, waiting, newer}, nil
		},
	}

	values, errs := batchReplacementFunc(stub, tenant)(t.Context(), []string{
		replaced.String(),
		untouched.String(),
		"bad",
		replaced.String(),
	})

	require.Len(t, values, 4)
	assert.Same(t, newer, values[0])
	assert.Nil(t, values[1])
	require.Error(t, errs[2])
	assert.Same(t, newer, values[3])
	assert.NoError(t, errs[0])
	assert.NoError(t, errs[1])
}

func TestReplacementBatch_AFailedReadFailsEveryKey(t *testing.T) {
	t.Parallel()

	failure := errors.New("database unavailable")
	stub := &stubMemoryReplacements{
		list: func(
			context.Context,
			repositories.ListAgentMemoryReplacementsRequest,
		) ([]*agent.Memory, error) {
			return nil, failure
		},
	}

	_, errs := batchReplacementFunc(stub, pagination.TenantInfo{})(t.Context(), []string{
		pulid.MustNew("amem_").String(),
		pulid.MustNew("amem_").String(),
	})

	for _, err := range errs {
		assert.ErrorIs(t, err, failure)
	}
}
