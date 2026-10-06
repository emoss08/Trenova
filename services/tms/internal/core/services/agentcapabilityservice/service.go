// Package agentcapabilityservice serves an agent's capabilities page: what the
// agent can look up and change, how far each change may go on its own, who it
// hands work to and the limits it runs under.
//
// The page speaks in three modes where the agent definition speaks in tiers.
// Allowed is AutoExecute, Ask first is ActWithApproval (and Propose, which a
// person meets the same way), and Off is a tool the agent does not hold. A row
// is locked, with the reason, where a mode cannot be chosen: a tool whose own
// ceiling, or the agent's, keeps it from running on its own, a change that
// cannot be undone, or a read another of the agent's tools depends on.
package agentcapabilityservice

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolpolicy"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger      *zap.Logger
	Definitions repositories.AgentDefinitionRepository
	Agents      services.AgentDefinitionService
	Tools       services.AgentToolRegistry
	QueryTools  services.AgentQueryToolRegistry
	Permissions services.PermissionEngine
	Budgets     services.AgentBudgetService       `optional:"true"`
	Providers   repositories.AIProviderRepository `optional:"true"`
	Users       repositories.UserRepository       `optional:"true"`
}

type Service struct {
	l           *zap.Logger
	definitions repositories.AgentDefinitionRepository
	agents      services.AgentDefinitionService
	tools       services.AgentToolRegistry
	queryTools  services.AgentQueryToolRegistry
	permissions services.PermissionEngine
	budgets     services.AgentBudgetService
	providers   repositories.AIProviderRepository
	users       repositories.UserRepository
}

func New(p Params) services.AgentCapabilityService {
	return &Service{
		l:           p.Logger.Named("service.agent-capability"),
		definitions: p.Definitions,
		agents:      p.Agents,
		tools:       p.Tools,
		queryTools:  p.QueryTools,
		permissions: p.Permissions,
		budgets:     p.Budgets,
		providers:   p.Providers,
		users:       p.Users,
	}
}

func (s *Service) Get(
	ctx context.Context,
	req *services.GetAgentCapabilitiesRequest,
	actor *services.RequestActor,
) (*services.AgentCapabilities, error) {
	definition, err := s.readable(ctx, req.TenantInfo, req.AgentID, actor)
	if err != nil {
		return nil, err
	}

	return s.describe(ctx, definition, actor)
}

func (s *Service) Update(
	ctx context.Context,
	req *services.UpdateAgentCapabilitiesRequest,
	actor *services.RequestActor,
) (*services.AgentCapabilities, error) {
	definition, err := s.readable(ctx, req.TenantInfo, req.AgentID, actor)
	if err != nil {
		return nil, err
	}
	editable, err := s.mayEdit(ctx, actor)
	if err != nil {
		return nil, err
	}
	if !editable {
		return nil, errortypes.NewAuthorizationError(
			"You can see what {0} can do, but changing it needs permission to update agents.",
			definition.Name,
		)
	}

	// Each requested mode is checked against the page as the person saw it,
	// so a locked row cannot be unlocked by sending the request by hand.
	rows := s.rows(definition)
	for _, change := range req.Tools {
		if err = checkChange(rows, change); err != nil {
			return nil, err
		}
	}

	saved, err := s.agents.Patch(ctx, &services.PatchAgentDefinitionRequest{
		ID:         definition.ID,
		TenantInfo: req.TenantInfo,
		Version:    req.Version,
		Comment:    "Agent capabilities changed",
		Edit: func(d *agentdefinition.Definition) error {
			return applyUpdate(d, req)
		},
	}, actor)
	if err != nil {
		return nil, err
	}

	return s.describe(ctx, saved, actor)
}

// readable is the agent, refused unless the reader may use it. Who may use
// an agent may read what it can do; nothing here is more than a conversation
// with it would show them.
func (s *Service) readable(
	ctx context.Context,
	tenant pagination.TenantInfo,
	agentID pulid.ID,
	actor *services.RequestActor,
) (*agentdefinition.Definition, error) {
	definition, err := s.definitions.GetByID(ctx, repositories.GetAgentDefinitionByIDRequest{
		ID:         agentID,
		TenantInfo: tenant,
	})
	if err != nil {
		return nil, err
	}
	if actor == nil || s.permissions == nil {
		return nil, errWithheld(definition)
	}
	allowed, err := s.permissions.MayUseAgent(ctx, actor, definition)
	if err != nil {
		return nil, fmt.Errorf("check access to agent %s: %w", definition.ID, err)
	}
	if !allowed {
		return nil, errWithheld(definition)
	}

	return definition, nil
}

func errWithheld(definition *agentdefinition.Definition) error {
	return errortypes.NewAuthorizationError(
		"You do not have access to {0}. An administrator can give one of your roles access to it.",
		definition.Name,
	)
}

// mayEdit says whether the reader may change the agent: the same permission
// the agent's own settings in AI Control need, so the page never widens it.
func (s *Service) mayEdit(ctx context.Context, actor *services.RequestActor) (bool, error) {
	if actor == nil || s.permissions == nil {
		return false, nil
	}

	result, err := s.permissions.Check(ctx, &services.PermissionCheckRequest{
		PrincipalType:  actor.PrincipalType,
		PrincipalID:    actor.PrincipalID,
		UserID:         actor.UserID,
		APIKeyID:       actor.APIKeyID,
		BusinessUnitID: actor.BusinessUnitID,
		OrganizationID: actor.OrganizationID,
		Resource:       permission.ResourceAgentDefinition.String(),
		Operation:      permission.OpUpdate,
	})
	if err != nil {
		return false, fmt.Errorf("check permission to change agents: %w", err)
	}

	return result.Allowed, nil
}

func (s *Service) describe(
	ctx context.Context,
	definition *agentdefinition.Definition,
	actor *services.RequestActor,
) (*services.AgentCapabilities, error) {
	tenant := pagination.TenantInfo{
		OrgID: definition.OrganizationID,
		BuID:  definition.BusinessUnitID,
	}
	editable, err := s.mayEdit(ctx, actor)
	if err != nil {
		return nil, err
	}

	out := &services.AgentCapabilities{
		AgentID:     definition.ID,
		Name:        definition.Name,
		Description: definition.Description,
		Template:    string(definition.Template),
		Icon:        definition.Icon,
		Accent:      definition.Accent,
		SystemKey:   definition.SystemKey,
		Enabled:     definition.Enabled,
		CanEdit:     editable,
		Version:     definition.Version,
		ReadTools:   []services.AgentCapabilityTool{},
		WriteTools:  []services.AgentCapabilityTool{},
		Model:       s.modelName(ctx, definition, tenant),
		SetUpBy:     s.personName(ctx, definition.CreatedByID, tenant),
	}
	for _, row := range s.rows(definition) {
		if row.Write {
			out.WriteTools = append(out.WriteTools, row)
		} else {
			out.ReadTools = append(out.ReadTools, row)
		}
	}

	if out.Handoffs, err = s.handoffs(ctx, definition, tenant, actor); err != nil {
		return nil, err
	}
	if out.Limits, err = s.limits(ctx, definition); err != nil {
		return nil, err
	}

	return out, nil
}

// rows are the tools the page lists: the ones the agent holds, in the order
// they were chosen, then the ones switched off on the page. The tools every
// agent holds are not listed, since none of them can be switched off.
func (s *Service) rows(definition *agentdefinition.Definition) []services.AgentCapabilityTool {
	prerequisites := s.prerequisiteReads(definition)
	rows := make([]services.AgentCapabilityTool, 0,
		len(definition.ToolNames)+len(definition.DisabledToolNames))
	seen := make(map[string]struct{})
	add := func(name string, held bool) {
		if _, dup := seen[name]; dup || agentdefinition.IsCoreTool(name) {
			return
		}
		seen[name] = struct{}{}
		if row, ok := s.row(definition, name, held, prerequisites); ok {
			rows = append(rows, row)
		}
	}
	for _, name := range definition.ToolNames {
		add(name, true)
	}
	for _, name := range definition.DisabledToolNames {
		add(name, false)
	}

	return rows
}

func (s *Service) row(
	definition *agentdefinition.Definition,
	name string,
	held bool,
	prerequisites map[string]string,
) (services.AgentCapabilityTool, bool) {
	row := services.AgentCapabilityTool{Key: name, Label: Label(name)}

	if _, ok := s.queryTools.Get(name); ok {
		row.Mode = services.AgentCapabilityAllowed
		if !held {
			row.Mode = services.AgentCapabilityOff
		}
		row.AllowedModes = []services.AgentCapabilityMode{
			services.AgentCapabilityAllowed, services.AgentCapabilityOff,
		}
		if needer, needed := prerequisites[name]; needed && held {
			row.AllowedModes = []services.AgentCapabilityMode{services.AgentCapabilityAllowed}
			row.LockReason = "Always on · " + needer + " needs it"
		}

		return row, true
	}

	tool, ok := s.tools.Get(name)
	if !ok {
		// A tool that is no longer registered, or comes with an extension
		// that is off, has nothing to show.
		return row, false
	}
	policy := tool.Policy()
	row.Write = true
	row.Mode = ModeForTier(agenttoolpolicy.StaticTier(definition, policy))
	if !held {
		row.Mode = services.AgentCapabilityOff
	}
	row.AllowedModes, row.LockReason = WriteModes(definition, policy)

	return row, true
}

// WriteModes are the modes a write tool may be given on this agent, and why
// Allowed is not among them when it is not.
func WriteModes(
	definition *agentdefinition.Definition,
	policy services.ToolPolicy,
) ([]services.AgentCapabilityMode, string) {
	all := []services.AgentCapabilityMode{
		services.AgentCapabilityAllowed,
		services.AgentCapabilityAskFirst,
		services.AgentCapabilityOff,
	}
	asks := []services.AgentCapabilityMode{
		services.AgentCapabilityAskFirst,
		services.AgentCapabilityOff,
	}

	switch {
	case !policy.Reversible:
		return asks, "Always asks · this can't be undone"
	case policy.EgressCeiling() != agent.TierAutoExecute:
		return asks, "Always asks · it reaches outside the organization"
	case agenttoolpolicy.Promotable(policy) != agent.TierAutoExecute:
		return asks, "Always asks · this change always needs a person"
	case definition.AutonomyCeiling != agent.TierAutoExecute:
		return asks, "Always asks · this agent asks before every change"
	default:
		return all, ""
	}
}

// ModeForTier is the page's word for a tier.
func ModeForTier(tier agent.AutonomyTier) services.AgentCapabilityMode {
	if tier == agent.TierAutoExecute {
		return services.AgentCapabilityAllowed
	}

	return services.AgentCapabilityAskFirst
}

// prerequisiteReads are the reads a held write takes its arguments from,
// keyed by read, valued by the label of a write that needs it. Such a read
// is held with the write whatever the page says, so it cannot be turned off.
func (s *Service) prerequisiteReads(definition *agentdefinition.Definition) map[string]string {
	needed := make(map[string]string)
	for _, name := range definition.ToolNames {
		tool, ok := s.tools.Get(name)
		if !ok {
			continue
		}
		dependent, depends := tool.(services.PrerequisiteTool)
		if !depends {
			continue
		}
		for _, read := range dependent.Prerequisites() {
			if _, already := needed[read]; !already {
				needed[read] = Label(name)
			}
		}
	}

	return needed
}

func checkChange(rows []services.AgentCapabilityTool, change services.AgentCapabilityToolChange) error {
	if !change.Mode.IsValid() {
		return errortypes.NewValidationError(
			"tools", errortypes.ErrInvalid, "Mode must be Allowed, AskFirst or Off",
		)
	}
	idx := slices.IndexFunc(rows, func(row services.AgentCapabilityTool) bool {
		return row.Key == change.Key
	})
	if idx < 0 {
		return errortypes.NewValidationError(
			"tools", errortypes.ErrInvalid,
			fmt.Sprintf("%q is not one of this agent's tools", change.Key),
		)
	}
	row := rows[idx]
	if !slices.Contains(row.AllowedModes, change.Mode) {
		reason := row.LockReason
		if reason == "" {
			reason = "that mode is not offered for this tool"
		}

		return errortypes.NewBusinessError("{0} cannot be set that way: {1}", row.Label, reason)
	}

	return nil
}

// applyUpdate makes the page's changes on a copy of the stored agent.
func applyUpdate(d *agentdefinition.Definition, req *services.UpdateAgentCapabilitiesRequest) error {
	if req.Enabled != nil {
		d.Enabled = *req.Enabled
	}
	for _, change := range req.Tools {
		applyMode(d, change)
	}
	if req.DailyRequestLimit != nil {
		d.DailyRunLimit = *req.DailyRequestLimit
	}
	switch {
	case req.ClearMonthlyBudget:
		d.MonthlyBudgetUSD = nil
	case req.MonthlyBudgetUSD != nil:
		budget := *req.MonthlyBudgetUSD
		d.MonthlyBudgetUSD = &budget
	}
	if req.MaxChangeItems != nil {
		d.MaxChangeItems = *req.MaxChangeItems
	}
	if req.BusinessHoursOnly != nil {
		d.BusinessHoursOnly = *req.BusinessHoursOnly
	}
	if req.BusinessHoursStart != nil {
		d.BusinessHoursStart = *req.BusinessHoursStart
	}
	if req.BusinessHoursEnd != nil {
		d.BusinessHoursEnd = *req.BusinessHoursEnd
	}
	if req.BusinessHoursTimezone != nil {
		d.BusinessHoursTimezone = strings.TrimSpace(*req.BusinessHoursTimezone)
	}
	if len(req.DelegateTopics) > 0 {
		topics := make(map[string]string, len(d.DelegateTopics)+len(req.DelegateTopics))
		for id, topic := range d.DelegateTopics {
			topics[id] = topic
		}
		for id, topic := range req.DelegateTopics {
			if !d.MayDelegateTo(id) {
				return errortypes.NewValidationError(
					"delegateTopics", errortypes.ErrInvalid,
					"A topic can only name an agent this one hands work to",
				)
			}
			if strings.TrimSpace(topic) == "" {
				delete(topics, id.String())
				continue
			}
			topics[id.String()] = strings.TrimSpace(topic)
		}
		d.DelegateTopics = topics
	}

	return nil
}

func applyMode(d *agentdefinition.Definition, change services.AgentCapabilityToolChange) {
	switch change.Mode {
	case services.AgentCapabilityOff:
		d.TurnToolOff(change.Key)
		return
	case services.AgentCapabilityAllowed, services.AgentCapabilityAskFirst:
		d.TurnToolOn(change.Key)
	}

	tiers := make(map[string]agent.AutonomyTier, len(d.ToolTiers)+1)
	for name, tier := range d.ToolTiers {
		tiers[name] = tier
	}
	switch change.Mode {
	case services.AgentCapabilityAllowed:
		tiers[change.Key] = agent.TierAutoExecute
	case services.AgentCapabilityAskFirst:
		tiers[change.Key] = agent.TierActWithApproval
	}
	d.ToolTiers = tiers
}

// modelName is the model the agent answers with: its preferred provider while
// that one still serves conversations, otherwise the first in the
// organization's order, which is the one the router tries first.
func (s *Service) modelName(
	ctx context.Context,
	definition *agentdefinition.Definition,
	tenant pagination.TenantInfo,
) string {
	if s.providers == nil {
		return ""
	}
	providers, err := s.providers.ListForTask(ctx, repositories.ListAIProvidersForTaskRequest{
		Task:       aiprovider.TaskAssistantChat,
		TenantInfo: tenant,
	})
	if err != nil {
		s.l.Warn("could not read the agent's model", zap.Error(err))
		return ""
	}

	var first *aiprovider.Provider
	for _, provider := range providers {
		if ok, _ := provider.CanServeTask(aiprovider.TaskAssistantChat); !ok {
			continue
		}
		if provider.ID == definition.PreferredProviderID {
			return providerName(provider)
		}
		if first == nil {
			first = provider
		}
	}
	if first == nil {
		return ""
	}

	return providerName(first)
}

func providerName(provider *aiprovider.Provider) string {
	if name := strings.TrimSpace(provider.Name); name != "" {
		return name
	}

	return provider.Model
}

func (s *Service) personName(
	ctx context.Context,
	id *pulid.ID,
	tenant pagination.TenantInfo,
) string {
	if id == nil || id.IsNil() || s.users == nil {
		return ""
	}
	user, err := s.users.GetByID(ctx, repositories.GetUserByIDRequest{
		TenantInfo:   tenant,
		LookupUserID: *id,
	})
	if err != nil || user == nil {
		return ""
	}

	return user.Name
}

// handoffs are the agents this one hands work to that the reader may use
// themselves, in the order configured.
func (s *Service) handoffs(
	ctx context.Context,
	definition *agentdefinition.Definition,
	tenant pagination.TenantInfo,
	actor *services.RequestActor,
) ([]services.AgentCapabilityHandoff, error) {
	out := []services.AgentCapabilityHandoff{}
	if len(definition.DelegateIDs) == 0 {
		return out, nil
	}

	delegates, err := s.definitions.ListByIDs(ctx, repositories.ListAgentDefinitionsByIDsRequest{
		IDs:        definition.DelegateIDs,
		TenantInfo: tenant,
	})
	if err != nil {
		return nil, err
	}
	usable, err := s.permissions.AgentsUsable(ctx, actor, permission.OpRead)
	if err != nil {
		return nil, fmt.Errorf("check which agents the person may use: %w", err)
	}
	byID := make(map[pulid.ID]*agentdefinition.Definition, len(delegates))
	for _, delegate := range delegates {
		byID[delegate.ID] = delegate
	}

	for _, id := range definition.DelegateIDs {
		delegate, ok := byID[id]
		if !ok || !delegate.Enabled || !usable.Allows(delegate) {
			continue
		}
		topic := definition.DelegateTopic(id)
		if topic == "" {
			topic = delegate.Description
		}
		out = append(out, services.AgentCapabilityHandoff{
			AgentID:     delegate.ID,
			Name:        delegate.Name,
			Description: delegate.Description,
			Icon:        delegate.Icon,
			Accent:      delegate.Accent,
			Template:    string(delegate.Template),
			Topic:       topic,
		})
	}

	return out, nil
}

func (s *Service) limits(
	ctx context.Context,
	definition *agentdefinition.Definition,
) (services.AgentCapabilityLimits, error) {
	limits := services.AgentCapabilityLimits{
		DailyRequestLimit:  definition.DailyRunLimit,
		MaxChangeItems:     definition.ChangeLimit(),
		BusinessHoursOnly:  definition.BusinessHoursOnly,
		BusinessHoursStart: definition.BusinessHoursStart,
		BusinessHoursEnd:   definition.BusinessHoursEnd,
		MonthlySpentUSD:    "0.00",
	}
	if definition.MonthlyBudgetUSD != nil {
		budget := definition.MonthlyBudgetUSD.StringFixed(2)
		limits.MonthlyBudgetUSD = &budget
	}
	if s.budgets == nil {
		limits.BusinessHoursTimezone = definition.BusinessHoursZone("UTC")
		return limits, nil
	}

	limits.BusinessHoursTimezone = definition.BusinessHoursZone(s.budgets.Timezone(ctx, definition))
	if limits.BusinessHoursTimezone == "" {
		limits.BusinessHoursTimezone = "UTC"
	}
	status, err := s.budgets.Status(ctx, definition)
	if err != nil {
		return limits, err
	}
	limits.RequestsToday = status.RunsToday
	limits.MonthlySpentUSD = status.SpentUSD
	limits.MonthStart = status.MonthStart
	zone, zErr := time.LoadLocation(limits.BusinessHoursTimezone)
	if zErr != nil {
		zone = time.UTC
	}
	limits.MonthResetsAt = time.Unix(status.MonthStart, 0).In(zone).AddDate(0, 1, 0).Unix()
	limits.DayResetsAt = time.Unix(status.DayStart, 0).In(zone).AddDate(0, 0, 1).Unix()

	return limits, nil
}

// Label is a tool's name as a person reads it: post_invoices is "Post
// invoices".
func Label(name string) string {
	words := strings.FieldsFunc(name, func(r rune) bool {
		return r == '_' || r == '-' || r == '.'
	})
	if len(words) == 0 {
		return name
	}
	for idx, word := range words {
		word = strings.ToLower(word)
		if idx == 0 {
			word = strings.ToUpper(word[:1]) + word[1:]
		}
		words[idx] = word
	}

	return strings.Join(words, " ")
}
