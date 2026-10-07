package agentdraftservice

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/aiusage"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	draftingTask        = aiprovider.TaskAssistantChat
	maxDescriptionRunes = 1000
	draftMaxTokens      = 4000
	tightenMaxTokens    = 8000
)

type Params struct {
	fx.In

	Logger        *zap.Logger
	Completion    serviceports.StructuredCompleter
	Definitions   serviceports.AgentDefinitionService
	Providers     repositories.AIProviderRepository
	Organizations repositories.OrganizationRepository
}

type Service struct {
	l             *zap.Logger
	completion    serviceports.StructuredCompleter
	definitions   serviceports.AgentDefinitionService
	providers     repositories.AIProviderRepository
	organizations repositories.OrganizationRepository
}

var _ serviceports.AgentDraftingService = (*Service)(nil)

func New(p Params) *Service {
	return &Service{
		l:             p.Logger.Named("service.agentdraft"),
		completion:    p.Completion,
		definitions:   p.Definitions,
		providers:     p.Providers,
		organizations: p.Organizations,
	}
}

func (s *Service) Available(ctx context.Context, tenantInfo pagination.TenantInfo) (bool, error) {
	providers, err := s.providers.ListForTask(ctx, repositories.ListAIProvidersForTaskRequest{
		Task:       draftingTask,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return false, err
	}

	return aiprovider.RouteFor(providers, draftingTask) != nil, nil
}

func (s *Service) DraftFromDescription(
	ctx context.Context,
	req *serviceports.DraftAgentRequest,
) (*serviceports.AgentDraft, error) {
	description, err := checkDescription(req.Description)
	if err != nil {
		return nil, err
	}

	catalog, err := s.definitions.ToolCatalog(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	timezone := s.timezone(ctx, req.TenantInfo)

	completion, err := s.completion.CompleteStructured(
		ctx,
		&serviceports.StructuredCompletionRequest{
			TenantInfo: req.TenantInfo,
			Task:       draftingTask,
			System:     draftSystemPrompt,
			Context: draftContext(&draftContextInput{
				catalog:     catalog,
				timezone:    timezone,
				description: description,
			}),
			OutputSchema: draftSchema(),
			SchemaName:   "agent_draft",
			MaxTokens:    draftMaxTokens,
			Attribution:  attribution(req.TenantInfo),
		},
	)
	if err != nil {
		return nil, err
	}

	reply, err := parseDraftReply(completion.Text)
	if err != nil {
		s.l.Warn("a drafted agent could not be read",
			zap.String("model", completion.ModelIdentifier),
			zap.Error(err),
		)

		return nil, errortypes.NewBusinessError(
			"The model did not return an agent. Try describing the job in a sentence or two.",
		).WithInternal(err)
	}

	request, notes, err := sanitizeDraft(&sanitizeInput{
		reply:       reply,
		catalog:     catalog,
		timezone:    timezone,
		description: description,
		tenantInfo:  req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	definition, err := s.definitions.Draft(ctx, request)
	if err != nil {
		s.l.Warn("a drafted agent failed the checks a save makes",
			zap.String("model", completion.ModelIdentifier),
			zap.Error(err),
		)

		return nil, errortypes.NewBusinessError(
			"The drafted agent did not pass the checks a save makes. Try describing the job differently.",
		).WithInternal(err)
	}

	return draftOf(definition, notes), nil
}

func (s *Service) TightenInstructions(
	ctx context.Context,
	req *serviceports.TightenAgentInstructionsRequest,
) (*serviceports.TightenedAgentInstructions, error) {
	instructions, err := checkInstructions(req.Instructions)
	if err != nil {
		return nil, err
	}

	completion, err := s.completion.CompleteStructured(
		ctx,
		&serviceports.StructuredCompletionRequest{
			TenantInfo:   req.TenantInfo,
			Task:         draftingTask,
			System:       tightenSystemPrompt,
			Context:      tightenContext(instructions),
			OutputSchema: tightenSchema(),
			SchemaName:   "agent_instructions",
			MaxTokens:    tightenMaxTokens,
			Attribution:  attribution(req.TenantInfo),
		},
	)
	if err != nil {
		return nil, err
	}

	text, err := parseTightenReply(completion.Text)
	if err != nil {
		s.l.Warn("tightened instructions could not be read",
			zap.String("model", completion.ModelIdentifier),
			zap.Error(err),
		)

		return nil, errortypes.NewBusinessError(
			"The model did not return instructions. Your instructions are unchanged.",
		).WithInternal(err)
	}

	return checkTightened(instructions, text)
}

func (s *Service) timezone(ctx context.Context, tenantInfo pagination.TenantInfo) string {
	if s.organizations == nil {
		return agentdefinition.DefaultCronTimezone
	}

	org, err := s.organizations.GetByID(ctx, repositories.GetOrganizationByIDRequest{
		TenantInfo: tenantInfo,
	})
	if err != nil {
		s.l.Warn("a drafted agent could not read the organization's zone",
			zap.String("organization", tenantInfo.OrgID.String()),
			zap.Error(err),
		)

		return agentdefinition.DefaultCronTimezone
	}
	if !agentdefinition.CronTimezoneValid(org.Timezone) {
		return agentdefinition.DefaultCronTimezone
	}

	return strings.TrimSpace(org.Timezone)
}

func attribution(tenantInfo pagination.TenantInfo) serviceports.AIUsageAttribution {
	return serviceports.AIUsageAttribution{
		UserID:  tenantInfo.UserID,
		Feature: aiusage.FeatureAgentDrafting,
	}
}

func checkDescription(text string) (string, error) {
	description := strings.TrimSpace(text)
	if description == "" {
		return "", errortypes.NewValidationError(
			"description", errortypes.ErrRequired, "Say what the agent should do.",
		)
	}
	if utf8.RuneCountInString(description) > maxDescriptionRunes {
		return "", errortypes.NewValidationError(
			"description", errortypes.ErrInvalid,
			"Keep the description under {0} characters.", maxDescriptionRunes,
		)
	}

	return description, nil
}

func checkInstructions(text string) (string, error) {
	instructions := strings.TrimSpace(text)
	if instructions == "" {
		return "", errortypes.NewValidationError(
			"instructions", errortypes.ErrRequired, "There are no instructions to tighten.",
		)
	}
	if utf8.RuneCountInString(instructions) > agentdefinition.MaxInstructionsRunes {
		return "", errortypes.NewValidationError(
			"instructions", errortypes.ErrInvalid,
			"Instructions cannot be longer than {0} characters.",
			agentdefinition.MaxInstructionsRunes,
		)
	}

	return instructions, nil
}

func draftOf(
	definition *agentdefinition.Definition,
	notes []serviceports.AgentDraftNote,
) *serviceports.AgentDraft {
	return &serviceports.AgentDraft{
		Name:                   definition.Name,
		Description:            definition.Description,
		Icon:                   definition.Icon,
		Accent:                 definition.Accent,
		Instructions:           definition.Instructions,
		Guardrails:             definition.Guardrails,
		TriggerMode:            definition.TriggerMode,
		CronExpression:         definition.CronExpression,
		CronTimezone:           definition.CronTimezone,
		EventKinds:             definition.EventKinds,
		IntervalSeconds:        definition.IntervalSeconds,
		ToolNames:              definition.ToolNames,
		ToolTiers:              definition.ToolTiers,
		AutonomyCeiling:        definition.AutonomyCeiling,
		DataAccessCeiling:      definition.DataAccessCeiling,
		OutputMode:             definition.OutputMode,
		Enabled:                definition.Enabled,
		ShadowMode:             definition.ShadowMode,
		DecisionTimeoutSeconds: definition.DecisionTimeoutSeconds,
		RunTimeoutSeconds:      definition.RunTimeoutSeconds,
		MaxToolCalls:           definition.MaxToolCalls,
		MaxConcurrentRuns:      definition.MaxConcurrentRuns,
		Notes:                  notes,
	}
}
