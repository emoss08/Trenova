package thumbnailjobs

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/documents/app/pagerender"
	"github.com/emoss08/trenova/internal/modules/documents/app/storedobject"
	"github.com/emoss08/trenova/internal/modules/documents/app/thumbnailservice"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/jobs/thumbnailjobs/thumbnailcontract"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/google/uuid"
	"go.temporal.io/sdk/activity"
	"go.uber.org/fx"
)

const logKeyDocumentID = "documentId"

type PrerenderProgress struct {
	PagesStored int `json:"pagesStored"`
}

type ActivitiesParams struct {
	fx.In

	DocumentRepository documentports.DocumentRepository
	Storage            kernelports.Client
	ThumbnailGenerator *thumbnailservice.Generator
	PageRenders        *pagerender.Store
	Encryption         *encryptionservice.Service
	Config             *config.Config
}

type Activities struct {
	docRepo            documentports.DocumentRepository
	storage            kernelports.Client
	thumbnailGenerator *thumbnailservice.Generator
	pageRenders        *pagerender.Store
	encryption         *encryptionservice.Service
	prerenderPages     int
	heartbeat          func(ctx context.Context, details ...any)
}

func NewActivities(p ActivitiesParams) *Activities {
	imagingConfig := &config.ImagingConfig{}
	if p.Config != nil {
		imagingConfig = &p.Config.Imaging
	}
	return &Activities{
		docRepo:            p.DocumentRepository,
		storage:            p.Storage,
		thumbnailGenerator: p.ThumbnailGenerator,
		pageRenders:        p.PageRenders,
		encryption:         p.Encryption,
		prerenderPages:     imagingConfig.GetPrerenderPages(),
		heartbeat:          activity.RecordHeartbeat,
	}
}

func (a *Activities) PrerenderPagesActivity(
	ctx context.Context,
	payload *thumbnailcontract.GenerateThumbnailPayload,
) (int, error) {
	doc, _, err := a.getDocument(ctx, payload)
	if err != nil {
		return 0, err
	}
	if !document.SupportsPreview(doc.FileType) {
		return 0, nil
	}
	if a.encryption == nil {
		return 0, a.retryable("Failed to read the document", storedobject.ErrNotAvailable)
	}

	a.heartbeat(ctx, PrerenderProgress{})
	stored, err := a.pageRenders.Prerender(
		ctx,
		pagination.TenantInfo{OrgID: payload.OrganizationID, BuID: payload.BusinessUnitID},
		doc,
		a.documentSource(doc, doc.FileType),
		a.prerenderPages,
		func(stored int) {
			a.heartbeat(ctx, PrerenderProgress{PagesStored: stored})
		},
	)
	if errors.Is(err, thumbnailservice.ErrUnreadable) ||
		errors.Is(err, thumbnailservice.ErrPDFHasNoPages) {
		return stored, temporaltype.NewDataIntegrityError(
			"The document's pages could not be rendered",
			map[string]any{logKeyDocumentID: payload.DocumentID.String(), "error": err.Error()},
		).ToTemporalError()
	}
	if err != nil {
		return stored, a.retryable("Failed to render page views", err)
	}
	return stored, nil
}

func (a *Activities) GenerateThumbnailActivity(
	ctx context.Context,
	payload *thumbnailcontract.GenerateThumbnailPayload,
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

	if a.encryption == nil {
		return a.failure(payload, "failed to read file data: %v", storedobject.ErrNotAvailable),
			a.retryable("Failed to read the document", storedobject.ErrNotAvailable)
	}

	thumbData, result, err := a.generateThumbnail(
		ctx,
		payload,
		a.documentSource(doc, payload.ContentType),
	)
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
	payload *thumbnailcontract.GenerateThumbnailPayload,
) (*document.Document, *GenerateThumbnailResult, error) {
	logger := activity.GetLogger(ctx)
	a.heartbeat(ctx, "loading document record")

	doc, err := a.docRepo.GetByID(ctx, documentports.GetDocumentByIDRequest{
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
	payload *thumbnailcontract.GenerateThumbnailPayload,
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

func (a *Activities) documentSource(
	doc *document.Document,
	contentType string,
) kernelports.ImagingSource {
	return storedobject.Source(
		a.storage,
		a.encryption,
		doc.StoragePath,
		documentStorageAAD(doc, doc.StoragePath),
		contentType,
		doc.FileSize,
	)
}

func (a *Activities) generateThumbnail(
	ctx context.Context,
	payload *thumbnailcontract.GenerateThumbnailPayload,
	source kernelports.ImagingSource,
) ([]byte, *GenerateThumbnailResult, error) {
	logger := activity.GetLogger(ctx)
	a.heartbeat(ctx, "generating thumbnail")

	thumbData, err := a.thumbnailGenerator.Generate(ctx, source, kernelports.ImagingBatch)
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

		if errors.Is(err, thumbnailservice.ErrUnreadable) {
			return nil, a.failure(
					payload,
					"thumbnail generation failed: %v",
					err,
				), temporaltype.NewDataIntegrityError(
					"PDF could not be read while generating thumbnail",
					map[string]any{
						"documentId": payload.DocumentID.String(),
						"error":      err.Error(),
					},
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
	payload *thumbnailcontract.GenerateThumbnailPayload,
	doc *document.Document,
	thumbData []byte,
) (string, *GenerateThumbnailResult, error) {
	logger := activity.GetLogger(ctx)
	a.heartbeat(ctx, "uploading thumbnail")

	previewPath := fmt.Sprintf("%s/thumbnails/%s/%s_thumb.webp",
		payload.OrganizationID.String(),
		payload.ResourceType,
		uuid.New().String(),
	)

	if _, err := storedobject.Write(ctx, a.storage, a.encryption, &storedobject.WriteRequest{
		Key:         previewPath,
		ContentType: "image/webp",
		AAD:         documentStorageAAD(doc, previewPath),
		Body:        bytes.NewReader(thumbData),
		Size:        int64(len(thumbData)),
	}); err != nil {
		logger.Error("Failed to store thumbnail", "error", err)
		return "", a.failure(
				payload,
				"failed to store thumbnail: %v",
				err,
			), a.retryable(
				"Failed to store thumbnail",
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
	payload *thumbnailcontract.GenerateThumbnailPayload,
	doc *document.Document,
	previewPath string,
) (*GenerateThumbnailResult, error) {
	logger := activity.GetLogger(ctx)
	a.heartbeat(ctx, "updating document record")

	doc.PreviewStoragePath = previewPath
	doc.PreviewStatus = document.PreviewStatusReady

	err := a.docRepo.UpdatePreview(ctx, &documentports.UpdateDocumentPreviewRequest{
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
	payload *thumbnailcontract.GenerateThumbnailPayload,
) error {
	doc, err := a.docRepo.GetByID(ctx, documentports.GetDocumentByIDRequest{
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

	return a.docRepo.UpdatePreview(ctx, &documentports.UpdateDocumentPreviewRequest{
		ID: doc.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: payload.OrganizationID,
			BuID:  payload.BusinessUnitID,
		},
		PreviewStatus:      doc.PreviewStatus,
		PreviewStoragePath: "",
	})
}
