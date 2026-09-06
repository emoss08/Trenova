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
	require.Error(t, errs[3])
	require.NoError(t, errs[4])
	assert.Equal(t, "Second", values[4].Name)
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

	values, errs := factory.batchFunc(pagination.TenantInfo{})(t.Context(), []string{
		"bad",
		locationID.String(),
	})

	require.Len(t, values, 2)
	require.Error(t, errs[0])
	require.ErrorIs(t, errs[1], repoErr)
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

func (s *stubLocationRepository) ListConnection(
	context.Context,
	*repositories.ListLocationConnectionRequest,
) (*pagination.CursorListResult[*location.Location], error) {
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
