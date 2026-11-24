package selfserviceservice_test

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/personphoto"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workforce/app/selfserviceservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerphotoservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/profilechange"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	storageutil "github.com/emoss08/trenova/shared/testutil/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeWorkerPhotos struct {
	staged    []*kernelports.StoredPhoto
	applied   []*workerphotoservice.ApplyRequest
	discarded []*kernelports.StoredPhoto
	applyErr  error
}

func (f *fakeWorkerPhotos) Stage(
	_ context.Context,
	req *workerphotoservice.StageRequest,
) (*kernelports.StoredPhoto, error) {
	prefix := req.TenantInfo.OrgID.String() + "/worker/photo/" + req.WorkerID.String() + "/" +
		pulid.MustNew("tst_").String()
	photo := &kernelports.StoredPhoto{Key: prefix + ".jpg", ThumbnailKey: prefix + "-thumb.jpg"}
	f.staged = append(f.staged, photo)
	return photo, nil
}

func (f *fakeWorkerPhotos) Apply(
	_ context.Context,
	req *workerphotoservice.ApplyRequest,
) (*worker.Worker, error) {
	f.applied = append(f.applied, req)
	if f.applyErr != nil {
		return nil, f.applyErr
	}
	return &worker.Worker{
		ID:                req.WorkerID,
		PhotoKey:          req.Photo.Key,
		PhotoThumbnailKey: req.Photo.ThumbnailKey,
	}, nil
}

func (f *fakeWorkerPhotos) Discard(
	_ context.Context,
	_ workerphotoservice.WorkerRef,
	photo *kernelports.StoredPhoto,
) {
	if photo != nil {
		f.discarded = append(f.discarded, photo)
	}
}

func (f *fakeWorkerPhotos) ProposedURLs(
	_ context.Context,
	_ workerphotoservice.WorkerRef,
	photo *kernelports.StoredPhoto,
) (*personphoto.URLs, error) {
	return &personphoto.URLs{
		URL:          "https://signed.test/" + photo.Key,
		ThumbnailURL: "https://signed.test/" + photo.ThumbnailKey,
	}, nil
}

func portrait() *selfserviceservice.SubmitPhotoChangeRequest {
	return &selfserviceservice.SubmitPhotoChangeRequest{
		Worker: employee(),
		File: storageutil.NewMockFileHeader(
			"me.jpg",
			[]byte("\xff\xd8\xff portrait"),
			"image/jpeg",
		),
		TenantInfo: tenant(),
	}
}

func pendingPhotoRequest(tenantInfo pagination.TenantInfo) *profilechange.WorkerProfileChangeRequest {
	prefix := tenantInfo.OrgID.String() + "/worker/photo/wrk_1/older"
	request := &profilechange.WorkerProfileChangeRequest{
		ID:             pulid.MustNew("wpcr_"),
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		WorkerID:       "wrk_1",
		Status:         profilechange.ProfileChangePending,
		Changes:        profilechange.PhotoChange(),
		SubmittedAt:    1_800_000_000,
	}
	request.ProposePhoto(prefix+".jpg", prefix+"-thumb.jpg")
	return request
}

func contactRequest() *profilechange.WorkerProfileChangeRequest {
	return &profilechange.WorkerProfileChangeRequest{
		ID:          pulid.MustNew("wpcr_"),
		WorkerID:    "wrk_1",
		Status:      profilechange.ProfileChangePending,
		SubmittedAt: 1_800_000_000,
		Changes: []profilechange.FieldChange{
			{Field: profilechange.ProfileFieldCity, From: "Springfield", To: "Shelbyville"},
		},
	}
}

func TestSubmitPhotoChange_OpensAPendingRequestWithTheStagedPhoto(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	photos := &fakeWorkerPhotos{}

	created, err := newServiceWithPhotos(repo, &fakeWorkers{}, photos).
		SubmitPhotoChange(t.Context(), portrait())

	require.NoError(t, err)
	require.Len(t, photos.staged, 1)
	assert.Equal(t, profilechange.ProfileChangePending, created.Status)
	assert.True(t, created.IsPhoto())
	assert.Equal(t, photos.staged[0].Key, created.ProposedPhotoKey)
	assert.Equal(t, photos.staged[0].ThumbnailKey, created.ProposedPhotoThumbnailKey)
	assert.Empty(t, photos.discarded)
}

func TestSubmitPhotoChange_RefusesANewPhotoWhileOneWaits(t *testing.T) {
	t.Parallel()

	tenantInfo := tenant()
	repo := &fakeRepo{
		pending: []*profilechange.WorkerProfileChangeRequest{
			contactRequest(),
			pendingPhotoRequest(tenantInfo),
		},
	}
	photos := &fakeWorkerPhotos{}
	req := portrait()
	req.TenantInfo = tenantInfo

	_, err := newServiceWithPhotos(repo, &fakeWorkers{}, photos).
		SubmitPhotoChange(t.Context(), req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Withdraw it")
	assert.Empty(t, photos.staged, "nothing is processed for a photo that cannot be asked for")
	assert.Nil(t, repo.created)
	assert.Nil(t, repo.updated)
}

func TestSubmitPhotoChange_AWaitingContactRequestDoesNotBlockAPhoto(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{pending: []*profilechange.WorkerProfileChangeRequest{contactRequest()}}

	created, err := newServiceWithPhotos(repo, &fakeWorkers{}, &fakeWorkerPhotos{}).
		SubmitPhotoChange(t.Context(), portrait())

	require.NoError(t, err)
	assert.True(t, created.IsPhoto())
	assert.NotNil(t, repo.created)
}

func TestSubmitChange_AWaitingPhotoRequestDoesNotBlockContactChanges(t *testing.T) {
	t.Parallel()

	wrk := employee()
	repo := &fakeRepo{
		pending: []*profilechange.WorkerProfileChangeRequest{pendingPhotoRequest(tenant())},
	}
	wanted := profilechange.ContactSnapshotOf(wrk)
	wanted.City = "Shelbyville"

	created, err := newService(repo, &fakeWorkers{wrk: wrk}).SubmitChange(
		t.Context(),
		&selfserviceservice.SubmitChangeRequest{Worker: wrk, Wanted: wanted, TenantInfo: tenant()},
	)

	require.NoError(t, err)
	assert.False(t, created.IsPhoto())
}

func TestDecideChange_ApprovingAPhotoMovesTheStagedObjectsOntoTheWorker(t *testing.T) {
	t.Parallel()

	tenantInfo := tenant()
	request := pendingPhotoRequest(tenantInfo)
	stagedKey, stagedThumbnail := request.ProposedPhotoKey, request.ProposedPhotoThumbnailKey
	repo := &fakeRepo{request: request}
	workers := &fakeWorkers{wrk: employee()}
	photos := &fakeWorkerPhotos{}

	decided, err := newServiceWithPhotos(repo, workers, photos).DecideChange(
		t.Context(),
		&selfserviceservice.DecideChangeRequest{
			ID:         request.ID,
			Approve:    true,
			TenantInfo: tenantInfo,
			UserID:     pulid.MustNew("usr_"),
		},
	)

	require.NoError(t, err)
	assert.Equal(t, profilechange.ProfileChangeApproved, decided.Status)
	require.Len(t, photos.applied, 1)
	assert.Equal(t, stagedKey, photos.applied[0].Photo.Key)
	assert.Equal(t, stagedThumbnail, photos.applied[0].Photo.ThumbnailKey)
	assert.Equal(t, request.WorkerID, photos.applied[0].WorkerID)
	assert.Empty(t, photos.discarded, "the approved objects are the worker's photo now")
	assert.False(t, decided.HasProposedPhoto(), "a decided request holds no staged photo")
	assert.Nil(t, workers.written, "a photo approval writes no contact fields")
}

func TestDecideChange_AFailedPhotoApprovalLeavesTheRequestWaiting(t *testing.T) {
	t.Parallel()

	tenantInfo := tenant()
	request := pendingPhotoRequest(tenantInfo)
	repo := &fakeRepo{request: request}
	photos := &fakeWorkerPhotos{applyErr: errors.New("version mismatch")}

	_, err := newServiceWithPhotos(repo, &fakeWorkers{}, photos).DecideChange(
		t.Context(),
		&selfserviceservice.DecideChangeRequest{
			ID:         request.ID,
			Approve:    true,
			TenantInfo: tenantInfo,
			UserID:     pulid.MustNew("usr_"),
		},
	)

	require.Error(t, err)
	assert.Nil(t, repo.updated)
	assert.Empty(t, photos.discarded)
}

func TestDecideChange_RejectingAPhotoDeletesTheStagedObjects(t *testing.T) {
	t.Parallel()

	tenantInfo := tenant()
	request := pendingPhotoRequest(tenantInfo)
	stagedKey := request.ProposedPhotoKey
	repo := &fakeRepo{request: request}
	photos := &fakeWorkerPhotos{}

	decided, err := newServiceWithPhotos(repo, &fakeWorkers{}, photos).DecideChange(
		t.Context(),
		&selfserviceservice.DecideChangeRequest{
			ID:         request.ID,
			Approve:    false,
			Note:       "Please send a photo without sunglasses",
			TenantInfo: tenantInfo,
			UserID:     pulid.MustNew("usr_"),
		},
	)

	require.NoError(t, err)
	assert.Equal(t, profilechange.ProfileChangeRejected, decided.Status)
	assert.Empty(t, photos.applied)
	require.Len(t, photos.discarded, 1)
	assert.Equal(t, stagedKey, photos.discarded[0].Key)
	assert.False(t, decided.HasProposedPhoto())
}

func TestWithdrawChange_WithdrawingAPhotoDeletesTheStagedObjects(t *testing.T) {
	t.Parallel()

	tenantInfo := tenant()
	request := pendingPhotoRequest(tenantInfo)
	stagedKey := request.ProposedPhotoKey
	repo := &fakeRepo{request: request}
	photos := &fakeWorkerPhotos{}

	withdrawn, err := newServiceWithPhotos(repo, &fakeWorkers{}, photos).WithdrawChange(
		t.Context(),
		&selfserviceservice.WithdrawChangeRequest{
			ID:            request.ID,
			ActorWorkerID: request.WorkerID,
			TenantInfo:    tenantInfo,
		},
	)

	require.NoError(t, err)
	assert.Equal(t, profilechange.ProfileChangeWithdrawn, withdrawn.Status)
	require.Len(t, photos.discarded, 1)
	assert.Equal(t, stagedKey, photos.discarded[0].Key)
}

func TestProposedPhoto_OnlyAWaitingRequestShowsOne(t *testing.T) {
	t.Parallel()

	tenantInfo := tenant()
	svc := newService(&fakeRepo{}, &fakeWorkers{})

	waiting := pendingPhotoRequest(tenantInfo)
	urls, err := svc.ProposedPhoto(t.Context(), waiting)
	require.NoError(t, err)
	require.NotNil(t, urls)
	assert.Contains(t, urls.URL, waiting.ProposedPhotoKey)

	decided := pendingPhotoRequest(tenantInfo)
	decided.Status = profilechange.ProfileChangeApproved
	urls, err = svc.ProposedPhoto(t.Context(), decided)
	require.NoError(t, err)
	assert.Nil(t, urls)
}
