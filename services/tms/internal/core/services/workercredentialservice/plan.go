package workercredentialservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

type CredentialChange = services.RecordChange[worker.WorkerCredential]

// CreatePlan is what Create would file: the credential with its type, and the
// active credential of the same type a renewal would archive.
type CreatePlan struct {
	Credential *worker.WorkerCredential
	Superseded *worker.WorkerCredential
}

func (s *Service) prepareCreate(
	ctx context.Context,
	entity *worker.WorkerCredential,
) (*worker.WorkerCredentialType, error) {
	credentialType, err := s.prepare(ctx, entity)
	if err != nil {
		return nil, err
	}
	if _, err = s.loadWorker(ctx, credentialTenant(entity), entity.WorkerID); err != nil {
		return nil, err
	}
	return credentialType, nil
}

// PlanCreate checks and fills the credential as Create does, without filing it.
// A worker already holding an active credential of the type is refused unless
// the call renews it, which is what the database would refuse.
func (s *Service) PlanCreate(ctx context.Context, req *CreateRequest) (*CreatePlan, error) {
	entity := *req.Entity
	credentialType, err := s.prepareCreate(ctx, &entity)
	if err != nil {
		return nil, err
	}
	entity.CredentialType = credentialType

	held, err := s.repo.ListForWorker(ctx, &repositories.ListWorkerCredentialsRequest{
		TenantInfo: credentialTenant(&entity),
		WorkerID:   entity.WorkerID,
	})
	if err != nil {
		return nil, err
	}
	plan := &CreatePlan{Credential: &entity}
	for _, current := range held {
		if current == nil || !current.IsActive() ||
			current.CredentialTypeID != entity.CredentialTypeID {
			continue
		}
		if !req.Renew {
			return nil, errortypes.NewValidationError(
				"credentialTypeId",
				errortypes.ErrDuplicate,
				"This worker already holds an active credential of this type; renew it instead",
			)
		}
		plan.Superseded = current
	}
	return plan, nil
}

func (s *Service) planUpdate(
	ctx context.Context,
	entity *worker.WorkerCredential,
) (*CredentialChange, *worker.WorkerCredentialType, error) {
	original, err := s.repo.GetByID(ctx, &repositories.GetWorkerCredentialByIDRequest{
		ID:         entity.ID,
		TenantInfo: credentialTenant(entity),
	})
	if err != nil {
		return nil, nil, err
	}
	if !original.IsActive() {
		return nil, nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"Archived credentials are read-only. Add a new credential instead",
		)
	}

	entity.WorkerID = original.WorkerID
	entity.CredentialTypeID = original.CredentialTypeID
	entity.Status = original.Status
	entity.VerifiedByID = original.VerifiedByID
	entity.VerifiedAt = original.VerifiedAt
	entity.ArchivedByID = original.ArchivedByID
	entity.ArchivedAt = original.ArchivedAt
	entity.ArchiveReason = original.ArchiveReason
	entity.CreatedAt = original.CreatedAt
	if entity.DocumentID.IsNil() {
		entity.DocumentID = original.DocumentID
	}

	credentialType, err := s.prepare(ctx, entity)
	if err != nil {
		return nil, nil, err
	}

	if s.factsChanged(original, entity) && original.IsVerified() {
		entity.VerifiedByID = pulid.Nil
		entity.VerifiedAt = nil
	}
	return &CredentialChange{Before: original, After: entity}, credentialType, nil
}

// PlanUpdate is what Update would leave the credential as; changing a fact a
// verifier vouched for clears the verification.
func (s *Service) PlanUpdate(
	ctx context.Context,
	entity *worker.WorkerCredential,
) (*CredentialChange, error) {
	planned := *entity
	change, credentialType, err := s.planUpdate(ctx, &planned)
	if err != nil {
		return nil, err
	}
	change.After.CredentialType = credentialType
	return change, nil
}

// PlanArchive is what Archive would leave the credential as. One already
// archived is left alone.
func (s *Service) PlanArchive(ctx context.Context, req *StatusRequest) (*CredentialChange, error) {
	original, err := s.loadForChange(ctx, req)
	if err != nil {
		return nil, err
	}
	updated := *original
	if original.IsActive() {
		now := timeutils.NowUnix()
		updated.Status = worker.CredentialStatusArchived
		updated.ArchivedAt = &now
		updated.ArchivedByID = req.UserID
		updated.ArchiveReason = strings.TrimSpace(req.Reason)
	}
	return &CredentialChange{Before: original, After: &updated}, nil
}

// PlanAttachDocument is what AttachDocument would leave the credential as,
// with the document on After.
func (s *Service) PlanAttachDocument(
	ctx context.Context,
	req *AttachDocumentRequest,
) (*CredentialChange, error) {
	original, err := s.loadForChange(ctx, &StatusRequest{ID: req.ID, TenantInfo: req.TenantInfo})
	if err != nil {
		return nil, err
	}
	if !original.IsActive() {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"Archived credentials cannot take new documents",
		)
	}

	doc, err := s.documentRepo.GetByID(ctx, repositories.GetDocumentByIDRequest{
		ID:         req.DocumentID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	ownedByCredential := doc.ResourceType == credentialResourceID &&
		doc.ResourceID == original.ID.String()
	ownedByWorker := doc.ResourceType == workerResourceType &&
		doc.ResourceID == original.WorkerID.String()
	if !ownedByCredential && !ownedByWorker {
		return nil, errortypes.NewValidationError(
			"documentId",
			errortypes.ErrInvalid,
			"Document does not belong to this worker",
		)
	}

	updated := *original
	updated.DocumentID = doc.ID
	updated.Document = doc
	if original.DocumentID != doc.ID && original.IsVerified() {
		updated.VerifiedByID = pulid.Nil
		updated.VerifiedAt = nil
	}
	return &CredentialChange{Before: original, After: &updated}, nil
}
