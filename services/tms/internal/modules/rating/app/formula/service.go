package formula

import (
	"context"
	"fmt"
	"maps"
	"time"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula/contextvariablecache"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula/effectiveversioncache"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula/environment"
	formulaerrors "github.com/emoss08/trenova/internal/modules/rating/app/formula/errors"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula/resolver"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula/schema"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratematrixports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/ratetablecache"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const RatingTimeVariable = "ratingTime"

type ServiceParams struct {
	fx.In

	Logger         *zap.Logger
	Registry       *schema.Registry
	Environment    *environment.Builder
	Resolver       *resolver.Resolver
	Repo           formulatemplateports.FormulaTemplateRepository
	VersionRepo    formulatemplateports.FormulaTemplateVersionRepository
	RateMatrixRepo ratematrixports.RateMatrixRepository

	ContextProviders []formulatemplatetypes.ContextVariableProvider `group:"formula_context_variables"`
}

type Service struct {
	l                *zap.Logger
	registry         *schema.Registry
	env              *environment.Builder
	repo             formulatemplateports.FormulaTemplateRepository
	versionRepo      formulatemplateports.FormulaTemplateVersionRepository
	rateMatrixRepo   ratematrixports.RateMatrixRepository
	contextProviders []formulatemplatetypes.ContextVariableProvider
	schemas          map[string][]rules.FormulaVariable
}

//nolint:gocritic // fx param structs are passed by value
func NewService(p ServiceParams) *Service {
	resolver.RegisterDefaultComputed(p.Resolver)

	schemas := make(map[string][]rules.FormulaVariable, 1)
	for _, schemaID := range p.Registry.IDs() {
		if definition, ok := p.Registry.Get(schemaID); ok {
			schemas[schemaID] = schemaVariables(definition, schemaID)
		}
	}

	return &Service{
		l:                p.Logger.Named("service.formula"),
		registry:         p.Registry,
		env:              p.Environment,
		repo:             p.Repo,
		versionRepo:      p.VersionRepo,
		rateMatrixRepo:   p.RateMatrixRepo,
		contextProviders: p.ContextProviders,
		schemas:          schemas,
	}
}

func (s *Service) ContextVariables(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) map[string]any {
	if len(s.contextProviders) == 0 {
		return nil
	}

	key := tenantInfo.OrgID.String() + "/" + tenantInfo.BuID.String()
	return contextvariablecache.Get(ctx, key, func(ctx context.Context) map[string]any {
		variables := make(map[string]any, 4)
		for _, provider := range s.contextProviders {
			provided, err := provider.ContextVariables(ctx, tenantInfo)
			if err != nil {
				s.l.Warn("context variable provider failed",
					zap.String("provider", fmt.Sprintf("%T", provider)),
					zap.Error(err),
				)
				continue
			}
			maps.Copy(variables, provided)
		}
		return variables
	})
}

func (s *Service) providedValues(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	entity any,
	ratingDate int64,
) map[string]any {
	contextual := s.ContextVariables(ctx, tenantInfo)
	provided := make(map[string]any, len(contextual)+1)
	maps.Copy(provided, contextual)
	provided[RatingTimeVariable] = time.Unix(ratingAsOf(entity, ratingDate), 0).UTC()
	return provided
}

func ratingAsOf(entity any, ratingDate int64) int64 {
	if ratingDate > 0 {
		return ratingDate
	}
	if rated, ok := resolver.Unwrap(entity).(*shipment.Shipment); ok {
		return rated.RatingAsOf(timeutils.NowUnix())
	}
	return timeutils.NowUnix()
}

func (s *Service) Calculate(
	ctx context.Context,
	req *formulatemplatetypes.CalculateRequest,
) (*formulatemplatetypes.CalculateResponse, error) {
	log := s.l.With(
		zap.String("operation", "Calculate"),
		zap.String("templateID", req.TemplateID.String()),
	)

	template, err := s.repo.GetByID(ctx, formulatemplateports.GetFormulaTemplateByIDRequest{
		TemplateID: req.TemplateID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		log.Error("failed to get formula template", zap.Error(err))
		return nil, err
	}

	asOf := ratingAsOf(req.Entity, req.RatingDate)
	billing, err := s.ResolveBillingVersion(ctx, template, req.TenantInfo, asOf)
	if err != nil {
		log.Error("failed to resolve the billing version", zap.Error(err))
		return nil, err
	}
	resolved, err := billingContent(template, billing)
	if err != nil {
		return nil, err
	}

	resp, err := s.Rate(ctx, &RateRequest{
		Template:    resolved,
		Entity:      req.Entity,
		Variables:   req.Variables,
		Overrides:   req.Overrides,
		RatingDate:  asOf,
		Unavailable: req.UnavailableInputs,
	})
	if err != nil {
		log.Warn(
			"failed to evaluate formula",
			append(formulaerrors.LogFields(err), zap.Error(err))...)
		return nil, err
	}
	if billing.InForce != nil && resp.Receipt != nil {
		start := billing.InForce.EffectiveStart()
		resp.Receipt.EffectiveFrom = &start
	}

	if resp.Guardrail != nil && resp.Guardrail.Applied {
		log.Warn("formula guardrail applied",
			zap.String("bound", resp.Guardrail.Bound),
			zap.String("rawAmount", resp.Guardrail.RawAmount.String()),
			zap.String("clampedAmount", resp.Amount.String()),
		)
	}

	return resp, nil
}

type BillingResolution struct {
	InForce   *formulatemplate.FormulaTemplateVersion
	Scheduled *formulatemplate.FormulaTemplateVersion
}

func billingContent(
	template *formulatemplate.FormulaTemplate,
	billing BillingResolution,
) (*formulatemplate.FormulaTemplate, error) {
	switch {
	case template.Status == formulatemplate.StatusInactive:
		return nil, errortypes.NewValidationError(
			"formulaTemplateId",
			errortypes.ErrInvalid,
			"Formula template {0} is archived and no longer rates shipments",
			template.Name,
		)
	case billing.InForce != nil:
		return template.ApplyVersion(billing.InForce), nil
	case billing.Scheduled != nil:
		return nil, errortypes.NewValidationError(
			"formulaTemplateId",
			errortypes.ErrInvalid,
			"Formula template {0} is approved to bill from {1}, so it cannot rate shipments before then",
			template.Name,
			timeutils.FormatCalendarDate(billing.Scheduled.EffectiveStart(), time.UTC),
		)
	case template.Status == formulatemplate.StatusActive:
		return template, nil
	default:
		return nil, errortypes.NewValidationError(
			"formulaTemplateId",
			errortypes.ErrInvalid,
			"Formula template {0} has never been approved, so it cannot rate shipments yet",
			template.Name,
		)
	}
}

func (s *Service) ResolveBillingVersion(
	ctx context.Context,
	template *formulatemplate.FormulaTemplate,
	tenantInfo pagination.TenantInfo,
	asOf int64,
) (BillingResolution, error) {
	approved, err := effectiveversioncache.GetVersions(
		ctx,
		template.ID,
		func(loadCtx context.Context) ([]*formulatemplate.FormulaTemplateVersion, error) {
			return s.versionRepo.ListApproved(
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

	inForce, scheduled := effectiveversioncache.EffectiveAsOf(approved, asOf, timeutils.NowUnix())
	return BillingResolution{InForce: inForce, Scheduled: scheduled}, nil
}

func (s *Service) RateTables(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*formulatemplatetypes.RateTables, error) {
	return ratetablecache.GetStamped(
		ctx,
		tenantInfo.OrgID,
		tenantInfo.BuID,
		func(ctx context.Context) (string, error) {
			return s.rateMatrixRepo.GetLookupStamp(ctx, tenantInfo)
		},
		func(ctx context.Context, stamp string) (*formulatemplatetypes.RateTables, error) {
			data, err := s.rateMatrixRepo.GetLookupData(
				ctx,
				&ratematrixports.GetRateMatrixLookupDataRequest{TenantInfo: tenantInfo},
			)
			if err != nil {
				return nil, err
			}

			return NewRateTables(data, stamp)
		},
	)
}

func (s *Service) tablesFor(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	sources ...string,
) (*formulatemplatetypes.TableSelection, error) {
	if tenantInfo.OrgID.IsNil() {
		return nil, nil //nolint:nilnil // no tenant means no tables were loaded
	}
	if !formulatemplatetypes.ReadsTables(sources...) {
		empty := &formulatemplatetypes.RateTables{}
		return empty.Select(sources...), nil
	}

	tables, err := s.RateTables(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	return tables.Select(sources...), nil
}

func (s *Service) RequiredPreloads(schemaID string) []string {
	return s.env.RequiredPreloads(schemaID)
}

func (s *Service) FieldRoots(schemaID string) map[string]struct{} {
	return s.env.FieldRoots(schemaID)
}
