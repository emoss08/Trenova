package workerphotoservice_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerphotoservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/permtest"
	"github.com/emoss08/trenova/internal/testutil/phototest"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	storageutil "github.com/emoss08/trenova/shared/testutil/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeWorkers struct {
	workerports.WorkerRepository

	stored        *worker.Worker
	writes        []*workerports.UpdateWorkerPhotoRequest
	photoKeyReads [][]pulid.ID
	updateErr     error
}

func (f *fakeWorkers) GetByID(
	_ context.Context,
	_ workerports.GetWorkerByIDRequest,
) (*worker.Worker, error) {
	copied := *f.stored
	return &copied, nil
}

func (f *fakeWorkers) UpdatePhoto(
	_ context.Context,
	req *workerports.UpdateWorkerPhotoRequest,
) (*worker.Worker, error) {
	f.writes = append(f.writes, req)
	if f.updateErr != nil {
		return nil, f.updateErr
	}
	if req.Version != f.stored.Version {
		return nil, dberror.CreateVersionMismatchError("Worker", req.WorkerID.String())
	}
	f.stored.SetPhoto(req.PhotoKey, req.PhotoThumbnailKey)
	f.stored.Version++
	copied := *f.stored
	return &copied, nil
}

func (f *fakeWorkers) GetPhotoKeys(
	_ context.Context,
	req workerports.GetWorkersByIDsRequest,
) ([]workerports.WorkerPhotoKeys, error) {
	f.photoKeyReads = append(f.photoKeyReads, req.WorkerIDs)
	out := make([]workerports.WorkerPhotoKeys, 0, len(req.WorkerIDs))
	for _, id := range req.WorkerIDs {
		if id == f.stored.ID && f.stored.HasPhoto() {
			out = append(out, workerports.WorkerPhotoKeys{
				WorkerID:          id,
				PhotoKey:          f.stored.PhotoKey,
				PhotoThumbnailKey: f.stored.PhotoThumbnailKey,
			})
		}
	}
	return out, nil
}

type rig struct {
	svc     *workerphotoservice.Service
	workers *fakeWorkers
	photos  *phototest.Fake
	tenant  pagination.TenantInfo
}

func newRig(stored *worker.Worker) *rig {
	return newRigWith(stored, permtest.AllowAll())
}

func newRigWith(stored *worker.Worker, engine *permtest.Engine) *rig {
	tenant := pagination.TenantInfo{
		OrgID:  stored.OrganizationID,
		BuID:   stored.BusinessUnitID,
		UserID: pulid.MustNew("usr_"),
	}
	r := &rig{
		workers: &fakeWorkers{stored: stored},
		photos:  &phototest.Fake{},
		tenant:  tenant,
	}
	r.svc = workerphotoservice.New(workerphotoservice.Params{
		Logger:       zap.NewNop(),
		Repo:         r.workers,
		Photos:       r.photos,
		AuditService: &mocks.NoopAuditService{},
		Realtime:     &mocks.NoopRealtimeService{},
		DB:           dbtest.NopConnection{},
		Permissions:  engine,
		Registry:     permissioncatalog.Default(),
	})
	return r
}

func storedWorker() *worker.Worker {
	orgID := pulid.MustNew("org_")
	workerID := pulid.MustNew("wrk_")
	return &worker.Worker{
		ID:                workerID,
		OrganizationID:    orgID,
		BusinessUnitID:    pulid.MustNew("bu_"),
		FirstName:         "Ada",
		LastName:          "Byron",
		PhotoKey:          orgID.String() + "/worker/photo/" + workerID.String() + "/old.jpg",
		PhotoThumbnailKey: orgID.String() + "/worker/photo/" + workerID.String() + "/old-thumb.jpg",
		Version:           4,
	}
}

func actor(tenant pagination.TenantInfo) *permissionports.RequestActor {
	return &permissionports.RequestActor{
		UserID:         tenant.UserID,
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
	}
}

func upload(r *rig, t *testing.T, version int64) (*workerphotoservice.Result, error) {
	t.Helper()
	return r.svc.Upload(t.Context(), &workerphotoservice.UploadRequest{
		TenantInfo: r.tenant,
		WorkerID:   r.workers.stored.ID,
		Version:    version,
		File: storageutil.NewMockFileHeader(
			"face.jpg",
			[]byte("\xff\xd8\xff portrait"),
			"image/jpeg",
		),
		Actor: actor(r.tenant),
	})
}

func TestUpload_RefusesAStaleVersionBeforeStoringAnything(t *testing.T) {
	t.Parallel()

	r := newRig(storedWorker())
	_, err := upload(r, t, 3)

	require.Error(t, err)
	assert.True(t, errortypes.IsVersionMismatchError(err), "a stale version is an edit conflict: %v", err)
	assert.Empty(t, r.photos.Stored, "no picture is processed for a write that cannot land")
	assert.Empty(t, r.workers.writes)
}

func TestUpload_WritesTheNewPhotoAndDeletesTheOldOne(t *testing.T) {
	t.Parallel()

	stored := storedWorker()
	oldKey, oldThumbnail := stored.PhotoKey, stored.PhotoThumbnailKey
	r := newRig(stored)

	result, err := upload(r, t, 4)

	require.NoError(t, err)
	require.Len(t, r.photos.Stored, 1)
	assert.Equal(t, "worker/photo", r.photos.Stored[0].Holder.Folder)
	assert.Equal(t, r.tenant.OrgID, r.photos.Stored[0].Holder.OrganizationID)
	assert.Equal(t, stored.ID, r.photos.Stored[0].Holder.ID)

	require.Len(t, r.workers.writes, 1)
	write := r.workers.writes[0]
	assert.Equal(t, int64(4), write.Version, "the write is held to the version the caller saw")
	assert.NotEmpty(t, write.PhotoKey)
	assert.NotEqual(t, oldKey, write.PhotoKey)

	assert.ElementsMatch(t, []string{oldKey, oldThumbnail}, r.photos.DiscardedKeys())
	assert.Equal(t, int64(5), result.Version)
	require.NotNil(t, result.Photo)
	assert.Contains(t, result.Photo.URL, write.PhotoKey)
	assert.Contains(t, result.Photo.ThumbnailURL, write.PhotoThumbnailKey)
}

func TestUpload_AFailedWriteDeletesTheNewObjects(t *testing.T) {
	t.Parallel()

	stored := storedWorker()
	r := newRig(stored)
	r.workers.updateErr = dberror.CreateVersionMismatchError("Worker", stored.ID.String())

	_, err := upload(r, t, 4)

	require.Error(t, err)
	require.Len(t, r.workers.writes, 1)
	assert.ElementsMatch(
		t,
		[]string{r.workers.writes[0].PhotoKey, r.workers.writes[0].PhotoThumbnailKey},
		r.photos.DiscardedKeys(),
		"the staged objects are cleaned up and the photo on file is kept",
	)
	assert.Equal(t, stored.PhotoKey, r.workers.stored.PhotoKey)
}

func TestUpload_AFailedStoreWritesNothing(t *testing.T) {
	t.Parallel()

	r := newRig(storedWorker())
	r.photos.StoreErr = errortypes.NewValidationError(
		"file",
		errortypes.ErrInvalidFormat,
		"The photo could not be read as an image",
	)

	_, err := upload(r, t, 4)

	require.Error(t, err)
	assert.Empty(t, r.workers.writes)
	assert.Empty(t, r.photos.DiscardedKeys())
}

func TestRemove_ClearsThePhotoAndDeletesItsObjects(t *testing.T) {
	t.Parallel()

	stored := storedWorker()
	oldKey, oldThumbnail := stored.PhotoKey, stored.PhotoThumbnailKey
	r := newRig(stored)

	result, err := r.svc.Remove(t.Context(), &workerphotoservice.RemoveRequest{
		TenantInfo: r.tenant,
		WorkerID:   stored.ID,
		Version:    4,
		Actor:      actor(r.tenant),
	})

	require.NoError(t, err)
	require.Len(t, r.workers.writes, 1)
	assert.Empty(t, r.workers.writes[0].PhotoKey)
	assert.Empty(t, r.workers.writes[0].PhotoThumbnailKey)
	assert.ElementsMatch(t, []string{oldKey, oldThumbnail}, r.photos.DiscardedKeys())
	assert.Nil(t, result.Photo)
	assert.Equal(t, int64(5), result.Version)
}

func TestRemove_WithoutAPhotoWritesNothing(t *testing.T) {
	t.Parallel()

	stored := storedWorker()
	stored.SetPhoto("", "")
	r := newRig(stored)

	result, err := r.svc.Remove(t.Context(), &workerphotoservice.RemoveRequest{
		TenantInfo: r.tenant,
		WorkerID:   stored.ID,
		Version:    4,
		Actor:      actor(r.tenant),
	})

	require.NoError(t, err)
	assert.Empty(t, r.workers.writes)
	assert.Equal(t, int64(4), result.Version)
}

func TestRemove_RefusesAStaleVersion(t *testing.T) {
	t.Parallel()

	r := newRig(storedWorker())
	_, err := r.svc.Remove(t.Context(), &workerphotoservice.RemoveRequest{
		TenantInfo: r.tenant,
		WorkerID:   r.workers.stored.ID,
		Version:    2,
		Actor:      actor(r.tenant),
	})

	require.Error(t, err)
	assert.Empty(t, r.workers.writes)
	assert.Empty(t, r.photos.DiscardedKeys())
}

func TestApply_MovesStagedObjectsOntoTheWorkerAtItsCurrentVersion(t *testing.T) {
	t.Parallel()

	stored := storedWorker()
	oldKey, oldThumbnail := stored.PhotoKey, stored.PhotoThumbnailKey
	r := newRig(stored)
	prefix := stored.OrganizationID.String() + "/worker/photo/" + stored.ID.String()
	staged := kernelports.StoredPhoto{
		Key:          prefix + "/new.jpg",
		ThumbnailKey: prefix + "/new-thumb.jpg",
	}

	updated, err := r.svc.Apply(t.Context(), &workerphotoservice.ApplyRequest{
		TenantInfo: r.tenant,
		WorkerID:   stored.ID,
		Photo:      staged,
		Actor:      actor(r.tenant),
	})

	require.NoError(t, err)
	assert.Equal(t, staged.Key, updated.PhotoKey)
	assert.Equal(t, staged.ThumbnailKey, updated.PhotoThumbnailKey)
	require.Len(t, r.workers.writes, 1)
	assert.Equal(t, int64(4), r.workers.writes[0].Version)
	assert.ElementsMatch(t, []string{oldKey, oldThumbnail}, r.photos.DiscardedKeys())
}

func TestURLs_AWorkerWithoutAPhotoHasNone(t *testing.T) {
	t.Parallel()

	stored := storedWorker()
	stored.SetPhoto("", "")
	r := newRig(stored)

	urls, err := r.svc.URLs(t.Context(), stored)

	require.NoError(t, err)
	assert.Nil(t, urls)
}

func TestURLsByWorker_ReadsEveryWorkerOnceAndSignsOnlyThoseWithAPhoto(t *testing.T) {
	t.Parallel()

	stored := storedWorker()
	r := newRig(stored)
	other := pulid.MustNew("wrk_")

	photos, err := r.svc.URLsByWorker(
		t.Context(),
		r.tenant,
		[]pulid.ID{stored.ID, other, stored.ID, pulid.Nil},
	)

	require.NoError(t, err)
	require.Len(t, r.workers.photoKeyReads, 1, "one read for the whole list")
	assert.ElementsMatch(t, []pulid.ID{stored.ID, other}, r.workers.photoKeyReads[0])
	require.Len(t, photos, 1)
	assert.Contains(t, photos[stored.ID].URL, stored.PhotoKey)
	assert.Nil(t, photos[other])
}

func TestURLsByWorker_NoWorkersReadsNothing(t *testing.T) {
	t.Parallel()

	r := newRig(storedWorker())
	photos, err := r.svc.URLsByWorker(t.Context(), r.tenant, nil)

	require.NoError(t, err)
	assert.Empty(t, photos)
	assert.Empty(t, r.workers.photoKeyReads)
}

func TestURLsByWorker_NothingForAReaderWhoCannotSeeWorkerPhotos(t *testing.T) {
	t.Parallel()

	for name, grant := range map[string]permtest.Grant{
		"no worker read": {MaxSensitivity: permission.SensitivityConfidential},
		"below the field's sensitivity": {
			Operations:     []permission.Operation{permission.OpRead},
			MaxSensitivity: permission.SensitivityPublic,
		},
	} {
		stored := storedWorker()
		engine := permtest.AllowAll()
		engine.Overrides[permission.ResourceWorker.String()] = grant
		r := newRigWith(stored, engine)

		photos, err := r.svc.URLsByWorker(t.Context(), r.tenant, []pulid.ID{stored.ID})

		require.NoError(t, err, name)
		assert.Empty(t, photos, name)
		assert.Empty(t, r.workers.photoKeyReads, "%s: nothing is read for a reader who may not see it", name)
	}
}
