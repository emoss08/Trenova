package formulatemplateservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

const effectiveFromGraceSeconds int64 = 3600

func (s *Service) UpdateVersionEffectiveDate(
	ctx context.Context,
	req *repositories.UpdateEffectiveDateRequest,
) (*formulatemplate.FormulaTemplateVersion, error) {
	log := s.l.With(
		zap.String("operation", "UpdateVersionEffectiveDate"),
		zap.String("templateID", req.TemplateID.String()),
		zap.Int64("versionNumber", req.VersionNumber),
	)

	if _, err := s.versionRepo.GetByTemplateAndVersion(ctx, &repositories.GetVersionRequest{
		TenantInfo:    req.TenantInfo,
		TemplateID:    req.TemplateID,
		VersionNumber: req.VersionNumber,
	}); err != nil {
		log.Error("failed to get version", zap.Error(err))
		return nil, err
	}

	template, err := s.getTemplateByIDWithTenant(ctx, req.TemplateID, req.TenantInfo)
	if err != nil {
		log.Error("failed to get formula template", zap.Error(err))
		return nil, err
	}

	if req.EffectiveFrom != nil {
		if template.Status != formulatemplate.StatusActive {
			return nil, errortypes.NewValidationError(
				"effectiveFrom",
				errortypes.ErrInvalid,
				"Only Active templates can have scheduled versions",
			)
		}

		if *req.EffectiveFrom <= timeutils.NowUnix()-effectiveFromGraceSeconds {
			return nil, errortypes.NewValidationError(
				"effectiveFrom",
				errortypes.ErrInvalid,
				"Effective date cannot be more than one hour in the past",
			)
		}
	}

	updated, err := s.versionRepo.UpdateEffectiveDate(ctx, req)
	if err != nil {
		log.Error("failed to update version effective date", zap.Error(err))
		return nil, err
	}

	s.logAuditAction(
		log,
		template,
		permission.OpUpdate,
		req.TenantInfo.UserID,
		nil,
		"Formula template version effective date updated",
		auditservice.WithMetadata(map[string]any{
			"versionNumber": req.VersionNumber,
			"effectiveFrom": req.EffectiveFrom,
		}),
	)

	return updated, nil
}

func (s *Service) ListScheduledVersions(
	ctx context.Context,
	req *repositories.ListScheduledVersionsRequest,
) ([]*formulatemplate.FormulaTemplateVersion, error) {
	return s.versionRepo.ListScheduled(ctx, req)
}
