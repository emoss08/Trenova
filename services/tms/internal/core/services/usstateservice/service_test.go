package usstateservice_test

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/usstateservice"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type testDeps struct {
	repo *mocks.MockUsStateRepository
	svc  *usstateservice.Service
}

func setupTest(t *testing.T) *testDeps {
	t.Helper()
	repo := mocks.NewMockUsStateRepository(t)
	svc := usstateservice.New(usstateservice.Params{
		Logger: zap.NewNop(),
		Repo:   repo,
	})
	return &testDeps{repo: repo, svc: svc}
}

func TestService_Get_Success(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	ussID := pulid.MustNew("uss_")
	expected := &usstate.UsState{
		ID:           ussID,
		Name:         "California",
		Abbreviation: "CA",
		CountryName:  "United States",
		CountryIso3:  "USA",
	}

	req := repositories.GetUsStateByIDRequest{StateID: ussID}
	deps.repo.On("GetByID", mock.Anything, req).Return(expected, nil)

	result, err := deps.svc.Get(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, ussID, result.ID)
	assert.Equal(t, "California", result.Name)
	assert.Equal(t, "CA", result.Abbreviation)
}

func TestService_Get_Error(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	dbErr := errors.New("database error")
	req := repositories.GetUsStateByIDRequest{StateID: pulid.MustNew("uss_")}
	deps.repo.On("GetByID", mock.Anything, req).Return(nil, dbErr)

	result, err := deps.svc.Get(ctx, req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, dbErr, err)
}

func TestService_SelectOptions_Success(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	expected := &pagination.ListResult[*usstate.UsState]{
		Items: []*usstate.UsState{
			{
				ID:           pulid.MustNew("uss_"),
				Name:         "California",
				Abbreviation: "CA",
				CountryName:  "United States",
				CountryIso3:  "USA",
			},
			{
				ID:           pulid.MustNew("uss_"),
				Name:         "Texas",
				Abbreviation: "TX",
				CountryName:  "United States",
				CountryIso3:  "USA",
			},
		},
		Total: 2,
	}

	req := &pagination.SelectQueryRequest{}
	deps.repo.On("SelectOptions", mock.Anything, req).Return(expected, nil)

	result, err := deps.svc.SelectOptions(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, 2, result.Total)
	assert.Len(t, result.Items, 2)
}

func TestService_SelectOptions_Error(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	dbErr := errors.New("connection failed")
	req := &pagination.SelectQueryRequest{}
	deps.repo.On("SelectOptions", mock.Anything, req).Return(nil, dbErr)

	result, err := deps.svc.SelectOptions(ctx, req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, dbErr, err)
}
