package formulatemplateservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplateservice/standardcatalog"
	"github.com/emoss08/trenova/internal/modules/rating/app/tariff"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type StandardTemplate struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Source      string `json:"source"`
}

func (s *Service) ListStandards() ([]StandardTemplate, error) {
	catalog, err := standardcatalog.Load()
	if err != nil {
		return nil, err
	}

	standards := make([]StandardTemplate, 0, len(catalog))
	for _, entry := range catalog {
		standards = append(standards, StandardTemplate{
			Name:        entry.Name,
			Description: entry.Description,
			Source:      entry.Source,
		})
	}
	return standards, nil
}

type InstallStandardsResponse struct {
	Installed []*formulatemplate.FormulaTemplate `json:"installed"`
	Skipped   []string                           `json:"skipped"`
}

func (s *Service) InstallStandards(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*InstallStandardsResponse, error) {
	log := s.l.With(zap.String("operation", "InstallStandards"))

	catalog, err := standardcatalog.Load()
	if err != nil {
		log.Error("failed to load standard template catalog", zap.Error(err))
		return nil, err
	}

	names := make([]string, 0, len(catalog))
	for _, entry := range catalog {
		names = append(names, entry.Name)
	}

	existing, err := s.repo.FindByNames(ctx, formulatemplateports.GetFormulaTemplatesByNamesRequest{
		TenantInfo: tenantInfo,
		Names:      names,
	})
	if err != nil {
		log.Error("failed to check existing templates", zap.Error(err))
		return nil, err
	}

	existingNames := make(map[string]struct{}, len(existing))
	for _, template := range existing {
		existingNames[template.Name] = struct{}{}
	}

	skipped := make([]string, 0, len(existing))
	pending := make([]*formulatemplate.FormulaTemplate, 0, len(catalog))
	for _, entry := range catalog {
		if _, ok := existingNames[entry.Name]; ok {
			skipped = append(skipped, entry.Name)
			continue
		}

		entity, buildErr := s.standardTemplate(ctx, tenantInfo, &entry)
		if buildErr != nil {
			log.Error("standard template failed its checks",
				zap.String("template", entry.Name),
				zap.Error(buildErr),
			)
			return nil, buildErr
		}
		pending = append(pending, entity)
	}

	now := timeutils.NowUnix()
	var installed []*formulatemplate.FormulaTemplate
	err = s.db.WithTx(ctx, kernelports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		installed = make([]*formulatemplate.FormulaTemplate, 0, len(pending))
		for _, entity := range pending {
			created, txErr := s.repo.Create(txCtx, entity)
			if txErr != nil {
				return txErr
			}

			version := formulatemplate.NewVersionFromTemplate(
				created,
				1,
				tenantInfo.UserID,
				"Standard template installed",
				nil,
			)
			version.EffectiveFrom = &now
			if _, txErr = s.versionRepo.Create(txCtx, version); txErr != nil {
				return txErr
			}

			installed = append(installed, created)
		}
		return nil
	})
	if err != nil {
		log.Error("failed to install standard templates", zap.Error(err))
		return nil, err
	}

	for _, entity := range installed {
		s.logAuditAction(
			ctx,
			log,
			entity,
			permission.OpCreate,
			tenantInfo.UserID,
			nil,
			"Standard template installed",
		)
	}

	return &InstallStandardsResponse{Installed: installed, Skipped: skipped}, nil
}

func (s *Service) standardTemplate(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	entry *standardcatalog.Template,
) (*formulatemplate.FormulaTemplate, error) {
	entity := formulatemplate.Seed{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		Name:           entry.Name,
		Description:    entry.Description,
		Source:         entry.Source,
		Status:         formulatemplate.StatusActive,
	}.Build()

	content, err := s.checkContent(ctx, &contentRequest{
		tenant: tenantInfo,
		source: entity.Source,
		mode:   tariff.LibrariesLatest,
	})
	if err != nil {
		return nil, err
	}
	if refusal := content.refusal(); refusal != nil {
		return nil, refusal
	}
	if !content.testsPass() {
		return nil, errortypes.NewBusinessError(
			"The standard template {0} does not pass its own tests on this server",
			entry.Name,
		)
	}

	applyContent(entity, content)
	now := timeutils.NowUnix()
	if tenantInfo.UserID.IsNotNil() {
		approver := tenantInfo.UserID
		entity.ApprovedByID = &approver
	}
	entity.ApprovedAt = &now
	return entity, nil
}
