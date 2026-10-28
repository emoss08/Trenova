package agentruntime

import (
	"context"
	"slices"
	"strings"
	"sync"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentturn"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentprofileports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/productguideports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/retrievalports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const deskPath = "/desk"

type ContextBuilderParams struct {
	fx.In

	Logger        *zap.Logger
	Organizations tenantports.OrganizationRepository
	Users         tenantports.UserRepository
	Runtime       agentruntimeports.AgentRuntime
	Memories      agentmemoryports.AgentMemoryService `optional:"true"`
	Guide         productguideports.ProductGuide      `optional:"true"`
	// Definitions and Permissions name the agents a conversation's agent may
	// hand work to. Without them it is offered none.
	Definitions agentdefinitionports.AgentDefinitionRepository `optional:"true"`
	Permissions permissionports.PermissionEngine               `optional:"true"`
	Vectorizer  retrievalports.QueryVectorizer                 `optional:"true"`
	// Profiles tells the turn what kind of business the organization is and
	// what the person works on. Without it the turn goes without.
	Profiles agentprofileports.Reader `optional:"true"`
}

type ContextBuilder struct {
	logger        *zap.Logger
	organizations tenantports.OrganizationRepository
	users         tenantports.UserRepository
	runtime       agentruntimeports.AgentRuntime
	memories      agentmemoryports.AgentMemoryService
	guide         productguideports.ProductGuide
	definitions   agentdefinitionports.AgentDefinitionRepository
	permissions   permissionports.PermissionEngine
	vectorizer    retrievalports.QueryVectorizer
	profiles      agentprofileports.Reader
}

func NewContextBuilder(p ContextBuilderParams) agentruntimeports.RuntimeContextBuilder {
	return &ContextBuilder{
		logger:        p.Logger.Named("service.agentruntime.context"),
		organizations: p.Organizations,
		users:         p.Users,
		runtime:       p.Runtime,
		memories:      p.Memories,
		guide:         p.Guide,
		definitions:   p.Definitions,
		permissions:   p.Permissions,
		vectorizer:    p.Vectorizer,
		profiles:      p.Profiles,
	}
}

func (b *ContextBuilder) Build(
	ctx context.Context,
	req *agentruntimeports.RuntimeContextRequest,
) (agentturn.RuntimeContext, error) {
	definition := req.Definition
	rc := agentturn.RuntimeContext{
		Now:              timeutils.NowUnix(),
		Trigger:          req.Trigger,
		Surface:          req.Surface,
		Subject:          req.Subject,
		Page:             req.Page,
		Attachments:      req.Attachments,
		Mentions:         req.Mentions,
		Facts:            req.Facts,
		DelegatorRecords: req.DelegatorRecords,
		Tools:            b.runtime.ToolSummaries(definition),
	}

	b.describeTrenova(&rc, req)
	tenant := req.Actor.TenantInfo()

	// The organization, the person, the agents this one may hand work to and
	// the memories read from different places, and none needs another's
	// answer; one after the other, the embedding the memories wait on waited
	// behind every read before it. Each runs on its own and writes only its
	// own result, which is put together below in the order it always was.
	var (
		wg        sync.WaitGroup
		delegates []agentturn.RuntimeDelegate
		org       organizationContext
		user      *userContext
		memories  *agentmemoryports.MemoryContext
		profiles  agentprofileports.PromptProfiles
	)
	delegator := strings.TrimSpace(req.DelegatedBy)
	if delegator == "" {
		wg.Go(func() { delegates = b.delegates(ctx, req) })
	}
	wg.Go(func() { org = b.describeOrganization(ctx, definition, tenant) })
	if definition.HasContextProvider(agentdefinition.ContextUser) {
		wg.Go(func() { user = b.describeUser(ctx, req.Actor.PersonUserID(), tenant) })
	}
	// Memory is read for every agent that asks for it, scoped to the
	// organization, to the agent's own tools, and to the records the turn is
	// about: a correction to dispatch_assign_move belongs in the prompt of an agent
	// that can assign, and what was recorded about Acme belongs in a turn
	// about one of Acme's shipments. The prompt keeps what its budget holds.
	if definition.HasContextProvider(agentdefinition.ContextMemory) && b.memories != nil {
		records := rc.MemoryRecords()
		toolNames := agentturn.ToolNamesOf(rc.Tools)
		wg.Go(func() { memories = b.readMemories(ctx, req, records, toolNames, tenant) })
	}
	if b.profiles != nil {
		wg.Go(
			func() { profiles = b.readProfiles(ctx, definition, req.Actor.PersonUserID(), tenant) },
		)
	}
	wg.Wait()

	if delegator != "" {
		rc.DelegatedBy = delegator
	} else {
		rc.Delegates = delegates
	}
	rc.Timezone = org.timezone
	rc.OrganizationName = org.name
	rc.BusinessUnitName = org.businessUnit
	if user != nil {
		rc.User = user.user
		if rc.Timezone == "" {
			rc.Timezone = user.timezone
		}
	}
	if memories != nil {
		rc.Memories = memories.Memories
		rc.MemorySubjects = memories.Subjects
		rc.MemoryRelevance = memories.Relevance
	}
	rc.OrganizationProfile = profiles.Organization
	rc.UserProfile = profiles.User

	return rc, nil
}

// readProfiles reads the organization's profile for an agent that is told
// about the organization, and the person's for one that is told about the
// person and has a person to be told about. A profile that cannot be read
// leaves the turn without it.
func (b *ContextBuilder) readProfiles(
	ctx context.Context,
	definition *agentdefinition.Definition,
	personID pulid.ID,
	tenant pagination.TenantInfo,
) agentprofileports.PromptProfiles {
	req := agentprofileports.PromptProfilesRequest{
		TenantInfo:   pagination.TenantInfo{OrgID: tenant.OrgID, BuID: tenant.BuID},
		Organization: definition.HasContextProvider(agentdefinition.ContextOrganization),
	}
	if definition.HasContextProvider(agentdefinition.ContextUser) {
		req.UserID = personID
	}
	if !req.Organization && req.UserID.IsNil() {
		return agentprofileports.PromptProfiles{}
	}

	profiles, err := b.profiles.ForPrompt(ctx, req)
	if err != nil {
		b.logger.Warn("agent context: a profile could not be read",
			zap.String("organization", tenant.OrgID.String()),
			zap.Error(err),
		)
	}

	return profiles
}

// organizationContext is what the turn is told about the organization.
type organizationContext struct {
	timezone     string
	name         string
	businessUnit string
}

// describeOrganization is read on every build, not only when the prompt
// wants its name. Its timezone is what every tool means by "today", and a
// tool draws the day boundary in UTC unless it is told otherwise — so an
// agent whose prompt providers happened not to include the organization got a
// clock a few hours off in every date filter it ran.
func (b *ContextBuilder) describeOrganization(
	ctx context.Context,
	definition *agentdefinition.Definition,
	tenant pagination.TenantInfo,
) organizationContext {
	describe := definition.HasContextProvider(agentdefinition.ContextOrganization) ||
		definition.HasContextProvider(agentdefinition.ContextClock)
	org, err := b.organizations.GetByID(ctx, tenantports.GetOrganizationByIDRequest{
		TenantInfo: tenant,
		IncludeBU:  describe,
	})
	if err != nil {
		b.logger.Warn("agent context: organization lookup failed",
			zap.String("organization", tenant.OrgID.String()),
			zap.Error(err),
		)

		return organizationContext{}
	}

	described := organizationContext{timezone: org.Timezone}
	if describe {
		described.name = org.Name
		if org.BusinessUnit != nil {
			described.businessUnit = org.BusinessUnit.Name
		}
	}

	return described
}

func (b *ContextBuilder) readMemories(
	ctx context.Context,
	req *agentruntimeports.RuntimeContextRequest,
	records []agent.EntityRef,
	toolNames []string,
	tenant pagination.TenantInfo,
) *agentmemoryports.MemoryContext {
	memories, err := b.memories.ForContext(ctx, agentmemoryports.MemoryContextRequest{
		TenantInfo:        tenant,
		AgentDefinitionID: req.Definition.ID,
		ReaderUserID:      req.Actor.PersonUserID(),
		ToolNames:         toolNames,
		Records:           records,
		Query:             b.memoryQuery(ctx, &req.Query),
		Text:              req.Query.Text,
		Focus:             req.Query.Focus,
	})
	if err != nil {
		b.logger.Warn("agent context: memory lookup failed",
			zap.String("organization", tenant.OrgID.String()),
			zap.Error(err),
		)

		return nil
	}

	return memories
}

// describeTrenova names the page the person is on from the product guide, and
// says whether the turn can answer questions about Trenova itself: only with
// a person in a conversation, the only turn that holds agent_find_page and
// agent_open_page.
func (b *ContextBuilder) describeTrenova(
	rc *agentturn.RuntimeContext,
	req *agentruntimeports.RuntimeContextRequest,
) {
	if b.guide == nil {
		return
	}

	rc.Guide = req.Trigger == agent.RunTriggerChat &&
		req.Actor != nil &&
		req.Actor.PrincipalType == permissionports.PrincipalTypeUser &&
		slices.Contains(req.Definition.EffectiveToolNames(), agentdefinition.CoreToolFindInTrenova)

	if req.Surface == agent.SurfaceDesk {
		rc.SurfaceGuide = b.runtimePage(deskPath)
	}

	if req.Page == nil {
		return
	}
	rc.PageGuide = b.runtimePage(req.Page.Path)
}

func (b *ContextBuilder) runtimePage(path string) *agentturn.RuntimePage {
	page, ok := b.guide.PageForPath(path)
	if !ok {
		return nil
	}

	return &agentturn.RuntimePage{
		Name:     page.Name,
		Location: page.Location(),
		Summary:  stringutils.FirstNonEmpty(page.Summary, page.Description),
	}
}

// delegates are the agents on the definition's allowlist that the person may
// hand work to through it: in the tenant, enabled, ones a person talks to,
// and only when the person may use the assistant at all. Each names the tools
// it holds that the person may use, so a delegate is never described by what
// the person could not have it do. A lookup that fails offers none; the turn
// still answers with its own tools.
func (b *ContextBuilder) delegates(
	ctx context.Context,
	req *agentruntimeports.RuntimeContextRequest,
) []agentturn.RuntimeDelegate {
	definition := req.Definition
	if req.Trigger != agent.RunTriggerChat || !definition.Delegates() ||
		b.definitions == nil || req.Actor == nil {
		return nil
	}
	usable := UsableAgentsFor(ctx, b.permissions, req.Actor, b.logger)
	if !usable.Assistant {
		return nil
	}

	found, err := b.definitions.ListByIDs(
		ctx,
		agentdefinitionports.ListAgentDefinitionsByIDsRequest{
			IDs:        definition.DelegateIDs,
			TenantInfo: req.Actor.TenantInfo(),
		},
	)
	if err != nil {
		b.logger.Warn("agent context: the agents it may hand work to could not be read",
			zap.String("agent", definition.ID.String()),
			zap.Error(err),
		)

		return nil
	}

	byID := make(map[pulid.ID]*agentdefinition.Definition, len(found))
	for _, delegate := range found {
		byID[delegate.ID] = delegate
	}

	delegates := make([]agentturn.RuntimeDelegate, 0, len(definition.DelegateIDs))
	for _, id := range definition.DelegateIDs {
		delegate, ok := byID[id]
		if !ok || definition.DelegateRefusal(delegate) != "" || !usable.Allows(delegate) {
			continue
		}
		named := agentturn.AsDelegate(delegate)
		named.Tools = b.delegateTools(ctx, req.Actor, delegate)
		delegates = append(delegates, named)
	}

	return delegates
}

// delegateTools names the lookups a delegate holds beyond the core ones every
// agent has, narrowed to those the person may use: a task handed to it only
// reads.
func (b *ContextBuilder) delegateTools(
	ctx context.Context,
	actor *permissionports.RequestActor,
	delegate *agentdefinition.Definition,
) []string {
	summaries := b.runtime.ToolSummaries(delegate)
	names := make([]string, 0, len(summaries))
	for _, summary := range summaries {
		if !agentdefinition.IsCoreTool(summary.Name) && summary.Lookup {
			names = append(names, summary.Name)
		}
	}

	return b.runtime.PermittedTools(ctx, actor, names)
}

func (b *ContextBuilder) memoryQuery(
	ctx context.Context,
	req *retrievalports.QueryVectorRequest,
) retrievalports.QueryVector {
	if b.vectorizer == nil || req == nil || strings.TrimSpace(req.Text) == "" ||
		req.TenantInfo.OrgID.IsNil() || req.TenantInfo.BuID.IsNil() {
		return retrievalports.QueryVector{}
	}

	query, err := b.vectorizer.Vectorize(ctx, req)
	if err != nil {
		b.logger.Warn("agent context: the turn could not be embedded; ranking memories by recency",
			zap.String("organization", req.TenantInfo.OrgID.String()),
			zap.Error(err),
		)

		return retrievalports.QueryVector{}
	}

	return query
}

// userContext is what the turn is told about the person, and their timezone
// for when the organization has none.
type userContext struct {
	user     *agentturn.RuntimeUser
	timezone string
}

func (b *ContextBuilder) describeUser(
	ctx context.Context,
	person pulid.ID,
	tenant pagination.TenantInfo,
) *userContext {
	if person.IsNil() {
		return nil
	}

	user, err := b.users.GetByID(ctx, tenantports.GetUserByIDRequest{
		TenantInfo:   tenant,
		LookupUserID: person,
	})
	if err != nil {
		b.logger.Warn("agent context: user lookup failed",
			zap.String("user", person.String()),
			zap.Error(err),
		)

		return nil
	}

	roles := make([]string, 0, len(user.Assignments))
	for _, assignment := range user.Assignments {
		if assignment != nil && assignment.Role != nil && assignment.Role.Name != "" {
			roles = append(roles, assignment.Role.Name)
		}
	}
	return &userContext{
		user: &agentturn.RuntimeUser{
			Name:     user.Name,
			Email:    user.EmailAddress,
			Roles:    roles,
			Timezone: user.Timezone,
		},
		timezone: user.Timezone,
	}
}
