package documenttemplateservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/domain/documenttemplate/starters"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListDocumentTemplatesRequest,
) (*pagination.ListResult[*documenttemplate.DocumentTemplate], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) Get(
	ctx context.Context,
	req *repositories.GetDocumentTemplateByIDRequest,
) (*documenttemplate.DocumentTemplate, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) ListVersions(
	ctx context.Context,
	req *repositories.ListDocumentTemplateVersionsRequest,
) ([]*documenttemplate.DocumentTemplateVersion, error) {
	return s.versionRepo.List(ctx, req)
}

func (s *Service) GetVersion(
	ctx context.Context,
	req *repositories.GetDocumentTemplateVersionByIDRequest,
) (*documenttemplate.DocumentTemplateVersion, error) {
	return s.versionRepo.GetByID(ctx, req)
}

func (s *Service) CreateTemplate(
	ctx context.Context,
	entity *documenttemplate.DocumentTemplate,
	userID pulid.ID,
) (*documenttemplate.DocumentTemplate, error) {
	// A new template has no content yet, so it cannot be live and cannot be the
	// organization default; both are earned by publishing.
	entity.ActiveVersionID = nil
	entity.CreatedByID = &userID
	entity.UpdatedByID = &userID

	if multiErr := s.validator.ValidateTemplate(ctx, entity, nil); multiErr != nil {
		return nil, multiErr
	}

	created, err := s.repo.Create(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.logAudit(ctx, created, nil, permission.OpCreate, userID, "Document template created")

	return created, nil
}

func (s *Service) UpdateTemplate(
	ctx context.Context,
	entity *documenttemplate.DocumentTemplate,
	userID pulid.ID,
) (*documenttemplate.DocumentTemplate, error) {
	original, err := s.repo.GetByID(ctx, &repositories.GetDocumentTemplateByIDRequest{
		DocumentTemplateID: entity.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	})
	if err != nil {
		return nil, err
	}

	// Kind and the live pointer are not editable through this path: changing a
	// kind would orphan every version against a different variable catalog, and
	// the pointer belongs to Publish.
	entity.Kind = original.Kind
	entity.ActiveVersionID = original.ActiveVersionID
	entity.CreatedByID = original.CreatedByID
	entity.UpdatedByID = &userID

	if multiErr := s.validator.ValidateTemplate(ctx, entity, original); multiErr != nil {
		return nil, multiErr
	}

	updated, err := s.repo.Update(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.logAudit(ctx, updated, original, permission.OpUpdate, userID, "Document template updated")

	return updated, nil
}

func (s *Service) DeleteTemplate(
	ctx context.Context,
	req *repositories.GetDocumentTemplateByIDRequest,
	userID pulid.ID,
) error {
	original, err := s.repo.GetByID(ctx, req)
	if err != nil {
		return err
	}

	if err = s.repo.Delete(ctx, req); err != nil {
		return err
	}

	s.logAudit(ctx, nil, original, permission.OpDelete, userID, "Document template deleted")

	return nil
}

// CreateVersion opens a new draft, optionally copied from an existing version.
func (s *Service) CreateVersion(
	ctx context.Context,
	entity *documenttemplate.DocumentTemplateVersion,
	sourceVersionID *pulid.ID,
	userID pulid.ID,
) (*documenttemplate.DocumentTemplateVersion, error) {
	template, err := s.repo.GetByID(ctx, &repositories.GetDocumentTemplateByIDRequest{
		DocumentTemplateID: entity.TemplateID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	})
	if err != nil {
		return nil, err
	}

	entity.CreatedByID = &userID
	entity.UpdatedByID = &userID
	s.stampContentHash(entity)

	// A copy takes its content from the source inside the repository's
	// transaction, so validation here would judge a body that is about to be
	// replaced.
	copying := sourceVersionID != nil && !sourceVersionID.IsNil()
	if !copying {
		if multiErr := s.validator.ValidateVersion(ctx, entity, template.Kind); multiErr != nil {
			return nil, multiErr
		}
	}

	created, err := s.versionRepo.Create(ctx, &repositories.CreateDocumentTemplateVersionRequest{
		Version:         entity,
		SourceVersionID: sourceVersionID,
	})
	if err != nil {
		return nil, err
	}

	s.logAudit(
		ctx, created, nil, permission.OpCreate, userID, "Document template version created",
	)

	return created, nil
}

func (s *Service) UpdateVersion(
	ctx context.Context,
	entity *documenttemplate.DocumentTemplateVersion,
	userID pulid.ID,
) (*documenttemplate.DocumentTemplateVersion, error) {
	tenantInfo := pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	}

	original, err := s.versionRepo.GetByID(
		ctx,
		&repositories.GetDocumentTemplateVersionByIDRequest{
			VersionID:  entity.ID,
			TenantInfo: tenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}

	if !original.IsEditable() {
		return nil, errortypes.NewBusinessError(
			"A published version cannot be edited. Create a new draft from it instead, " +
				"so the bytes a customer already received stay reproducible.",
		)
	}

	// Lifecycle fields belong to Publish and Archive. Accepting them here would
	// let a caller activate a version without passing the gate.
	entity.TemplateID = original.TemplateID
	entity.VersionNumber = original.VersionNumber
	entity.Status = original.Status
	entity.SourceVersionID = original.SourceVersionID
	entity.StarterHash = original.StarterHash
	entity.PublishedAt = original.PublishedAt
	entity.PublishedByID = original.PublishedByID
	entity.ArchivedAt = original.ArchivedAt
	entity.ArchivedByID = original.ArchivedByID
	entity.CreatedByID = original.CreatedByID
	entity.UpdatedByID = &userID
	s.stampContentHash(entity)

	kind := s.kindOf(ctx, original, tenantInfo)
	if multiErr := s.validator.ValidateVersion(ctx, entity, kind); multiErr != nil {
		return nil, multiErr
	}

	updated, err := s.versionRepo.Update(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.logAudit(
		ctx, updated, original, permission.OpUpdate, userID, "Document template version updated",
	)

	return updated, nil
}

// Rollback republishes an earlier version by copying it forward.
//
// History is never reactivated in place. The dialog says so, and the reason is
// that a generated document names the version that produced it: reactivating an
// archived row would make that record describe bytes sent at two different times
// under two different approvals.
func (s *Service) Rollback(
	ctx context.Context,
	req *services.PublishVersionRequest,
) (*documenttemplate.DocumentTemplateVersion, error) {
	target, err := s.versionRepo.GetByID(
		ctx,
		&repositories.GetDocumentTemplateVersionByIDRequest{
			VersionID:  req.VersionID,
			TenantInfo: req.TenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}

	if target.Status == documenttemplate.VersionStatusActive {
		return nil, errortypes.NewBusinessError(
			"That version is already live. There is nothing to roll back to.",
		)
	}

	draft := &documenttemplate.DocumentTemplateVersion{
		OrganizationID: target.OrganizationID,
		BusinessUnitID: target.BusinessUnitID,
		TemplateID:     target.TemplateID,
		PublishNotes:   req.Notes,
	}

	created, err := s.CreateVersion(ctx, draft, &req.VersionID, req.UserID)
	if err != nil {
		return nil, err
	}

	return s.PublishVersion(ctx, &services.PublishVersionRequest{
		TenantInfo: req.TenantInfo,
		VersionID:  created.ID,
		UserID:     req.UserID,
		Notes: fmt.Sprintf(
			"Rolled back to version %d. %s", target.VersionNumber, req.Notes,
		),
	})
}

func (s *Service) ArchiveVersion(
	ctx context.Context,
	req *repositories.ArchiveDocumentTemplateVersionRequest,
) (*documenttemplate.DocumentTemplateVersion, error) {
	original, err := s.versionRepo.GetByID(
		ctx,
		&repositories.GetDocumentTemplateVersionByIDRequest{
			VersionID:  req.VersionID,
			TenantInfo: req.TenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}

	archived, err := s.versionRepo.Archive(ctx, req)
	if err != nil {
		return nil, err
	}

	s.logAudit(
		ctx,
		archived,
		original,
		permission.OpArchive,
		req.ArchivedByID,
		"Document template version archived",
	)

	return archived, nil
}

func (s *Service) DeleteVersion(
	ctx context.Context,
	req *repositories.GetDocumentTemplateVersionByIDRequest,
	userID pulid.ID,
) error {
	original, err := s.versionRepo.GetByID(ctx, req)
	if err != nil {
		return err
	}

	if err = s.versionRepo.Delete(ctx, req); err != nil {
		return err
	}

	s.logAudit(
		ctx, nil, original, permission.OpDelete, userID, "Document template version deleted",
	)

	return nil
}

// Assign points one customer at one template.
func (s *Service) Assign(
	ctx context.Context,
	entity *documenttemplate.DocumentTemplateAssignment,
	userID pulid.ID,
) (*documenttemplate.DocumentTemplateAssignment, error) {
	template, err := s.repo.GetByID(ctx, &repositories.GetDocumentTemplateByIDRequest{
		DocumentTemplateID: entity.TemplateID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	})
	if err != nil {
		return nil, err
	}

	// Kind is denormalized onto the assignment so resolution can filter without
	// joining. Taking it from the template rather than the caller keeps the two
	// from ever disagreeing.
	entity.Kind = template.Kind
	entity.AssignedByID = &userID

	if multiErr := s.validator.ValidateAssignment(ctx, entity, template); multiErr != nil {
		return nil, multiErr
	}

	assigned, err := s.assignmentRepo.Assign(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.logAudit(ctx, assigned, nil, permission.OpAssign, userID, "Document template assigned")

	return assigned, nil
}

func (s *Service) Unassign(
	ctx context.Context,
	req *repositories.UnassignDocumentTemplateRequest,
	userID pulid.ID,
) error {
	if err := s.assignmentRepo.Unassign(ctx, req); err != nil {
		return err
	}

	s.logAudit(ctx, nil, nil, permission.OpUnassign, userID, fmt.Sprintf(
		"Document template unassigned from customer %s for %s", req.CustomerID, req.Kind,
	))

	return nil
}

func (s *Service) ListAssignments(
	ctx context.Context,
	req *repositories.ListDocumentTemplateAssignmentsRequest,
) (*pagination.ListResult[*documenttemplate.DocumentTemplateAssignment], error) {
	return s.assignmentRepo.List(ctx, req)
}

func (s *Service) ListConnection(
	ctx context.Context,
	req *repositories.ListDocumentTemplateConnectionRequest,
) (*pagination.CursorListResult[*documenttemplate.DocumentTemplate], error) {
	return s.repo.ListConnection(ctx, req)
}

func (s *Service) ListAssignmentsConnection(
	ctx context.Context,
	req *repositories.ListDocumentTemplateAssignmentConnectionRequest,
) (*pagination.CursorListResult[*documenttemplate.DocumentTemplateAssignment], error) {
	return s.assignmentRepo.ListConnection(ctx, req)
}

// KindOf reports the catalog a version is written against, loading the template
// when the relation was not fetched with it.
func (s *Service) KindOf(
	ctx context.Context,
	version *documenttemplate.DocumentTemplateVersion,
	tenantInfo pagination.TenantInfo,
) documenttemplate.Kind {
	return s.kindOf(ctx, version, tenantInfo)
}

func (s *Service) GetCustomerAssignments(
	ctx context.Context,
	req *repositories.GetCustomerAssignmentsRequest,
) ([]*documenttemplate.DocumentTemplateAssignment, error) {
	return s.assignmentRepo.GetForCustomer(ctx, req)
}

// stampContentHash records the identity of the bytes being saved. The column is
// NOT NULL because a version with no hash could not be matched to the documents
// it produced.
func (s *Service) stampContentHash(entity *documenttemplate.DocumentTemplateVersion) {
	content := contentFromVersion(entity)
	entity.ContentHash = ContentHash(&content)
}

// kindOf finds the catalog a version is written against.
func (s *Service) kindOf(
	ctx context.Context,
	version *documenttemplate.DocumentTemplateVersion,
	tenantInfo pagination.TenantInfo,
) documenttemplate.Kind {
	if version.Template != nil && version.Template.Kind != "" {
		return version.Template.Kind
	}

	template, err := s.repo.GetByID(ctx, &repositories.GetDocumentTemplateByIDRequest{
		DocumentTemplateID: version.TemplateID,
		TenantInfo:         tenantInfo,
	})
	if err != nil {
		s.l.Warn("failed to load the template a version belongs to",
			zap.String("versionId", version.ID.String()), zap.Error(err))
		return ""
	}

	return template.Kind
}

// StarterDrifted reports whether the built-in has changed since a version was
// seeded from it, which is what lets the editor say an organization's copy is
// behind the shipped default rather than leaving it silently stale.
func (s *Service) StarterDrifted(
	version *documenttemplate.DocumentTemplateVersion,
	kind documenttemplate.Kind,
) bool {
	if version.StarterHash == "" {
		return false
	}

	current, err := starters.Hash(kind)
	if err != nil {
		return false
	}

	return current != version.StarterHash
}

//nolint:unparam // ctx is carried for consistency with every other logAudit here.
func (s *Service) logAudit(
	ctx context.Context,
	current any,
	previous any,
	operation permission.Operation,
	userID pulid.ID,
	comment string,
	extra ...services.LogOption,
) {
	// A delete has no current state, so identity comes from what was removed.
	subject := current
	if subject == nil {
		subject = previous
	}

	orgID, buID := pulid.Nil, pulid.Nil
	resourceID := ""

	if entity, ok := subject.(interface{ GetOrganizationID() pulid.ID }); ok {
		orgID = entity.GetOrganizationID()
	}
	if entity, ok := subject.(interface{ GetBusinessUnitID() pulid.ID }); ok {
		buID = entity.GetBusinessUnitID()
	}
	if entity, ok := subject.(interface{ GetID() pulid.ID }); ok {
		resourceID = entity.GetID().String()
	}

	currentState := map[string]any{}
	if current != nil {
		currentState = jsonutils.MustToJSON(current)
	}

	previousState := map[string]any{}
	if previous != nil {
		previousState = jsonutils.MustToJSON(previous)
	}

	options := append([]services.LogOption{
		auditservice.WithComment(comment),
		auditservice.WithDiff(previous, current),
	}, extra...)

	if err := s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceDocumentTemplate,
		ResourceID:     resourceID,
		Operation:      operation,
		UserID:         userID,
		CurrentState:   currentState,
		PreviousState:  previousState,
		OrganizationID: orgID,
		BusinessUnitID: buID,
	}, options...); err != nil {
		s.l.Warn("failed to log document template audit entry", zap.Error(err))
	}
}

// RecordGeneratedDocument logs one rendered artifact against the version that
// produced it.
//
// The record is written as Completed with the file already stored, because
// callers reach here after the upload succeeded. A render that failed never gets
// a row: the point of the table is reproducing what a customer received, and a
// document nobody received is noise in that record.
func (s *Service) RecordGeneratedDocument(
	ctx context.Context,
	req *services.RecordGeneratedDocumentRequest,
) (*documenttemplate.GeneratedDocument, error) {
	if req.Rendered == nil {
		return nil, errortypes.NewValidationError(
			"rendered", errortypes.ErrRequired,
			"A generated document must name the render that produced it",
		)
	}

	entity := &documenttemplate.GeneratedDocument{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		Kind:           req.Kind,
		// Nil for a built-in render, which has no row. ContentHash still
		// identifies the exact content in that case.
		TemplateID:        req.Rendered.TemplateID,
		TemplateVersionID: req.Rendered.VersionID,
		DocumentID:        req.DocumentID,
		ReferenceType:     req.ReferenceType,
		ReferenceID:       req.ReferenceID,
		FileName:          req.FileName,
		FileSize:          req.FileSize,
		Checksum:          req.Checksum,
		ContentHash:       req.Rendered.ContentHash,
		// Whatever the render dropped travels with the record, so an omission is
		// discoverable months later rather than only in that day's logs.
		Warnings: req.Rendered.Warnings,
		// Set here rather than left to BeforeAppendModel: that hook runs at
		// insert, which is after the validation below, so an unset value would
		// fail a check the insert would have satisfied. Delivery is recorded
		// separately when the document actually reaches someone.
		DeliveryMethod: documenttemplate.DeliveryMethodNone,
	}

	if req.UserID.IsNotNil() {
		id := req.UserID
		entity.GeneratedByID = &id
	}

	entity.MarkCompleted(req.FilePath, req.FileSize, req.Checksum)

	// The same rules the database CHECK constraints enforce, checked here so a
	// caller gets a field error rather than an opaque constraint violation.
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return s.generatedRepo.Create(ctx, entity)
}
