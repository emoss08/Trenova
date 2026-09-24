package agentdefinitionservice

import (
	"context"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/emoss08/trenova/shared/typeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	Repo         repositories.AgentDefinitionRepository
	Tools        services.AgentToolRegistry
	QueryTools   services.AgentQueryToolRegistry
	Contexts     services.RuntimeContextBuilder
	AuditService services.AuditService
	// Schedules keeps the schedule behind a scheduled or continuous agent
	// in line with the agent as it is saved.
	Schedules services.AgentDefinitionScheduler
	// Extensions says which extensions the organization has on, so their
	// tools are offered only while they are.
	Extensions services.AgentExtensionGate `optional:"true"`
	// Access sets who may use an agent in the transaction that saves it.
	Access      services.AgentAccessService
	Permissions services.PermissionEngine `optional:"true"`
}

type Service struct {
	l           *zap.Logger
	repo        repositories.AgentDefinitionRepository
	tools       services.AgentToolRegistry
	queryTools  services.AgentQueryToolRegistry
	contexts    services.RuntimeContextBuilder
	audit       services.AuditService
	schedules   services.AgentDefinitionScheduler
	extensions  services.AgentExtensionGate
	access      services.AgentAccessService
	permissions services.PermissionEngine
}

func New(p Params) services.AgentDefinitionService {
	return &Service{
		l:           p.Logger.Named("service.agentdefinition"),
		repo:        p.Repo,
		tools:       p.Tools,
		queryTools:  p.QueryTools,
		contexts:    p.Contexts,
		audit:       p.AuditService,
		schedules:   p.Schedules,
		extensions:  p.Extensions,
		access:      p.Access,
		permissions: p.Permissions,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListAgentDefinitionRequest,
) (*pagination.ListResult[*agentdefinition.Definition], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) ListConnection(
	ctx context.Context,
	req *repositories.ListAgentDefinitionConnectionRequest,
) (*pagination.CursorListResult[*agentdefinition.Definition], error) {
	return s.repo.ListConnection(ctx, req)
}

func (s *Service) GetByID(
	ctx context.Context,
	req repositories.GetAgentDefinitionByIDRequest,
) (*agentdefinition.Definition, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) GetBySystemKey(
	ctx context.Context,
	req repositories.GetAgentDefinitionBySystemKeyRequest,
) (*agentdefinition.Definition, error) {
	return s.repo.GetBySystemKey(ctx, req)
}

func (s *Service) Create(
	ctx context.Context,
	req *services.SaveAgentDefinitionRequest,
	actor *services.RequestActor,
) (*agentdefinition.Definition, error) {
	definition := &agentdefinition.Definition{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
	}
	apply(definition, req)

	if err := s.validate(ctx, definition, nil); err != nil {
		return nil, err
	}
	if err := s.checkDataAccess(ctx, definition, nil, actor); err != nil {
		return nil, err
	}
	if err := s.schedule(definition); err != nil {
		return nil, err
	}

	created, err := s.save(ctx, req, actor, func(saveCtx context.Context) (
		*agentdefinition.Definition,
		error,
	) {
		return s.repo.Create(saveCtx, definition)
	})
	if err != nil {
		return nil, err
	}

	s.schedules.Sync(ctx, created)
	s.logAudit(created, nil, permission.OpCreate, actor, "Agent created")

	return created, nil
}

func (s *Service) Update(
	ctx context.Context,
	req *services.SaveAgentDefinitionRequest,
	actor *services.RequestActor,
) (*agentdefinition.Definition, error) {
	existing, err := s.repo.GetByID(ctx, repositories.GetAgentDefinitionByIDRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	previous := *existing
	updated := *existing
	updated.Version = req.Version
	apply(&updated, req)

	if err = s.validate(ctx, &updated, &previous); err != nil {
		return nil, err
	}
	if err = s.checkDataAccess(ctx, &updated, &previous, actor); err != nil {
		return nil, err
	}
	if scheduleChanged(&previous, &updated) {
		if err = s.schedule(&updated); err != nil {
			return nil, err
		}
	}

	saved, err := s.save(ctx, req, actor, func(saveCtx context.Context) (
		*agentdefinition.Definition,
		error,
	) {
		return s.repo.Update(saveCtx, &updated)
	})
	if err != nil {
		return nil, err
	}

	if scheduleChanged(&previous, saved) || !typeutils.EqualPtr(previous.EndsAt, saved.EndsAt) ||
		previous.Name != saved.Name {
		s.schedules.Sync(ctx, saved)
	}
	s.logAudit(saved, &previous, permission.OpUpdate, actor, "Agent updated")

	return saved, nil
}

// save writes the agent and, when the request says who may use it, sets
// that in the same transaction through the access service, which holds the
// rule for who may change it.
func (s *Service) save(
	ctx context.Context,
	req *services.SaveAgentDefinitionRequest,
	actor *services.RequestActor,
	write func(ctx context.Context) (*agentdefinition.Definition, error),
) (*agentdefinition.Definition, error) {
	if req.Access == nil {
		return write(ctx)
	}
	if s.access == nil {
		return nil, errortypes.NewBusinessError(
			"Who can use an agent cannot be set here: agent access is not available",
		)
	}

	return s.access.SaveWithAccess(ctx, &services.SaveAgentWithAccessRequest{
		TenantInfo: req.TenantInfo,
		Access:     *req.Access,
		Save:       write,
	}, actor)
}

func (s *Service) Delete(
	ctx context.Context,
	req repositories.DeleteAgentDefinitionRequest,
	actor *services.RequestActor,
) error {
	existing, err := s.repo.GetByID(ctx, repositories.GetAgentDefinitionByIDRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return err
	}

	if existing.IsSystem() {
		return errortypes.NewBusinessError(
			"{0} is a system agent. Disable it instead of deleting it.", existing.Name,
		)
	}

	if err = s.repo.Delete(ctx, req); err != nil {
		return err
	}

	s.schedules.Remove(ctx, existing.ID)
	s.logAudit(existing, existing, permission.OpDelete, actor, "Agent deleted")

	return nil
}

func (s *Service) Templates() []services.AgentTemplateDescriptor {
	templates := agentdefinition.AllTemplates()
	descriptors := make([]services.AgentTemplateDescriptor, 0, len(templates))

	for _, template := range templates {
		descriptors = append(descriptors, services.AgentTemplateDescriptor{
			Template:             template,
			Label:                template.Label(),
			Description:          template.Description(),
			StarterInstructions:  template.StarterInstructions(),
			StarterTools:         registeredStarterTools(template, s.tools, s.queryTools),
			StarterTrigger:       template.StarterTrigger(),
			StarterEvents:        template.StarterEvents(),
			StarterCron:          template.StarterCron(),
			StarterCeiling:       template.StarterCeiling(),
			StarterDataAccess:    template.StarterDataAccess(),
			StarterOutput:        starterOutput(template),
			StarterDailyRunLimit: template.StarterDailyRunLimit(),
			ContextProviders:     agentdefinition.AllContextProviders(),
		})
	}

	return descriptors
}

func starterOutput(template agentdefinition.Template) agentdefinition.OutputMode {
	if template.StarterTrigger() == agentdefinition.TriggerChat {
		return agentdefinition.OutputConversational
	}

	return agentdefinition.OutputReport
}

func (s *Service) ToolCatalog(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]services.ToolCatalogEntry, error) {
	extensions, err := s.activeExtensions(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	return buildToolCatalog(s.tools, s.queryTools, extensions), nil
}

func (s *Service) activeExtensions(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (extensionState, error) {
	if s.extensions == nil {
		return extensionState{}, nil
	}

	active, err := s.extensions.ActiveExtensions(ctx, tenantInfo)
	if err != nil {
		return nil, errortypes.NewBusinessError(
			"could not read which extensions are turned on",
		).WithInternal(err)
	}

	return active, nil
}

func (s *Service) EventKinds() []agent.EventDescriptor {
	return agent.KnownEvents()
}

func (s *Service) PreviewPrompt(
	ctx context.Context,
	req *services.PreviewPromptRequest,
) (string, error) {
	definition := &agentdefinition.Definition{
		OrganizationID: req.Definition.TenantInfo.OrgID,
		BusinessUnitID: req.Definition.TenantInfo.BuID,
	}
	apply(definition, req.Definition)

	runtimeContext := agentdefinition.RuntimeContext{
		Trigger: definition.TriggerMode.RunTrigger(),
	}
	if s.contexts != nil {
		built, err := s.contexts.Build(ctx, &services.RuntimeContextRequest{
			Definition: definition,
			Actor:      req.Actor,
			Trigger:    definition.TriggerMode.RunTrigger(),
		})
		if err != nil {
			return "", err
		}
		runtimeContext = built
	}

	return definition.BuildSystemPrompt(runtimeContext), nil
}

// validate checks the definition as it would be saved. previous is the
// definition as stored, nil for a new one.
func (s *Service) validate(
	ctx context.Context,
	definition, previous *agentdefinition.Definition,
) error {
	multiErr := errortypes.NewMultiError()
	definition.Validate(multiErr)
	extensions, err := s.activeExtensions(ctx, pagination.TenantInfo{
		OrgID: definition.OrganizationID,
		BuID:  definition.BusinessUnitID,
	})
	if err != nil {
		return err
	}
	validateToolSelection(toolSelection{
		definition: definition,
		previous:   previous,
		actions:    s.tools,
		queries:    s.queryTools,
		extensions: extensions,
	}, multiErr)
	if err := s.validateDelegates(ctx, definition, previous, multiErr); err != nil {
		return err
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (s *Service) schedule(definition *agentdefinition.Definition) error {
	switch definition.TriggerMode {
	case agentdefinition.TriggerScheduled, agentdefinition.TriggerContinuous:
		if !definition.Enabled {
			definition.NextRunAt = nil
			return nil
		}
		next, err := definition.ComputeNextRun(timeutils.NowUnix())
		if err != nil {
			return errortypes.NewBusinessError(
				"The schedule could not be computed: {0}",
				err.Error(),
			)
		}
		definition.NextRunAt = &next
	default:
		definition.NextRunAt = nil
	}

	return nil
}

func scheduleChanged(previous, updated *agentdefinition.Definition) bool {
	return previous.TriggerMode != updated.TriggerMode ||
		previous.CronExpression != updated.CronExpression ||
		previous.CronTimezone != updated.CronTimezone ||
		previous.IntervalSeconds != updated.IntervalSeconds ||
		previous.Enabled != updated.Enabled ||
		updated.NextRunAt == nil
}

func apply(definition *agentdefinition.Definition, req *services.SaveAgentDefinitionRequest) {
	definition.Name = strings.TrimSpace(req.Name)
	definition.Description = strings.TrimSpace(req.Description)
	definition.Template = req.Template
	definition.Instructions = strings.TrimSpace(req.Instructions)
	definition.Guardrails = trimAll(req.Guardrails)
	definition.ToolNames = agentdefinition.WithoutCoreTools(trimAll(req.ToolNames))
	definition.ToolTiers = copyTiers(req.ToolTiers)
	definition.AutonomyCeiling = req.AutonomyCeiling
	if req.DataAccessCeiling != "" {
		definition.DataAccessCeiling = req.DataAccessCeiling
	}
	definition.Enabled = req.Enabled
	definition.ShadowMode = req.ShadowMode
	definition.DecisionTimeoutSeconds = req.DecisionTimeoutSeconds
	definition.TriggerMode = req.TriggerMode
	definition.CronExpression = strings.TrimSpace(req.CronExpression)
	definition.CronTimezone = strings.TrimSpace(req.CronTimezone)
	definition.EventKinds = req.EventKinds
	definition.IntervalSeconds = req.IntervalSeconds
	definition.EndsAt = req.EndsAt
	definition.MaxConcurrentRuns = req.MaxConcurrentRuns
	definition.RunTimeoutSeconds = req.RunTimeoutSeconds
	definition.MaxToolCalls = req.MaxToolCalls
	definition.MonthlyBudgetUSD = req.MonthlyBudgetUSD
	definition.DailyRunLimit = req.DailyRunLimit
	definition.ToolDailyLimits = copyLimits(req.ToolDailyLimits)
	definition.SimulationMode = req.SimulationMode
	definition.MemoryTokenBudget = req.MemoryTokenBudget
	definition.Icon = strings.TrimSpace(req.Icon)
	definition.Accent = strings.TrimSpace(req.Accent)
	definition.ContextProviders = req.ContextProviders
	definition.OutputMode = req.OutputMode
	definition.PreferredProviderID = req.PreferredProviderID
	if req.DelegateIDs != nil {
		definition.DelegateIDs = slices.Clone(*req.DelegateIDs)
	}
	definition.ApplyDefaults()

	if definition.TriggerMode != agentdefinition.TriggerScheduled {
		definition.CronExpression = ""
	}
	if definition.TriggerMode != agentdefinition.TriggerEvent {
		definition.EventKinds = nil
	}
	if definition.TriggerMode != agentdefinition.TriggerContinuous {
		definition.IntervalSeconds = 0
	}
}

// copyLimits keeps only the caps that mean something, so a cleared field
// does not linger as a zero.
func copyLimits(limits map[string]int) map[string]int {
	out := make(map[string]int, len(limits))
	for tool, limit := range limits {
		tool = strings.TrimSpace(tool)
		if tool == "" || limit <= 0 {
			continue
		}
		out[tool] = limit
	}
	if len(out) == 0 {
		return nil
	}

	return out
}

func trimAll(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, strings.TrimSpace(value))
	}

	return out
}

func copyTiers(tiers map[string]agent.AutonomyTier) map[string]agent.AutonomyTier {
	if len(tiers) == 0 {
		return nil
	}

	out := make(map[string]agent.AutonomyTier, len(tiers))
	for tool, tier := range tiers {
		out[strings.TrimSpace(tool)] = tier
	}

	return out
}

func (s *Service) logAudit(
	definition *agentdefinition.Definition,
	previous *agentdefinition.Definition,
	operation permission.Operation,
	actor *services.RequestActor,
	comment string,
) {
	auditActor := actor.AuditActor()

	var previousState map[string]any
	if previous != nil {
		previousState = jsonutils.MustToJSON(previous)
	}

	if err := s.audit.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceAgentDefinition,
		ResourceID:     definition.GetID().String(),
		Operation:      operation,
		UserID:         auditActor.UserID,
		PrincipalType:  auditActor.PrincipalType,
		PrincipalID:    auditActor.PrincipalID,
		APIKeyID:       auditActor.APIKeyID,
		CurrentState:   jsonutils.MustToJSON(definition),
		PreviousState:  previousState,
		OrganizationID: definition.OrganizationID,
		BusinessUnitID: definition.BusinessUnitID,
	}, auditservice.WithComment(comment)); err != nil {
		s.l.Error("failed to log agent definition audit", zap.Error(err))
	}
}
