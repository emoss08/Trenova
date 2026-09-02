package formulatemplateservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

// clearScheduledVersions drops any pending effective dates on a template whose
// approved content is no longer what it was. A schedule set against the old
// approval would otherwise fire, unreviewed, the moment the template is
// approved again.
func (s *Service) clearScheduledVersions(
	ctx context.Context,
	template *formulatemplate.FormulaTemplate,
) (int64, error) {
	return s.versionRepo.ClearScheduled(ctx, &repositories.ListScheduledVersionsRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: template.OrganizationID,
			BuID:  template.BusinessUnitID,
		},
		TemplateID: template.ID,
	})
}

func withClearedSchedules(comment string, cleared int64) string {
	switch cleared {
	case 0:
		return comment
	case 1:
		return comment + "; cleared 1 scheduled version"
	default:
		return fmt.Sprintf("%s; cleared %d scheduled versions", comment, cleared)
	}
}

func (s *Service) CreateVersion(
	ctx context.Context,
	req *repositories.CreateVersionRequest,
) (*formulatemplate.FormulaTemplateVersion, error) {
	log := s.l.With(
		zap.String("operation", "CreateVersion"),
		zap.String("templateID", req.TemplateID.String()),
	)

	template, err := s.getTemplateByIDWithTenant(ctx, req.TemplateID, req.TenantInfo)
	if err != nil {
		log.Error("failed to get template", zap.Error(err))
		return nil, err
	}

	if err = s.validateTemplate(ctx, template); err != nil {
		return nil, err
	}

	var changeSummary map[string]jsonutils.FieldChange
	if template.CurrentVersionNumber >= 1 {
		prevVersion, verr := s.versionRepo.GetByTemplateAndVersion(
			ctx,
			&repositories.GetVersionRequest{
				TenantInfo:    req.TenantInfo,
				TemplateID:    req.TemplateID,
				VersionNumber: template.CurrentVersionNumber,
			},
		)
		if verr == nil && prevVersion != nil {
			var diffErr error
			changeSummary, diffErr = jsonutils.JSONDiff(
				prevVersion,
				template,
				&jsonutils.DiffOptions{
					IgnoreFields: versionDiffIgnoreFields,
				},
			)
			if diffErr != nil {
				log.Warn("failed to compute change summary for version", zap.Error(diffErr))
			}
		}
	}

	previous := *template
	newVersionNumber := template.CurrentVersionNumber + 1
	template.CurrentVersionNumber = newVersionNumber

	var createdVersion *formulatemplate.FormulaTemplateVersion
	err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		if _, txErr := s.repo.Update(txCtx, template); txErr != nil {
			log.Error("failed to update template version number", zap.Error(txErr))
			return txErr
		}

		version := formulatemplate.NewVersionFromTemplate(
			template,
			newVersionNumber,
			req.TenantInfo.UserID,
			req.ChangeMessage,
			changeSummary,
		)

		created, txErr := s.versionRepo.Create(txCtx, version)
		if txErr != nil {
			log.Error("failed to create version", zap.Error(txErr))
			return txErr
		}

		createdVersion = created
		return nil
	})
	if err != nil {
		return nil, err
	}

	s.logAuditAction(
		log,
		template,
		permission.OpUpdate,
		req.TenantInfo.UserID,
		&previous,
		fmt.Sprintf("Version %d created", newVersionNumber),
	)

	return createdVersion, nil
}

func (s *Service) ListVersions(
	ctx context.Context,
	req *repositories.ListVersionsRequest,
) (*pagination.ListResult[*formulatemplate.FormulaTemplateVersion], error) {
	return s.versionRepo.List(ctx, req)
}

func (s *Service) GetVersion(
	ctx context.Context,
	req *repositories.GetVersionRequest,
) (*formulatemplate.FormulaTemplateVersion, error) {
	return s.versionRepo.GetByTemplateAndVersion(ctx, req)
}

func (s *Service) Rollback(
	ctx context.Context,
	req *repositories.RollbackRequest,
) (*formulatemplate.FormulaTemplate, error) {
	log := s.l.With(
		zap.String("operation", "Rollback"),
		zap.String("templateID", req.TemplateID.String()),
		zap.Int64("targetVersion", req.TargetVersion),
	)

	targetVersion, err := s.versionRepo.GetByTemplateAndVersion(
		ctx,
		&repositories.GetVersionRequest{
			TenantInfo:    req.TenantInfo,
			TemplateID:    req.TemplateID,
			VersionNumber: req.TargetVersion,
		},
	)
	if err != nil {
		log.Error("failed to get target version", zap.Error(err))
		return nil, err
	}

	currentTemplate, err := s.getTemplateByIDWithTenant(ctx, req.TemplateID, req.TenantInfo)
	if err != nil {
		log.Error("failed to get current template", zap.Error(err))
		return nil, err
	}

	resolved := currentTemplate.ApplyVersionFull(targetVersion)
	resolved.CurrentVersionNumber = currentTemplate.CurrentVersionNumber + 1

	// Old content can name a rate table that has since been removed; a
	// rollback must fail on that here, not at the next approval.
	if err = s.validateTemplate(ctx, resolved); err != nil {
		return nil, err
	}

	changeMessage := req.ChangeMessage
	if changeMessage == "" {
		changeMessage = fmt.Sprintf("Rolled back to version %d", req.TargetVersion)
	}

	auditComment := changeMessage
	if (currentTemplate.Status == formulatemplate.StatusActive ||
		currentTemplate.Status == formulatemplate.StatusInReview) &&
		resolved.HasMaterialChange(currentTemplate) {
		resolved.Status = formulatemplate.StatusDraft
		clearApprovalFields(resolved)
		auditComment = changeMessage + "; rollback reverted approval"
	}

	var updatedTemplate *formulatemplate.FormulaTemplate
	err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		updated, txErr := s.repo.Update(txCtx, resolved)
		if txErr != nil {
			log.Error("failed to update template", zap.Error(txErr))
			return txErr
		}

		if txErr = s.createVersionSnapshot(
			txCtx,
			updated,
			updated.CurrentVersionNumber,
			req.TenantInfo.UserID,
			changeMessage,
			nil,
		); txErr != nil {
			log.Error("failed to create version snapshot", zap.Error(txErr))
			return txErr
		}

		updatedTemplate = updated
		return nil
	})
	if err != nil {
		return nil, err
	}

	s.logAuditAction(
		log,
		updatedTemplate,
		permission.OpUpdate,
		req.TenantInfo.UserID,
		currentTemplate,
		auditComment,
	)

	return updatedTemplate, nil
}

func (s *Service) Fork(
	ctx context.Context,
	req *repositories.ForkTemplateRequest,
) (*formulatemplate.FormulaTemplate, error) {
	log := s.l.With(
		zap.String("operation", "Fork"),
		zap.String("sourceTemplateID", req.SourceTemplateID.String()),
	)

	sourceTemplate, err := s.getTemplateByIDWithTenant(ctx, req.SourceTemplateID, req.TenantInfo)
	if err != nil {
		log.Error("failed to get source template", zap.Error(err))
		return nil, err
	}

	snapshot, sourceVersionNum, err := s.resolveTemplateSnapshot(
		ctx,
		log,
		sourceTemplate,
		req.SourceVersion,
		req.TenantInfo,
	)
	if err != nil {
		log.Error("failed to resolve requested source version", zap.Error(err))
		return nil, err
	}

	forkedTemplate := &formulatemplate.FormulaTemplate{
		OrganizationID:       req.TenantInfo.OrgID,
		BusinessUnitID:       req.TenantInfo.BuID,
		Name:                 req.NewName,
		Description:          snapshot.Description,
		Type:                 snapshot.Type,
		Expression:           snapshot.Expression,
		Status:               formulatemplate.StatusDraft,
		SchemaID:             snapshot.SchemaID,
		VariableDefinitions:  snapshot.VariableDefinitions,
		BreakdownDefinitions: snapshot.BreakdownDefinitions,
		MinCharge:            snapshot.MinCharge,
		MaxCharge:            snapshot.MaxCharge,
		RoundingMode:         snapshot.RoundingMode,
		RoundingPrecision:    snapshot.RoundingPrecision,
		Metadata:             snapshot.Metadata,
		SourceTemplateID:     &req.SourceTemplateID,
		SourceVersionNumber:  &sourceVersionNum,
		CurrentVersionNumber: 1,
	}

	changeMessage := req.ChangeMessage
	if changeMessage == "" {
		changeMessage = fmt.Sprintf("Forked from template %s", sourceTemplate.Name)
	}

	var createdTemplate *formulatemplate.FormulaTemplate
	err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		created, txErr := s.repo.Create(txCtx, forkedTemplate)
		if txErr != nil {
			log.Error("failed to create forked template", zap.Error(txErr))
			return txErr
		}

		if txErr = s.createVersionSnapshot(
			txCtx, created, 1, req.TenantInfo.UserID, changeMessage, nil,
		); txErr != nil {
			log.Error("failed to create version snapshot", zap.Error(txErr))
			return txErr
		}

		createdTemplate = created
		return nil
	})
	if err != nil {
		return nil, err
	}

	s.logAuditAction(
		log,
		createdTemplate,
		permission.OpCreate,
		req.TenantInfo.UserID,
		nil,
		changeMessage,
	)

	return createdTemplate, nil
}

func (s *Service) CompareVersions(
	ctx context.Context,
	req *repositories.CompareVersionsRequest,
) (*formulatemplate.VersionDiff, error) {
	log := s.l.With(
		zap.String("operation", "CompareVersions"),
		zap.String("templateID", req.TemplateID.String()),
		zap.Int64("fromVersion", req.FromVersion),
		zap.Int64("toVersion", req.ToVersion),
	)

	versions, err := s.versionRepo.GetVersionRange(ctx, &repositories.GetVersionRangeRequest{
		TenantInfo:  req.TenantInfo,
		TemplateID:  req.TemplateID,
		FromVersion: req.FromVersion,
		ToVersion:   req.ToVersion,
	})
	if err != nil {
		log.Error("failed to get version range", zap.Error(err))
		return nil, err
	}

	if len(versions) != 2 {
		return nil, errortypes.NewValidationError(
			"versions",
			errortypes.ErrInvalid,
			"Both versions must exist for comparison",
		)
	}

	fromVer, toVer := extractVersionPair(versions, req.FromVersion, req.ToVersion)
	if fromVer == nil || toVer == nil {
		return nil, errortypes.NewValidationError(
			"versions",
			errortypes.ErrNotFound,
			"One or both versions not found in the retrieved range",
		)
	}

	changes, err := jsonutils.JSONDiff(fromVer, toVer, &jsonutils.DiffOptions{
		IgnoreFields: versionDiffIgnoreFields,
	})
	if err != nil {
		log.Error("failed to compute diff", zap.Error(err))
		return nil, err
	}

	return &formulatemplate.VersionDiff{
		FromVersion: req.FromVersion,
		ToVersion:   req.ToVersion,
		Changes:     changes,
		ChangeCount: len(changes),
	}, nil
}

func (s *Service) GetLineage(
	ctx context.Context,
	req *repositories.GetLineageRequest,
) (*formulatemplate.ForkLineage, error) {
	log := s.l.With(
		zap.String("operation", "GetLineage"),
		zap.String("templateID", req.TemplateID.String()),
	)

	template, err := s.getTemplateByIDWithTenant(ctx, req.TemplateID, req.TenantInfo)
	if err != nil {
		log.Error("failed to get template", zap.Error(err))
		return nil, err
	}

	forkedTemplates, err := s.versionRepo.GetForkedTemplates(
		ctx,
		&repositories.GetForkedTemplatesRequest{
			TenantInfo:       req.TenantInfo,
			SourceTemplateID: req.TemplateID,
		},
	)
	if err != nil {
		log.Error("failed to get forked templates", zap.Error(err))
		return nil, err
	}

	return buildLineage(template, forkedTemplates), nil
}

func (s *Service) UpdateVersionTags(
	ctx context.Context,
	req *repositories.UpdateVersionTagsRequest,
) (*formulatemplate.FormulaTemplateVersion, error) {
	log := s.l.With(
		zap.String("operation", "UpdateVersionTags"),
		zap.String("templateID", req.TemplateID.String()),
		zap.Int64("versionNumber", req.VersionNumber),
	)

	req.Tags = sliceutils.DedupeStrings(req.Tags)
	for _, tag := range req.Tags {
		if !formulatemplate.VersionTag(tag).IsValid() {
			return nil, errortypes.NewValidationError(
				"tags",
				errortypes.ErrInvalid,
				fmt.Sprintf("Invalid tag: %s", tag),
			)
		}
	}

	template, err := s.getTemplateByIDWithTenant(ctx, req.TemplateID, req.TenantInfo)
	if err != nil {
		log.Error("failed to get template", zap.Error(err))
		return nil, err
	}

	version, err := s.versionRepo.UpdateTags(ctx, req)
	if err != nil {
		log.Error("failed to update version tags", zap.Error(err))
		return nil, err
	}

	s.logAuditAction(
		log,
		template,
		permission.OpUpdate,
		req.TenantInfo.UserID,
		nil,
		fmt.Sprintf(
			"Version %d tags updated to [%s]",
			req.VersionNumber,
			strings.Join(req.Tags, ", "),
		),
	)

	return version, nil
}

func (s *Service) createVersionSnapshot(
	ctx context.Context,
	template *formulatemplate.FormulaTemplate,
	versionNumber int64,
	userID pulid.ID,
	changeMessage string,
	changeSummary map[string]jsonutils.FieldChange,
) error {
	version := formulatemplate.NewVersionFromTemplate(
		template,
		versionNumber,
		userID,
		changeMessage,
		changeSummary,
	)

	_, err := s.versionRepo.Create(ctx, version)
	return err
}

// resolveTemplateSnapshot picks the content a fork starts from. A version the
// caller asked for by number must exist: forking "v3" and quietly receiving
// the latest content instead would be a copy of the wrong thing with the
// right label on it. Only when no version was named does the latest snapshot,
// and failing that the row itself, stand in.
func (s *Service) resolveTemplateSnapshot(
	ctx context.Context,
	log *zap.Logger,
	template *formulatemplate.FormulaTemplate,
	requestedVersion *int64,
	tenant pagination.TenantInfo,
) (templateSnapshot, int64, error) {
	if requestedVersion != nil {
		version, err := s.versionRepo.GetByTemplateAndVersion(ctx, &repositories.GetVersionRequest{
			TenantInfo:    tenant,
			TemplateID:    template.ID,
			VersionNumber: *requestedVersion,
		})
		if err != nil {
			return templateSnapshot{}, 0, err
		}
		return snapshotFromVersion(version), version.VersionNumber, nil
	}

	version, err := s.versionRepo.GetLatestVersion(ctx, template.ID, tenant)
	if err == nil && version != nil {
		return snapshotFromVersion(version), version.VersionNumber, nil
	}
	if err != nil {
		log.Warn("no version snapshot found, forking the template row", zap.Error(err))
	}

	return snapshotFromTemplate(template), template.CurrentVersionNumber, nil
}
