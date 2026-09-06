package driverportalservice

import (
	"context"
	"mime/multipart"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/documentservice"
	"github.com/emoss08/trenova/internal/core/services/workercredentialservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

const credentialResourceType = "worker_credential"

// PortalCredential is the driver's view of one credential slot: enough to know
// what is due and to upload a renewal, without the office-only fields.
type PortalCredential struct {
	ID               pulid.ID                  `json:"id"`
	CredentialTypeID pulid.ID                  `json:"credentialTypeId"`
	Name             string                    `json:"name"`
	Category         worker.CredentialCategory `json:"category"`
	Health           worker.CredentialHealth   `json:"health"`
	DaysUntilExpiry  *int64                    `json:"daysUntilExpiry"`
	ExpiresAt        *int64                    `json:"expiresAt"`
	NumberMasked     string                    `json:"numberMasked"`
	Required         bool                      `json:"required"`
	Verified         bool                      `json:"verified"`
	RequiresDocument bool                      `json:"requiresDocument"`
	DocumentID       pulid.ID                  `json:"documentId"`
}

func (s *Service) MyCredentials(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*PortalCredential, error) {
	wrk, err := s.ResolveWorker(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	summary, err := s.credentials.Summary(ctx, tenantInfo, wrk.ID)
	if err != nil {
		return nil, err
	}

	views := make([]*PortalCredential, 0, len(summary.Items))
	for _, item := range summary.Items {
		view := &PortalCredential{
			CredentialTypeID: item.CredentialType.ID,
			Name:             item.CredentialType.Name,
			Category:         item.CredentialType.Category,
			Health:           item.Health,
			DaysUntilExpiry:  item.DaysUntilExpiry,
			Required:         item.Required,
			RequiresDocument: item.CredentialType.RequiresDocument,
		}
		if cred := item.Credential; cred != nil {
			view.ID = cred.ID
			view.ExpiresAt = cred.ExpiresAt
			view.NumberMasked = stringutils.MaskTail(cred.Number, 4)
			view.Verified = cred.IsVerified()
			view.DocumentID = cred.DocumentID
		}
		views = append(views, view)
	}
	return views, nil
}

// UploadMyCredentialDocument files a driver's photo of a renewed card against
// the credential so the office can verify and renew it. The upload is refused
// when the carrier has switched profile uploads off, and the credential must
// belong to the signed-in driver.
func (s *Service) UploadMyCredentialDocument(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	credentialID pulid.ID,
	file *multipart.FileHeader,
	actor *serviceports.RequestActor,
) (*PortalDocument, error) {
	wrk, err := s.ResolveWorker(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	if _, err = s.requireFeature(ctx, tenantInfo,
		func(control *tenant.DashControl) bool { return control.AllowProfileDocumentUpload },
		"Your carrier collects qualification documents outside Dash — see your fleet manager.",
	); err != nil {
		return nil, err
	}
	if actor == nil || actor.UserID.IsNil() {
		return nil, errortypes.NewAuthorizationError(
			"Document upload requires an authenticated user",
		)
	}

	cred, err := s.credentials.Get(ctx, tenantInfo, credentialID)
	if err != nil {
		return nil, err
	}
	if cred.WorkerID != wrk.ID {
		return nil, errortypes.NewNotFoundError("Credential not found")
	}
	if !cred.IsActive() {
		return nil, errortypes.NewValidationError(
			"credentialId",
			errortypes.ErrInvalidOperation,
			"This credential has been archived",
		)
	}

	result, err := s.documentService.Upload(ctx, &documentservice.UploadRequest{
		TenantInfo:   tenantInfo,
		Actor:        *actor,
		File:         file,
		ResourceID:   cred.ID.String(),
		ResourceType: credentialResourceType,
	})
	if err != nil {
		return nil, err
	}

	if _, err = s.credentials.AttachDocument(ctx, &workercredentialservice.AttachDocumentRequest{
		ID:         cred.ID,
		DocumentID: result.Document.ID,
		TenantInfo: tenantInfo,
		UserID:     actor.UserID,
	}); err != nil {
		return nil, err
	}

	s.notifyCredentialUpload(ctx, tenantInfo, wrk, cred)
	return portalDocumentView(result.Document), nil
}

func (s *Service) notifyCredentialUpload(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	wrk *worker.Worker,
	cred *worker.WorkerCredential,
) {
	name := cred.ID.String()
	if cred.CredentialType != nil {
		name = cred.CredentialType.Name
	}
	s.notifyDispatch(
		ctx,
		tenantInfo,
		"credential_document_uploaded",
		"Credential document uploaded",
		wrk.FirstName+" "+wrk.LastName+" uploaded a document for their "+name+
			". Review it and renew the credential.",
		"/hr/workers?tab=credentials",
		map[string]any{"workerId": wrk.ID.String(), "credentialId": cred.ID.String()},
	)
}
