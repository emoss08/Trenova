package loaders

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPayProfileActiveAssignmentCountBatchFunc_MapsCountsAndDefaultsMissingToZero(
	t *testing.T,
) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}
	firstID := pulid.MustNew("pp_")
	secondID := pulid.MustNew("pp_")
	repo := &stubPayProfileRepository{
		countByIDs: func(_ context.Context, req repositories.CountActivePayAssignmentsRequest) (map[pulid.ID]int, error) {
			assert.Equal(t, tenantInfo, req.TenantInfo)
			assert.Equal(t, []pulid.ID{firstID, secondID}, req.ProfileIDs)
			return map[pulid.ID]int{firstID: 4}, nil
		},
	}
	factory := &PayProfileActiveAssignmentCountLoaderFactory{payProfileRepo: repo}

	values, errs := factory.batchFunc(tenantInfo)(t.Context(), []string{
		firstID.String(),
		"bad",
		secondID.String(),
		firstID.String(),
	})

	require.Len(t, values, 4)
	require.NoError(t, errs[0])
	assert.Equal(t, 4, values[0])
	require.Error(t, errs[1])
	require.NoError(t, errs[2])
	assert.Zero(t, values[2])
	require.NoError(t, errs[3])
	assert.Equal(t, 4, values[3])
}

func TestPayProfileActiveAssignmentCountBatchFunc_RepositoryErrorFillsValidResults(
	t *testing.T,
) {
	t.Parallel()

	profileID := pulid.MustNew("pp_")
	repoErr := errors.New("repository failed")
	repo := &stubPayProfileRepository{
		countByIDs: func(context.Context, repositories.CountActivePayAssignmentsRequest) (map[pulid.ID]int, error) {
			return nil, repoErr
		},
	}
	factory := &PayProfileActiveAssignmentCountLoaderFactory{payProfileRepo: repo}

	values, errs := factory.batchFunc(pagination.TenantInfo{})(t.Context(), []string{
		"bad",
		profileID.String(),
	})

	require.Len(t, values, 2)
	require.Error(t, errs[0])
	require.NotErrorIs(t, errs[0], repoErr)
	require.ErrorIs(t, errs[1], repoErr)
}

type stubPayProfileRepository struct {
	repositories.PayProfileRepository

	countByIDs func(context.Context, repositories.CountActivePayAssignmentsRequest) (map[pulid.ID]int, error)
}

func (s *stubPayProfileRepository) CountActiveAssignmentsByIDs(
	ctx context.Context,
	req repositories.CountActivePayAssignmentsRequest,
) (map[pulid.ID]int, error) {
	return s.countByIDs(ctx, req)
}
