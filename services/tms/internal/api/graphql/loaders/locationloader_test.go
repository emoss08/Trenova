package loaders

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocationBatchFunc_PreservesOrderAndReportsMissingIDs(t *testing.T) {
	t.Parallel()

	firstID := pulid.MustNew("loc_")
	secondID := pulid.MustNew("loc_")
	missingID := pulid.MustNew("loc_")
	repo := &stubLocationRepository{
		getByIDs: func(_ context.Context, req repositories.GetLocationsByIDsRequest) ([]*location.Location, error) {
			assert.Equal(t, []pulid.ID{secondID, firstID, missingID}, req.LocationIDs)
			return []*location.Location{
				{ID: firstID, Name: "First"},
				{ID: secondID, Name: "Second"},
			}, nil
		},
	}
	factory := &LocationByIDLoaderFactory{locationRepo: repo}

	results := factory.batchFunc(pagination.TenantInfo{})(t.Context(), []string{
		secondID.String(),
		"bad",
		firstID.String(),
		missingID.String(),
		secondID.String(),
	})

	require.Len(t, results, 5)
	require.NoError(t, results[0].Error)
	assert.Equal(t, "Second", results[0].Data.Name)
	require.Error(t, results[1].Error)
	require.NoError(t, results[2].Error)
	assert.Equal(t, "First", results[2].Data.Name)
	require.Error(t, results[3].Error)
	require.NoError(t, results[4].Error)
	assert.Equal(t, "Second", results[4].Data.Name)
}

func TestLocationBatchFunc_RepositoryErrorFillsValidResults(t *testing.T) {
	t.Parallel()

	locationID := pulid.MustNew("loc_")
	repoErr := errors.New("repository failed")
	repo := &stubLocationRepository{
		getByIDs: func(context.Context, repositories.GetLocationsByIDsRequest) ([]*location.Location, error) {
			return nil, repoErr
		},
	}
	factory := &LocationByIDLoaderFactory{locationRepo: repo}

	results := factory.batchFunc(pagination.TenantInfo{})(t.Context(), []string{
		"bad",
		locationID.String(),
	})

	require.Len(t, results, 2)
	require.Error(t, results[0].Error)
	require.ErrorIs(t, results[1].Error, repoErr)
}

type stubLocationRepository struct {
	getByIDs func(context.Context, repositories.GetLocationsByIDsRequest) ([]*location.Location, error)
}

func (s *stubLocationRepository) List(
	context.Context,
	*repositories.ListLocationRequest,
) (*pagination.ListResult[*location.Location], error) {
	return nil, nil
}

func (s *stubLocationRepository) GetByID(
	context.Context,
	repositories.GetLocationByIDRequest,
) (*location.Location, error) {
	return nil, nil
}

func (s *stubLocationRepository) GetByIDs(
	ctx context.Context,
	req repositories.GetLocationsByIDsRequest,
) ([]*location.Location, error) {
	return s.getByIDs(ctx, req)
}

func (s *stubLocationRepository) Create(
	context.Context,
	*location.Location,
) (*location.Location, error) {
	return nil, nil
}

func (s *stubLocationRepository) Update(
	context.Context,
	*location.Location,
) (*location.Location, error) {
	return nil, nil
}

func (s *stubLocationRepository) BulkUpdateStatus(
	context.Context,
	*repositories.BulkUpdateLocationStatusRequest,
) ([]*location.Location, error) {
	return nil, nil
}

func (s *stubLocationRepository) SelectOptions(
	context.Context,
	*repositories.LocationSelectOptionsRequest,
) (*pagination.ListResult[*location.Location], error) {
	return nil, nil
}
