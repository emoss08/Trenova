package performancereviewservice

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
	req *repositories.ListReviewTemplatesRequest,
) (*pagination.CursorListResult[*worker.PerformanceReviewTemplate], error) {
	return s.repo.ListTemplates(ctx, req)
}

func (s *Service) GetTemplate(
	ctx context.Context,
	req *repositories.GetReviewTemplateByIDRequest,
) (*worker.PerformanceReviewTemplate, error) {
	return s.repo.GetTemplateByID(ctx, req)
}

func (s *Service) ActiveTemplates(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*worker.PerformanceReviewTemplate, error) {
	return s.repo.ListActiveTemplates(ctx, tenantInfo)
}

func (s *Service) TemplateSelectOptions(
	ctx context.Context,
	req *repositories.ReviewTemplateSelectOptionsRequest,
) (*pagination.ListResult[*worker.PerformanceReviewTemplate], error) {
	return s.repo.TemplateSelectOptions(ctx, req)
}

func (s *Service) validateTemplate(
	ctx context.Context,
	entity *worker.PerformanceReviewTemplate,
) error {
	entity.NormalizeCode()
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)

	if entity.Code != "" {
		exists, err := s.repo.TemplateCodeExists(ctx, &repositories.ReviewTemplateCodeExistsRequest{
			TenantInfo: templateTenant(entity),
			Code:       entity.Code,
			ExcludeID:  entity.ID,
		})
		if err != nil {
			return err
		}
		if exists {
			multiErr.Add(
				"code",
				errortypes.ErrDuplicate,
				"A review template with this code already exists",
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
	entity *worker.PerformanceReviewTemplate,
	userID pulid.ID,
) (*worker.PerformanceReviewTemplate, error) {
	log := s.l.With(zap.String("operation", "CreateTemplate"))

	if err := s.validateTemplate(ctx, entity); err != nil {
		return nil, err
	}
	if entity.IsDefault {
		if err := s.repo.ClearDefaultTemplate(ctx, templateTenant(entity), pulid.Nil); err != nil {
			return nil, err
		}
	}

	created, err := s.repo.CreateTemplate(ctx, entity)
	if err != nil {
		log.Error("failed to create review template", zap.Error(err))
		return nil, err
	}

	s.audit(&auditParams{
		resource: permission.ResourcePerformanceReviewTemplate, resourceID: created.GetResourceID(),
		operation: permission.OpCreate, userID: userID, tenant: templateTenant(created),
		current: created, comment: "Review template created", log: log,
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
	entity *worker.PerformanceReviewTemplate,
	userID pulid.ID,
) (*worker.PerformanceReviewTemplate, error) {
	log := s.l.With(zap.String("operation", "UpdateTemplate"), zap.String("id", entity.ID.String()))

	original, err := s.repo.GetTemplateByID(ctx, &repositories.GetReviewTemplateByIDRequest{
		ID:         entity.ID,
		TenantInfo: templateTenant(entity),
	})
	if err != nil {
		return nil, err
	}
	if err = s.validateTemplate(ctx, entity); err != nil {
		return nil, err
	}
	if original.Status == domaintypes.StatusActive && entity.Status != domaintypes.StatusActive {
		if err = s.requireTemplateRetirable(ctx, original); err != nil {
			return nil, err
		}
	}
	if entity.IsDefault && !original.IsDefault {
		if err = s.repo.ClearDefaultTemplate(ctx, templateTenant(entity), entity.ID); err != nil {
			return nil, err
		}
	}

	updated, err := s.repo.UpdateTemplate(ctx, entity)
	if err != nil {
		log.Error("failed to update review template", zap.Error(err))
		return nil, err
	}

	s.audit(&auditParams{
		resource: permission.ResourcePerformanceReviewTemplate, resourceID: updated.GetResourceID(),
		operation: permission.OpUpdate, userID: userID, tenant: templateTenant(updated),
		current: updated, previous: original, comment: "Review template updated", log: log,
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

type TemplateStatusChangeRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
	Version    int64
	UserID     pulid.ID
}

func (s *Service) ArchiveTemplate(
	ctx context.Context,
	req *TemplateStatusChangeRequest,
) (*worker.PerformanceReviewTemplate, error) {
	return s.changeTemplateStatus(ctx, req, domaintypes.StatusInactive, permission.OpArchive)
}

func (s *Service) RestoreTemplate(
	ctx context.Context,
	req *TemplateStatusChangeRequest,
) (*worker.PerformanceReviewTemplate, error) {
	return s.changeTemplateStatus(ctx, req, domaintypes.StatusActive, permission.OpRestore)
}

func (s *Service) changeTemplateStatus(
	ctx context.Context,
	req *TemplateStatusChangeRequest,
	target domaintypes.Status,
	operation permission.Operation,
) (*worker.PerformanceReviewTemplate, error) {
	log := s.l.With(zap.String("operation", string(operation)), zap.String("id", req.ID.String()))

	original, err := s.repo.GetTemplateByID(ctx, &repositories.GetReviewTemplateByIDRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
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
			"Template was changed by someone else. Reload and try again",
		)
	}
	if target == domaintypes.StatusInactive {
		if err = s.requireTemplateRetirable(ctx, original); err != nil {
			return nil, err
		}
	}

	updated := *original
	updated.Status = target
	if target == domaintypes.StatusInactive {
		updated.IsDefault = false
	}
	saved, err := s.repo.UpdateTemplate(ctx, &updated)
	if err != nil {
		log.Error("failed to change review template status", zap.Error(err))
		return nil, err
	}

	s.audit(&auditParams{
		resource: permission.ResourcePerformanceReviewTemplate, resourceID: saved.GetResourceID(),
		operation: operation, userID: req.UserID, tenant: req.TenantInfo,
		current: saved, previous: original, comment: fmt.Sprintf("Review template %s", target), log: log,
	})
	s.publish(ctx, req.TenantInfo, realtimeTemplate, operation, saved.ID, req.UserID)
	return saved, nil
}

func (s *Service) requireTemplateRetirable(
	ctx context.Context,
	entity *worker.PerformanceReviewTemplate,
) error {
	count, err := s.repo.CountReviewsByTemplate(ctx, &repositories.CountReviewsByTemplateRequest{
		TenantInfo: templateTenant(entity),
		TemplateID: entity.ID,
		OpenOnly:   true,
	})
	if err != nil {
		return err
	}
	if count > 0 {
		return errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"{0} open review{1} still use this template. Close those first", count, plural(count),
		)
	}
	return nil
}

func (s *Service) CountOpenReviews(
	ctx context.Context,
	entity *worker.PerformanceReviewTemplate,
) (int, error) {
	return s.repo.CountReviewsByTemplate(ctx, &repositories.CountReviewsByTemplateRequest{
		TenantInfo: templateTenant(entity),
		TemplateID: entity.ID,
		OpenOnly:   true,
	})
}

func (s *Service) CountOpenReviewsByTemplates(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	templateIDs []pulid.ID,
) (map[pulid.ID]int, error) {
	return s.repo.CountReviewsByTemplateIDs(ctx, &repositories.CountReviewsByTemplateIDsRequest{
		TenantInfo:  tenantInfo,
		TemplateIDs: templateIDs,
		OpenOnly:    true,
	})
}

func plural(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}
