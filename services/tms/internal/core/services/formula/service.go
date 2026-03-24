package formula

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/formula/engine"
	"github.com/emoss08/trenova/internal/core/services/formula/resolver"
	"github.com/emoss08/trenova/internal/core/services/formula/schema"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ServiceParams struct {
	fx.In

	Logger   *zap.Logger
	Registry *schema.Registry
	Engine   *engine.Engine
	Resolver *resolver.Resolver
	Repo     repositories.FormulaTemplateRepository
}

type Service struct {
	l        *zap.Logger
	registry *schema.Registry
	engine   *engine.Engine
	resolver *resolver.Resolver
	repo     repositories.FormulaTemplateRepository
}

func NewService(p ServiceParams) *Service {
	resolver.RegisterDefaultComputed(p.Resolver)

	return &Service{
		l:        p.Logger.Named("service.formula"),
		registry: p.Registry,
		engine:   p.Engine,
		resolver: p.Resolver,
		repo:     p.Repo,
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

	result, err := s.engine.Evaluate(&formulatemplatetypes.EvaluationRequest{
		Template:  template,
		Entity:    req.Entity,
		Variables: req.Variables,
	})
	if err != nil {
		log.Error("failed to evaluate formula", zap.Error(err))
		return nil, err
	}

	return &formulatemplatetypes.CalculateResponse{
		Amount:    result.Value,
		Variables: result.Variables,
	}, nil
}

type EvaluateExpressionRequest struct {
	Expression string
	Entity     any
	SchemaID   string
	Variables  map[string]any
}

func (s *Service) EvaluateExpression(
	req *EvaluateExpressionRequest,
) (*formulatemplatetypes.CalculateResponse, error) {
	log := s.l.With(
		zap.String("operation", "EvaluateExpression"),
		zap.String("schemaID", req.SchemaID),
	)

	result, err := s.engine.EvaluateExpression(
		req.Expression,
		req.Entity,
		req.SchemaID,
		req.Variables,
	)
	if err != nil {
		log.Error("failed to evaluate expression", zap.Error(err))
		return nil, err
	}

	return &formulatemplatetypes.CalculateResponse{
		Amount:    result.Value,
		Variables: result.Variables,
	}, nil
}

func (s *Service) ValidateExpression(expression, schemaID string) error {
	return s.engine.ValidateExpression(expression, schemaID)
}

func (s *Service) EvaluateWithEnv(
	expression string,
	env map[string]any,
) (*formulatemplatetypes.CalculateResponse, error) {
	result, err := s.engine.EvaluateWithEnv(expression, env)
	if err != nil {
		return nil, err
	}

	return &formulatemplatetypes.CalculateResponse{
		Amount:    result.Value,
		Variables: result.Variables,
	}, nil
}

func (s *Service) ValidateExpressionWithEnv(expression string, env map[string]any) error {
	return s.engine.ValidateExpressionWithEnv(expression, env)
}

func (s *Service) BuildValidationEnvironment(
	schemaID string,
	variables map[string]any,
) (map[string]any, error) {
	return s.engine.GetEnvironmentBuilder().BuildValidationEnvironment(schemaID, variables)
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
