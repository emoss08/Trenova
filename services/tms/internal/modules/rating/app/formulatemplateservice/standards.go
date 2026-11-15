package formulatemplateservice

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplateservice/standardcatalog"
	"github.com/emoss08/trenova/internal/modules/rating/app/tariff"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/maypok86/otter/v2"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
)

const (
	standardsInstalledCacheSize = 10_000
	standardsInstalledTTL       = 6 * time.Hour
	standardNameSuffix          = " (Standard)"
	standardInstallComment      = "Standard template installed"
)

type standardsLedger struct {
	installed *otter.Cache[standardsTenant, struct{}]
	group     singleflight.Group
}

type standardsTenant struct {
	orgID pulid.ID
	buID  pulid.ID
}

func newStandardsLedger() *standardsLedger {
	return &standardsLedger{
		installed: otter.Must(&otter.Options[standardsTenant, struct{}]{
			MaximumSize:      standardsInstalledCacheSize,
			ExpiryCalculator: otter.ExpiryWriting[standardsTenant, struct{}](standardsInstalledTTL),
		}),
	}
}

func (s *Service) EnsureStandards(ctx context.Context, tenantInfo pagination.TenantInfo) error {
	if tenantInfo.OrgID.IsNil() || tenantInfo.BuID.IsNil() || tenantInfo.UserID.IsNil() {
		return nil
	}

	key := standardsTenant{orgID: tenantInfo.OrgID, buID: tenantInfo.BuID}
	if _, done := s.standards.installed.GetIfPresent(key); done {
		return nil
	}

	_, err, _ := s.standards.group.Do(
		key.orgID.String()+"/"+key.buID.String(),
		func() (any, error) {
			if installErr := s.installMissingStandards(ctx, tenantInfo); installErr != nil {
				return nil, installErr
			}
			s.standards.installed.Set(key, struct{}{})
			return nil, nil //nolint:nilnil // the group shares only the error
		},
	)

	return err
}

func (s *Service) installMissingStandards(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) error {
	log := s.l.With(
		zap.String("operation", "EnsureStandards"),
		zap.String("orgID", tenantInfo.OrgID.String()),
	)

	missing, err := s.missingStandards(ctx, tenantInfo)
	if err != nil || len(missing) == 0 {
		return err
	}

	pending, err := s.buildStandards(ctx, tenantInfo, missing)
	if err != nil {
		log.Error("failed to build the standard templates", zap.Error(err))
		return err
	}

	now := timeutils.NowUnix()
	installed := make([]*formulatemplate.FormulaTemplate, 0, len(pending))
	err = s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		for _, entity := range pending {
			created, txErr := s.repo.Create(txCtx, entity)
			if txErr != nil {
				return txErr
			}

			version := formulatemplate.NewVersionFromTemplate(
				created,
				1,
				tenantInfo.UserID,
				standardInstallComment,
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
		still, recheckErr := s.missingStandards(ctx, tenantInfo)
		if recheckErr == nil && len(still) == 0 {
			return nil
		}
		log.Error("failed to install the standard templates", zap.Error(err))
		return err
	}

	for _, entity := range installed {
		s.LogAuditAction(
			ctx,
			log,
			entity,
			permission.OpCreate,
			tenantInfo.UserID,
			nil,
			standardInstallComment,
		)
	}

	return nil
}

func (s *Service) missingStandards(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]standardcatalog.Template, error) {
	catalog, err := standardcatalog.Load()
	if err != nil {
		return nil, err
	}

	keys, err := s.repo.ListStandardKeys(ctx, formulatemplateports.ListStandardKeysRequest{
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	present := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		present[key] = struct{}{}
	}

	missing := make([]standardcatalog.Template, 0, len(catalog))
	for _, entry := range catalog {
		if _, ok := present[entry.Key]; !ok {
			missing = append(missing, entry)
		}
	}

	return missing, nil
}

func (s *Service) buildStandards(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	entries []standardcatalog.Template,
) ([]*formulatemplate.FormulaTemplate, error) {
	names, err := s.standardNames(ctx, tenantInfo, entries)
	if err != nil {
		return nil, err
	}

	built := make([]*formulatemplate.FormulaTemplate, 0, len(entries))
	for i := range entries {
		entity, buildErr := s.standardTemplate(ctx, tenantInfo, &entries[i], names[i])
		if buildErr != nil {
			return nil, buildErr
		}
		built = append(built, entity)
	}

	return built, nil
}

func (s *Service) standardNames(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	entries []standardcatalog.Template,
) ([]string, error) {
	catalogNames := make([]string, len(entries))
	for i := range entries {
		catalogNames[i] = entries[i].Name
	}

	existing, err := s.repo.FindByNames(ctx, formulatemplateports.GetFormulaTemplatesByNamesRequest{
		TenantInfo: tenantInfo,
		Names:      catalogNames,
	})
	if err != nil {
		return nil, err
	}
	if len(existing) == 0 {
		return catalogNames, nil
	}

	bases := make([]string, 0, len(existing))
	for _, template := range existing {
		bases = append(bases, template.Name+standardNameSuffix)
	}
	taken, err := s.repo.TakenNames(ctx, &formulatemplateports.ListTakenNamesRequest{
		TenantInfo: tenantInfo,
		Bases:      bases,
	})
	if err != nil {
		return nil, err
	}
	for _, template := range existing {
		taken[template.Name] = struct{}{}
	}

	names := make([]string, len(entries))
	for i, name := range catalogNames {
		if _, used := taken[name]; !used {
			names[i] = name
			continue
		}
		names[i] = nextAvailableName(taken, name+standardNameSuffix)
		taken[names[i]] = struct{}{}
	}

	return names, nil
}

func (s *Service) standardTemplate(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	entry *standardcatalog.Template,
	name string,
) (*formulatemplate.FormulaTemplate, error) {
	source := entry.Source
	if name != entry.Name {
		renamed, err := s.Renamed(ctx, source, name)
		if err != nil {
			return nil, err
		}
		source = renamed
	}

	entity := formulatemplate.Seed{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		Name:           name,
		Description:    entry.Description,
		Source:         source,
		Status:         formulatemplate.StatusActive,
		StandardKey:    entry.Key,
	}.Build()

	content, err := s.CheckContent(ctx, &ContentRequest{
		Tenant: tenantInfo,
		Source: entity.Source,
		Mode:   tariff.LibrariesLatest,
	})
	if err != nil {
		return nil, err
	}
	if refusal := content.Refusal(); refusal != nil {
		return nil, refusal
	}
	if !content.TestsPass() {
		return nil, fmt.Errorf(
			"the standard template %s does not pass its own tests on this server: %v",
			entry.Name,
			content.FailingTests(),
		)
	}

	ApplyContent(entity, content)
	approver := tenantInfo.UserID
	now := timeutils.NowUnix()
	entity.ApprovedByID = &approver
	entity.ApprovedAt = &now

	return entity, nil
}

func refuseStandardChanges(templates ...*formulatemplate.FormulaTemplate) error {
	for _, template := range templates {
		if err := template.RefuseStandardChange(); err != nil {
			return err
		}
	}

	return nil
}
