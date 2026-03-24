package usstatehandler_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/api/handlers/usstatehandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/internal/core/services/usstateservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var errNotFound = errors.New("us state not found")

func setupUsStateHandler(t *testing.T, repo *mocks.MockUsStateRepository) *usstatehandler.Handler {
	t.Helper()

	logger := zap.NewNop()

	service := usstateservice.New(usstateservice.Params{
		Logger: logger,
		Repo:   repo,
	})

	cfg := &config.Config{
		App: config.AppConfig{
			Debug: true,
		},
	}

	errorHandler := helpers.NewErrorHandler(helpers.ErrorHandlerParams{
		Logger: logger,
		Config: cfg,
	})

	return usstatehandler.New(usstatehandler.Params{
		Service:      service,
		ErrorHandler: errorHandler,
	})
}

func TestUsStateHandler_SelectOptions_Success(t *testing.T) {
	t.Parallel()

	ussID := pulid.MustNew("uss_")
	repo := mocks.NewMockUsStateRepository(t)
	repo.On("SelectOptions", mock.Anything, mock.Anything).
		Return(&pagination.ListResult[*usstate.UsState]{
			Items: []*usstate.UsState{
				{
					ID:           ussID,
					Name:         "California",
					Abbreviation: "CA",
					CountryName:  "United States",
					CountryIso3:  "USA",
				},
			},
			Total: 1,
		}, nil)

	handler := setupUsStateHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/us-states/select-options/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp pagination.Response[[]map[string]any]
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, 1, resp.Count)
	assert.Len(t, resp.Results, 1)
}

func TestUsStateHandler_GetOption_Success(t *testing.T) {
	t.Parallel()

	ussID := pulid.MustNew("uss_")
	repo := mocks.NewMockUsStateRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(&usstate.UsState{
		ID:           ussID,
		Name:         "California",
		Abbreviation: "CA",
		CountryName:  "United States",
		CountryIso3:  "USA",
	}, nil)

	handler := setupUsStateHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/us-states/select-options/" + ussID.String()).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "California", resp["name"])
}

func TestUsStateHandler_GetOption_InvalidID(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockUsStateRepository(t)
	handler := setupUsStateHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/us-states/select-options/invalid-id").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestUsStateHandler_GetOption_NotFound(t *testing.T) {
	t.Parallel()

	ussID := pulid.MustNew("uss_")
	repo := mocks.NewMockUsStateRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(nil, errNotFound)

	handler := setupUsStateHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/us-states/select-options/" + ussID.String()).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}
