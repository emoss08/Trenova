package formula

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/formula/engine"
	"github.com/emoss08/trenova/internal/core/services/formula/resolver"
	"github.com/emoss08/trenova/internal/core/services/formula/schema"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ServiceParams struct {
	fx.In

	Logger        *zap.Logger
	Registry      *schema.Registry
	Engine        *engine.Engine
	Resolver      *resolver.Resolver
	Repo          repositories.FormulaTemplateRepository
	VersionRepo   repositories.FormulaTemplateVersionRepository
	RateTableRepo repositories.RateTableRepository
}

type Service struct {
	l             *zap.Logger
	registry      *schema.Registry
	engine        *engine.Engine
	resolver      *resolver.Resolver
	repo          repositories.FormulaTemplateRepository
	versionRepo   repositories.FormulaTemplateVersionRepository
	rateTableRepo repositories.RateTableRepository
}

//nolint:gocritic // fx param structs are passed by value
func NewService(p ServiceParams) *Service {
	resolver.RegisterDefaultComputed(p.Resolver)

	return &Service{
		l:             p.Logger.Named("service.formula"),
		registry:      p.Registry,
		engine:        p.Engine,
		resolver:      p.Resolver,
		repo:          p.Repo,
		versionRepo:   p.VersionRepo,
		rateTableRepo: p.RateTableRepo,
	}
}

func (s *Service) Calculate(
	ctx context.Context,
	req *formulatemplatetypes.CalculateRequest,
) (*formulatemplatetypes.CalculateResponse, error) {
	log := s.l.With(
		zap.String("operation", "Calculate"),
		zap.String("templateID", req.TemplateID.String()),
	)

	template, err := s.repo.GetByID(ctx, repositories.GetFormulaTemplateByIDRequest{
		TemplateID: req.TemplateID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		log.Error("failed to get formula template", zap.Error(err))
		return nil, err
	}

	if template.Status != formulatemplate.StatusActive {
		return nil, errortypes.NewValidationError(
			"formulaTemplateId",
			errortypes.ErrInvalid,
			"Formula template must be Active to rate shipments",
		)
	}

	resolved, err := s.resolveEffectiveTemplate(ctx, template, req.TenantInfo, req.RatingDate)
	if err != nil {
		log.Error("failed to resolve effective template version", zap.Error(err))
		return nil, err
	}

	resp, err := s.Rate(ctx, &RateRequest{
		Template:  resolved,
		Entity:    req.Entity,
		Variables: req.Variables,
	})
	if err != nil {
		log.Error("failed to evaluate formula", zap.Error(err))
		return nil, err
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

func (s *Service) resolveEffectiveTemplate(
	ctx context.Context,
	template *formulatemplate.FormulaTemplate,
	tenantInfo pagination.TenantInfo,
	ratingDate int64,
) (*formulatemplate.FormulaTemplate, error) {
	asOf := ratingDate
	if asOf == 0 {
		asOf = timeutils.NowUnix()
	}

	version, err := s.versionRepo.GetEffectiveVersion(
		ctx,
		&repositories.GetEffectiveVersionRequest{
			TenantInfo: tenantInfo,
			TemplateID: template.ID,
			AsOf:       asOf,
		},
	)
	if err != nil {
		return nil, err
	}

	if version == nil {
		return template, nil
	}

	return template.ApplyVersion(version), nil
}

type RateRequest struct {
	Template  *formulatemplate.FormulaTemplate
	Entity    any
	Variables map[string]any
}

func (s *Service) Rate(
	ctx context.Context,
	req *RateRequest,
) (*formulatemplatetypes.CalculateResponse, error) {
	lookup, err := s.buildLookup(ctx, pagination.TenantInfo{
		OrgID: req.Template.OrganizationID,
		BuID:  req.Template.BusinessUnitID,
	})
	if err != nil {
		return nil, err
	}

	result, err := s.engine.Evaluate(ctx, &formulatemplatetypes.EvaluationRequest{
		Template:  req.Template,
		Entity:    req.Entity,
		Variables: req.Variables,
		Lookup:    lookup,
	})
	if err != nil {
		return nil, err
	}

	amount, guardrail := applyGuardrails(req.Template, result.Value)

	return &formulatemplatetypes.CalculateResponse{
		Amount:              amount,
		Variables:           result.Variables,
		FormulaTemplateID:   req.Template.ID.String(),
		FormulaTemplateName: req.Template.Name,
		Expression:          req.Template.Expression,
		Breakdown:           result.Breakdown,
		Guardrail:           guardrail,
		VersionNumber:       req.Template.CurrentVersionNumber,
	}, nil
}

func (s *Service) buildLookup(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (formulatemplatetypes.RateTableLookup, error) {
	tables, err := s.rateTableRepo.GetLookupData(ctx, &repositories.GetRateTableLookupDataRequest{
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	return NewRateTableLookup(tables), nil
}

func applyGuardrails(
	template *formulatemplate.FormulaTemplate,
	rawAmount decimal.Decimal,
) (decimal.Decimal, *formulatemplatetypes.GuardrailResult) {
	if !template.MinCharge.Valid && !template.MaxCharge.Valid {
		return rawAmount, nil
	}

	guardrail := &formulatemplatetypes.GuardrailResult{RawAmount: rawAmount}
	if template.MinCharge.Valid {
		minCharge := template.MinCharge.Decimal
		guardrail.MinCharge = &minCharge
	}
	if template.MaxCharge.Valid {
		maxCharge := template.MaxCharge.Decimal
		guardrail.MaxCharge = &maxCharge
	}

	amount := rawAmount
	switch {
	case template.MinCharge.Valid && rawAmount.LessThan(template.MinCharge.Decimal):
		amount = template.MinCharge.Decimal
		guardrail.Applied = true
		guardrail.Bound = "min"
	case template.MaxCharge.Valid && rawAmount.GreaterThan(template.MaxCharge.Decimal):
		amount = template.MaxCharge.Decimal
		guardrail.Applied = true
		guardrail.Bound = "max"
	}

	return amount, guardrail
}

type EvaluateExpressionRequest struct {
	Expression string
	Entity     any
	SchemaID   string
	Variables  map[string]any
	Breakdowns []*formulatypes.BreakdownDefinition
	TenantInfo pagination.TenantInfo
}

func (s *Service) EvaluateExpression(
	ctx context.Context,
	req *EvaluateExpressionRequest,
) (*formulatemplatetypes.CalculateResponse, error) {
	log := s.l.With(
		zap.String("operation", "EvaluateExpression"),
		zap.String("schemaID", req.SchemaID),
	)

	var lookup formulatemplatetypes.RateTableLookup
	if !req.TenantInfo.OrgID.IsNil() {
		builtLookup, err := s.buildLookup(ctx, req.TenantInfo)
		if err != nil {
			log.Error("failed to build rate table lookup", zap.Error(err))
			return nil, err
		}
		lookup = builtLookup
	}

	result, err := s.engine.EvaluateExpression(
		ctx,
		&formulatemplatetypes.ExpressionEvaluationRequest{
			Expression: req.Expression,
			Entity:     req.Entity,
			SchemaID:   req.SchemaID,
			Variables:  req.Variables,
			Breakdowns: req.Breakdowns,
			Lookup:     lookup,
		},
	)
	if err != nil {
		log.Error("failed to evaluate expression", zap.Error(err))
		return nil, err
	}

	return &formulatemplatetypes.CalculateResponse{
		Amount:    result.Value,
		Variables: result.Variables,
		Breakdown: result.Breakdown,
	}, nil
}

func (s *Service) ValidateExpression(ctx context.Context, expression, schemaID string) error {
	return s.engine.ValidateExpression(ctx, expression, schemaID)
}

func (s *Service) EvaluateWithEnv(
	ctx context.Context,
	expression string,
	env map[string]any,
) (*formulatemplatetypes.CalculateResponse, error) {
	result, err := s.engine.EvaluateWithEnv(ctx, expression, env)
	if err != nil {
		return nil, err
	}

	return &formulatemplatetypes.CalculateResponse{
		Amount:    result.Value,
		Variables: result.Variables,
	}, nil
}

func (s *Service) ValidateExpressionWithEnv(
	ctx context.Context,
	expression string,
	env map[string]any,
) error {
	return s.engine.ValidateExpressionWithEnv(ctx, expression, env)
}

func (s *Service) ValidateLookupTables(
	ctx context.Context,
	expression string,
	tenantInfo pagination.TenantInfo,
) error {
	tables, err := engine.ExtractLookupTables(expression)
	if err != nil || len(tables) == 0 {
		return nil //nolint:nilerr // unparseable expressions are rejected by compile validation
	}

	existing, err := s.rateTableRepo.GetByKeys(ctx, &repositories.GetRateTablesByKeysRequest{
		TenantInfo: tenantInfo,
		Keys:       tables,
	})
	if err != nil {
		return err
	}

	known := make(map[string]struct{}, len(existing))
	for _, table := range existing {
		known[table.Key] = struct{}{}
	}

	multiErr := errortypes.NewMultiError()
	for _, table := range tables {
		if _, ok := known[table]; !ok {
			multiErr.Add(
				"expression",
				errortypes.ErrInvalid,
				"Unknown rate table: "+table,
			)
		}
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (s *Service) BuildValidationEnvironment(
	schemaID string,
	variables map[string]any,
) (map[string]any, error) {
	env, _, err := s.engine.GetEnvironmentBuilder().
		BuildValidationEnvironment(schemaID, variables)
	return env, err
}

func (s *Service) GetAvailableVariables(schemaID string) []*formulatypes.FieldSource {
	return s.engine.GetEnvironmentBuilder().GetAvailableVariables(schemaID)
}

func (s *Service) GetRequiredPreloads(schemaID string) []string {
	return s.engine.GetEnvironmentBuilder().GetRequiredPreloads(schemaID)
}

func (s *Service) GetEngine() *engine.Engine {
	return s.engine
}

func (s *Service) GetResolver() *resolver.Resolver {
	return s.resolver
}
