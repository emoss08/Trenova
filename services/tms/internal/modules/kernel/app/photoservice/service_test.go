package photoservice_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config/storageconfig"
	"github.com/emoss08/trenova/internal/infrastructure/imaging/imagingfake"
	"github.com/emoss08/trenova/internal/modules/kernel/app/photoservice"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	storageutil "github.com/emoss08/trenova/shared/testutil/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type scanner struct {
	mu       sync.Mutex
	err      error
	requests []*kernelports.UploadScanRequest
}

func (s *scanner) ScanUpload(_ context.Context, req *kernelports.UploadScanRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests = append(s.requests, req)
	return s.err
}

type normalized struct {
	mu       sync.Mutex
	requests []kernelports.ImagingNormalizeRequest
	sources  [][]byte
}

func (n *normalized) record(req *kernelports.ImagingNormalizeRequest, source []byte) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.requests = append(n.requests, *req)
	n.sources = append(n.sources, source)
}

type rig struct {
	svc        *photoservice.Service
	storage    *storageutil.MockStorageClient
	imaging    *imagingfake.Fake
	scanner    *scanner
	normalized *normalized
	tenant     pagination.TenantInfo
	holder     kernelports.PhotoHolder
	now        int64
}

func newRig(t *testing.T) *rig {
	t.Helper()

	r := &rig{
		storage:    storageutil.NewMockStorageClient(),
		imaging:    imagingfake.New(),
		scanner:    &scanner{},
		normalized: &normalized{},
		tenant: pagination.TenantInfo{
			OrgID:  pulid.MustNew("org_"),
			BuID:   pulid.MustNew("bu_"),
			UserID: pulid.MustNew("usr_"),
		},
		now: 1_800_000_123,
	}
	r.holder = kernelports.PhotoHolder{
		OrganizationID: r.tenant.OrgID,
		Folder:         "worker/photo",
		ID:             pulid.MustNew("wrk_"),
	}
	r.imaging.NormalizeImageFunc = func(
		ctx context.Context,
		req *kernelports.ImagingNormalizeRequest,
		yield func(*kernelports.ImagingArtifact) error,
	) (*kernelports.ImagingNormalizeSummary, error) {
		source, err := imagingfake.Read(ctx, req.Source)
		if err != nil {
			return nil, err
		}
		r.normalized.record(req, source)
		out := []byte("jpeg:" + strings.Repeat("x", req.MaxWidth))
		if err = yield(&kernelports.ImagingArtifact{
			ContentType: "image/jpeg",
			Width:       req.MaxWidth,
			Height:      req.MaxHeight,
			Size:        int64(len(out)),
			Body:        bytes.NewReader(out),
		}); err != nil {
			return nil, err
		}
		return &kernelports.ImagingNormalizeSummary{SourceFormat: "png"}, nil
	}
	r.svc = photoservice.New(photoservice.Params{
		Logger:  zap.NewNop(),
		Clock:   func() int64 { return r.now },
		Storage: r.storage,
		StorageConfig: &storageconfig.Config{
			MaxFileSize:        5 << 20,
			PresignedURLExpiry: 15 * time.Minute,
		},
		Imaging: r.imaging,
		Scanner: r.scanner,
	})
	return r
}

var pngPortrait = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR portrait")

func (r *rig) store(t *testing.T, name, contentType string) (*kernelports.StoredPhoto, error) {
	t.Helper()
	return r.storeBytes(t, name, contentType, pngPortrait)
}

func (r *rig) storeBytes(
	t *testing.T,
	name, contentType string,
	content []byte,
) (*kernelports.StoredPhoto, error) {
	t.Helper()
	return r.svc.Store(t.Context(), &kernelports.StorePhotoRequest{
		TenantInfo:   r.tenant,
		File:         storageutil.NewMockFileHeader(name, content, contentType),
		Holder:       r.holder,
		ResourceType: "worker-photo",
	})
}

func (r *rig) keys(stored *kernelports.StoredPhoto) *kernelports.PhotoKeys {
	return &kernelports.PhotoKeys{
		Holder:       r.holder,
		Key:          stored.Key,
		ThumbnailKey: stored.ThumbnailKey,
	}
}

func fieldErrorOn(t *testing.T, err error, field string) {
	t.Helper()
	var multi *errortypes.MultiError
	if errors.As(err, &multi) {
		for _, fieldErr := range multi.Errors {
			if fieldErr.Path.String() == field {
				return
			}
		}
	}
	var single *errortypes.Error
	if errors.As(err, &single) && single.Path.String() == field {
		return
	}
	t.Fatalf("expected a field error on %q, got %v", field, err)
}

func TestStore_NormalizesAPictureAndAThumbnailAndUploadsBoth(t *testing.T) {
	t.Parallel()

	r := newRig(t)
	stored, err := r.store(t, "me.png", "image/png")
	require.NoError(t, err)

	require.Len(t, r.normalized.requests, 2)
	full, thumbnail := r.normalized.requests[0], r.normalized.requests[1]
	assert.Equal(t, kernelports.ImagingJPEG, full.Format)
	assert.Equal(t, 1024, full.MaxWidth)
	assert.Equal(t, 1024, full.MaxHeight)
	assert.Equal(t, kernelports.ImagingJPEG, thumbnail.Format)
	assert.Equal(t, 256, thumbnail.MaxWidth)
	assert.Equal(t, 256, thumbnail.MaxHeight)
	assert.Equal(t, pngPortrait, r.normalized.sources[0],
		"the picture is made from the upload")
	assert.Equal(t, "jpeg:"+strings.Repeat("x", 1024), string(r.normalized.sources[1]),
		"the thumbnail is made from the normalized picture, not the upload again")

	require.NotEqual(t, stored.Key, stored.ThumbnailKey)
	for _, key := range []string{stored.Key, stored.ThumbnailKey} {
		assert.True(
			t,
			strings.HasPrefix(key, r.tenant.OrgID.String()+"/worker/photo/"+r.holder.ID.String()+"/"),
			key,
		)
		assert.True(t, strings.HasSuffix(key, ".jpg"), key)
		file, ok := r.storage.GetFile(key)
		require.True(t, ok, key)
		assert.Equal(t, "image/jpeg", file.ContentType)
	}
	thumbnailFile, _ := r.storage.GetFile(stored.ThumbnailKey)
	assert.Equal(t, "jpeg:"+strings.Repeat("x", 256), string(thumbnailFile.Data))
}

func TestStore_RefusesAFileThatIsNotAnImageBeforeScanningIt(t *testing.T) {
	t.Parallel()

	r := newRig(t)
	_, err := r.store(t, "notes.txt", "text/plain")

	require.Error(t, err)
	fieldErrorOn(t, err, "file")
	assert.Empty(t, r.scanner.requests)
	assert.Zero(t, r.imaging.CallCount("normalize_image"))
	assert.Empty(t, r.storage.GetUploads())
}

func TestStore_RefusesAFileTheScannerRefuses(t *testing.T) {
	t.Parallel()

	r := newRig(t)
	r.scanner.err = errortypes.NewValidationError("file", errortypes.ErrInvalid, "infected")
	_, err := r.store(t, "me.png", "image/png")

	require.Error(t, err)
	require.Len(t, r.scanner.requests, 1)
	assert.Equal(t, pngPortrait, r.scanner.requests[0].Content)
	assert.Equal(t, "file", r.scanner.requests[0].Field)
	assert.Zero(t, r.imaging.CallCount("normalize_image"))
	assert.Empty(t, r.storage.GetUploads())
}

func TestStore_AnUnreadableImageIsAFieldError(t *testing.T) {
	t.Parallel()

	r := newRig(t)
	r.imaging.NormalizeImageFunc = func(
		context.Context,
		*kernelports.ImagingNormalizeRequest,
		func(*kernelports.ImagingArtifact) error,
	) (*kernelports.ImagingNormalizeSummary, error) {
		return nil, kernelports.ErrImagingInvalidDocument
	}
	_, err := r.store(t, "me.png", "image/png")

	require.Error(t, err)
	fieldErrorOn(t, err, "file")
	assert.Empty(t, r.storage.GetUploads())
}

func TestStore_AFailedThumbnailUploadDeletesThePicture(t *testing.T) {
	t.Parallel()

	r := newRig(t)
	uploads := 0
	r.storage.UploadFunc = func(
		_ context.Context,
		params *kernelports.UploadParams,
	) (*kernelports.FileInfo, error) {
		uploads++
		if uploads == 2 {
			return nil, errors.New("storage is down")
		}
		return &kernelports.FileInfo{Key: params.Key}, nil
	}
	_, err := r.store(t, "me.png", "image/png")

	require.Error(t, err)
	require.Len(t, r.storage.GetDeletes(), 1)
	assert.True(t, strings.HasPrefix(r.storage.GetDeletes()[0], r.tenant.OrgID.String()+"/"))
}

func TestStore_SniffsTheBytesRatherThanTrustingTheHeader(t *testing.T) {
	t.Parallel()

	r := newRig(t)
	stored, err := r.store(t, "me.png", "application/octet-stream")
	require.NoError(t, err)
	require.NotNil(t, stored)
	assert.Equal(t, "image/png", r.normalized.requests[0].Source.ContentType,
		"the imaging worker is told what the bytes are, not what the browser said")
}

func TestStore_RefusesBytesThatAreNotAnImageWhateverTheHeaderSays(t *testing.T) {
	t.Parallel()

	r := newRig(t)
	_, err := r.storeBytes(t, "me.png", "image/png", []byte("%PDF-1.7 not a picture"))

	require.Error(t, err)
	fieldErrorOn(t, err, "file")
	assert.Empty(t, r.scanner.requests)
	assert.Empty(t, r.storage.GetUploads())
}

func TestSign_NoPhotoIsNil(t *testing.T) {
	t.Parallel()

	r := newRig(t)
	urls, err := r.svc.Sign(t.Context(), &kernelports.PhotoKeys{Holder: r.holder})

	require.NoError(t, err)
	assert.Nil(t, urls)
}

func TestSign_SignsBothObjects(t *testing.T) {
	t.Parallel()

	r := newRig(t)
	stored, err := r.store(t, "me.png", "image/png")
	require.NoError(t, err)

	urls, err := r.svc.Sign(t.Context(), r.keys(stored))

	require.NoError(t, err)
	require.NotNil(t, urls)
	assert.Contains(t, urls.URL, stored.Key)
	assert.Contains(t, urls.ThumbnailURL, stored.ThumbnailKey)
}

func TestSign_RefusesAnObjectAnotherHolderOwns(t *testing.T) {
	t.Parallel()

	r := newRig(t)
	stored, err := r.store(t, "me.png", "image/png")
	require.NoError(t, err)

	for name, holder := range map[string]kernelports.PhotoHolder{
		"another organization": {
			OrganizationID: pulid.MustNew("org_"),
			Folder:         r.holder.Folder,
			ID:             r.holder.ID,
		},
		"another worker": {
			OrganizationID: r.holder.OrganizationID,
			Folder:         r.holder.Folder,
			ID:             pulid.MustNew("wrk_"),
		},
		"another folder": {
			OrganizationID: r.holder.OrganizationID,
			Folder:         "user/profile-picture",
			ID:             r.holder.ID,
		},
	} {
		urls, signErr := r.svc.Sign(t.Context(), &kernelports.PhotoKeys{
			Holder:       holder,
			Key:          stored.Key,
			ThumbnailKey: stored.ThumbnailKey,
		})
		require.NoError(t, signErr, name)
		assert.Nil(t, urls, "%s: a key its holder does not own is never signed", name)
	}
}

func TestSign_IsStableWithinAWindowAndAlwaysValidForAnHour(t *testing.T) {
	t.Parallel()

	r := newRig(t)
	var signed []kernelports.PresignedURLParams
	r.storage.GetPresignedURLFunc = func(
		_ context.Context,
		params *kernelports.PresignedURLParams,
	) (string, error) {
		signed = append(signed, *params)
		return "https://signed.test/" + params.Key + "?at=" + params.SignedAt.String(), nil
	}
	stored, err := r.store(t, "me.png", "image/png")
	require.NoError(t, err)

	first, err := r.svc.Sign(t.Context(), r.keys(stored))
	require.NoError(t, err)
	r.now += 5 * 60
	second, err := r.svc.Sign(t.Context(), r.keys(stored))
	require.NoError(t, err)

	assert.Equal(t, first.URL, second.URL, "a link read twice in a window is the same link")
	for _, params := range signed {
		require.False(t, params.SignedAt.IsZero())
		assert.Zero(t, params.SignedAt.Unix()%600, "signed at the start of a ten-minute window")
		validUntil := params.SignedAt.Add(params.Expiry)
		assert.GreaterOrEqual(t, validUntil.Unix(), r.now+3600,
			"every link is valid for at least an hour from now")
	}
}

func TestSign_PassesAnExternalPictureThrough(t *testing.T) {
	t.Parallel()

	r := newRig(t)
	urls, err := r.svc.Sign(t.Context(), &kernelports.PhotoKeys{
		Holder: r.holder,
		Key:    "https://example.com/avatar.png",
	})

	require.NoError(t, err)
	require.NotNil(t, urls)
	assert.Equal(t, "https://example.com/avatar.png", urls.URL)
	assert.Equal(t, "https://example.com/avatar.png", urls.ThumbnailURL)
}

func TestDiscard_DeletesOnlyTheHoldersOwnObjects(t *testing.T) {
	t.Parallel()

	r := newRig(t)
	own := r.tenant.OrgID.String() + "/worker/photo/" + r.holder.ID.String() + "/a.jpg"
	r.svc.Discard(t.Context(), &kernelports.PhotoKeys{
		Holder:       r.holder,
		Key:          own,
		ThumbnailKey: "https://example.com/avatar.png",
	})
	r.svc.Discard(t.Context(), &kernelports.PhotoKeys{
		Holder: r.holder,
		Key:    pulid.MustNew("org_").String() + "/worker/photo/" + r.holder.ID.String() + "/b.jpg",
	})
	r.svc.Discard(t.Context(), &kernelports.PhotoKeys{
		Holder: r.holder,
		Key:    r.tenant.OrgID.String() + "/worker/photo/" + pulid.MustNew("wrk_").String() + "/c.jpg",
	})

	assert.Equal(t, []string{own}, r.storage.GetDeletes())
}

func TestDiscard_StillDeletesWhenTheRequestIsCancelled(t *testing.T) {
	t.Parallel()

	r := newRig(t)
	var deleteErrs []error
	r.storage.DeleteFunc = func(ctx context.Context, _ string) error {
		deleteErrs = append(deleteErrs, ctx.Err())
		return nil
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	r.svc.Discard(ctx, &kernelports.PhotoKeys{
		Holder: r.holder,
		Key:    r.tenant.OrgID.String() + "/worker/photo/" + r.holder.ID.String() + "/a.jpg",
	})

	require.Len(t, deleteErrs, 1)
	assert.NoError(t, deleteErrs[0], "cleanup outlives the request that asked for it")
}
