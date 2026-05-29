package thumbnailjobs

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/storage"
	"github.com/emoss08/trenova/internal/core/services/encryptionservice"
	"github.com/emoss08/trenova/internal/core/services/thumbnailservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/google/uuid"
	"go.temporal.io/sdk/activity"
	"go.uber.org/fx"
)

type ActivitiesParams struct {
	fx.In

	DocumentRepository repositories.DocumentRepository
	Storage            storage.Client
	ThumbnailGenerator *thumbnailservice.Generator
	Encryption         *encryptionservice.Service
}

type Activities struct {
	docRepo            repositories.DocumentRepository
	storage            storage.Client
	thumbnailGenerator *thumbnailservice.Generator
	encryption         *encryptionservice.Service
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		docRepo:            p.DocumentRepository,
		storage:            p.Storage,
		thumbnailGenerator: p.ThumbnailGenerator,
		encryption:         p.Encryption,
	}
}

func (a *Activities) GenerateThumbnailActivity(
	ctx context.Context,
	payload *GenerateThumbnailPayload,
) (*GenerateThumbnailResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Starting thumbnail generation",
		"documentId", payload.DocumentID.String(),
		"contentType", payload.ContentType,
	)

	doc, result, err := a.getDocument(ctx, payload)
	if err != nil {
		return result, err
	}

	fileData, result, err := a.downloadFileData(ctx, payload, doc)
	if err != nil {
		return result, err
	}

	thumbData, result, err := a.generateThumbnail(ctx, payload, fileData)
	if err != nil {
		return result, err
	}

	previewPath, result, err := a.uploadThumbnail(ctx, payload, doc, thumbData)
	if err != nil {
		return result, err
	}

	if result, err = a.updateDocumentPreview(ctx, payload, doc, previewPath); err != nil {
		return result, err
	}

	logger.Info("Thumbnail generated successfully",
		"documentId", payload.DocumentID.String(),
		"previewPath", previewPath,
	)

	return &GenerateThumbnailResult{
		DocumentID:         payload.DocumentID,
		PreviewStoragePath: previewPath,
		Success:            true,
	}, nil
}

func (a *Activities) getDocument(
	ctx context.Context,
	payload *GenerateThumbnailPayload,
) (*document.Document, *GenerateThumbnailResult, error) {
	logger := activity.GetLogger(ctx)
	activity.RecordHeartbeat(ctx, "loading document record")

	doc, err := a.docRepo.GetByID(ctx, repositories.GetDocumentByIDRequest{
		ID: payload.DocumentID,
		TenantInfo: pagination.TenantInfo{
			OrgID: payload.OrganizationID,
			BuID:  payload.BusinessUnitID,
		},
	})
	if err != nil {
		logger.Error("Failed to get document", "error", err)
		return nil, a.failure(
				payload,
				"failed to get document: %v",
				err,
			), a.retryable(
				"Failed to get document",
				err,
			)
	}

	return doc, nil, nil
}

func (a *Activities) failure(
	payload *GenerateThumbnailPayload,
	format string,
	err error,
) *GenerateThumbnailResult {
	return &GenerateThumbnailResult{
		DocumentID: payload.DocumentID,
		Success:    false,
		Error:      fmt.Sprintf(format, err),
	}
}

func (a *Activities) retryable(message string, err error) error {
	return temporaltype.NewRetryableError(message, err).ToTemporalError()
}

func (a *Activities) downloadFileData(
	ctx context.Context,
	payload *GenerateThumbnailPayload,
	doc *document.Document,
) ([]byte, *GenerateThumbnailResult, error) {
	logger := activity.GetLogger(ctx)
	activity.RecordHeartbeat(ctx, "downloading original file")

	downloadResult, err := a.storage.Download(ctx, doc.StoragePath)
	if err != nil {
		logger.Error("Failed to download original file", "error", err)
		return nil, a.failure(
				payload,
				"failed to download file: %v",
				err,
			), a.retryable(
				"Failed to download file",
				err,
			)
	}
	defer downloadResult.Body.Close()

	fileData, err := io.ReadAll(downloadResult.Body)
	if err != nil {
		logger.Error("Failed to read file data", "error", err)
		return nil, a.failure(
				payload,
				"failed to read file data: %v",
				err,
			), a.retryable(
				"Failed to read file data",
				err,
			)
	}

	if a.encryption == nil {
		err = errors.New("document encryption is not configured")
		logger.Error("Failed to decrypt file data", "error", err)
		return nil, a.failure(
				payload,
				"failed to decrypt file data: %v",
				err,
			), a.retryable(
				"Failed to decrypt file data",
				err,
			)
	}
	if !encryptionservice.IsEnvelope(string(fileData)) {
		err = encryptionservice.ErrInvalidEnvelope
		logger.Error("Failed to decrypt file data", "error", err)
		return nil, a.failure(
				payload,
				"failed to decrypt file data: %v",
				err,
			), a.retryable(
				"Failed to decrypt file data",
				err,
			)
	}

	fileData, err = a.encryption.DecryptBytesWithAAD(
		string(fileData),
		documentStorageAAD(doc, doc.StoragePath),
	)
	if err != nil {
		logger.Error("Failed to decrypt file data", "error", err)
		return nil, a.failure(
				payload,
				"failed to decrypt file data: %v",
				err,
			), a.retryable(
				"Failed to decrypt file data",
				err,
			)
	}

	return fileData, nil, nil
}

func (a *Activities) generateThumbnail(
	ctx context.Context,
	payload *GenerateThumbnailPayload,
	fileData []byte,
) ([]byte, *GenerateThumbnailResult, error) {
	logger := activity.GetLogger(ctx)
	activity.RecordHeartbeat(ctx, "generating thumbnail")

	thumbData, err := a.thumbnailGenerator.Generate(bytes.NewReader(fileData), payload.ContentType)
	if err != nil {
		logger.Warn("Thumbnail generation failed", "error", err)

		if errors.Is(err, thumbnailservice.ErrPDFHasNoPages) {
			return nil, a.failure(
					payload,
					"thumbnail generation failed: %v",
					err,
				), temporaltype.NewDataIntegrityError(
					"PDF has no pages for thumbnail generation",
					map[string]any{"documentId": payload.DocumentID.String()},
				).
					ToTemporalError()
		}

		return nil, a.failure(payload, "thumbnail generation failed: %v", err), a.retryable(
			"Failed to generate thumbnail",
			err,
		)
	}

	return thumbData, nil, nil
}

func (a *Activities) uploadThumbnail(
	ctx context.Context,
	payload *GenerateThumbnailPayload,
	doc *document.Document,
	thumbData []byte,
) (string, *GenerateThumbnailResult, error) {
	logger := activity.GetLogger(ctx)
	activity.RecordHeartbeat(ctx, "uploading thumbnail")

	previewPath := fmt.Sprintf("%s/thumbnails/%s/%s_thumb.webp",
		payload.OrganizationID.String(),
		payload.ResourceType,
		uuid.New().String(),
	)

	if a.encryption == nil {
		err := errors.New("document encryption is not configured")
		return "", a.failure(
				payload,
				"failed to encrypt thumbnail: %v",
				err,
			), a.retryable(
				"Failed to encrypt thumbnail",
				err,
			)
	}

	encryptedThumb, err := a.encryption.EncryptBytesWithAAD(
		thumbData,
		documentStorageAAD(doc, previewPath),
	)
	if err != nil {
		logger.Error("Failed to encrypt thumbnail", "error", err)
		return "", a.failure(
				payload,
				"failed to encrypt thumbnail: %v",
				err,
			), a.retryable(
				"Failed to encrypt thumbnail",
				err,
			)
	}
	uploadBody := []byte(encryptedThumb)

	_, err = a.storage.Upload(ctx, &storage.UploadParams{
		Key:         previewPath,
		ContentType: "image/webp",
		Size:        int64(len(uploadBody)),
		Body:        bytes.NewReader(uploadBody),
		Metadata: map[string]string{
			"crypto_mode": encryptionservice.CryptoModeForCiphertext(encryptedThumb),
		},
	})
	if err != nil {
		logger.Error("Failed to upload thumbnail", "error", err)
		return "", a.failure(
				payload,
				"failed to upload thumbnail: %v",
				err,
			), a.retryable(
				"Failed to upload thumbnail",
				err,
			)
	}

	return previewPath, nil, nil
}

func (a *Activities) cleanupThumbnail(ctx context.Context, previewPath string) {
	logger := activity.GetLogger(ctx)
	if delErr := a.storage.Delete(ctx, previewPath); delErr != nil {
		logger.Error("Failed to cleanup thumbnail", "error", delErr)
	}
}

func (a *Activities) updateDocumentPreview(
	ctx context.Context,
	payload *GenerateThumbnailPayload,
	doc *document.Document,
	previewPath string,
) (*GenerateThumbnailResult, error) {
	logger := activity.GetLogger(ctx)
	activity.RecordHeartbeat(ctx, "updating document record")

	doc.PreviewStoragePath = previewPath
	doc.PreviewStatus = document.PreviewStatusReady

	err := a.docRepo.UpdatePreview(ctx, &repositories.UpdateDocumentPreviewRequest{
		ID: doc.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: payload.OrganizationID,
			BuID:  payload.BusinessUnitID,
		},
		PreviewStatus:      document.PreviewStatusReady,
		PreviewStoragePath: previewPath,
	})
	if err != nil {
		logger.Error("Failed to update document", "error", err)
		a.cleanupThumbnail(ctx, previewPath)
		return a.failure(
				payload,
				"failed to update document: %v",
				err,
			), a.retryable(
				"Failed to update document",
				err,
			)
	}

	return nil, nil //nolint:nilnil // nil is valid for no error
}

func documentStorageAAD(
	doc *document.Document,
	storagePath string,
) encryptionservice.AAD {
	return encryptionservice.AAD{
		Purpose:        encryptionservice.PurposeDocument,
		OrganizationID: doc.OrganizationID,
		BusinessUnitID: doc.BusinessUnitID,
		ResourceID:     storagePath,
	}
}

func (a *Activities) MarkThumbnailFailedActivity(
	ctx context.Context,
	payload *GenerateThumbnailPayload,
) error {
	doc, err := a.docRepo.GetByID(ctx, repositories.GetDocumentByIDRequest{
		ID: payload.DocumentID,
		TenantInfo: pagination.TenantInfo{
			OrgID: payload.OrganizationID,
			BuID:  payload.BusinessUnitID,
		},
	})
	if err != nil {
		return err
	}

	if !document.SupportsPreview(doc.FileType) {
		doc.PreviewStatus = document.PreviewStatusUnsupported
	} else {
		doc.PreviewStatus = document.PreviewStatusFailed
	}

	return a.docRepo.UpdatePreview(ctx, &repositories.UpdateDocumentPreviewRequest{
		ID: doc.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: payload.OrganizationID,
			BuID:  payload.BusinessUnitID,
		},
		PreviewStatus:      doc.PreviewStatus,
		PreviewStoragePath: "",
	})
}
