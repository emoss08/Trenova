package casechecklistservice

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/deskcase"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	Checklists   repositories.CaseChecklistRepository
	AuditService services.AuditService
}

// Service keeps how an organization, and any customer of it, wants case
// checklists laid out. A save is checked whole: every built-in step of the
// kind once, locked steps left required, added steps named and checked by
// something that can tick them, and their document types the
// organization's.
type Service struct {
	checklists repositories.CaseChecklistRepository
	audit      services.AuditService
	l          *zap.Logger
}

//nolint:gocritic // fx passes params by value
func New(p Params) services.CaseChecklistService {
	return &Service{
		checklists: p.Checklists,
		audit:      p.AuditService,
		l:          p.Logger.Named("service.case-checklist"),
	}
}

// List is the organization's template of a kind (the default, unsaved,
// when it has none) and every customer's, each with its steps brought up
// to date with the steps Trenova defines.
func (s *Service) List(
	ctx context.Context,
	req *services.ListCaseChecklistTemplatesRequest,
) (*services.CaseChecklistTemplates, error) {
	if !req.Kind.IsValid() {
		return nil, errortypes.NewValidationError(
			"kind", errortypes.ErrInvalid, "Choose ready to bill or ready to close",
		)
	}

	templates, err := s.checklists.ListTemplates(ctx, &repositories.ListCaseChecklistTemplatesRequest{
		TenantInfo: req.TenantInfo,
		Kind:       req.Kind,
	})
	if err != nil {
		return nil, err
	}

	out := &services.CaseChecklistTemplates{
		Kind:      req.Kind,
		Customers: make([]*deskcase.ChecklistTemplate, 0, len(templates)),
		Locked:    deskcase.LockedKeys(req.Kind),
	}
	for _, template := range templates {
		template.Items = deskcase.Normalize(req.Kind, template.Items)
		if template.CustomerID.IsNil() {
			out.Organization = template
			continue
		}
		out.Customers = append(out.Customers, template)
	}
	if out.Organization == nil {
		out.Organization = &deskcase.ChecklistTemplate{
			OrganizationID: req.TenantInfo.OrgID,
			BusinessUnitID: req.TenantInfo.BuID,
			Kind:           req.Kind,
			Items:          deskcase.DefaultItems(req.Kind),
		}
	}

	return out, nil
}

func (s *Service) Save(
	ctx context.Context,
	req *services.SaveCaseChecklistTemplateRequest,
) (*deskcase.ChecklistTemplate, error) {
	entity := &deskcase.ChecklistTemplate{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		Kind:           req.Kind,
		CustomerID:     req.CustomerID,
		Items:          req.Items,
		UpdatedByID:    req.UserID,
	}

	var previous *deskcase.ChecklistTemplate
	if req.ID.IsNotNil() {
		stored, err := s.checklists.GetTemplate(ctx, &repositories.GetCaseChecklistTemplateRequest{
			TenantInfo: req.TenantInfo,
			ID:         req.ID,
		})
		if err != nil {
			return nil, err
		}
		previous = stored
		entity.ID = stored.ID
		entity.Kind = stored.Kind
		entity.CustomerID = stored.CustomerID
		entity.Version = req.Version
		entity.CreatedAt = stored.CreatedAt
	}

	entity.Clean()
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}
	if err := s.checkDocumentTypes(ctx, req, entity); err != nil {
		return nil, err
	}

	saved, err := s.checklists.SaveTemplate(ctx, entity)
	switch {
	case errors.Is(err, repositories.ErrCaseChecklistTemplateStale):
		return nil, errortypes.NewConflictError(
			"Someone else saved this checklist while you were editing it. Reload it and make your change again",
		)
	case dberror.IsForeignKeyConstraintViolation(err):
		return nil, errortypes.NewValidationError(
			"customerId", errortypes.ErrNotFound, "There is no such customer",
		)
	case err != nil:
		return nil, err
	}

	s.record(saved, previous, req.UserID, permission.OpUpdate, "Case checklist saved")

	return saved, nil
}

// checkDocumentTypes refuses an added step that waits on a document type
// that is not the organization's.
func (s *Service) checkDocumentTypes(
	ctx context.Context,
	req *services.SaveCaseChecklistTemplateRequest,
	entity *deskcase.ChecklistTemplate,
) error {
	ids := entity.DocumentTypeIDs()
	if len(ids) == 0 {
		return nil
	}

	count, err := s.checklists.CountDocumentTypes(ctx, &repositories.CountCaseChecklistDocumentTypesRequest{
		TenantInfo: req.TenantInfo,
		IDs:        ids,
	})
	if err != nil {
		return err
	}
	if count != len(ids) {
		return errortypes.NewValidationError(
			"items", errortypes.ErrNotFound,
			"A step waits on a document type that does not exist",
		)
	}

	return nil
}

// Delete removes a template: a customer's goes back to the organization's,
// the organization's back to the default.
func (s *Service) Delete(ctx context.Context, req *services.DeleteCaseChecklistTemplateRequest) error {
	stored, err := s.checklists.GetTemplate(ctx, &repositories.GetCaseChecklistTemplateRequest{
		TenantInfo: req.TenantInfo,
		ID:         req.ID,
	})
	if err != nil {
		return err
	}
	if err = s.checklists.DeleteTemplate(ctx, &repositories.GetCaseChecklistTemplateRequest{
		TenantInfo: req.TenantInfo,
		ID:         req.ID,
	}); err != nil {
		return err
	}

	s.record(stored, nil, req.UserID, permission.OpDelete, "Case checklist removed")

	return nil
}

func (s *Service) record(
	current, previous *deskcase.ChecklistTemplate,
	userID pulid.ID,
	operation permission.Operation,
	comment string,
) {
	params := &services.LogActionParams{
		Resource:       permission.ResourceBillingControl,
		ResourceID:     current.ID.String(),
		Operation:      operation,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(current),
		OrganizationID: current.OrganizationID,
		BusinessUnitID: current.BusinessUnitID,
	}
	options := []services.LogOption{auditservice.WithComment(comment)}
	if previous != nil {
		params.PreviousState = jsonutils.MustToJSON(previous)
		options = append(options, auditservice.WithDiff(previous, current))
	}
	if err := s.audit.LogAction(params, options...); err != nil {
		s.l.Error("failed to log a case checklist change", zap.Error(err))
	}
}
