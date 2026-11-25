package workerhandler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/mocks/workerportsmock"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func photoFile() testutil.MultipartFile {
	return testutil.MultipartFile{
		FieldName: "file",
		Filename:  "face.jpg",
		Data:      []byte("\xff\xd8\xff portrait"),
	}
}

func TestWorkerHandler_UploadPhoto_RequiresTheWorkerUpdatePermission(t *testing.T) {
	t.Parallel()

	repo := workerportsmock.NewMockWorkerRepository(t)
	deps := setupWorkerHandlerWithEngines(
		t,
		repo,
		&mocks.DenyResourcesPermissionEngine{Denied: []string{permission.ResourceWorker.String()}},
		&mocks.AllowAllPermissionEngine{},
	)
	workerID := pulid.MustNew("wrk_")

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/workers/"+workerID.String()+"/photo/").
		WithDefaultAuthContext().
		WithMultipartForm(map[string]string{"version": "1"}, photoFile())

	deps.handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusForbidden, ginCtx.ResponseCode())
	assert.Empty(t, deps.photos.Stored, "nothing is processed for a caller who may not edit workers")
}

func TestWorkerHandler_RemovePhoto_RequiresTheWorkerUpdatePermission(t *testing.T) {
	t.Parallel()

	repo := workerportsmock.NewMockWorkerRepository(t)
	deps := setupWorkerHandlerWithEngines(
		t,
		repo,
		&mocks.DenyResourcesPermissionEngine{Denied: []string{permission.ResourceWorker.String()}},
		&mocks.AllowAllPermissionEngine{},
	)
	workerID := pulid.MustNew("wrk_")

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodDelete).
		WithPath("/api/v1/workers/" + workerID.String() + "/photo/").
		WithQuery(map[string]string{"version": "1"}).
		WithDefaultAuthContext()

	deps.handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusForbidden, ginCtx.ResponseCode())
}

func TestWorkerHandler_UploadPhoto_RequiresTheVersion(t *testing.T) {
	t.Parallel()

	repo := workerportsmock.NewMockWorkerRepository(t)
	deps := setupWorkerHandler(t, repo)
	workerID := pulid.MustNew("wrk_")

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/workers/"+workerID.String()+"/photo/").
		WithDefaultAuthContext().
		WithMultipartForm(map[string]string{}, photoFile())

	deps.handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.GreaterOrEqual(t, ginCtx.ResponseCode(), http.StatusBadRequest)
	assert.Less(t, ginCtx.ResponseCode(), http.StatusInternalServerError)
	assert.Empty(t, deps.photos.Stored)
}

func TestWorkerHandler_UploadPhoto_ReturnsTheSignedPhotoAndTheNewVersion(t *testing.T) {
	t.Parallel()

	workerID := pulid.MustNew("wrk_")
	repo := workerportsmock.NewMockWorkerRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).
		Return(func(_ context.Context, req workerports.GetWorkerByIDRequest) *worker.Worker {
			return &worker.Worker{
				ID:             workerID,
				OrganizationID: req.TenantInfo.OrgID,
				BusinessUnitID: req.TenantInfo.BuID,
				Version:        3,
			}
		}, nil)
	repo.On("UpdatePhoto", mock.Anything, mock.Anything).
		Return(func(_ context.Context, req *workerports.UpdateWorkerPhotoRequest) *worker.Worker {
			return &worker.Worker{
				ID:                workerID,
				OrganizationID:    req.TenantInfo.OrgID,
				BusinessUnitID:    req.TenantInfo.BuID,
				PhotoKey:          req.PhotoKey,
				PhotoThumbnailKey: req.PhotoThumbnailKey,
				Version:           req.Version + 1,
			}
		}, nil)
	deps := setupWorkerHandler(t, repo)
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/workers/"+workerID.String()+"/photo/").
		WithDefaultAuthContext().
		WithMultipartForm(map[string]string{"version": "3"}, photoFile())

	deps.handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	require.Equal(t, http.StatusOK, ginCtx.ResponseCode(), string(ginCtx.ResponseBody()))
	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.NotEmpty(t, resp["url"])
	assert.NotEmpty(t, resp["thumbnailUrl"])
	assert.InDelta(t, 4, resp["version"], 0)
	require.Len(t, deps.photos.Stored, 1)
}
