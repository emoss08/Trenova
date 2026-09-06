package loaders

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShiftTemplateActiveAssignmentCountBatchFunc_MapsCountsAndDefaultsMissingToZero(
	t *testing.T,
) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}
	firstID := pulid.MustNew("sft_")
	secondID := pulid.MustNew("sft_")
	counter := stubTenantCounter{
		count: func(_ context.Context, ti pagination.TenantInfo, ids []pulid.ID) (map[pulid.ID]int, error) {
			assert.Equal(t, tenantInfo, ti)
			assert.Equal(t, []pulid.ID{firstID, secondID}, ids)
			return map[pulid.ID]int{firstID: 7}, nil
		},
	}
	factory := &ShiftTemplateActiveAssignmentCountLoaderFactory{counter: counter}

	values, errs := factory.batchFunc(tenantInfo)(t.Context(), []string{
		firstID.String(),
		"bad",
		secondID.String(),
		firstID.String(),
	})

	require.Len(t, values, 4)
	require.NoError(t, errs[0])
	assert.Equal(t, 7, values[0])
	require.Error(t, errs[1])
	require.NoError(t, errs[2])
	assert.Zero(t, values[2])
	require.NoError(t, errs[3])
	assert.Equal(t, 7, values[3])
}

func TestShiftTemplateActiveAssignmentCountBatchFunc_RepositoryErrorFillsValidResults(
	t *testing.T,
) {
	t.Parallel()

	templateID := pulid.MustNew("sft_")
	repoErr := errors.New("repository failed")
	counter := stubTenantCounter{
		count: func(context.Context, pagination.TenantInfo, []pulid.ID) (map[pulid.ID]int, error) {
			return nil, repoErr
		},
	}
	factory := &ShiftTemplateActiveAssignmentCountLoaderFactory{counter: counter}

	values, errs := factory.batchFunc(pagination.TenantInfo{})(t.Context(), []string{
		"bad",
		templateID.String(),
	})

	require.Len(t, values, 2)
	require.Error(t, errs[0])
	require.NotErrorIs(t, errs[0], repoErr)
	require.ErrorIs(t, errs[1], repoErr)
}
