package documentuploadjobs

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/documentupload"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/ports/storage"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/core/services/thumbnailservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs/thumbnailjobs"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	goredis "github.com/redis/go-redis/v9"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/fx"
)

type ActivitiesParams struct {
	fx.In

	SessionRepository    repositories.DocumentUploadSessionRepository
	DocumentRepository   repositories.DocumentRepository
	Storage              storage.Client
	AuditService         services.AuditService
	SearchProjection     services.DocumentSearchProjectionService
	ThumbnailGenerator   *thumbnailservice.Generator
	WorkflowStarter      services.WorkflowStarter
	Redis                *goredis.Client `optional:"true"`
	DocumentIntelligence services.DocumentContentService
}

type Activities struct {
	sessionRepo          repositories.DocumentUploadSessionRepository
	documentRepo         repositories.DocumentRepository
	storage              storage.Client
	auditService         services.AuditService
	searchProjection     services.DocumentSearchProjectionService
	thumbnailGenerator   *thumbnailservice.Generator
	workflowStarter      services.WorkflowStarter
	redis                *goredis.Client
	documentIntelligence services.DocumentContentService
}

//nolint:gocritic // dependency injection param
func NewActivities(p ActivitiesParams) *Activities {
	documentIntelligence := p.DocumentIntelligence
	if documentIntelligence == nil {
		documentIntelligence = noopDocumentContentService{}
	}

	searchProjection := p.SearchProjection
	if searchProjection == nil {
		searchProjection = noopDocumentSearchProjectionService{}
	}

	workflowStarter := p.WorkflowStarter
	if workflowStarter == nil {
		workflowStarter = noopWorkflowStarter{}
	}

	return &Activities{
		sessionRepo:          p.SessionRepository,
		documentRepo:         p.DocumentRepository,
		storage:              p.Storage,
		auditService:         p.AuditService,
		searchProjection:     searchProjection,
		thumbnailGenerator:   p.ThumbnailGenerator,
		workflowStarter:      workflowStarter,
		redis:                p.Redis,
		documentIntelligence: documentIntelligence,
	}
}

func (a *Activities) FinalizeUploadActivity(
	ctx context.Context,
	payload *FinalizeUploadPayload,
) (*FinalizeUploadResult, error) {
	tenantInfo := pagination.TenantInfo{
		OrgID:  payload.OrganizationID,
		BuID:   payload.BusinessUnitID,
		UserID: payload.UserID,
	}

	session, err := a.sessionRepo.GetByID(ctx, repositories.GetDocumentUploadSessionByIDRequest{
		ID:         payload.SessionID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, temporal.NewNonRetryableApplicationError(
				"Document upload session no longer exists",
				"document-upload-session-not-found",
				nil,
			)
		}
		return nil, err
	}

	defer a.releaseCompletionLease(ctx, session.ID.String())

	if res, done, earlyErr := a.finalizeUploadEarlyOutcomes(ctx, session); earlyErr != nil || done {
		return res, earlyErr
	}

	if err = a.updateStatus(ctx, session, documentupload.StatusVerifying); err != nil {
		return nil, err
	}
	activity.RecordHeartbeat(ctx, "verifying-upload")

	parts, err := a.getUploadedParts(ctx, session)
	if err != nil {
		return nil, err
	}

	if err = a.finalizeValidateStoredObject(ctx, session, parts); err != nil {
		return nil, err
	}

	if err = a.updateStatus(ctx, session, documentupload.StatusFinalizing); err != nil {
		return nil, err
	}
	activity.RecordHeartbeat(ctx, "finalizing-document")

	doc, err := a.ensureDocument(ctx, session, payload.UserID)
	if err != nil {
		return nil, a.failSession(
			ctx,
			session,
			documentupload.FailureDocumentCreateFailed.String(),
			"Failed to create document record",
		)
	}

	return a.finalizeUploadPersistDocument(ctx, session, parts, doc, payload)
}

func (a *Activities) finalizeUploadEarlyOutcomes(
	ctx context.Context,
	session *documentupload.DocumentUploadSession,
) (*FinalizeUploadResult, bool, error) {
	superseded, supersededErr := a.cancelSupersededSession(ctx, session)
	if supersededErr != nil {
		return nil, false, supersededErr
	}
	if superseded {
		return &FinalizeUploadResult{
			SessionID: session.ID,
			Status:    string(session.Status),
		}, true, nil
	}

	if session.DocumentID != nil && session.Status.IsDocumentReady() {
		previewPath := a.ensureThumbnailForSession(ctx, session)
		return &FinalizeUploadResult{
			SessionID:   session.ID,
			DocumentID:  session.DocumentID,
			Status:      string(session.Status),
			PreviewPath: previewPath,
		}, true, nil
	}

	if session.DocumentID != nil {
		if doc := a.getDocumentForSession(ctx, session); doc != nil {
			session.Status = documentupload.StatusAvailable
			session.FailureCode = ""
			session.FailureMessage = ""
			session.LastActivityAt = timeutils.NowUnix()
			if _, err := a.sessionRepo.Update(ctx, session); err != nil {
				return nil, false, err
			}

			previewPath := a.ensureThumbnailForDocument(ctx, doc)
			return &FinalizeUploadResult{
				SessionID:   session.ID,
				DocumentID:  &doc.ID,
				Status:      string(session.Status),
				PreviewPath: previewPath,
			}, true, nil
		}
	}

	return nil, false, nil
}

func (a *Activities) finalizeValidateStoredObject(
	ctx context.Context,
	session *documentupload.DocumentUploadSession,
	parts []storage.UploadedPart,
) error {
	if session.Strategy == documentupload.StrategyMultipart {
		if len(parts) == 0 {
			return a.failSession(
				ctx,
				session,
				documentupload.FailureNoUploadParts.String(),
				"No uploaded parts were found for this session",
			)
		}

		if err := a.storage.CompleteMultipartUpload(ctx, &storage.CompleteMultipartUploadParams{
			Key:      session.StoragePath,
			UploadID: session.StorageProviderUploadID,
			Parts:    parts,
		}); err != nil && !storage.IsMissingMultipartUploadError(err) {
			return a.failSession(
				ctx,
				session,
				documentupload.FailureMultipartCompleteFailed.String(),
				"Failed to finalize multipart upload",
			)
		}
	}

	fileInfo, err := a.storage.GetFileInfo(ctx, session.StoragePath)
	if err != nil {
		return a.failSession(
			ctx,
			session,
			documentupload.FailureFileInfoFailed.String(),
			"Failed to verify uploaded file",
		)
	}

	if fileInfo.Size != session.FileSize {
		return a.failSession(
			ctx,
			session,
			documentupload.FailureFileSizeMismatch.String(),
			"Uploaded file size does not match the original file",
		)
	}

	return nil
}

func (a *Activities) finalizeUploadPersistDocument(
	ctx context.Context,
	session *documentupload.DocumentUploadSession,
	parts []storage.UploadedPart,
	doc *document.Document,
	payload *FinalizeUploadPayload,
) (*FinalizeUploadResult, error) {
	session.DocumentID = &doc.ID
	session.UploadedParts = toDomainUploadedParts(parts)
	session.Status = documentupload.StatusAvailable
	session.FailureCode = ""
	session.FailureMessage = ""
	session.LastActivityAt = timeutils.NowUnix()
	if _, err := a.sessionRepo.Update(ctx, session); err != nil {
		return nil, err
	}

	previewPath := ""
	if a.thumbnailGenerator != nil && a.thumbnailGenerator.SupportsThumbnail(doc.FileType) {
		previewPath = a.startThumbnailWorkflow(ctx, doc)
	}
	if doc.ProcessingProfile.SupportsIntelligence() {
		_ = a.documentIntelligence.EnqueueExtraction(ctx, doc, payload.UserID)
	}

	return &FinalizeUploadResult{
		SessionID:   session.ID,
		DocumentID:  &doc.ID,
		Status:      session.Status.String(),
		PreviewPath: previewPath,
	}, nil
}

func (a *Activities) ReconcileUploadsActivity(
	ctx context.Context,
	payload *ReconcileUploadsPayload,
) (*ReconcileUploadsResult, error) {
	result := &ReconcileUploadsResult{}
	now := time.Now()
	staleBefore := now.Add(-time.Duration(payload.StaleAfterSeconds) * time.Second).Unix()
	expiresBefore := now.Unix()
	previewBefore := now.Add(-time.Duration(payload.PendingAfterSeconds) * time.Second).Unix()

	sessions, err := a.sessionRepo.ListForReconciliation(
		ctx,
		staleBefore,
		expiresBefore,
		payload.Limit,
	)
	if err != nil {
		return nil, err
	}

	for _, session := range sessions {
		result.StaleSessionsProcessed++

		expired, finalize, recErr := a.reconcileSession(ctx, session, expiresBefore)
		if recErr != nil {
			activity.GetLogger(ctx).Warn("document upload reconciliation failed",
				"sessionId", session.ID.String(),
				"error", recErr,
			)
			continue
		}
		if expired {
			result.SessionsExpired++
		}
		if finalize {
			result.FinalizationsStarted++
		}
	}

	docs, err := a.documentRepo.ListPendingPreviewReconciliation(ctx, previewBefore, payload.Limit)
	if err != nil {
		return nil, err
	}

	for _, doc := range docs {
		if a.ensureThumbnailForDocument(ctx, doc) != "" ||
			doc.PreviewStatus == document.PreviewStatusPending {
			result.PreviewRetriesStarted++
		}
	}

	return result, nil
}

func (a *Activities) CleanupDocumentStorageActivity(
	ctx context.Context,
	payload *CleanupDocumentStoragePayload,
) error {
	for _, path := range payload.Paths {
		if path == "" {
			continue
		}
		if err := a.deleteStoredObject(ctx, path); err != nil {
			return temporaltype.NewRetryableError("Failed to cleanup document storage", err).
				ToTemporalError()
		}
	}

	return nil
}

func (a *Activities) deleteStoredObject(ctx context.Context, key string) error {
	exists, err := a.storage.Exists(ctx, key)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	fileInfo, err := a.storage.GetFileInfo(ctx, key)
	if err != nil {
		return err
	}

	return a.storage.DeleteObject(ctx, &storage.DeleteObjectParams{
		Key:       key,
		VersionID: fileInfo.VersionID,
	})
}

func (a *Activities) updateStatus(
	ctx context.Context,
	session *documentupload.DocumentUploadSession,
	status documentupload.Status,
) error {
	session.Status = status
	session.LastActivityAt = timeutils.NowUnix()
	_, err := a.sessionRepo.Update(ctx, session)
	return err
}

func (a *Activities) failSession(
	ctx context.Context,
	session *documentupload.DocumentUploadSession,
	code string,
	message string,
) error {
	session.Status = documentupload.StatusFailed
	session.FailureCode = code
	session.FailureMessage = message
	session.LastActivityAt = timeutils.NowUnix()
	_, err := a.sessionRepo.Update(ctx, session)
	if err != nil {
		return err
	}
	return temporal.NewNonRetryableApplicationError(message, "document-upload-finalization", nil)
}

func (a *Activities) ensureDocument(
	ctx context.Context,
	session *documentupload.DocumentUploadSession,
	userID pulid.ID,
) (*document.Document, error) {
	if session.DocumentID != nil {
		return a.documentRepo.GetByID(ctx, repositories.GetDocumentByIDRequest{
			ID: *session.DocumentID,
			TenantInfo: pagination.TenantInfo{
				OrgID: session.OrganizationID,
				BuID:  session.BusinessUnitID,
			},
		})
	}

	existing, err := a.documentRepo.GetByStoragePath(
		ctx,
		repositories.GetDocumentByStoragePathRequest{
			StoragePath: session.StoragePath,
			TenantInfo: pagination.TenantInfo{
				OrgID: session.OrganizationID,
				BuID:  session.BusinessUnitID,
			},
		},
	)
	if err == nil {
		return existing, nil
	}
	if !errortypes.IsNotFoundError(err) {
		return nil, err
	}

	fileInfo, err := a.storage.GetFileInfo(ctx, session.StoragePath)
	if err != nil {
		return nil, err
	}

	return a.createDocumentWithRetryLoop(ctx, session, userID, fileInfo)
}

func (a *Activities) createDocumentWithRetryLoop(
	ctx context.Context,
	session *documentupload.DocumentUploadSession,
	userID pulid.ID,
	fileInfo *storage.FileInfo,
) (*document.Document, error) {
	const maxCreateAttempts = 3
	for attempt := 0; attempt < maxCreateAttempts; attempt++ {
		lineageID, versionNumber, previousDoc, resolveErr := a.resolveLineageState(ctx, session)
		if resolveErr != nil {
			return nil, resolveErr
		}

		doc := a.newDocumentForUpload(
			session,
			fileInfo,
			lineageID,
			versionNumber,
			previousDoc,
			userID,
		)

		createdDoc, createErr := a.documentRepo.Create(ctx, doc)
		if createErr != nil {
			recovered, retry, dupErr := a.recoverDocumentCreateAfterUniqueConflict(
				ctx,
				session,
				createErr,
				attempt,
				maxCreateAttempts,
			)
			if dupErr != nil {
				return nil, dupErr
			}
			if recovered != nil {
				return recovered, nil
			}
			if retry {
				continue
			}
		}

		if err := a.finalizeCreatedDocumentUpload(
			ctx,
			session,
			userID,
			createdDoc,
			previousDoc,
			lineageID,
		); err != nil {
			return nil, err
		}

		return createdDoc, nil
	}

	return nil, temporal.NewNonRetryableApplicationError(
		"Failed to create document record after concurrent upload retries",
		"document-upload-finalization",
		nil,
	)
}

func (a *Activities) newDocumentForUpload(
	session *documentupload.DocumentUploadSession,
	fileInfo *storage.FileInfo,
	lineageID pulid.ID,
	versionNumber int64,
	previousDoc *document.Document,
	userID pulid.ID,
) *document.Document {
	return &document.Document{
		LineageID:          lineageID,
		VersionNumber:      versionNumber,
		IsCurrentVersion:   previousDoc == nil,
		OrganizationID:     session.OrganizationID,
		BusinessUnitID:     session.BusinessUnitID,
		FileName:           filepath.Base(session.StoragePath),
		OriginalName:       session.OriginalName,
		FileSize:           session.FileSize,
		FileType:           session.ContentType,
		StoragePath:        session.StoragePath,
		StorageVersionID:   fileInfo.VersionID,
		Status:             document.StatusActive,
		Description:        session.Description,
		ResourceID:         session.ResourceID,
		ResourceType:       session.ResourceType,
		ProcessingProfile:  session.ProcessingProfile,
		Tags:               session.Tags,
		UploadedByID:       userID,
		PreviewStoragePath: "",
		PreviewStatus:      previewStatusForFileType(session.ContentType),
		DocumentTypeID:     session.DocumentTypeID,
	}
}

func (a *Activities) finalizeCreatedDocumentUpload(
	ctx context.Context,
	session *documentupload.DocumentUploadSession,
	userID pulid.ID,
	createdDoc *document.Document,
	previousDoc *document.Document,
	lineageID pulid.ID,
) error {
	if previousDoc != nil {
		if err := a.documentRepo.PromoteVersion(ctx, &repositories.PromoteDocumentVersionRequest{
			LineageID:         lineageID,
			CurrentDocumentID: createdDoc.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID: session.OrganizationID,
				BuID:  session.BusinessUnitID,
			},
		}); err != nil {
			return err
		}
		_ = a.searchProjection.Delete(ctx, previousDoc.ID, pagination.TenantInfo{
			OrgID: session.OrganizationID,
			BuID:  session.BusinessUnitID,
		})
	}
	createdDoc.IsCurrentVersion = true

	_ = a.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceDocument,
		ResourceID:     createdDoc.GetID().String(),
		Operation:      permission.OpCreate,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(createdDoc),
		OrganizationID: createdDoc.OrganizationID,
		BusinessUnitID: createdDoc.BusinessUnitID,
	}, auditservice.WithComment("Document uploaded"))
	a.syncSearchProjection(ctx, createdDoc, "")
	return nil
}

func (a *Activities) recoverDocumentCreateAfterUniqueConflict(
	ctx context.Context,
	session *documentupload.DocumentUploadSession,
	createErr error,
	attempt, maxAttempts int,
) (*document.Document, bool, error) {
	if !dberror.IsUniqueConstraintViolation(createErr) {
		return nil, false, createErr
	}

	existingDoc, getErr := a.documentRepo.GetByStoragePath(
		ctx,
		repositories.GetDocumentByStoragePathRequest{
			StoragePath: session.StoragePath,
			TenantInfo: pagination.TenantInfo{
				OrgID: session.OrganizationID,
				BuID:  session.BusinessUnitID,
			},
		},
	)
	if getErr == nil {
		return existingDoc, false, nil
	}

	if attempt < maxAttempts-1 {
		return nil, true, nil
	}

	return nil, false, createErr
}

func (a *Activities) resolveLineageState(
	ctx context.Context,
	session *documentupload.DocumentUploadSession,
) (pulid.ID, int64, *document.Document, error) {
	var (
		lineageID     pulid.ID
		versionNumber int64 = 1
		previousDoc   *document.Document
	)

	if session.LineageID != nil && !session.LineageID.IsNil() {
		lineageID = *session.LineageID
		versions, err := a.documentRepo.ListVersions(ctx, repositories.ListDocumentVersionsRequest{
			LineageID: lineageID,
			TenantInfo: pagination.TenantInfo{
				OrgID: session.OrganizationID,
				BuID:  session.BusinessUnitID,
			},
		})
		if err != nil {
			return "", 0, nil, err
		}
		for _, version := range versions {
			if version.VersionNumber >= versionNumber {
				versionNumber = version.VersionNumber + 1
			}
			if version.IsCurrentVersion {
				previousDoc = version
			}
		}
	}

	return lineageID, versionNumber, previousDoc, nil
}

func (a *Activities) syncSearchProjection(
	ctx context.Context,
	doc *document.Document,
	contentText string,
) {
	_ = a.searchProjection.Upsert(ctx, doc, contentText)
}

func (a *Activities) reconcileSession(
	ctx context.Context,
	session *documentupload.DocumentUploadSession,
	expiresBefore int64,
) (expired, finalize bool, err error) {
	if superseded, supersededErr := a.cancelSupersededSession(ctx, session); supersededErr != nil {
		return false, false, supersededErr
	} else if superseded {
		return false, false, nil
	}

	switch session.Status { //nolint:exhaustive // we only want to check for the finalizable statuses
	case documentupload.StatusUploaded,
		documentupload.StatusVerifying,
		documentupload.StatusFinalizing:
		return false, a.startFinalizeWorkflowForSession(ctx, session), nil
	case documentupload.StatusInitiated,
		documentupload.StatusUploading,
		documentupload.StatusPaused:
		if session.ExpiresAt <= expiresBefore {
			return true, false, a.expireSession(ctx, session)
		}

		ready, readyErr := a.isSessionReadyToFinalize(ctx, session)
		if readyErr != nil {
			return false, false, readyErr
		}
		if !ready {
			return false, false, nil
		}

		session.Status = documentupload.StatusUploaded
		session.FailureCode = ""
		session.FailureMessage = ""
		session.LastActivityAt = timeutils.NowUnix()
		if _, err = a.sessionRepo.Update(ctx, session); err != nil {
			return false, false, err
		}

		return false, a.startFinalizeWorkflowForSession(ctx, session), nil
	default:
		return false, false, nil
	}
}

func (a *Activities) isSessionReadyToFinalize(
	ctx context.Context,
	session *documentupload.DocumentUploadSession,
) (bool, error) {
	if session.Strategy == documentupload.StrategySingle {
		info, err := a.storage.GetFileInfo(ctx, session.StoragePath)
		if err != nil {
			return false, nil //nolint:nilerr // we want to return false if we can't get the file info
		}

		return info.Size == session.FileSize, nil
	}

	parts, err := a.getUploadedParts(ctx, session)
	if err != nil {
		if storage.IsMissingMultipartUploadError(err) {
			return false, nil
		}
		return false, err
	}

	if len(parts) == 0 {
		return false, nil
	}

	totalSize := int64(0)
	for _, part := range parts {
		totalSize += part.Size
	}

	if totalSize != session.FileSize {
		return false, nil
	}

	session.UploadedParts = toDomainUploadedParts(parts)
	return true, nil
}

func (a *Activities) expireSession(
	ctx context.Context,
	session *documentupload.DocumentUploadSession,
) error {
	if session.Strategy == documentupload.StrategyMultipart &&
		session.StorageProviderUploadID != "" {
		_ = a.storage.AbortMultipartUpload(ctx, &storage.AbortMultipartUploadParams{
			Key:      session.StoragePath,
			UploadID: session.StorageProviderUploadID,
		})
	}

	session.Status = documentupload.StatusExpired
	session.FailureCode = documentupload.FailureSessionExpired.String()
	session.FailureMessage = "Upload session expired before completion"
	session.LastActivityAt = timeutils.NowUnix()
	_, err := a.sessionRepo.Update(ctx, session)
	return err
}

func (a *Activities) getUploadedParts(
	ctx context.Context,
	session *documentupload.DocumentUploadSession,
) ([]storage.UploadedPart, error) {
	if session.Strategy == documentupload.StrategySingle {
		fileInfo, err := a.storage.GetFileInfo(ctx, session.StoragePath)
		if err != nil {
			return nil, err
		}
		return []storage.UploadedPart{{PartNumber: 1, Size: fileInfo.Size}}, nil
	}

	if len(session.UploadedParts) > 0 {
		return storage.ToDomainParts(session.UploadedParts), nil
	}

	return a.storage.ListMultipartUploadParts(ctx, &storage.ListMultipartUploadPartsParams{
		Key:      session.StoragePath,
		UploadID: session.StorageProviderUploadID,
	})
}

func (a *Activities) startFinalizeWorkflowForSession(
	ctx context.Context,
	session *documentupload.DocumentUploadSession,
) bool {
	if !a.workflowStarter.Enabled() {
		return false
	}

	_, err := a.workflowStarter.StartWorkflow(
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
		&FinalizeUploadPayload{
			BasePayload: temporaltype.BasePayload{
				OrganizationID: session.OrganizationID,
				BusinessUnitID: session.BusinessUnitID,
				Timestamp:      timeutils.NowUnix(),
			},
			SessionID: session.ID,
		},
	)
	if err != nil {
		var started *serviceerror.WorkflowExecutionAlreadyStarted
		return errors.As(err, &started)
	}

	return true
}

func (a *Activities) startThumbnailWorkflow(ctx context.Context, doc *document.Document) string {
	if !a.workflowStarter.Enabled() {
		return ""
	}

	if doc.PreviewStatus != document.PreviewStatusPending || doc.PreviewStoragePath != "" {
		doc.PreviewStatus = document.PreviewStatusPending
		doc.PreviewStoragePath = ""
		if err := a.documentRepo.UpdatePreview(ctx, &repositories.UpdateDocumentPreviewRequest{
			ID: doc.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID: doc.OrganizationID,
				BuID:  doc.BusinessUnitID,
			},
			PreviewStatus:      document.PreviewStatusPending,
			PreviewStoragePath: "",
		}); err != nil {
			return ""
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

	_, err := a.workflowStarter.StartWorkflow(
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
		var started *serviceerror.WorkflowExecutionAlreadyStarted
		if errors.As(err, &started) {
			return doc.PreviewStoragePath
		}
		_ = a.documentRepo.UpdatePreview(ctx, &repositories.UpdateDocumentPreviewRequest{
			ID: doc.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID: doc.OrganizationID,
				BuID:  doc.BusinessUnitID,
			},
			PreviewStatus:      document.PreviewStatusFailed,
			PreviewStoragePath: "",
		})
		return ""
	}

	return doc.PreviewStoragePath
}

func (a *Activities) cancelSupersededSession(
	ctx context.Context,
	session *documentupload.DocumentUploadSession,
) (bool, error) {
	if session == nil || session.LineageID == nil || session.LineageID.IsNil() ||
		session.Status.IsTerminal() {
		return false, nil
	}

	superseded, err := a.isSupersededByNewerVersion(ctx, session)
	if err != nil || !superseded {
		return false, err
	}

	session.MarkSuperseded(timeutils.NowUnix())
	if _, err = a.sessionRepo.Update(ctx, session); err != nil {
		return false, err
	}

	return true, nil
}

func (a *Activities) isSupersededByNewerVersion(
	ctx context.Context,
	session *documentupload.DocumentUploadSession,
) (bool, error) {
	activeSessions, err := a.sessionRepo.ListActive(
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

	versions, err := a.documentRepo.ListVersions(ctx, repositories.ListDocumentVersionsRequest{
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

func (a *Activities) getDocumentForSession(
	ctx context.Context,
	session *documentupload.DocumentUploadSession,
) *document.Document {
	if session.DocumentID == nil {
		return nil
	}

	doc, err := a.documentRepo.GetByID(ctx, repositories.GetDocumentByIDRequest{
		ID: *session.DocumentID,
		TenantInfo: pagination.TenantInfo{
			OrgID: session.OrganizationID,
			BuID:  session.BusinessUnitID,
		},
	})
	if err != nil {
		return nil
	}

	return doc
}

func (a *Activities) ensureThumbnailForSession(
	ctx context.Context,
	session *documentupload.DocumentUploadSession,
) string {
	doc := a.getDocumentForSession(ctx, session)
	if doc == nil {
		return ""
	}

	return a.ensureThumbnailForDocument(ctx, doc)
}

func (a *Activities) ensureThumbnailForDocument(
	ctx context.Context,
	doc *document.Document,
) string {
	if doc == nil || a.thumbnailGenerator == nil {
		return ""
	}

	if doc.PreviewStatus == document.PreviewStatusReady || !document.SupportsPreview(doc.FileType) {
		return doc.PreviewStoragePath
	}

	return a.startThumbnailWorkflow(ctx, doc)
}

func previewStatusForFileType(contentType string) document.PreviewStatus {
	if document.SupportsPreview(contentType) {
		return document.PreviewStatusPending
	}

	return document.PreviewStatusUnsupported
}

func (a *Activities) releaseCompletionLease(ctx context.Context, sessionID string) {
	if a.redis == nil {
		return
	}
	_ = a.redis.Del(ctx, completionLeaseKey(sessionID)).Err()
}

func completionLeaseKey(sessionID string) string {
	return "document-upload:completion:" + sessionID
}

func toDomainUploadedParts(parts []storage.UploadedPart) []storage.UploadedPart {
	result := storage.ToDomainParts(parts)

	slices.SortFunc(result, func(a, b storage.UploadedPart) int {
		return a.PartNumber - b.PartNumber
	})

	return result
}
