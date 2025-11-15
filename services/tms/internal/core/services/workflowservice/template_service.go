package workflowservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/workflow"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils/jsonutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type TemplateServiceParams struct {
	fx.In

	Logger       *zap.Logger
	Repo         repositories.WorkflowTemplateRepository
	AuditService services.AuditService
}

type TemplateService struct {
	l    *zap.Logger
	repo repositories.WorkflowTemplateRepository
	as   services.AuditService
}

func NewTemplateService(p TemplateServiceParams) *TemplateService {
	return &TemplateService{
		l:    p.Logger.Named("service.workflow-template"),
		repo: p.Repo,
		as:   p.AuditService,
	}
}

func (s *TemplateService) List(
	ctx context.Context,
	req *repositories.ListWorkflowTemplateRequest,
) (*pagination.ListResult[*workflow.WorkflowTemplate], error) {
	return s.repo.List(ctx, req)
}

func (s *TemplateService) Get(
	ctx context.Context,
	req repositories.GetWorkflowTemplateByIDRequest,
) (*workflow.WorkflowTemplate, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *TemplateService) Create(
	ctx context.Context,
	entity *workflow.WorkflowTemplate,
	userID pulid.ID,
) (*workflow.WorkflowTemplate, error) {
	log := s.l.With(
		zap.String("operation", "Create"),
		zap.String("name", entity.Name),
		zap.String("buID", entity.BusinessUnitID.String()),
		zap.String("orgID", entity.OrganizationID.String()),
		zap.String("userID", userID.String()),
	)

	// Validate
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	// Set created by
	entity.CreatedBy = userID

	createdEntity, err := s.repo.Create(ctx, entity)
	if err != nil {
		log.Error("failed to create workflow template", zap.Error(err))
		return nil, err
	}

	// Audit log
	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceWorkflowTemplate,
			ResourceID:     createdEntity.GetID(),
			Operation:      permission.OpCreate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(createdEntity),
			OrganizationID: createdEntity.OrganizationID,
			BusinessUnitID: createdEntity.BusinessUnitID,
		},
		audit.WithComment("Workflow template created"),
	)
	if err != nil {
		log.Error("failed to log template creation", zap.Error(err))
	}

	return createdEntity, nil
}

func (s *TemplateService) Update(
	ctx context.Context,
	entity *workflow.WorkflowTemplate,
	userID pulid.ID,
) (*workflow.WorkflowTemplate, error) {
	log := s.l.With(
		zap.String("operation", "Update"),
		zap.String("templateID", entity.ID.String()),
		zap.String("userID", userID.String()),
	)

	// Validate
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	// Get original
	original, err := s.repo.GetByID(ctx, repositories.GetWorkflowTemplateByIDRequest{
		ID:     entity.ID,
		OrgID:  entity.OrganizationID,
		BuID:   entity.BusinessUnitID,
		UserID: userID,
	})
	if err != nil {
		log.Error("failed to get original template", zap.Error(err))
		return nil, err
	}

	// Don't allow updating system templates
	if original.IsSystemTemplate {
		return nil, errortypes.NewBusinessError("Cannot update system templates")
	}

	updatedEntity, err := s.repo.Update(ctx, entity)
	if err != nil {
		log.Error("failed to update workflow template", zap.Error(err))
		return nil, err
	}

	// Audit log
	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceWorkflowTemplate,
			ResourceID:     updatedEntity.GetID(),
			Operation:      permission.OpUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(updatedEntity),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: updatedEntity.OrganizationID,
			BusinessUnitID: updatedEntity.BusinessUnitID,
		},
		audit.WithComment("Workflow template updated"),
		audit.WithDiff(original, updatedEntity),
	)
	if err != nil {
		log.Error("failed to log template update", zap.Error(err))
	}

	return updatedEntity, nil
}

func (s *TemplateService) Delete(
	ctx context.Context,
	id, orgID, buID, userID pulid.ID,
) error {
	log := s.l.With(
		zap.String("operation", "Delete"),
		zap.String("templateID", id.String()),
		zap.String("userID", userID.String()),
	)

	// Get template to check if it can be deleted
	template, err := s.repo.GetByID(ctx, repositories.GetWorkflowTemplateByIDRequest{
		ID:     id,
		OrgID:  orgID,
		BuID:   buID,
		UserID: userID,
	})
	if err != nil {
		return err
	}

	// Don't allow deletion of system templates
	if template.IsSystemTemplate {
		return errortypes.NewBusinessError("Cannot delete system templates")
	}

	err = s.repo.Delete(ctx, id, orgID, buID)
	if err != nil {
		log.Error("failed to delete workflow template", zap.Error(err))
		return err
	}

	// Audit log
	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceWorkflowTemplate,
			ResourceID:     id.String(),
			Operation:      permission.OpDelete,
			UserID:         userID,
			PreviousState:  jsonutils.MustToJSON(template),
			OrganizationID: orgID,
			BusinessUnitID: buID,
		},
		audit.WithComment("Workflow template deleted"),
	)
	if err != nil {
		log.Error("failed to log template deletion", zap.Error(err))
	}

	return nil
}

// System and Public Templates

func (s *TemplateService) GetSystemTemplates(
	ctx context.Context,
) ([]*workflow.WorkflowTemplate, error) {
	return s.repo.GetSystemTemplates(ctx)
}

func (s *TemplateService) GetPublicTemplates(
	ctx context.Context,
	orgID, buID pulid.ID,
) ([]*workflow.WorkflowTemplate, error) {
	return s.repo.GetPublicTemplates(ctx, orgID, buID)
}

// Usage tracking

func (s *TemplateService) UseTemplate(
	ctx context.Context,
	id, orgID, buID, userID pulid.ID,
) (*workflow.WorkflowTemplate, error) {
	log := s.l.With(
		zap.String("operation", "UseTemplate"),
		zap.String("templateID", id.String()),
		zap.String("userID", userID.String()),
	)

	// Get template
	template, err := s.repo.GetByID(ctx, repositories.GetWorkflowTemplateByIDRequest{
		ID:     id,
		OrgID:  orgID,
		BuID:   buID,
		UserID: userID,
	})
	if err != nil {
		return nil, err
	}

	// Increment usage count
	err = s.repo.IncrementUsage(ctx, id, orgID, buID)
	if err != nil {
		log.Error("failed to increment template usage", zap.Error(err))
		// Don't fail the operation if usage tracking fails
	}

	return template, nil
}
