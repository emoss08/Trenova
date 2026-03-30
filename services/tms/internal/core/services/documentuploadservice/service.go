package documentuploadservice

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/documentupload"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/ports/storage"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/core/services/documentservice"
	"github.com/emoss08/trenova/internal/core/services/thumbnailservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs/documentuploadjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/thumbnailjobs"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	multipartThreshold = 8 * 1024 * 1024
	defaultPartSize    = 5 * 1024 * 1024
	sessionTTL         = 24 * time.Hour
)

type Params struct {
	fx.In

	Logger             *zap.Logger
	SessionRepo        repositories.DocumentUploadSessionRepository
	DocumentRepo       repositories.DocumentRepository
	Storage            storage.Client
	Validator          *documentservice.Validator
	AuditService       services.AuditService
	Config             *config.Config
	ThumbnailGenerator *thumbnailservice.Generator
	WorkflowStarter    services.WorkflowStarter
	Redis              *goredis.Client `optional:"true"`
}

type Service struct {
	l                  *zap.Logger
	sessionRepo        repositories.DocumentUploadSessionRepository
	documentRepo       repositories.DocumentRepository
	storage            storage.Client
	validator          *documentservice.Validator
	auditService       services.AuditService
	config             *config.StorageConfig
	thumbnailGenerator *thumbnailservice.Generator
	workflowStarter    services.WorkflowStarter
	redis              *goredis.Client
}

type CreateSessionRequest struct {
	TenantInfo     pagination.TenantInfo
	ResourceID     string
	ResourceType   string
	FileName       string
	FileSize       int64
	ContentType    string
	Description    string
	Tags           []string
	DocumentTypeID string
	LineageID      string
}

type PartRequest struct {
	TenantInfo   pagination.TenantInfo
	SessionID    pulid.ID
	PartNumbers  []int
	ResourceID   string
	ResourceType string
}

type CompletionRequest struct {
	TenantInfo pagination.TenantInfo
	SessionID  pulid.ID
}

type CancelRequest struct {
	TenantInfo pagination.TenantInfo
	SessionID  pulid.ID
}

type PartUploadTarget struct {
	PartNumber int    `json:"partNumber"`
	URL        string `json:"url"`
}

type SessionState struct {
	Session *documentupload.Session `json:"session"`
	Parts   []storage.UploadedPart  `json:"parts"`
}

func New(p Params) *Service {
	return &Service{
		l:                  p.Logger.Named("service.document-upload"),
		sessionRepo:        p.SessionRepo,
		documentRepo:       p.DocumentRepo,
		storage:            p.Storage,
		validator:          p.Validator,
		auditService:       p.AuditService,
		config:             p.Config.GetStorageConfig(),
		thumbnailGenerator: p.ThumbnailGenerator,
		workflowStarter:    p.WorkflowStarter,
		redis:              p.Redis,
	}
}

func (s *Service) CreateSession(
	ctx context.Context,
	req *CreateSessionRequest,
) (*documentupload.Session, error) {
	if multiErr := s.validator.ValidateUploadMetadata(req.FileName, req.FileSize, req.ContentType); multiErr != nil {
		return nil, multiErr
	}

	session := &documentupload.Session{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		ResourceID:     req.ResourceID,
		ResourceType:   req.ResourceType,
		OriginalName:   req.FileName,
		ContentType:    req.ContentType,
		FileSize:       req.FileSize,
		StoragePath:    s.generateStoragePath(req.TenantInfo.OrgID.String(), req.ResourceType, req.FileName),
		Description:    req.Description,
		Tags:           make([]string, 0, len(req.Tags)),
		UploadedParts:  make([]documentupload.UploadedPart, 0),
		ExpiresAt:      time.Now().Add(sessionTTL).Unix(),
		LastActivityAt: time.Now().Unix(),
	}
	session.Tags = append(session.Tags, req.Tags...)

	if strings.TrimSpace(req.LineageID) != "" {
		lineageID, err := pulid.MustParse(req.LineageID)
		if err != nil {
			return nil, errortypes.NewValidationError("lineageId", errortypes.ErrInvalid, "Invalid lineage ID")
		}
		session.LineageID = &lineageID
	}

	if req.DocumentTypeID != "" {
		docTypeID, err := pulid.MustParse(req.DocumentTypeID)
		if err != nil {
			return nil, errortypes.NewValidationError(
				"documentTypeId", errortypes.ErrInvalid, "Invalid document type ID",
			)
		}
		session.DocumentTypeID = &docTypeID
	}

	if req.FileSize >= multipartThreshold {
		session.Strategy = documentupload.StrategyMultipart
		session.PartSize = defaultPartSize
		uploadID, err := s.storage.InitiateMultipartUpload(ctx, &storage.MultipartUploadParams{
			Key:         session.StoragePath,
			ContentType: req.ContentType,
			Metadata: map[string]string{
				"original_name": req.FileName,
				"resource_id":   req.ResourceID,
				"resource_type": req.ResourceType,
			},
		})
		if err != nil {
			return nil, errortypes.NewDatabaseError("Failed to initialize multipart upload").WithInternal(err)
		}
		session.StorageProviderUploadID = uploadID
		session.Status = documentupload.StatusUploading
	} else {
		session.Strategy = documentupload.StrategySingle
		session.PartSize = req.FileSize
		session.Status = documentupload.StatusInitiated
	}

	return s.sessionRepo.Create(ctx, session)
}

func (s *Service) ListActive(
	ctx context.Context,
	req *repositories.ListActiveDocumentUploadSessionsRequest,
) ([]*documentupload.Session, error) {
	sessions, err := s.sessionRepo.ListActive(ctx, req)
	if err != nil {
		return nil, err
	}

	for _, session := range sessions {
		normalizeSession(session)
	}

	return sessions, nil
}

func (s *Service) GetSessionState(
	ctx context.Context,
	req repositories.GetDocumentUploadSessionByIDRequest,
) (*SessionState, error) {
	session, err := s.sessionRepo.GetByID(ctx, req)
	if err != nil {
		return nil, err
	}

	parts, err := s.getUploadedParts(ctx, session)
	if err != nil {
		return nil, err
	}

	session.UploadedParts = toDomainParts(parts)
	normalizeSession(session)
	if parts == nil {
		parts = make([]storage.UploadedPart, 0)
	}

	return &SessionState{
		Session: session,
		Parts:   parts,
	}, nil
}

func (s *Service) GetPartUploadTargets(
	ctx context.Context,
	req *PartRequest,
) ([]PartUploadTarget, error) {
	session, err := s.sessionRepo.GetByID(ctx, repositories.GetDocumentUploadSessionByIDRequest{
		ID:         req.SessionID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	if session.Status.IsTerminal() {
		return nil, errortypes.NewConflictError("Upload session is no longer active")
	}

	partNumbers := req.PartNumbers
	if len(partNumbers) == 0 {
		partNumbers = []int{1}
	}

	targets := make([]PartUploadTarget, 0, len(partNumbers))
	if session.Strategy == documentupload.StrategySingle {
		url, err := s.storage.GetPresignedUploadURL(ctx, &storage.PresignedUploadURLParams{
			Key:         session.StoragePath,
			Expiry:      s.config.GetPresignedURLExpiry(),
			ContentType: session.ContentType,
		})
		if err != nil {
			return nil, errortypes.NewDatabaseError("Failed to generate upload URL").WithInternal(err)
		}

		targets = append(targets, PartUploadTarget{PartNumber: 1, URL: url})
	} else {
		for _, partNumber := range partNumbers {
			url, err := s.storage.GetMultipartUploadPartURL(ctx, &storage.MultipartUploadPartURLParams{
				Key:        session.StoragePath,
				UploadID:   session.StorageProviderUploadID,
				PartNumber: partNumber,
				Expiry:     s.config.GetPresignedURLExpiry(),
			})
			if err != nil {
				if isMissingMultipartUploadError(err) {
					_, _ = s.markSessionFailed(
						ctx,
						session,
						"MULTIPART_UPLOAD_MISSING",
						"Upload session is no longer active",
					)
					return nil, errortypes.NewConflictError("Upload session is no longer active")
				}
				return nil, errortypes.NewDatabaseError("Failed to generate upload URL").WithInternal(err)
			}
			targets = append(targets, PartUploadTarget{PartNumber: partNumber, URL: url})
		}
	}

	return targets, nil
}

func (s *Service) Complete(
	ctx context.Context,
	req *CompletionRequest,
) (*documentupload.Session, error) {
	session, err := s.sessionRepo.GetByID(ctx, repositories.GetDocumentUploadSessionByIDRequest{
		ID:         req.SessionID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	if session.DocumentID != nil && (session.Status == documentupload.StatusAvailable || session.Status == documentupload.StatusCompleted) {
		s.ensureThumbnailForSession(ctx, session)
		normalizeSession(session)
		return session, nil
	}

	if superseded, err := s.cancelSupersededSession(ctx, session); err != nil {
		return nil, err
	} else if superseded {
		return nil, errortypes.NewConflictError("Upload session was superseded by a newer version upload")
	}

	uploadedParts, err := s.getUploadedParts(ctx, session)
	if err != nil {
		return nil, err
	}

	if session.Strategy == documentupload.StrategyMultipart {
		if len(uploadedParts) == 0 {
			return nil, errortypes.NewConflictError("No uploaded parts were found for this session")
		}

		if sumUploadedPartSizes(uploadedParts) != session.FileSize {
			return nil, errortypes.NewConflictError("Uploaded file size does not match the original file")
		}
	}

	session.Status = documentupload.StatusUploaded
	session.UploadedParts = toDomainParts(uploadedParts)
	session.FailureCode = ""
	session.FailureMessage = ""
	session.LastActivityAt = time.Now().Unix()
	if _, err = s.sessionRepo.Update(ctx, session); err != nil {
		return nil, err
	}

	if !s.workflowStarter.Enabled() {
		if _, err = s.runSynchronousFinalization(ctx, req, session); err != nil {
			return nil, err
		}
		return s.sessionRepo.GetByID(ctx, repositories.GetDocumentUploadSessionByIDRequest{
			ID:         req.SessionID,
			TenantInfo: req.TenantInfo,
		})
	}

	if acquired, leaseErr := s.acquireCompletionLease(ctx, session.ID.String()); leaseErr != nil {
		s.l.Warn("failed to acquire upload completion lease", zap.Error(leaseErr), zap.String("sessionId", session.ID.String()))
		if err = s.startFinalizeWorkflow(ctx, req, session); err != nil {
			if _, updateErr := s.markSessionFailed(
				ctx,
				session,
				"UPLOAD_FINALIZATION_FAILED",
				"Upload finalization failed",
			); updateErr != nil {
				s.l.Warn("failed to mark upload session as failed", zap.Error(updateErr), zap.String("sessionId", session.ID.String()))
			}
			return nil, err
		}
	} else if acquired {
		if err = s.startFinalizeWorkflow(ctx, req, session); err != nil {
			if _, updateErr := s.markSessionFailed(
				ctx,
				session,
				"UPLOAD_FINALIZATION_FAILED",
				"Upload finalization failed",
			); updateErr != nil {
				s.l.Warn("failed to mark upload session as failed", zap.Error(updateErr), zap.String("sessionId", session.ID.String()))
			}
			return nil, err
		}
	}

	normalizeSession(session)
	return session, nil
}

func (s *Service) Cancel(ctx context.Context, req *CancelRequest) error {
	session, err := s.sessionRepo.GetByID(ctx, repositories.GetDocumentUploadSessionByIDRequest{
		ID:         req.SessionID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return err
	}

	if session.Status == documentupload.StatusCompleted {
		return errortypes.NewConflictError("Completed uploads cannot be canceled")
	}

	if session.Strategy == documentupload.StrategyMultipart && session.StorageProviderUploadID != "" {
		if err = s.storage.AbortMultipartUpload(ctx, &storage.AbortMultipartUploadParams{
			Key:      session.StoragePath,
			UploadID: session.StorageProviderUploadID,
		}); err != nil {
			return errortypes.NewDatabaseError("Failed to cancel multipart upload").WithInternal(err)
		}
	} else {
		_ = s.storage.Delete(ctx, session.StoragePath)
	}

	session.Status = documentupload.StatusCanceled
	session.LastActivityAt = time.Now().Unix()
	_, err = s.sessionRepo.Update(ctx, session)
	return err
}

func (s *Service) getUploadedParts(
	ctx context.Context,
	session *documentupload.Session,
) ([]storage.UploadedPart, error) {
	if session.Status == documentupload.StatusCompleted ||
		session.Status == documentupload.StatusAvailable ||
		session.Status == documentupload.StatusQuarantined ||
		session.Status == documentupload.StatusFailed ||
		session.Status == documentupload.StatusCanceled ||
		session.Status == documentupload.StatusExpired {
		if len(session.UploadedParts) > 0 {
			return fromDomainParts(session.UploadedParts), nil
		}
	}

	if session.Strategy == documentupload.StrategySingle {
		exists, err := s.storage.Exists(ctx, session.StoragePath)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, nil
		}

		fileInfo, err := s.storage.GetFileInfo(ctx, session.StoragePath)
		if err != nil {
			return nil, err
		}

		return []storage.UploadedPart{{
			PartNumber: 1,
			Size:       fileInfo.Size,
		}}, nil
	}

	parts, err := s.storage.ListMultipartUploadParts(ctx, &storage.ListMultipartUploadPartsParams{
		Key:      session.StoragePath,
		UploadID: session.StorageProviderUploadID,
	})
	if err != nil {
		if isMissingMultipartUploadError(err) {
			if len(session.UploadedParts) > 0 {
				return fromDomainParts(session.UploadedParts), nil
			}

			exists, existsErr := s.storage.Exists(ctx, session.StoragePath)
			if existsErr != nil {
				return nil, existsErr
			}
			if !exists {
				return nil, nil
			}

			fileInfo, fileInfoErr := s.storage.GetFileInfo(ctx, session.StoragePath)
			if fileInfoErr != nil {
				return nil, fileInfoErr
			}

			return []storage.UploadedPart{{
				PartNumber: 1,
				Size:       fileInfo.Size,
			}}, nil
		}

		return nil, err
	}

	return parts, nil
}

func (s *Service) markSessionFailed(
	ctx context.Context,
	session *documentupload.Session,
	code string,
	message string,
) (*documentupload.Session, error) {
	session.Status = documentupload.StatusFailed
	session.FailureCode = code
	session.FailureMessage = message
	session.LastActivityAt = time.Now().Unix()
	return s.sessionRepo.Update(ctx, session)
}

func fromDomainParts(parts []documentupload.UploadedPart) []storage.UploadedPart {
	result := make([]storage.UploadedPart, 0, len(parts))
	for _, part := range parts {
		result = append(result, storage.UploadedPart{
			PartNumber: part.PartNumber,
			ETag:       part.ETag,
			Size:       part.Size,
		})
	}
	return result
}

func sumUploadedPartSizes(parts []storage.UploadedPart) int64 {
	var total int64
	for _, part := range parts {
		total += part.Size
	}
	return total
}

func isMissingMultipartUploadError(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "multipart upload does not exist")
}

func normalizeSession(session *documentupload.Session) {
	if session == nil {
		return
	}

	if session.Tags == nil {
		session.Tags = make([]string, 0)
	}

	if session.UploadedParts == nil {
		session.UploadedParts = make([]documentupload.UploadedPart, 0)
	}
}

func (s *Service) acquireCompletionLease(ctx context.Context, sessionID string) (bool, error) {
	if s.redis == nil {
		return true, nil
	}

	return s.redis.SetNX(ctx, completionLeaseKey(sessionID), "1", 5*time.Minute).Result()
}

func (s *Service) startFinalizeWorkflow(
	ctx context.Context,
	req *CompletionRequest,
	session *documentupload.Session,
) error {
	_, err := s.workflowStarter.StartWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:                                       fmt.Sprintf("document-upload-finalize-%s", session.ID.String()),
			TaskQueue:                                temporaltype.UploadTaskQueue,
			WorkflowExecutionErrorWhenAlreadyStarted: true,
			StaticSummary:                            fmt.Sprintf("Finalizing upload %s", session.ID.String()),
		},
		"FinalizeDocumentUploadWorkflow",
		&documentuploadjobs.FinalizeUploadPayload{
			BasePayload: temporaltype.BasePayload{
				OrganizationID: req.TenantInfo.OrgID,
				BusinessUnitID: req.TenantInfo.BuID,
				UserID:         req.TenantInfo.UserID,
				Timestamp:      time.Now().Unix(),
			},
			SessionID: session.ID,
		},
	)
	if err != nil {
		var alreadyStarted *serviceerror.WorkflowExecutionAlreadyStarted
		if errors.As(err, &alreadyStarted) {
			return nil
		}

		return errortypes.NewDatabaseError("Failed to start upload finalization").WithInternal(err)
	}

	return nil
}

func (s *Service) runSynchronousFinalization(
	ctx context.Context,
	req *CompletionRequest,
	session *documentupload.Session,
) (*document.Document, error) {
	uploadedParts, err := s.getUploadedParts(ctx, session)
	if err != nil {
		return nil, err
	}

	if session.Strategy == documentupload.StrategyMultipart {
		if err = s.storage.CompleteMultipartUpload(ctx, &storage.CompleteMultipartUploadParams{
			Key:      session.StoragePath,
			UploadID: session.StorageProviderUploadID,
			Parts:    uploadedParts,
		}); err != nil {
			return nil, errortypes.NewDatabaseError("Failed to finalize multipart upload").WithInternal(err)
		}
	}

	fileInfo, err := s.storage.GetFileInfo(ctx, session.StoragePath)
	if err != nil {
		return nil, errortypes.NewDatabaseError("Failed to verify uploaded file").WithInternal(err)
	}

	if fileInfo.Size != session.FileSize {
		return nil, errortypes.NewConflictError("Uploaded file size does not match the original file")
	}

	doc := &document.Document{
		OrganizationID:     session.OrganizationID,
		BusinessUnitID:     session.BusinessUnitID,
		FileName:           filepath.Base(session.StoragePath),
		OriginalName:       session.OriginalName,
		FileSize:           session.FileSize,
		FileType:           session.ContentType,
		StoragePath:        session.StoragePath,
		Status:             document.StatusActive,
		Description:        session.Description,
		ResourceID:         session.ResourceID,
		ResourceType:       session.ResourceType,
		Tags:               session.Tags,
		UploadedByID:       req.TenantInfo.UserID,
		PreviewStoragePath: "",
		PreviewStatus:      previewStatusForFileType(session.ContentType),
	}
	if session.DocumentTypeID != nil {
		doc.DocumentTypeID = session.DocumentTypeID
	}

	createdDoc, err := s.documentRepo.Create(ctx, doc)
	if err != nil {
		return nil, err
	}

	session.DocumentID = &createdDoc.ID
	session.Status = documentupload.StatusAvailable
	session.UploadedParts = toDomainParts(uploadedParts)
	session.LastActivityAt = time.Now().Unix()
	if _, err = s.sessionRepo.Update(ctx, session); err != nil {
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
		s.l.Error("failed to log document upload completion", zap.Error(err))
	}

	if s.thumbnailGenerator.SupportsThumbnail(createdDoc.FileType) {
		s.startThumbnailWorkflow(ctx, createdDoc)
	}

	return createdDoc, nil
}

func completionLeaseKey(sessionID string) string {
	return "document-upload:completion:" + sessionID
}

func (s *Service) cancelSupersededSession(
	ctx context.Context,
	session *documentupload.Session,
) (bool, error) {
	if session == nil || session.LineageID == nil || session.LineageID.IsNil() || session.Status.IsTerminal() {
		return false, nil
	}

	superseded, err := s.isSupersededByNewerVersion(ctx, session)
	if err != nil || !superseded {
		return false, err
	}

	session.MarkSuperseded(time.Now().Unix())
	if _, err = s.sessionRepo.Update(ctx, session); err != nil {
		return false, err
	}

	return true, nil
}

func (s *Service) isSupersededByNewerVersion(
	ctx context.Context,
	session *documentupload.Session,
) (bool, error) {
	if session == nil || session.LineageID == nil || session.LineageID.IsNil() {
		return false, nil
	}

	activeSessions, err := s.sessionRepo.ListActive(ctx, &repositories.ListActiveDocumentUploadSessionsRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: session.OrganizationID,
			BuID:  session.BusinessUnitID,
		},
		ResourceID:   session.ResourceID,
		ResourceType: session.ResourceType,
	})
	if err != nil {
		return false, err
	}

	versions, err := s.documentRepo.ListVersions(ctx, repositories.ListDocumentVersionsRequest{
		LineageID: *session.LineageID,
		TenantInfo: pagination.TenantInfo{
			OrgID: session.OrganizationID,
			BuID:  session.BusinessUnitID,
		},
	})
	if err != nil {
		return false, err
	}

	return session.IsSupersededByNewerArtifacts(activeSessions, versions), nil
}

func toDomainParts(parts []storage.UploadedPart) []documentupload.UploadedPart {
	result := make([]documentupload.UploadedPart, 0, len(parts))
	for _, part := range parts {
		result = append(result, documentupload.UploadedPart{
			PartNumber: part.PartNumber,
			ETag:       part.ETag,
			Size:       part.Size,
		})
	}
	slices.SortFunc(result, func(a, b documentupload.UploadedPart) int {
		return a.PartNumber - b.PartNumber
	})
	return result
}

func (s *Service) startThumbnailWorkflow(ctx context.Context, doc *document.Document) {
	if !s.workflowStarter.Enabled() {
		return
	}

	if doc.PreviewStatus != document.PreviewStatusPending || doc.PreviewStoragePath != "" {
		doc.PreviewStatus = document.PreviewStatusPending
		doc.PreviewStoragePath = ""
		if err := s.documentRepo.UpdatePreview(ctx, &repositories.UpdateDocumentPreviewRequest{
			ID: doc.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID: doc.OrganizationID,
				BuID:  doc.BusinessUnitID,
			},
			PreviewStatus:      document.PreviewStatusPending,
			PreviewStoragePath: "",
		}); err != nil {
			s.l.Warn("failed to mark document preview as pending", zap.Error(err), zap.String("documentId", doc.ID.String()))
		}
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
			WorkflowIDReusePolicy:                    enumspb.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
			WorkflowExecutionErrorWhenAlreadyStarted: true,
			StaticSummary:                            fmt.Sprintf("Generating thumbnail for document %s", doc.ID),
		},
		"GenerateThumbnailWorkflow",
		payload,
	)
	if err != nil {
		s.l.Warn("failed to start thumbnail workflow", zap.Error(err), zap.String("documentId", doc.ID.String()))
		var alreadyStarted *serviceerror.WorkflowExecutionAlreadyStarted
		if !errors.As(err, &alreadyStarted) {
			if updateErr := s.documentRepo.UpdatePreview(ctx, &repositories.UpdateDocumentPreviewRequest{
				ID: doc.ID,
				TenantInfo: pagination.TenantInfo{
					OrgID: doc.OrganizationID,
					BuID:  doc.BusinessUnitID,
				},
				PreviewStatus:      document.PreviewStatusFailed,
				PreviewStoragePath: "",
			}); updateErr != nil {
				s.l.Warn("failed to mark document preview as failed", zap.Error(updateErr), zap.String("documentId", doc.ID.String()))
			}
		}
	}
}

func (s *Service) generateStoragePath(orgID, resourceType, filename string) string {
	ext := filepath.Ext(filename)
	uniqueID := uuid.New().String()
	return fmt.Sprintf("%s/%s/%s%s", orgID, resourceType, uniqueID, strings.ToLower(ext))
}

func previewStatusForFileType(contentType string) document.PreviewStatus {
	if document.SupportsPreview(contentType) {
		return document.PreviewStatusPending
	}

	return document.PreviewStatusUnsupported
}

func (s *Service) ensureThumbnailForSession(
	ctx context.Context,
	session *documentupload.Session,
) {
	if session.DocumentID == nil || s.thumbnailGenerator == nil {
		return
	}

	doc, err := s.documentRepo.GetByID(ctx, repositories.GetDocumentByIDRequest{
		ID: *session.DocumentID,
		TenantInfo: pagination.TenantInfo{
			OrgID: session.OrganizationID,
			BuID:  session.BusinessUnitID,
		},
	})
	if err != nil {
		s.l.Warn("failed to load document for thumbnail reconciliation",
			zap.Error(err),
			zap.String("sessionId", session.ID.String()),
		)
		return
	}

	if doc.PreviewStatus == document.PreviewStatusReady || !document.SupportsPreview(doc.FileType) {
		return
	}

	s.startThumbnailWorkflow(ctx, doc)
}
