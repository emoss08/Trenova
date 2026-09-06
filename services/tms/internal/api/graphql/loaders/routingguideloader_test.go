package loaders

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoutingGuideBatchFunc_PreservesOrderAndRequestsEntries(t *testing.T) {
	t.Parallel()

	firstID := pulid.MustNew("rgd_")
	secondID := pulid.MustNew("rgd_")
	missingID := pulid.MustNew("rgd_")
	getter := &stubRoutingGuidesByIDsGetter{
		getByIDs: func(_ context.Context, req repositories.GetRoutingGuidesByIDsRequest) ([]*tender.RoutingGuide, error) {
			assert.Equal(t, []pulid.ID{secondID, firstID, missingID}, req.RoutingGuideIDs)
			assert.True(t, req.IncludeEntries)
			return []*tender.RoutingGuide{
				{ID: firstID, Name: "First", Entries: []*tender.RoutingGuideEntry{{Rank: 1}}},
				{ID: secondID, Name: "Second"},
			}, nil
		},
	}
	factory := &RoutingGuideWithEntriesByIDLoaderFactory{routingGuides: getter}

	values, errs := factory.batchFunc(pagination.TenantInfo{})(t.Context(), []string{
		secondID.String(),
		"bad",
		firstID.String(),
		missingID.String(),
		secondID.String(),
	})

	require.Len(t, values, 5)
	require.NoError(t, errs[0])
	assert.Equal(t, "Second", values[0].Name)
	require.Error(t, errs[1])
	require.NoError(t, errs[2])
	assert.Equal(t, "First", values[2].Name)
	assert.Len(t, values[2].Entries, 1)
	require.Error(t, errs[3])
	assert.True(t, errortypes.IsNotFoundError(errs[3]))
	require.NoError(t, errs[4])
	assert.Equal(t, "Second", values[4].Name)
}

func TestRoutingGuideBatchFunc_RepositoryErrorFillsValidResults(t *testing.T) {
	t.Parallel()

	guideID := pulid.MustNew("rgd_")
	repoErr := errors.New("repository failed")
	getter := &stubRoutingGuidesByIDsGetter{
		getByIDs: func(context.Context, repositories.GetRoutingGuidesByIDsRequest) ([]*tender.RoutingGuide, error) {
			return nil, repoErr
		},
	}
	factory := &RoutingGuideWithEntriesByIDLoaderFactory{routingGuides: getter}

	values, errs := factory.batchFunc(pagination.TenantInfo{})(t.Context(), []string{
		"bad",
		guideID.String(),
	})

	require.Len(t, values, 2)
	require.Error(t, errs[0])
	require.ErrorIs(t, errs[1], repoErr)
}

type stubRoutingGuidesByIDsGetter struct {
	getByIDs func(context.Context, repositories.GetRoutingGuidesByIDsRequest) ([]*tender.RoutingGuide, error)
}

func (s *stubRoutingGuidesByIDsGetter) GetByIDs(
	ctx context.Context,
	req repositories.GetRoutingGuidesByIDsRequest,
) ([]*tender.RoutingGuide, error) {
	return s.getByIDs(ctx, req)
}
