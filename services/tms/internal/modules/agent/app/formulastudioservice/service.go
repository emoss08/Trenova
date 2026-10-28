package formulastudioservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/formulastudioports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplateservice"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const studioTask = aiprovider.TaskAssistantChat

var errNoProvider = errors.New("no completion service is wired")

type formulaEngine interface {
	GetByID(
		ctx context.Context,
		req formulatemplateports.GetFormulaTemplateByIDRequest,
	) (*formulatemplate.FormulaTemplate, error)
	Preview(
		ctx context.Context,
		req *formulatemplateservice.PreviewRequest,
	) (*formulatemplateports.PreviewResult, error)
	TestSource(
		ctx context.Context,
		req *formulatemplateservice.TestSourceRequest,
	) (*formulatemplateservice.TestSourceResult, error)
	ReviewDiff(
		ctx context.Context,
		req *formulatemplateservice.ReviewDiffRequest,
	) (*formulatemplateservice.ReviewDiffResponse, error)
	Backtest(
		ctx context.Context,
		req *formulatemplateservice.BacktestRequest,
	) (*formulatemplateservice.BacktestResponse, error)
}

type agentAccess interface {
	MayUseAgent(
		ctx context.Context,
		actor *permissionports.RequestActor,
		definition *agentdefinition.Definition,
	) (bool, error)
}

type Params struct {
	fx.In

	Logger       *zap.Logger
	Templates    *formulatemplateservice.Service
	Permissions  permissionports.PermissionEngine
	SystemAgents agentdefinitionports.SystemAgentProvisioner `optional:"true"`
	Completion   llmports.StructuredCompleter                `optional:"true"`
}

type Service struct {
	l            *zap.Logger
	engine       formulaEngine
	access       agentAccess
	systemAgents agentdefinitionports.SystemAgentProvisioner
	completion   llmports.StructuredCompleter
}

var _ formulastudioports.FormulaStudioAssist = (*Service)(nil)

func New(p Params) *Service { //nolint:gocritic // fx param structs are passed by value
	return &Service{
		l:            p.Logger.Named("service.formulastudio"),
		engine:       p.Templates,
		access:       p.Permissions,
		systemAgents: p.SystemAgents,
		completion:   p.Completion,
	}
}

func (s *Service) assistant(
	ctx context.Context,
	tenant pagination.TenantInfo,
	actor *permissionports.RequestActor,
) (*agentdefinition.Definition, error) {
	if actor == nil || actor.UserID.IsNil() {
		return nil, errortypes.NewAuthorizationError(
			"You don't have permission to perform this action: {0} {1}",
			"assistant", "create",
		)
	}
	if s.systemAgents == nil {
		return nil, errortypes.NewBusinessError(
			"The formula assistant is not available on this server",
		)
	}

	definition, err := s.systemAgents.EnsureSystem(
		ctx,
		agentdefinitionports.EnsureSystemAgentRequest{
			TenantInfo: tenant,
			SystemKey:  agentdefinition.SystemKeyFormulaAssistant,
		},
	)
	if err != nil {
		return nil, err
	}

	if !definition.Enabled {
		return nil, errortypes.NewBusinessError(
			"{0} is turned off. An administrator can turn it on in AI Control.", definition.Name,
		)
	}

	if s.access == nil {
		return nil, agentWithheld(definition)
	}
	allowed, err := s.access.MayUseAgent(ctx, actor, definition)
	if err != nil {
		return nil, fmt.Errorf("check access to agent %s: %w", definition.ID, err)
	}
	if !allowed {
		return nil, agentWithheld(definition)
	}

	return definition, nil
}

func agentWithheld(definition *agentdefinition.Definition) error {
	return errortypes.NewAuthorizationError(
		"You do not have access to {0}. An administrator can give one of your roles access to it.",
		definition.Name,
	)
}

type completion struct {
	tenant     pagination.TenantInfo
	definition *agentdefinition.Definition
	feature    aiusage.Feature
	system     string
	sections   []llmports.ContextSection
	schema     map[string]any
	schemaName string
	maxTokens  int
}

func (s *Service) complete(ctx context.Context, call *completion, out any) error {
	if s.completion == nil {
		return errNoProvider
	}

	version := call.definition.Version
	result, err := s.completion.CompleteStructured(ctx, &llmports.StructuredCompletionRequest{
		TenantInfo:          call.tenant,
		Task:                studioTask,
		System:              call.system,
		Context:             llmports.DelimitedContext{Sections: call.sections},
		OutputSchema:        call.schema,
		SchemaName:          call.schemaName,
		MaxTokens:           call.maxTokens,
		PreferredProviderID: call.definition.PreferredProviderID,
		Attribution: aiusageports.AIUsageAttribution{
			UserID:            call.tenant.UserID,
			AgentDefinitionID: call.definition.ID,
			Feature:           call.feature,
			DefinitionVersion: &version,
		},
	})
	if err != nil {
		return err
	}

	if err = sonic.UnmarshalString(result.Text, out); err != nil {
		s.l.Warn("the formula assistant's answer could not be read",
			zap.String("schema", call.schemaName),
			zap.String("model", result.ModelIdentifier),
			zap.Error(err),
		)
		return fmt.Errorf("%w: %w", llmports.ErrModelSchemaValidation, err)
	}

	return nil
}

func modelFailure(definition *agentdefinition.Definition, err error) error {
	if errors.Is(err, errNoProvider) || errors.Is(err, llmports.ErrNoProviderConfigured) {
		return errortypes.NewBusinessError(
			"No AI provider is configured for the formula assistant",
		).WithInternal(err)
	}

	return errortypes.NewBusinessError(
		"{0} could not answer just now. Try again in a moment.", definition.Name,
	).WithInternal(err)
}

func noProvider(err error) bool {
	return errors.Is(err, errNoProvider) || errors.Is(err, llmports.ErrNoProviderConfigured)
}
