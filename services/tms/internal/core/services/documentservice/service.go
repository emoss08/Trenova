package documentservice

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/ports/storage"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/core/services/thumbnailservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs/thumbnailjobs"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/google/uuid"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger             *zap.Logger
	Repo               repositories.DocumentRepository
	CacheRepo          repositories.DocumentCacheRepository
	Storage            storage.Client
	Validator          *Validator
	AuditService       services.AuditService
	Config             *config.Config
	ThumbnailGenerator *thumbnailservice.Generator
	TemporalClient     client.Client
}

type Service struct {
	l                  *zap.Logger
	repo               repositories.DocumentRepository
	cacheRepo          repositories.DocumentCacheRepository
	storage            storage.Client
	validator          *Validator
	auditService       services.AuditService
	config             *config.StorageConfig
	thumbnailGenerator *thumbnailservice.Generator
	temporalClient     client.Client
}

//nolint:gocritic // dependency injection param
func New(p Params) *Service {
	return &Service{
		l:                  p.Logger.Named("service.document"),
		repo:               p.Repo,
		cacheRepo:          p.CacheRepo,
		storage:            p.Storage,
		validator:          p.Validator,
		auditService:       p.AuditService,
		config:             p.Config.GetStorageConfig(),
		thumbnailGenerator: p.ThumbnailGenerator,
		temporalClient:     p.TemporalClient,
	}
}

type UploadRequest struct {
	TenantInfo     pagination.TenantInfo
	File           *multipart.FileHeader
	ResourceID     string
	ResourceType   string
	Description    string
	Tags           []string
	DocumentTypeID string
}

type UploadResult struct {
	Document *document.Document
}

type BulkUploadRequest struct {
	TenantInfo   pagination.TenantInfo
	Files        []*multipart.FileHeader
	ResourceID   string
	ResourceType string
}

type BulkUploadResult struct {
	Documents []*document.Document
	Errors    []error
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListDocumentsRequest,
) (*pagination.ListResult[*document.Document], error) {
	log := s.l.With(
		zap.String("operation", "List"),
		zap.Any("request", req),
	)

	log.Info("listing documents")
	return s.repo.List(ctx, req)
}

func (s *Service) Get(
	ctx context.Context,
	req repositories.GetDocumentByIDRequest,
) (*document.Document, error) {
	entity, err := s.cacheRepo.GetByID(ctx, req)
	if err == nil {
		return entity, nil
	}
	if !errors.Is(err, repositories.ErrCacheMiss) {
		s.l.Warn("failed to load document from cache", zap.Error(err), zap.String("documentID", req.ID.String()))
	}

	return s.repo.GetByID(ctx, req)
}

func (s *Service) GetByResource(
	ctx context.Context,
	req *repositories.GetDocumentsByResourceRequest,
) ([]*document.Document, error) {
	return s.repo.GetByResourceID(ctx, req)
}

func (s *Service) Upload(
	ctx context.Context,
	req *UploadRequest,
) (*UploadResult, error) {
	log := s.l.With(
		zap.String("operation", "Upload"),
		zap.String("resourceId", req.ResourceID),
		zap.String("resourceType", req.ResourceType),
	)

	if multiErr := s.validator.ValidateFile(req.File); multiErr != nil {
		return nil, multiErr
	}

	file, err := req.File.Open()
	if err != nil {
		log.Error("failed to open file", zap.Error(err))
		return nil, errortypes.NewDatabaseError("Failed to process uploaded file").WithInternal(err)
	}
	defer file.Close()

	contentType := req.File.Header.Get("Content-Type")
	storagePath := s.generateStoragePath(
		req.TenantInfo.OrgID.String(),
		req.ResourceType,
		req.File.Filename,
	)

	_, err = s.storage.Upload(ctx, &storage.UploadParams{
		Key:         storagePath,
		ContentType: contentType,
		Size:        req.File.Size,
		Body:        file,
		Metadata: map[string]string{
			"original_name": req.File.Filename,
			"resource_id":   req.ResourceID,
			"resource_type": req.ResourceType,
		},
	})
	if err != nil {
		log.Error("failed to upload file to storage", zap.Error(err))
		return nil, errortypes.NewDatabaseError("Failed to upload file").WithInternal(err)
	}

	doc := &document.Document{
		OrganizationID:     req.TenantInfo.OrgID,
		BusinessUnitID:     req.TenantInfo.BuID,
		FileName:           filepath.Base(storagePath),
		OriginalName:       req.File.Filename,
		FileSize:           req.File.Size,
		FileType:           contentType,
		StoragePath:        storagePath,
		PreviewStoragePath: "",
		Status:             document.StatusActive,
		Description:        req.Description,
		ResourceID:         req.ResourceID,
		ResourceType:       req.ResourceType,
		Tags:               req.Tags,
		UploadedByID:       req.TenantInfo.UserID,
	}

	if req.DocumentTypeID != "" {
		docTypeID, parseErr := pulid.MustParse(req.DocumentTypeID)
		if parseErr != nil {
			return nil, errortypes.NewValidationError(
				"documentTypeId", errortypes.ErrInvalid, "Invalid document type ID",
			)
		}
		doc.DocumentTypeID = &docTypeID
	}

	createdDoc, err := s.repo.Create(ctx, doc)
	if err != nil {
		log.Error("failed to create document record", zap.Error(err))
		if delErr := s.storage.Delete(ctx, storagePath); delErr != nil {
			log.Error("failed to cleanup storage after db failure", zap.Error(delErr))
		}
		return nil, err
	}

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceDocument,
		ResourceID:     createdDoc.GetID().String(),
		Operation:      permission.OpCreate,
		UserID:         req.TenantInfo.UserID,
		CurrentState:   jsonutils.MustToJSON(createdDoc),
		OrganizationID: createdDoc.OrganizationID,
		BusinessUnitID: createdDoc.BusinessUnitID,
	}, auditservice.WithComment("Document uploaded")); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	if s.thumbnailGenerator.SupportsThumbnail(contentType) {
		s.startThumbnailWorkflow(ctx, log, createdDoc)
	}

	return &UploadResult{Document: createdDoc}, nil
}

func (s *Service) startThumbnailWorkflow(
	ctx context.Context,
	log *zap.Logger,
	doc *document.Document,
) {
	if s.temporalClient == nil {
		log.Debug("temporal client not configured, skipping thumbnail generation")
		return
	}

	workflowID := fmt.Sprintf("thumbnail-%s", doc.ID.String())

	payload := &thumbnailjobs.GenerateThumbnailPayload{
		DocumentID:     doc.ID,
		OrganizationID: doc.OrganizationID,
		BusinessUnitID: doc.BusinessUnitID,
		StoragePath:    doc.StoragePath,
		ContentType:    doc.FileType,
		ResourceType:   doc.ResourceType,
	}

	_, err := s.temporalClient.ExecuteWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:            workflowID,
			TaskQueue:     temporaltype.ThumbnailTaskQueue,
			RetryPolicy:   temporaltype.DefaultRetryPolicy,
			StaticSummary: fmt.Sprintf("Generating thumbnail for document %s", doc.ID),
		},
		"GenerateThumbnailWorkflow",
		payload,
	)
	if err != nil {
		log.Warn("failed to start thumbnail workflow",
			zap.String("documentId", doc.ID.String()),
			zap.Error(err),
		)
	}
}

func (s *Service) BulkUpload(
	ctx context.Context,
	req *BulkUploadRequest,
) (*BulkUploadResult, error) {
	log := s.l.With(
		zap.String("operation", "BulkUpload"),
		zap.String("resourceId", req.ResourceID),
		zap.String("resourceType", req.ResourceType),
		zap.Int("fileCount", len(req.Files)),
	)

	if multiErr := s.validator.ValidateFiles(req.Files); multiErr != nil {
		return nil, multiErr
	}

	result := &BulkUploadResult{
		Documents: make([]*document.Document, 0, len(req.Files)),
		Errors:    make([]error, 0),
	}

	for _, file := range req.Files {
		uploadResult, err := s.Upload(ctx, &UploadRequest{
			TenantInfo:   req.TenantInfo,
			File:         file,
			ResourceID:   req.ResourceID,
			ResourceType: req.ResourceType,
		})
		if err != nil {
			log.Warn(
				"failed to upload file in bulk operation",
				zap.String("filename", file.Filename),
				zap.Error(err),
			)
			result.Errors = append(result.Errors, fmt.Errorf("%s: %w", file.Filename, err))
			continue
		}
		result.Documents = append(result.Documents, uploadResult.Document)
	}

	return result, nil
}

func (s *Service) GetDownloadURL(
	ctx context.Context,
	req repositories.GetDocumentByIDRequest,
) (string, error) {
	log := s.l.With(
		zap.String("operation", "GetDownloadURL"),
		zap.String("documentId", req.ID.String()),
	)

	doc, err := s.repo.GetByID(ctx, req)
	if err != nil {
		return "", err
	}

	url, err := s.storage.GetPresignedURL(ctx, &storage.PresignedURLParams{
		Key:                doc.StoragePath,
		Expiry:             s.config.GetPresignedURLExpiry(),
		ContentDisposition: fmt.Sprintf("attachment; filename=\"%q\"", doc.OriginalName),
	})
	if err != nil {
		log.Error("failed to generate presigned URL", zap.Error(err))
		return "", errortypes.NewDatabaseError("Failed to generate download URL").WithInternal(err)
	}

	return url, nil
}

func (s *Service) GetViewURL(
	ctx context.Context,
	req repositories.GetDocumentByIDRequest,
) (string, error) {
	log := s.l.With(
		zap.String("operation", "GetViewURL"),
		zap.String("documentId", req.ID.String()),
	)

	doc, err := s.repo.GetByID(ctx, req)
	if err != nil {
		return "", err
	}

	url, err := s.storage.GetPresignedURL(ctx, &storage.PresignedURLParams{
		Key:                doc.StoragePath,
		Expiry:             s.config.GetPresignedURLExpiry(),
		ContentDisposition: fmt.Sprintf("inline; filename=\"%q\"", doc.OriginalName),
	})
	if err != nil {
		log.Error("failed to generate presigned URL for viewing", zap.Error(err))
		return "", errortypes.NewDatabaseError("Failed to generate view URL").WithInternal(err)
	}

	return url, nil
}

func (s *Service) Delete(
	ctx context.Context,
	req repositories.DeleteDocumentRequest,
	userID pulid.ID,
) error {
	log := s.l.With(
		zap.String("operation", "Delete"),
		zap.String("documentId", req.ID.String()),
	)

	doc, err := s.repo.GetByID(ctx, repositories.GetDocumentByIDRequest(req))
	if err != nil {
		return err
	}

	if err = s.storage.Delete(ctx, doc.StoragePath); err != nil {
		log.Error("failed to delete file from storage", zap.Error(err))
	}

	if doc.PreviewStoragePath != "" {
		if err = s.storage.Delete(ctx, doc.PreviewStoragePath); err != nil {
			log.Error("failed to delete thumbnail from storage", zap.Error(err))
		}
	}

	if err = s.repo.Delete(ctx, req); err != nil {
		return err
	}

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceDocument,
		ResourceID:     doc.GetID().String(),
		Operation:      permission.OpDelete,
		UserID:         userID,
		PreviousState:  jsonutils.MustToJSON(doc),
		OrganizationID: doc.OrganizationID,
		BusinessUnitID: doc.BusinessUnitID,
	}, auditservice.WithComment("Document deleted")); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	return nil
}

type BulkDeleteRequest struct {
	IDs        []pulid.ID
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
}

type BulkDeleteResult struct {
	DeletedCount int
	Errors       []error
}

func (s *Service) BulkDelete(
	ctx context.Context,
	req *BulkDeleteRequest,
) (*BulkDeleteResult, error) {
	log := s.l.With(
		zap.String("operation", "BulkDelete"),
		zap.Int("count", len(req.IDs)),
	)

	if len(req.IDs) == 0 {
		return &BulkDeleteResult{DeletedCount: 0}, nil
	}

	docs, err := s.repo.GetByIDs(ctx, repositories.BulkDeleteDocumentRequest{
		IDs:        req.IDs,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		log.Error("failed to get documents for bulk delete", zap.Error(err))
		return nil, err
	}

	result := &BulkDeleteResult{
		Errors: make([]error, 0),
	}

	for _, doc := range docs {
		if err = s.storage.Delete(ctx, doc.StoragePath); err != nil {
			log.Warn("failed to delete file from storage during bulk delete",
				zap.String("documentId", doc.ID.String()),
				zap.Error(err),
			)
			result.Errors = append(
				result.Errors,
				fmt.Errorf("failed to delete file %s: %w", doc.OriginalName, err),
			)
		}

		if doc.PreviewStoragePath != "" {
			if err = s.storage.Delete(ctx, doc.PreviewStoragePath); err != nil {
				log.Warn("failed to delete thumbnail from storage during bulk delete",
					zap.String("documentId", doc.ID.String()),
					zap.Error(err),
				)
			}
		}
	}

	if err = s.repo.BulkDelete(ctx, repositories.BulkDeleteDocumentRequest{
		IDs:        req.IDs,
		TenantInfo: req.TenantInfo,
	}); err != nil {
		log.Error("failed to bulk delete documents from database", zap.Error(err))
		return nil, err
	}

	result.DeletedCount = len(docs)

	for _, doc := range docs {
		if err = s.auditService.LogAction(&services.LogActionParams{
			Resource:       permission.ResourceDocument,
			ResourceID:     doc.GetID().String(),
			Operation:      permission.OpDelete,
			UserID:         req.UserID,
			PreviousState:  jsonutils.MustToJSON(doc),
			OrganizationID: doc.OrganizationID,
			BusinessUnitID: doc.BusinessUnitID,
		}, auditservice.WithComment("Document deleted (bulk)")); err != nil {
			log.Warn("failed to log audit action for bulk delete",
				zap.String("documentId", doc.ID.String()),
				zap.Error(err),
			)
		}
	}

	return result, nil
}

func (s *Service) GetPreviewURL(
	ctx context.Context,
	req repositories.GetDocumentByIDRequest,
) (string, error) {
	log := s.l.With(
		zap.String("operation", "GetPreviewURL"),
		zap.String("documentId", req.ID.String()),
	)

	doc, err := s.repo.GetByID(ctx, req)
	if err != nil {
		return "", err
	}

	if doc.PreviewStoragePath == "" {
		return "", errortypes.NewNotFoundError("Preview not available for this document")
	}

	url, err := s.storage.GetPresignedURL(ctx, &storage.PresignedURLParams{
		Key:    doc.PreviewStoragePath,
		Expiry: s.config.GetPresignedURLExpiry(),
	})
	if err != nil {
		log.Error("failed to generate presigned URL for preview", zap.Error(err))
		return "", errortypes.NewDatabaseError("Failed to generate preview URL").WithInternal(err)
	}

	return url, nil
}

func (s *Service) generateStoragePath(orgID, resourceType, filename string) string {
	ext := filepath.Ext(filename)
	uniqueID := uuid.New().String()
	return fmt.Sprintf("%s/%s/%s%s", orgID, resourceType, uniqueID, strings.ToLower(ext))
}
