package workerchecklistservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

func (s *Service) ListTemplates(
	ctx context.Context,
	req *repositories.ListChecklistTemplatesRequest,
) (*pagination.CursorListResult[*worker.WorkerChecklistTemplate], error) {
	return s.repo.ListTemplates(ctx, req)
}

func (s *Service) ActiveTemplates(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*worker.WorkerChecklistTemplate, error) {
	return s.repo.ListActiveTemplates(ctx, tenantInfo)
}

func (s *Service) GetTemplate(
	ctx context.Context,
	req *repositories.GetChecklistTemplateByIDRequest,
) (*worker.WorkerChecklistTemplate, error) {
	return s.repo.GetTemplateByID(ctx, req)
}

func (s *Service) validateTemplate(
	ctx context.Context,
	entity *worker.WorkerChecklistTemplate,
) error {
	entity.NormalizeCode()
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)

	if entity.Code != "" {
		exists, err := s.repo.TemplateCodeExists(
			ctx,
			&repositories.ChecklistTemplateCodeExistsRequest{
				TenantInfo: templateTenant(entity),
				Code:       entity.Code,
				ExcludeID:  entity.ID,
			},
		)
		if err != nil {
			return err
		}
		if exists {
			multiErr.Add(
				"code",
				errortypes.ErrDuplicate,
				"A checklist template with this code already exists",
			)
		}
	}

	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func (s *Service) CreateTemplate(
	ctx context.Context,
	entity *worker.WorkerChecklistTemplate,
	userID pulid.ID,
) (*worker.WorkerChecklistTemplate, error) {
	log := s.l.With(zap.String("operation", "CreateTemplate"))

	if err := s.validateTemplate(ctx, entity); err != nil {
		return nil, err
	}

	created, err := s.repo.CreateTemplate(ctx, entity)
	if err != nil {
		log.Error("failed to create checklist template", zap.Error(err))
		return nil, err
	}

	s.audit(&auditParams{
		resource:   permission.ResourceWorkerChecklistTemplate,
		resourceID: created.GetResourceID(),
		operation:  permission.OpCreate,
		userID:     userID,
		tenant:     templateTenant(created),
		current:    created,
		comment:    "Checklist template created",
		log:        log,
	})
	s.publish(
		ctx,
		templateTenant(created),
		realtimeTemplate,
		permission.OpCreate,
		created.ID,
		userID,
	)

	return created, nil
}

func (s *Service) UpdateTemplate(
	ctx context.Context,
	entity *worker.WorkerChecklistTemplate,
	userID pulid.ID,
) (*worker.WorkerChecklistTemplate, error) {
	log := s.l.With(zap.String("operation", "UpdateTemplate"), zap.String("id", entity.ID.String()))

	original, err := s.repo.GetTemplateByID(ctx, &repositories.GetChecklistTemplateByIDRequest{
		ID:           entity.ID,
		TenantInfo:   templateTenant(entity),
		IncludeItems: true,
	})
	if err != nil {
		return nil, err
	}
	if err = s.validateTemplate(ctx, entity); err != nil {
		return nil, err
	}

	updated, err := s.repo.UpdateTemplate(ctx, entity)
	if err != nil {
		log.Error("failed to update checklist template", zap.Error(err))
		return nil, err
	}

	s.audit(&auditParams{
		resource:   permission.ResourceWorkerChecklistTemplate,
		resourceID: updated.GetResourceID(),
		operation:  permission.OpUpdate,
		userID:     userID,
		tenant:     templateTenant(updated),
		current:    updated,
		previous:   original,
		comment:    "Checklist template updated",
		log:        log,
	})
	s.publish(
		ctx,
		templateTenant(updated),
		realtimeTemplate,
		permission.OpUpdate,
		updated.ID,
		userID,
	)

	return updated, nil
}

type TemplateStatusRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
	Version    int64
	UserID     pulid.ID
}

func (s *Service) ArchiveTemplate(
	ctx context.Context,
	req *TemplateStatusRequest,
) (*worker.WorkerChecklistTemplate, error) {
	return s.changeTemplateStatus(ctx, req, domaintypes.StatusInactive, permission.OpArchive)
}

func (s *Service) RestoreTemplate(
	ctx context.Context,
	req *TemplateStatusRequest,
) (*worker.WorkerChecklistTemplate, error) {
	return s.changeTemplateStatus(ctx, req, domaintypes.StatusActive, permission.OpRestore)
}

func (s *Service) changeTemplateStatus(
	ctx context.Context,
	req *TemplateStatusRequest,
	target domaintypes.Status,
	operation permission.Operation,
) (*worker.WorkerChecklistTemplate, error) {
	log := s.l.With(zap.String("operation", string(operation)), zap.String("id", req.ID.String()))

	original, err := s.repo.GetTemplateByID(ctx, &repositories.GetChecklistTemplateByIDRequest{
		ID:           req.ID,
		TenantInfo:   req.TenantInfo,
		IncludeItems: true,
	})
	if err != nil {
		return nil, err
	}
	if original.Status == target {
		return original, nil
	}
	if req.Version > 0 && original.Version != req.Version {
		return nil, errortypes.NewValidationError(
			"version",
			errortypes.ErrVersionMismatch,
			"Checklist template was changed by someone else. Reload and try again",
		)
	}

	updated := *original
	updated.Items = original.Items
	updated.Status = target
	if target == domaintypes.StatusInactive {
		updated.IsDefault = false
	}

	saved, err := s.repo.UpdateTemplate(ctx, &updated)
	if err != nil {
		log.Error("failed to change checklist template status", zap.Error(err))
		return nil, err
	}

	s.audit(&auditParams{
		resource:   permission.ResourceWorkerChecklistTemplate,
		resourceID: saved.GetResourceID(),
		operation:  operation,
		userID:     req.UserID,
		tenant:     templateTenant(saved),
		current:    saved,
		previous:   original,
		comment:    fmt.Sprintf("Checklist template %s", target),
		log:        log,
	})
	s.publish(ctx, templateTenant(saved), realtimeTemplate, operation, saved.ID, req.UserID)

	return saved, nil
}

func (s *Service) CountOpenChecklistsForTemplate(
	ctx context.Context,
	entity *worker.WorkerChecklistTemplate,
) (int, error) {
	return s.repo.CountOpenChecklists(ctx, &repositories.CountOpenChecklistsRequest{
		TenantInfo: templateTenant(entity),
		TemplateID: entity.ID,
	})
}

func (s *Service) CountOpenChecklistsForTemplates(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	templateIDs []pulid.ID,
) (map[pulid.ID]int, error) {
	return s.repo.CountOpenChecklistsByTemplateIDs(
		ctx,
		&repositories.CountOpenChecklistsByTemplateIDsRequest{
			TenantInfo:  tenantInfo,
			TemplateIDs: templateIDs,
		},
	)
}
