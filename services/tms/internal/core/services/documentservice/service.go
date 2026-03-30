package documentservice

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/ports/storage"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/core/services/thumbnailservice"
	workflowstarterservice "github.com/emoss08/trenova/internal/core/services/workflowstarter"
	"github.com/emoss08/trenova/internal/core/temporaljobs/documentuploadjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/thumbnailjobs"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger               *zap.Logger
	DB                   ports.DBConnection
	Repo                 repositories.DocumentRepository
	PacketRuleRepo       repositories.DocumentPacketRuleRepository
	DocumentTypeRepo     repositories.DocumentTypeRepository
	CacheRepo            repositories.DocumentCacheRepository
	SessionRepo          repositories.DocumentUploadSessionRepository
	Storage              storage.Client
	Validator            *Validator
	AuditService         services.AuditService
	DocumentIntelligence services.DocumentContentService
	SearchProjection     services.DocumentSearchProjectionService
	WorkflowStarter      services.WorkflowStarter
	Config               *config.Config
	ThumbnailGenerator   *thumbnailservice.Generator
}

type Service struct {
	l                    *zap.Logger
	db                   ports.DBConnection
	repo                 repositories.DocumentRepository
	packetRuleRepo       repositories.DocumentPacketRuleRepository
	documentTypeRepo     repositories.DocumentTypeRepository
	cacheRepo            repositories.DocumentCacheRepository
	sessionRepo          repositories.DocumentUploadSessionRepository
	storage              storage.Client
	validator            *Validator
	auditService         services.AuditService
	documentIntelligence services.DocumentContentService
	searchProjection     services.DocumentSearchProjectionService
	workflowStarter      services.WorkflowStarter
	config               *config.StorageConfig
	thumbnailGenerator   *thumbnailservice.Generator
}

//nolint:gocritic // dependency injection param
func New(p Params) *Service {
	workflowStarter := p.WorkflowStarter
	if workflowStarter == nil {
		workflowStarter = workflowstarterservice.New(workflowstarterservice.Params{})
	}

	documentIntelligence := p.DocumentIntelligence
	if documentIntelligence == nil {
		documentIntelligence = noopDocumentContentService{}
	}

	searchProjection := p.SearchProjection
	if searchProjection == nil {
		searchProjection = noopDocumentSearchProjectionService{}
	}

	return &Service{
		l:                    p.Logger.Named("service.document"),
		db:                   p.DB,
		repo:                 p.Repo,
		packetRuleRepo:       p.PacketRuleRepo,
		documentTypeRepo:     p.DocumentTypeRepo,
		cacheRepo:            p.CacheRepo,
		sessionRepo:          p.SessionRepo,
		storage:              p.Storage,
		validator:            p.Validator,
		auditService:         p.AuditService,
		documentIntelligence: documentIntelligence,
		searchProjection:     searchProjection,
		workflowStarter:      workflowStarter,
		config:               p.Config.GetStorageConfig(),
		thumbnailGenerator:   p.ThumbnailGenerator,
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
	LineageID      string
}

type UploadResult struct {
	Document *document.Document
}

type BulkUploadRequest struct {
	TenantInfo   pagination.TenantInfo
	Files        []*multipart.FileHeader
	ResourceID   string
	ResourceType string
	LineageID    string
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
	docID := pulid.MustNew("doc_")
	lineageID := docID
	var lineageInfo *document.Document
	if strings.TrimSpace(req.LineageID) != "" {
		parsedLineageID, parseErr := pulid.MustParse(req.LineageID)
		if parseErr != nil {
			return nil, errortypes.NewValidationError("lineageId", errortypes.ErrInvalid, "Invalid lineage ID")
		}
		lineageInfo, err = s.resolveLineageForUpload(ctx, parsedLineageID, req.TenantInfo)
		if err != nil {
			return nil, err
		}
		if lineageInfo.ResourceID != req.ResourceID || lineageInfo.ResourceType != req.ResourceType {
			return nil, errortypes.NewConflictError("Document lineage does not belong to the selected resource")
		}
		lineageID = lineageInfo.LineageID
	}

	hasher := sha256.New()
	tee := io.TeeReader(file, hasher)
	fileInfo, err := s.storage.Upload(ctx, &storage.UploadParams{
		Key:         storagePath,
		ContentType: contentType,
		Size:        req.File.Size,
		Body:        tee,
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
		ID:                 docID,
		OrganizationID:     req.TenantInfo.OrgID,
		BusinessUnitID:     req.TenantInfo.BuID,
		LineageID:          lineageID,
		VersionNumber:      s.nextVersionNumber(lineageInfo),
		IsCurrentVersion:   true,
		FileName:           filepath.Base(storagePath),
		OriginalName:       req.File.Filename,
		FileSize:           req.File.Size,
		FileType:           contentType,
		StoragePath:        storagePath,
		ChecksumSHA256:     fmt.Sprintf("%x", hasher.Sum(nil)),
		StorageVersionID:   fileInfo.VersionID,
		PreviewStoragePath: "",
		Status:             document.StatusActive,
		Description:        req.Description,
		ResourceID:         req.ResourceID,
		ResourceType:       req.ResourceType,
		Tags:               req.Tags,
		UploadedByID:       req.TenantInfo.UserID,
		PreviewStatus:      previewStatusForFileType(contentType),
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

	if err = s.makeCurrentDocumentVersion(ctx, doc, lineageInfo); err != nil {
		log.Error("failed to create document record", zap.Error(err))
		if delErr := s.deleteStoredObject(ctx, storagePath, fileInfo.VersionID); delErr != nil {
			log.Error("failed to cleanup storage after db failure", zap.Error(delErr))
		}
		return nil, err
	}
	doc.IsCurrentVersion = true

	createdDoc := doc

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
	if lineageInfo != nil {
		s.deleteSearchProjection(ctx, lineageInfo)
	}
	s.syncSearchProjection(ctx, log, createdDoc, "")

	if s.thumbnailGenerator.SupportsThumbnail(contentType) {
		s.startThumbnailWorkflow(ctx, log, createdDoc)
	}
	_ = s.documentIntelligence.EnqueueExtraction(ctx, createdDoc, req.TenantInfo.UserID)

	return &UploadResult{Document: createdDoc}, nil
}

func (s *Service) startThumbnailWorkflow(
	ctx context.Context,
	log *zap.Logger,
	doc *document.Document,
) {
	if !s.workflowStarter.Enabled() {
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

	_, err := s.workflowStarter.StartWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:                                       workflowID,
			TaskQueue:                                temporaltype.ThumbnailTaskQueue,
			WorkflowExecutionErrorWhenAlreadyStarted: true,
			StaticSummary:                            fmt.Sprintf("Generating thumbnail for document %s", doc.ID),
		},
		"GenerateThumbnailWorkflow",
		payload,
	)
	if err != nil {
		var alreadyStarted *serviceerror.WorkflowExecutionAlreadyStarted
		if errors.As(err, &alreadyStarted) {
			return
		}
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
			LineageID:    req.LineageID,
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

	if doc.LineageID.IsNil() {
		if err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
			if txErr := s.detachUploadSessionsForDocuments(txCtx, []pulid.ID{doc.ID}, req.TenantInfo); txErr != nil {
				log.Error("failed to clear upload session document reference", zap.Error(txErr))
				return errortypes.NewDatabaseError("Failed to detach upload sessions from document").WithInternal(txErr)
			}
			return s.repo.Delete(txCtx, req)
		}); err != nil {
			return err
		}

		s.cleanupDocumentStorage(ctx, log, doc)
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

	versions, err := s.repo.ListVersions(ctx, repositories.ListDocumentVersionsRequest{
		LineageID:  doc.LineageID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return err
	}

	if err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		versionIDs := documentIDs(versions)
		lineageIDs := []pulid.ID{doc.LineageID}
		if txErr := s.detachUploadSessionsForDocuments(txCtx, versionIDs, req.TenantInfo); txErr != nil {
			log.Error("failed to clear upload session document reference", zap.Error(txErr))
			return errortypes.NewDatabaseError("Failed to detach upload sessions from document").WithInternal(txErr)
		}

		return s.repo.DeleteByLineageIDs(txCtx, repositories.DeleteDocumentLineageRequest{
			LineageIDs: lineageIDs,
			TenantInfo: req.TenantInfo,
		})
	}); err != nil {
		return err
	}

	for _, version := range versions {
		s.cleanupDocumentStorage(ctx, log, version)
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

	if slices.ContainsFunc(docs, func(doc *document.Document) bool { return doc.LineageID.IsNil() }) {
		if err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
			if txErr := s.detachUploadSessionsForDocuments(txCtx, documentIDs(docs), req.TenantInfo); txErr != nil {
				log.Error("failed to clear upload session document references during bulk delete", zap.Error(txErr))
				return errortypes.NewDatabaseError("Failed to detach upload sessions from documents").WithInternal(txErr)
			}
			return s.repo.BulkDelete(txCtx, repositories.BulkDeleteDocumentRequest{
				IDs:        req.IDs,
				TenantInfo: req.TenantInfo,
			})
		}); err != nil {
			log.Error("failed to bulk delete documents from database", zap.Error(err))
			return nil, err
		}

		result.DeletedCount = len(docs)
		for _, doc := range docs {
			s.cleanupDocumentStorage(ctx, log, doc)
		}
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

	docIDs := make([]pulid.ID, 0, len(docs))
	lineageIDs := make([]pulid.ID, 0, len(docs))
	lineageSet := make(map[string]struct{}, len(docs))
	versionsByLineage := make(map[string][]*document.Document, len(docs))
	for _, doc := range docs {
		versions, versionErr := s.repo.ListVersions(ctx, repositories.ListDocumentVersionsRequest{
			LineageID:  doc.LineageID,
			TenantInfo: req.TenantInfo,
		})
		if versionErr != nil {
			log.Error("failed to get document versions for bulk delete", zap.Error(versionErr))
			return nil, versionErr
		}
		versionsByLineage[doc.LineageID.String()] = versions
		for _, version := range versions {
			docIDs = append(docIDs, version.ID)
		}
		if _, ok := lineageSet[doc.LineageID.String()]; ok {
			continue
		}
		lineageSet[doc.LineageID.String()] = struct{}{}
		lineageIDs = append(lineageIDs, doc.LineageID)
	}

	if err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		if txErr := s.detachUploadSessionsForDocuments(txCtx, docIDs, req.TenantInfo); txErr != nil {
			log.Error("failed to clear upload session document references during bulk delete", zap.Error(txErr))
			return errortypes.NewDatabaseError("Failed to detach upload sessions from documents").WithInternal(txErr)
		}

		return s.repo.DeleteByLineageIDs(txCtx, repositories.DeleteDocumentLineageRequest{
			LineageIDs: lineageIDs,
			TenantInfo: req.TenantInfo,
		})
	}); err != nil {
		log.Error("failed to bulk delete documents from database", zap.Error(err))
		return nil, err
	}

	result.DeletedCount = len(docIDs)

	for _, versions := range versionsByLineage {
		for _, doc := range versions {
			s.cleanupDocumentStorage(ctx, log, doc)
		}
	}

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

func (s *Service) detachUploadSessionsForDocuments(
	ctx context.Context,
	documentIDs []pulid.ID,
	tenantInfo pagination.TenantInfo,
) error {
	if len(documentIDs) == 0 {
		return nil
	}

	db := s.db.DBForContext(ctx)
	if db == nil {
		if len(documentIDs) == 1 {
			return s.sessionRepo.ClearDocumentReference(ctx, documentIDs[0], tenantInfo)
		}
		return s.sessionRepo.ClearDocumentReferences(ctx, documentIDs, tenantInfo)
	}

	documentIDStrings := make([]string, 0, len(documentIDs))
	for _, id := range documentIDs {
		documentIDStrings = append(documentIDStrings, id.String())
	}

	if _, err := db.
		NewUpdate().
		Table("document_upload_sessions").
		Set("document_id = NULL").
		Where("document_id IN (?)", bun.In(documentIDStrings)).
		Where("organization_id = ?", tenantInfo.OrgID).
		Where("business_unit_id = ?", tenantInfo.BuID).
		Exec(ctx); err != nil {
		return err
	}

	remaining, err := db.
		NewSelect().
		Table("document_upload_sessions").
		Where("document_id IN (?)", bun.In(documentIDStrings)).
		Where("organization_id = ?", tenantInfo.OrgID).
		Where("business_unit_id = ?", tenantInfo.BuID).
		Count(ctx)
	if err != nil {
		return err
	}

	if remaining > 0 {
		return fmt.Errorf("document upload session references remain after detach: %d", remaining)
	}

	return nil
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

	if doc.PreviewStatus != document.PreviewStatusReady || doc.PreviewStoragePath == "" {
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

func (s *Service) cleanupDocumentStorage(ctx context.Context, log *zap.Logger, doc *document.Document) {
	failedPaths := make([]string, 0, 2)

	if err := s.deleteStoredObject(ctx, doc.StoragePath, doc.StorageVersionID); err != nil {
		log.Warn("failed to delete file from storage after document delete",
			zap.String("documentId", doc.ID.String()),
			zap.String("storagePath", doc.StoragePath),
			zap.Error(err),
		)
		failedPaths = append(failedPaths, doc.StoragePath)
	}

	if doc.PreviewStoragePath == "" {
		s.enqueueStorageCleanup(ctx, log, doc.ID, failedPaths)
		return
	}

	if err := s.deleteStoredObject(ctx, doc.PreviewStoragePath, ""); err != nil {
		log.Warn("failed to delete preview from storage after document delete",
			zap.String("documentId", doc.ID.String()),
			zap.String("previewStoragePath", doc.PreviewStoragePath),
			zap.Error(err),
		)
		failedPaths = append(failedPaths, doc.PreviewStoragePath)
	}

	s.enqueueStorageCleanup(ctx, log, doc.ID, failedPaths)
}

func (s *Service) deleteStoredObject(ctx context.Context, key, versionID string) error {
	if strings.TrimSpace(key) == "" {
		return nil
	}

	if strings.TrimSpace(versionID) == "" {
		exists, err := s.storage.Exists(ctx, key)
		if err != nil {
			return err
		}
		if !exists {
			return nil
		}

		fileInfo, err := s.storage.GetFileInfo(ctx, key)
		if err != nil {
			return err
		}
		versionID = fileInfo.VersionID
	}

	return s.storage.DeleteObject(ctx, &storage.DeleteObjectParams{
		Key:       key,
		VersionID: versionID,
	})
}

func (s *Service) syncSearchProjection(
	ctx context.Context,
	log *zap.Logger,
	doc *document.Document,
	contentText string,
) {
	if err := s.searchProjection.Upsert(ctx, doc, contentText); err != nil {
		log.Warn("failed to sync document search projection",
			zap.String("documentId", doc.ID.String()),
			zap.Error(err),
		)
	}
}

func previewStatusForFileType(contentType string) document.PreviewStatus {
	if document.SupportsPreview(contentType) {
		return document.PreviewStatusPending
	}

	return document.PreviewStatusUnsupported
}

func (s *Service) enqueueStorageCleanup(
	ctx context.Context,
	log *zap.Logger,
	documentID pulid.ID,
	paths []string,
) {
	if len(paths) == 0 || !s.workflowStarter.Enabled() {
		return
	}

	paths = slices.Compact(paths)
	_, err := s.workflowStarter.StartWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:            fmt.Sprintf("document-storage-cleanup-%s-%d", documentID.String(), time.Now().Unix()),
			TaskQueue:     temporaltype.UploadTaskQueue,
			StaticSummary: fmt.Sprintf("Cleaning up document storage for %s", documentID.String()),
		},
		"CleanupDocumentStorageWorkflow",
		&documentuploadjobs.CleanupDocumentStoragePayload{
			BasePayload: temporaltype.BasePayload{
				Timestamp: time.Now().Unix(),
			},
			DocumentID: documentID,
			Paths:      paths,
		},
	)
	if err != nil {
		log.Warn("failed to enqueue document storage cleanup",
			zap.String("documentId", documentID.String()),
			zap.Strings("paths", paths),
			zap.Error(err),
		)
	}
}
