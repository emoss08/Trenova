package formulatemplateservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/formulatemplateservice/standardcatalog"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type StandardTemplate struct {
	Name                string                             `json:"name"`
	Description         string                             `json:"description"`
	Type                formulatemplate.TemplateType       `json:"type"`
	Expression          string                             `json:"expression"`
	SchemaID            string                             `json:"schemaId"`
	VariableDefinitions []*formulatypes.VariableDefinition `json:"variableDefinitions"`
}

// ListStandards exposes the vendor-curated catalog without installing it, so
// the studio can offer every standard as a starting point for a new template.
func (s *Service) ListStandards() ([]StandardTemplate, error) {
	catalog, err := standardcatalog.Load()
	if err != nil {
		return nil, err
	}

	standards := make([]StandardTemplate, 0, len(catalog))
	for _, entry := range catalog {
		variables := entry.VariableDefinitions
		if variables == nil {
			variables = []*formulatypes.VariableDefinition{}
		}
		standards = append(standards, StandardTemplate{
			Name:                entry.Name,
			Description:         entry.Description,
			Type:                entry.Type,
			Expression:          entry.Expression,
			SchemaID:            entry.SchemaID,
			VariableDefinitions: variables,
		})
	}

	return standards, nil
}

type InstallStandardsResponse struct {
	Installed []*formulatemplate.FormulaTemplate `json:"installed"`
	Skipped   []string                           `json:"skipped"`
}

// InstallStandards creates the standard template library for a tenant. It is
// idempotent by name: templates the organization already has are skipped, so a
// re-run only fills gaps and never duplicates the set an organization's
// contracts already point at. The catalog is vendor-curated and validated at
// install time, so the templates land Active without a review cycle.
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

	existing, err := s.repo.FindByNames(ctx, repositories.GetFormulaTemplatesByNamesRequest{
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

		entity := &formulatemplate.FormulaTemplate{
			OrganizationID:       tenantInfo.OrgID,
			BusinessUnitID:       tenantInfo.BuID,
			Name:                 entry.Name,
			Description:          entry.Description,
			Type:                 entry.Type,
			Expression:           entry.Expression,
			Status:               formulatemplate.StatusActive,
			SchemaID:             entry.SchemaID,
			VariableDefinitions:  entry.VariableDefinitions,
			CurrentVersionNumber: 1,
		}

		if vErr := s.validateTemplate(ctx, entity); vErr != nil {
			log.Error("standard template failed validation",
				zap.String("template", entry.Name),
				zap.Error(vErr),
			)
			return nil, vErr
		}

		pending = append(pending, entity)
	}

	var installed []*formulatemplate.FormulaTemplate
	err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		installed = make([]*formulatemplate.FormulaTemplate, 0, len(pending))
		for _, entity := range pending {
			created, txErr := s.repo.Create(txCtx, entity)
			if txErr != nil {
				log.Error("failed to create standard template",
					zap.String("template", entity.Name),
					zap.Error(txErr),
				)
				return txErr
			}

			if txErr = s.createVersionSnapshot(
				txCtx,
				created,
				1,
				tenantInfo.UserID,
				"Standard template installed",
				nil,
			); txErr != nil {
				log.Error("failed to create version snapshot", zap.Error(txErr))
				return txErr
			}

			installed = append(installed, created)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	for _, entity := range installed {
		s.logAuditAction(
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
