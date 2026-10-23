package formulatemplateservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/rating/app/tariff"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const versionNumberField = "versionNumber"

const effectiveFromGraceSeconds int64 = 3600

func validateEffectiveFrom(effectiveFrom int64) error {
	if effectiveFrom <= timeutils.NowUnix()-effectiveFromGraceSeconds {
		return errortypes.NewValidationError(
			"effectiveFrom",
			errortypes.ErrInvalid,
			"Effective date cannot be more than one hour in the past; earlier shipments keep the version that billed them",
		)
	}
	return nil
}

func (s *Service) UpdateVersionEffectiveDate(
	ctx context.Context,
	req *formulatemplateports.UpdateEffectiveDateRequest,
) (*formulatemplate.FormulaTemplateVersion, error) {
	log := s.l.With(
		zap.String("operation", "UpdateVersionEffectiveDate"),
		zap.String("templateID", req.TemplateID.String()),
		zap.Int64(versionNumberField, req.VersionNumber),
	)

	version, err := s.versionRepo.GetByTemplateAndVersion(
		ctx,
		&formulatemplateports.GetVersionRequest{
			TenantInfo:    req.TenantInfo,
			TemplateID:    req.TemplateID,
			VersionNumber: req.VersionNumber,
		},
	)
	if err != nil {
		log.Error("failed to get version", zap.Error(err))
		return nil, err
	}

	template, err := s.getTemplateByIDWithTenant(ctx, req.TemplateID, req.TenantInfo)
	if err != nil {
		log.Error("failed to get formula template", zap.Error(err))
		return nil, err
	}

	if template.Status != formulatemplate.StatusActive {
		return nil, errortypes.NewValidationError(
			"effectiveFrom",
			errortypes.ErrInvalid,
			"Only Active templates can have scheduled versions",
		)
	}

	if version.Status != formulatemplate.StatusActive {
		return nil, errortypes.NewValidationError(
			versionNumberField,
			errortypes.ErrInvalid,
			"Only approved snapshots can be scheduled; this version was captured while the template was {0}",
			version.Status.String(),
		)
	}

	now := timeutils.NowUnix()
	previousStart := version.EffectiveStart()
	started := previousStart <= now

	switch {
	case req.EffectiveFrom == nil && started:
		return nil, errortypes.NewValidationError(
			"effectiveFrom",
			errortypes.ErrInvalid,
			"Version {0} has already billed shipments; a window that has started keeps its date",
			version.VersionNumber,
		)
	case req.EffectiveFrom == nil:
		req.EffectiveFrom = &now
	default:
		if err = validateEffectiveFrom(*req.EffectiveFrom); err != nil {
			return nil, err
		}
	}

	if started {
		return s.rescheduleBilledContent(ctx, log, template, version, req)
	}

	updated, err := s.versionRepo.UpdateEffectiveDate(ctx, req)
	if err != nil {
		log.Error("failed to update version effective date", zap.Error(err))
		return nil, err
	}

	s.logAuditAction(
		ctx,
		log,
		template,
		permission.OpUpdate,
		req.TenantInfo.UserID,
		nil,
		"Formula template version effective date updated",
		auditservice.WithMetadata(map[string]any{
			versionNumberField:      req.VersionNumber,
			"previousEffectiveFrom": previousStart,
			"effectiveFrom":         *req.EffectiveFrom,
		}),
		auditservice.WithCritical(),
	)

	return updated, nil
}

func (s *Service) rescheduleBilledContent(
	ctx context.Context,
	log *zap.Logger,
	template *formulatemplate.FormulaTemplate,
	version *formulatemplate.FormulaTemplateVersion,
	req *formulatemplateports.UpdateEffectiveDateRequest,
) (*formulatemplate.FormulaTemplateVersion, error) {
	resolved := template.ApplyVersionFull(version)
	resolved.CurrentVersionNumber = template.CurrentVersionNumber + 1

	content, err := s.checkContent(ctx, &contentRequest{
		tenant: templateTenant(resolved),
		source: resolved.Source,
		mode:   tariff.LibrariesPinned,
		pins:   version.Imports,
		tables: version.Tables,
	})
	if err != nil {
		return nil, err
	}
	if refusal := content.refusal(); refusal != nil {
		return nil, refusal
	}
	if !content.testsPass() {
		return nil, errortypes.NewValidationError(
			versionNumberField,
			errortypes.ErrInvalid,
			"Version {0}'s tests no longer pass with the libraries it pinned, so it cannot bill again",
			version.VersionNumber,
		)
	}

	changeMessage := fmt.Sprintf("Version %d scheduled to bill again", version.VersionNumber)

	var scheduled *formulatemplate.FormulaTemplateVersion
	err = s.db.WithTx(ctx, kernelports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		updated, txErr := s.repo.Update(txCtx, resolved)
		if txErr != nil {
			return txErr
		}

		snapshot := formulatemplate.NewVersionFromTemplate(
			updated,
			updated.CurrentVersionNumber,
			req.TenantInfo.UserID,
			changeMessage,
			nil,
		)
		snapshot.Status = formulatemplate.StatusActive
		snapshot.EffectiveFrom = req.EffectiveFrom

		created, txErr := s.versionRepo.Create(txCtx, snapshot)
		if txErr != nil {
			return txErr
		}

		scheduled = created
		return nil
	})
	if err != nil {
		log.Error("failed to schedule billed content again", zap.Error(err))
		return nil, err
	}

	s.logAuditAction(
		ctx,
		log,
		resolved,
		permission.OpUpdate,
		req.TenantInfo.UserID,
		template,
		changeMessage,
		auditservice.WithMetadata(map[string]any{
			"sourceVersionNumber": version.VersionNumber,
			versionNumberField:    scheduled.VersionNumber,
			"effectiveFrom":       *req.EffectiveFrom,
		}),
		auditservice.WithCritical(),
	)

	return scheduled, nil
}

func (s *Service) ListScheduledVersions(
	ctx context.Context,
	req *formulatemplateports.ListScheduledVersionsRequest,
) ([]*formulatemplate.FormulaTemplateVersion, error) {
	return s.versionRepo.ListScheduled(ctx, req)
}
