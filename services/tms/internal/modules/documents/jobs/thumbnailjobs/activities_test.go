package thumbnailjobs

import (
	"github.com/emoss08/trenova/internal/testutil/mocks/documentportsmock"
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"io"
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/core/services/encryptionservice"
	"github.com/emoss08/trenova/internal/core/services/thumbnailservice"
	"github.com/emoss08/trenova/internal/modules/documents/jobs/thumbnailjobs/thumbnailcontract"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

type thumbnailStorage struct {
	downloadKey string
	download    []byte
	uploadKey   string
	uploadBody  []byte
	metadata    map[string]string
}

func (s *thumbnailStorage) Upload(
	_ context.Context,
	params *kernelports.UploadParams,
) (*kernelports.FileInfo, error) {
	body, err := io.ReadAll(params.Body)
	if err != nil {
		return nil, err
	}
	s.uploadKey = params.Key
	s.uploadBody = body
	s.metadata = params.Metadata
	return &kernelports.FileInfo{
		Key:         params.Key,
		Size:        int64(len(body)),
		ContentType: params.ContentType,
	}, nil
}

func (s *thumbnailStorage) Download(
	_ context.Context,
	key string,
) (*kernelports.DownloadResult, error) {
	s.downloadKey = key
	return &kernelports.DownloadResult{Body: io.NopCloser(bytes.NewReader(s.download))}, nil
}

func (s *thumbnailStorage) Delete(context.Context, string) error { return nil }
func (s *thumbnailStorage) DeleteObject(context.Context, *kernelports.DeleteObjectParams) error {
	return nil
}

func (s *thumbnailStorage) GetPresignedURL(
	context.Context,
	*kernelports.PresignedURLParams,
) (string, error) {
	return "", nil
}

func (s *thumbnailStorage) GetPresignedUploadURL(
	context.Context,
	*kernelports.PresignedUploadURLParams,
) (string, error) {
	return "", nil
}

func (s *thumbnailStorage) InitiateMultipartUpload(
	context.Context,
	*kernelports.MultipartUploadParams,
) (string, error) {
	return "", nil
}

func (s *thumbnailStorage) GetMultipartUploadPartURL(
	context.Context,
	*kernelports.MultipartUploadPartURLParams,
) (string, error) {
	return "", nil
}

func (s *thumbnailStorage) CompleteMultipartUpload(
	context.Context,
	*kernelports.CompleteMultipartUploadParams,
) error {
	return nil
}

func (s *thumbnailStorage) AbortMultipartUpload(
	context.Context,
	*kernelports.AbortMultipartUploadParams,
) error {
	return nil
}

func (s *thumbnailStorage) ListMultipartUploadParts(
	context.Context,
	*kernelports.ListMultipartUploadPartsParams,
) ([]kernelports.UploadedPart, error) {
	return nil, nil
}
func (s *thumbnailStorage) Exists(context.Context, string) (bool, error) { return false, nil }
func (s *thumbnailStorage) GetFileInfo(context.Context, string) (*kernelports.FileInfo, error) {
	return nil, nil
}

func TestGenerateThumbnailActivityDecryptsOriginalAndStoresEncryptedPreview(t *testing.T) {
	t.Parallel()

	enc := encryptionservice.NewWithKeyManager(
		encryptionservice.NewLocalKeyManager(
			"thumbnail-test-encryption-key-with-at-least-32-bytes",
		),
	)
	doc := thumbnailDocument()
	encrypted, err := enc.EncryptBytesWithAAD(
		testPNG(t),
		documentStorageAAD(doc, doc.StoragePath),
	)
	require.NoError(t, err)

	storageClient := &thumbnailStorage{download: []byte(encrypted)}
	repo := documentportsmock.NewMockDocumentRepository(t)
	repo.EXPECT().GetByID(mock.Anything, mockGetDocumentRequest(doc)).Return(doc, nil)
	repo.EXPECT().
		UpdatePreview(mock.Anything, mockUpdatePreviewRequest(doc)).
		Return(nil)

	activities := NewActivities(ActivitiesParams{
		DocumentRepository: repo,
		Storage:            storageClient,
		ThumbnailGenerator: thumbnailservice.NewGenerator(nil),
		Encryption:         enc,
	})

	payload := thumbnailPayload(doc)
	payload.StoragePath = "stale/workflow/payload.pdf"
	payload.ContentType = doc.FileType

	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestActivityEnvironment()
	env.RegisterActivity(activities.GenerateThumbnailActivity)
	result, err := env.ExecuteActivity(activities.GenerateThumbnailActivity, payload)
	require.NoError(t, err)

	var response *GenerateThumbnailResult
	require.NoError(t, result.Get(&response))
	require.True(t, response.Success)
	previewPath := response.PreviewStoragePath
	require.NotEmpty(t, previewPath)
	require.Equal(t, doc.StoragePath, storageClient.downloadKey)
	require.Equal(t, previewPath, storageClient.uploadKey)
	require.Equal(t, encryptionservice.CryptoModeEnvelopeV1, storageClient.metadata["crypto_mode"])
	require.True(t, encryptionservice.IsEnvelope(string(storageClient.uploadBody)))

	_, err = enc.DecryptBytesWithAAD(
		string(storageClient.uploadBody),
		documentStorageAAD(doc, previewPath),
	)
	require.NoError(t, err)
}

func thumbnailDocument() *document.Document {
	return &document.Document{
		ID:             pulid.MustNew("doc_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		StoragePath:    "org/trailer/original.png",
		FileType:       "image/png",
		ResourceType:   "trailer",
	}
}

func thumbnailPayload(doc *document.Document) *thumbnailcontract.GenerateThumbnailPayload {
	return &thumbnailcontract.GenerateThumbnailPayload{
		DocumentID:     doc.ID,
		OrganizationID: doc.OrganizationID,
		BusinessUnitID: doc.BusinessUnitID,
		StoragePath:    doc.StoragePath,
		ContentType:    doc.FileType,
		ResourceType:   doc.ResourceType,
	}
}

func mockGetDocumentRequest(doc *document.Document) any {
	return mock.MatchedBy(func(r documentports.GetDocumentByIDRequest) bool {
		return r.ID == doc.ID &&
			r.TenantInfo.OrgID == doc.OrganizationID &&
			r.TenantInfo.BuID == doc.BusinessUnitID
	})
}

func mockUpdatePreviewRequest(doc *document.Document) any {
	return mock.MatchedBy(func(r *documentports.UpdateDocumentPreviewRequest) bool {
		return r != nil &&
			r.ID == doc.ID &&
			r.TenantInfo.OrgID == doc.OrganizationID &&
			r.TenantInfo.BuID == doc.BusinessUnitID &&
			r.PreviewStatus == document.PreviewStatusReady &&
			r.PreviewStoragePath != ""
	})
}

func testPNG(t *testing.T) []byte {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for x := range 8 {
		for y := range 8 {
			img.Set(x, y, color.RGBA{R: 30, G: 120, B: 200, A: 255})
		}
	}

	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}
