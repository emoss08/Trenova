package tariff

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/rating/app/tariff/effectiveversioncache"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulalibraryports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/tariffports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
)

const FieldFormulaTemplateID = "formulaTemplateId"

type templateReader interface {
	GetByID(
		ctx context.Context,
		req formulatemplateports.GetFormulaTemplateByIDRequest,
	) (*formulatemplate.FormulaTemplate, error)
}

type approvedVersions interface {
	ListApproved(
		ctx context.Context,
		req *formulatemplateports.ListScheduledVersionsRequest,
	) ([]*formulatemplate.FormulaTemplateVersion, error)
}

type TemplateResolverParams struct {
	fx.In

	Templates       formulatemplateports.FormulaTemplateRepository
	Versions        formulatemplateports.FormulaTemplateVersionRepository
	LibraryVersions formulalibraryports.FormulaLibraryVersionRepository
}

type TemplateResolver struct {
	templates templateReader
	versions  approvedVersions
	libraries libraryLoader
	now       func() int64
}

var (
	_ tariffports.Resolver        = (*TemplateResolver)(nil)
	_ tariffports.LibraryResolver = (*TemplateResolver)(nil)
)

func NewTemplateResolver(p TemplateResolverParams) *TemplateResolver {
	return &TemplateResolver{
		templates: p.Templates,
		versions:  p.Versions,
		libraries: libraryLoader{versions: p.LibraryVersions},
		now:       timeutils.NowUnix,
	}
}

type BillingResolution struct {
	InForce   *formulatemplate.FormulaTemplateVersion
	Scheduled *formulatemplate.FormulaTemplateVersion
}

func (r *TemplateResolver) BillingVersion(
	ctx context.Context,
	template *formulatemplate.FormulaTemplate,
	tenantInfo pagination.TenantInfo,
	asOf int64,
) (BillingResolution, error) {
	approved, err := effectiveversioncache.GetVersions(
		ctx,
		template.ID,
		func(loadCtx context.Context) ([]*formulatemplate.FormulaTemplateVersion, error) {
			return r.versions.ListApproved(
				loadCtx,
				&formulatemplateports.ListScheduledVersionsRequest{
					TenantInfo: tenantInfo,
					TemplateID: template.ID,
				},
			)
		},
	)
	if err != nil {
		return BillingResolution{}, err
	}

	inForce, scheduled := effectiveversioncache.EffectiveAsOf(approved, asOf, r.now())
	return BillingResolution{InForce: inForce, Scheduled: scheduled}, nil
}

func BillingContent(
	template *formulatemplate.FormulaTemplate,
	billing BillingResolution,
) (*formulatemplate.FormulaTemplate, error) {
	switch {
	case template.Status == formulatemplate.StatusInactive:
		return nil, errortypes.NewValidationError(
			FieldFormulaTemplateID,
			errortypes.ErrInvalid,
			"Formula template {0} is archived and no longer rates shipments",
			template.Name,
		)
	case billing.InForce != nil:
		return template.ApplyVersion(billing.InForce), nil
	case billing.Scheduled != nil:
		return nil, errortypes.NewValidationError(
			FieldFormulaTemplateID,
			errortypes.ErrInvalid,
			"Formula template {0} is approved to bill from {1}, so it cannot rate shipments before then",
			template.Name,
			timeutils.FormatCalendarDate(billing.Scheduled.EffectiveStart(), time.UTC),
		)
	case template.Status == formulatemplate.StatusActive:
		return template, nil
	default:
		return nil, errortypes.NewValidationError(
			FieldFormulaTemplateID,
			errortypes.ErrInvalid,
			"Formula template {0} has never been approved, so it cannot rate shipments yet",
			template.Name,
		)
	}
}

func (r *TemplateResolver) ResolveBilling(
	ctx context.Context,
	req *tariffports.ResolveRequest,
) (*tariffports.Template, error) {
	template, err := effectiveversioncache.GetTemplate(
		ctx,
		req.TemplateID,
		func(loadCtx context.Context) (*formulatemplate.FormulaTemplate, error) {
			return r.templates.GetByID(loadCtx, formulatemplateports.GetFormulaTemplateByIDRequest{
				TemplateID: req.TemplateID,
				TenantInfo: req.TenantInfo,
			})
		},
	)
	if err != nil {
		return nil, err
	}

	asOf := req.AsOf
	if asOf == 0 {
		asOf = r.now()
	}
	billing, err := r.BillingVersion(ctx, template, req.TenantInfo, asOf)
	if err != nil {
		return nil, err
	}

	content, err := BillingContent(template, billing)
	if err != nil {
		return nil, err
	}

	libraries, err := r.PinnedLibraries(ctx, req.TenantInfo, content.Imports)
	if err != nil {
		return nil, err
	}

	return &tariffports.Template{
		TemplateID:    content.ID,
		VersionNumber: content.CurrentVersionNumber,
		Name:          content.Name,
		Source:        content.Source,
		Inputs:        content.Inputs,
		Imports:       content.Imports,
		Tables:        content.Tables,
		Libraries:     libraries,
	}, nil
}

func (r *TemplateResolver) PinnedLibraries(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	imports []tarifftypes.Import,
) ([]rules.TariffLibrary, error) {
	set, err := r.libraries.pinned(ctx, tenantInfo, imports)
	if err != nil {
		return nil, err
	}
	return set.Context(), nil
}

func (r *TemplateResolver) LatestLibraries(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	sources ...string,
) (LibrarySet, error) {
	return r.libraries.latest(ctx, tenantInfo, sources...)
}

func (r *TemplateResolver) PinnedSet(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	imports []tarifftypes.Import,
) (LibrarySet, error) {
	return r.libraries.pinned(ctx, tenantInfo, imports)
}

func (r *TemplateResolver) ResolveLibraries(
	ctx context.Context,
	req *tariffports.LibrariesRequest,
) ([]rules.TariffLibrary, error) {
	set, err := r.libraries.latest(ctx, req.TenantInfo, req.Sources...)
	if err != nil {
		return nil, err
	}
	return set.Context(), nil
}
