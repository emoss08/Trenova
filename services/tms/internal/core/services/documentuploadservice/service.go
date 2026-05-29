package documentuploadservice

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
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
	"github.com/emoss08/trenova/internal/core/services/encryptionservice"
	"github.com/emoss08/trenova/internal/core/services/thumbnailservice"
	"github.com/emoss08/trenova/internal/core/services/usageservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs/documentuploadjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/thumbnailjobs"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/fileutils"
	"github.com/emoss08/trenova/shared/hashutils"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
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
	UsageProvider      services.UsageProvider `optional:"true"`
	DocTypeRepo        repositories.DocumentTypeRepository
	Redis              *goredis.Client `optional:"true"`
	Encryption         *encryptionservice.Service
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
	usageProvider      services.UsageProvider
	docTypeRepo        repositories.DocumentTypeRepository
	redis              *goredis.Client
	encryption         *encryptionservice.Service
}

//nolint:gocritic // dependency injection param
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
		usageProvider:      p.UsageProvider,
		docTypeRepo:        p.DocTypeRepo,
		redis:              p.Redis,
		encryption:         p.Encryption,
	}
}

type UploadPartRequest struct {
	TenantInfo pagination.TenantInfo
	SessionID  pulid.ID
	PartNumber int
	Body       io.Reader
	Size       int64
}

func (s *Service) CreateSession(
	ctx context.Context,
	req *services.CreateSessionRequest,
) (*documentupload.DocumentUploadSession, error) {
	processingProfile, err := document.NormalizeProcessingProfile(req.ProcessingProfile)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"processingProfile",
			errortypes.ErrInvalid,
			"Invalid processing profile",
		)
	}

	if multiErr := s.validator.ValidateUploadMetadata(
		req.FileName,
		req.FileSize,
		req.ContentType,
	); multiErr != nil {
		return nil, multiErr
	}

	tags := make([]string, 0, len(req.Tags))
	session := &documentupload.DocumentUploadSession{
		ID:                pulid.MustNew("dus_"),
		OrganizationID:    req.TenantInfo.OrgID,
		BusinessUnitID:    req.TenantInfo.BuID,
		ResourceID:        req.ResourceID,
		ResourceType:      req.ResourceType,
		ProcessingProfile: processingProfile,
		OriginalName:      req.FileName,
		ContentType:       req.ContentType,
		FileSize:          req.FileSize,
		StoragePath: fileutils.GenerateStoragePath(
			req.TenantInfo.OrgID.String(),
			req.ResourceType,
			req.FileName,
		),
		Description:    req.Description,
		Tags:           tags,
		UploadedParts:  make([]storage.UploadedPart, 0),
		ExpiresAt:      timeutils.NowAddDuration(sessionTTL),
		LastActivityAt: timeutils.NowUnix(),
	}
	session.Tags = append(session.Tags, req.Tags...)

	if err = usageservice.CheckDocumentUploadLimit(
		ctx,
		s.usageProvider,
		usageservice.DocumentUploadUsageParams{
			TenantInfo:     req.TenantInfo,
			Actor:          req.Actor,
			IdempotencyKey: "document-upload-session:" + session.ID.String(),
		},
	); err != nil {
		return nil, err
	}

	if strings.TrimSpace(req.LineageID) != "" {
		lineageID, lineageErr := pulid.MustParse(req.LineageID)
		if lineageErr != nil {
			return nil, errortypes.NewValidationError(
				"lineageId",
				errortypes.ErrInvalid,
				"Invalid lineage ID",
			)
		}

		session.LineageID = &lineageID
	}

	if req.DocumentTypeID != "" {
		docTypeID, docTypeIDErr := pulid.MustParse(req.DocumentTypeID)
		if docTypeIDErr != nil {
			return nil, errortypes.NewValidationError(
				"documentTypeId", errortypes.ErrInvalid, "Invalid document type ID",
			)
		}
		session.DocumentTypeID = &docTypeID
	}

	if session.DocumentTypeID == nil &&
		processingProfile == document.ProcessingProfileRateConfirmationImport {
		dt, dtErr := s.docTypeRepo.GetByCode(ctx, repositories.GetDocumentTypeByCodeRequest{
			Code:       "RATECONF",
			TenantInfo: req.TenantInfo,
		})
		if dtErr == nil {
			session.DocumentTypeID = &dt.ID
		}
	}

	session.Strategy = documentupload.StrategySingle
	session.PartSize = req.FileSize
	session.Status = documentupload.StatusInitiated

	return s.sessionRepo.Create(ctx, session)
}

func (s *Service) ListActive(
	ctx context.Context,
	req *repositories.ListActiveDocumentUploadSessionsRequest,
) ([]*documentupload.DocumentUploadSession, error) {
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
) (*services.SessionState, error) {
	session, err := s.sessionRepo.GetByID(ctx, req)
	if err != nil {
		return nil, err
	}

	parts, err := s.getUploadedParts(ctx, session)
	if err != nil {
		return nil, err
	}

	session.UploadedParts = storage.ToDomainParts(parts)
	normalizeSession(session)
	if parts == nil {
		parts = make([]storage.UploadedPart, 0)
	}

	return &services.SessionState{
		Session: session,
		Parts:   parts,
	}, nil
}

func (s *Service) GetPartUploadTargets(
	ctx context.Context,
	req *services.PartRequest,
) ([]services.PartUploadTarget, error) {
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

	if session.Strategy == documentupload.StrategySingle {
		return s.partUploadTargetsSingle(ctx, session)
	}
	return s.partUploadTargetsMultipart(session, partNumbers)
}

func (s *Service) partUploadTargetsSingle(
	ctx context.Context,
	session *documentupload.DocumentUploadSession,
) ([]services.PartUploadTarget, error) {
	_ = ctx
	return []services.PartUploadTarget{{
		PartNumber: 1,
		URL:        uploadPartAPIPath(session.ID, 1),
		Method:     "PUT",
	}}, nil
}

func (s *Service) partUploadTargetsMultipart(
	session *documentupload.DocumentUploadSession,
	partNumbers []int,
) ([]services.PartUploadTarget, error) {
	targets := make([]services.PartUploadTarget, 0, len(partNumbers))
	for _, partNumber := range partNumbers {
		targets = append(targets, services.PartUploadTarget{
			PartNumber: partNumber,
			URL:        uploadPartAPIPath(session.ID, partNumber),
			Method:     "PUT",
		})
	}
	return targets, nil
}

func (s *Service) UploadPart(
	ctx context.Context,
	req *UploadPartRequest,
) (*documentupload.DocumentUploadSession, error) {
	if req.PartNumber != 1 {
		return nil, errortypes.NewValidationError(
			"partNumber",
			errortypes.ErrInvalid,
			"Encrypted upload sessions accept a single API-mediated part",
		)
	}

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
	if req.Size > 0 && req.Size != session.FileSize {
		return nil, errortypes.NewConflictError(
			"Uploaded file size does not match the original file",
		)
	}

	plaintext, err := io.ReadAll(io.LimitReader(req.Body, session.FileSize+1))
	if err != nil {
		return nil, errortypes.NewDatabaseError("Failed to read uploaded file").WithInternal(err)
	}
	if int64(len(plaintext)) != session.FileSize {
		return nil, errortypes.NewConflictError(
			"Uploaded file size does not match the original file",
		)
	}

	if s.encryption == nil {
		return nil, errortypes.NewBusinessError("Document encryption is not configured")
	}
	encrypted, encErr := s.encryption.EncryptBytesWithAAD(
		plaintext,
		uploadSessionStorageAAD(session),
	)
	if encErr != nil {
		return nil, errortypes.NewBusinessError("Failed to encrypt document upload").
			WithInternal(encErr)
	}
	uploadBody := []byte(encrypted)
	cryptoMode := encryptionservice.CryptoModeForCiphertext(encrypted)
	cryptoVersion := int16(1)

	if _, err = s.storage.Upload(ctx, &storage.UploadParams{
		Key:         session.StoragePath,
		ContentType: session.ContentType,
		Size:        int64(len(uploadBody)),
		Body:        bytes.NewReader(uploadBody),
		Metadata: map[string]string{
			"original_name": session.OriginalName,
			"resource_id":   session.ResourceID,
			"resource_type": session.ResourceType,
			"crypto_mode":   cryptoMode,
		},
	}); err != nil {
		return nil, errortypes.NewDatabaseError("Failed to upload document").WithInternal(err)
	}

	session.Status = documentupload.StatusUploaded
	session.ChecksumSHA256 = hashutils.SHA256BytesHex(plaintext)
	session.CryptoMode = cryptoMode
	session.CryptoVersion = cryptoVersion
	session.UploadedParts = []storage.UploadedPart{{
		PartNumber: 1,
		Size:       session.FileSize,
	}}
	session.LastActivityAt = timeutils.NowUnix()
	return s.sessionRepo.Update(ctx, session)
}

func (s *Service) Complete(
	ctx context.Context,
	req *services.CompletionRequest,
) (*documentupload.DocumentUploadSession, error) {
	session, err := s.sessionRepo.GetByID(ctx, repositories.GetDocumentUploadSessionByIDRequest{
		ID:         req.SessionID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	if session.DocumentID != nil && session.Status.IsDocumentReady() {
		s.ensureThumbnailForSession(ctx, session)
		normalizeSession(session)
		return session, nil
	}

	if err = usageservice.CheckDocumentUploadLimit(
		ctx,
		s.usageProvider,
		usageservice.DocumentUploadUsageParams{
			TenantInfo:     req.TenantInfo,
			Actor:          req.Actor,
			IdempotencyKey: "document-upload-session:" + req.SessionID.String(),
		},
	); err != nil {
		return nil, err
	}

	if superseded, supersededErr := s.cancelSupersededSession(ctx, session); supersededErr != nil {
		return nil, supersededErr
	} else if superseded {
		return nil, errortypes.NewConflictError(
			"Upload session was superseded by a newer version upload",
		)
	}

	uploadedParts, err := s.getUploadedParts(ctx, session)
	if err != nil {
		return nil, err
	}

	if session.Strategy == documentupload.StrategyMultipart {
		if len(uploadedParts) == 0 {
			return nil, errortypes.NewConflictError("No uploaded parts were found for this session")
		}

		if storage.SumUploadedPartSizes(uploadedParts) != session.FileSize {
			return nil, errortypes.NewConflictError(
				"Uploaded file size does not match the original file",
			)
		}
	}

	session.Status = documentupload.StatusUploaded
	session.UploadedParts = storage.ToDomainParts(uploadedParts)
	session.FailureCode = ""
	session.FailureMessage = ""
	session.LastActivityAt = timeutils.NowUnix()
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

	acquired, leaseErr := s.acquireCompletionLease(ctx, session.ID.String())
	if leaseErr != nil {
		s.l.Warn(
			"failed to acquire upload completion lease",
			zap.Error(leaseErr),
			zap.String("sessionId", session.ID.String()),
		)
	}
	if leaseErr != nil || acquired {
		if err = s.startFinalizeWorkflowOrMarkFailed(ctx, req, session); err != nil {
			return nil, err
		}
	}

	normalizeSession(session)
	return session, nil
}

func (s *Service) Cancel(ctx context.Context, req *services.CancelRequest) error {
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

	if session.Strategy == documentupload.StrategyMultipart &&
		session.StorageProviderUploadID != "" {
		if err = s.storage.AbortMultipartUpload(ctx, &storage.AbortMultipartUploadParams{
			Key:      session.StoragePath,
			UploadID: session.StorageProviderUploadID,
		}); err != nil {
			return errortypes.NewDatabaseError("Failed to cancel multipart upload").
				WithInternal(err)
		}
	} else {
		_ = s.storage.Delete(ctx, session.StoragePath)
	}

	session.Status = documentupload.StatusCanceled
	session.LastActivityAt = timeutils.NowUnix()
	_, err = s.sessionRepo.Update(ctx, session)
	return err
}

func (s *Service) getUploadedParts(
	ctx context.Context,
	session *documentupload.DocumentUploadSession,
) ([]storage.UploadedPart, error) {
	if len(session.UploadedParts) > 0 {
		return storage.ToDomainParts(session.UploadedParts), nil
	}

	if session.Strategy == documentupload.StrategySingle {
		return s.uploadedPartsFromSingleObject(ctx, session.StoragePath)
	}

	parts, err := s.storage.ListMultipartUploadParts(ctx, &storage.ListMultipartUploadPartsParams{
		Key:      session.StoragePath,
		UploadID: session.StorageProviderUploadID,
	})
	if err == nil {
		return parts, nil
	}
	if !storage.IsMissingMultipartUploadError(err) {
		return nil, err
	}
	if len(session.UploadedParts) > 0 {
		return storage.ToDomainParts(session.UploadedParts), nil
	}

	return s.uploadedPartsFromSingleObject(ctx, session.StoragePath)
}

func (s *Service) uploadedPartsFromSingleObject(
	ctx context.Context,
	storagePath string,
) ([]storage.UploadedPart, error) {
	exists, err := s.storage.Exists(ctx, storagePath)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}

	fileInfo, err := s.storage.GetFileInfo(ctx, storagePath)
	if err != nil {
		return nil, err
	}

	return []storage.UploadedPart{{
		PartNumber: 1,
		Size:       fileInfo.Size,
	}}, nil
}

func (s *Service) markSessionFailed(
	ctx context.Context,
	session *documentupload.DocumentUploadSession,
	code string,
	message string,
) (*documentupload.DocumentUploadSession, error) {
	session.Status = documentupload.StatusFailed
	session.FailureCode = code
	session.FailureMessage = message
	session.LastActivityAt = timeutils.NowUnix()
	return s.sessionRepo.Update(ctx, session)
}

func normalizeSession(session *documentupload.DocumentUploadSession) {
	if session == nil {
		return
	}

	if session.Tags == nil {
		session.Tags = make([]string, 0)
	}

	if session.UploadedParts == nil {
		session.UploadedParts = make([]storage.UploadedPart, 0)
	}
}

func (s *Service) acquireCompletionLease(ctx context.Context, sessionID string) (bool, error) {
	if s.redis == nil {
		return true, nil
	}

	_, err := s.redis.SetArgs(ctx, completionLeaseKey(sessionID), "1", goredis.SetArgs{
		Mode: "NX",
		TTL:  5 * time.Minute,
	}).Result()
	if err != nil {
		if errors.Is(err, goredis.Nil) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (s *Service) startFinalizeWorkflowOrMarkFailed(
	ctx context.Context,
	req *services.CompletionRequest,
	session *documentupload.DocumentUploadSession,
) error {
	err := s.startFinalizeWorkflow(ctx, req, session)
	if err == nil {
		return nil
	}
	if _, updateErr := s.markSessionFailed(
		ctx,
		session,
		documentupload.FailureCodeUploadFinalizationFailed.String(),
		"Upload finalization failed",
	); updateErr != nil {
		s.l.Warn(
			"failed to mark upload session as failed",
			zap.Error(updateErr),
			zap.String("sessionId", session.ID.String()),
		)
	}
	return err
}

func (s *Service) startFinalizeWorkflow(
	ctx context.Context,
	req *services.CompletionRequest,
	session *documentupload.DocumentUploadSession,
) error {
	_, err := s.workflowStarter.StartWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID: fmt.Sprintf(
				"document-upload-finalize-%s",
				session.ID.String(),
			),
			TaskQueue:                                temporaltype.UploadTaskQueue,
			WorkflowExecutionErrorWhenAlreadyStarted: true,
			StaticSummary: fmt.Sprintf(
				"Finalizing upload %s",
				session.ID.String(),
			),
		},
		"FinalizeDocumentUploadWorkflow",
		&documentuploadjobs.FinalizeUploadPayload{
			BasePayload: temporaltype.BasePayload{
				OrganizationID: req.TenantInfo.OrgID,
				BusinessUnitID: req.TenantInfo.BuID,
				UserID:         req.TenantInfo.UserID,
				Timestamp:      timeutils.NowUnix(),
			},
			SessionID:     session.ID,
			PrincipalType: req.Actor.PrincipalType,
			PrincipalID:   req.Actor.PrincipalID,
			APIKeyID:      req.Actor.APIKeyID,
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
	req *services.CompletionRequest,
	session *documentupload.DocumentUploadSession,
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
			return nil, errortypes.NewDatabaseError("Failed to finalize multipart upload").
				WithInternal(err)
		}
	}

	if _, err := s.storage.GetFileInfo(ctx, session.StoragePath); err != nil {
		return nil, errortypes.NewDatabaseError("Failed to verify uploaded file").WithInternal(err)
	}

	if session.CryptoMode != encryptionservice.CryptoModeEnvelopeV1 {
		return nil, errortypes.NewConflictError("Uploaded file is not encrypted")
	}
	if len(session.UploadedParts) == 0 {
		return nil, errortypes.NewConflictError(
			"Encrypted upload session is missing verified plaintext part metadata",
		)
	}

	doc := &document.Document{
		OrganizationID:     session.OrganizationID,
		BusinessUnitID:     session.BusinessUnitID,
		FileName:           filepath.Base(session.StoragePath),
		OriginalName:       session.OriginalName,
		FileSize:           session.FileSize,
		FileType:           session.ContentType,
		StoragePath:        session.StoragePath,
		ChecksumSHA256:     session.ChecksumSHA256,
		CryptoMode:         session.CryptoMode,
		CryptoVersion:      session.CryptoVersion,
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
	session.UploadedParts = storage.ToDomainParts(uploadedParts)
	session.LastActivityAt = timeutils.NowUnix()
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
	s.recordDocumentUploadUsage(ctx, createdDoc, req.TenantInfo, &req.Actor)

	return createdDoc, nil
}

func (s *Service) recordDocumentUploadUsage(
	ctx context.Context,
	doc *document.Document,
	tenantInfo pagination.TenantInfo,
	actor *services.RequestActor,
) {
	if _, err := usageservice.RecordDocumentUpload(
		ctx,
		s.usageProvider,
		usageservice.DocumentUploadUsageParams{
			TenantInfo: tenantInfo,
			Actor:      *actor,
			DocumentID: doc.ID,
		},
	); err != nil {
		s.l.Warn(
			"failed to record document upload usage",
			zap.String("documentId", doc.ID.String()),
			zap.Error(err),
		)
	}
}

func completionLeaseKey(sessionID string) string {
	return "document-upload:completion:" + sessionID
}

func uploadPartAPIPath(sessionID pulid.ID, partNumber int) string {
	return fmt.Sprintf("/api/v1/documents/uploads/%s/parts/%d/", sessionID.String(), partNumber)
}

func uploadSessionStorageAAD(session *documentupload.DocumentUploadSession) encryptionservice.AAD {
	return encryptionservice.AAD{
		Purpose:        encryptionservice.PurposeDocument,
		OrganizationID: session.OrganizationID,
		BusinessUnitID: session.BusinessUnitID,
		ResourceID:     session.StoragePath,
	}
}

func (s *Service) cancelSupersededSession(
	ctx context.Context,
	session *documentupload.DocumentUploadSession,
) (bool, error) {
	if session == nil || session.LineageID == nil || session.LineageID.IsNil() ||
		session.Status.IsTerminal() {
		return false, nil
	}

	superseded, err := s.isSupersededByNewerVersion(ctx, session)
	if err != nil || !superseded {
		return false, err
	}

	session.MarkSuperseded(timeutils.NowUnix())
	if _, err = s.sessionRepo.Update(ctx, session); err != nil {
		return false, err
	}

	return true, nil
}

func (s *Service) isSupersededByNewerVersion(
	ctx context.Context,
	session *documentupload.DocumentUploadSession,
) (bool, error) {
	if session == nil || session.LineageID == nil || session.LineageID.IsNil() {
		return false, nil
	}

	activeSessions, err := s.sessionRepo.ListActive(
		ctx,
		&repositories.ListActiveDocumentUploadSessionsRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: session.OrganizationID,
				BuID:  session.BusinessUnitID,
			},
			ResourceID:   session.ResourceID,
			ResourceType: session.ResourceType,
		},
	)
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
			s.l.Warn(
				"failed to mark document preview as pending",
				zap.Error(err),
				zap.String("documentId", doc.ID.String()),
			)
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
			StaticSummary: fmt.Sprintf(
				"Generating thumbnail for document %s",
				doc.ID,
			),
		},
		"GenerateThumbnailWorkflow",
		payload,
	)
	if err != nil {
		s.l.Warn(
			"failed to start thumbnail workflow",
			zap.Error(err),
			zap.String("documentId", doc.ID.String()),
		)
		var alreadyStarted *serviceerror.WorkflowExecutionAlreadyStarted
		if !errors.As(err, &alreadyStarted) {
			if updateErr := s.documentRepo.UpdatePreview(
				ctx,
				&repositories.UpdateDocumentPreviewRequest{
					ID: doc.ID,
					TenantInfo: pagination.TenantInfo{
						OrgID: doc.OrganizationID,
						BuID:  doc.BusinessUnitID,
					},
					PreviewStatus:      document.PreviewStatusFailed,
					PreviewStoragePath: "",
				},
			); updateErr != nil {
				s.l.Warn(
					"failed to mark document preview as failed",
					zap.Error(updateErr),
					zap.String("documentId", doc.ID.String()),
				)
			}
		}
	}
}

func previewStatusForFileType(contentType string) document.PreviewStatus {
	if document.SupportsPreview(contentType) {
		return document.PreviewStatusPending
	}

	return document.PreviewStatusUnsupported
}

func (s *Service) ensureThumbnailForSession(
	ctx context.Context,
	session *documentupload.DocumentUploadSession,
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
