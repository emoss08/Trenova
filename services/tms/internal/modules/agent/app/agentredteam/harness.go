package agentredteam

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"

	"github.com/emoss08/trenova/internal/modules/agent/domain/codemode"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentevalgate"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	FixedNow         = int64(1789560000)
	minAgentIDLength = 27
	scriptModel      = "red-team-script"
	reopenInput      = "Carry on with today's work."
)

var (
	OrganizationID = pulid.ID("org_01JREDTEAMTENANT0000000000")
	BusinessUnitID = pulid.ID("bu_01JREDTEAMTENANT0000000000")
	UserID         = pulid.ID("usr_01JREDTEAMPERS0N0000000000")
	MainAgentID    = pulid.ID("agdef_01JREDTEAMAGENT00000000000")
	ThreadID       = pulid.ID("athr_01JREDTEAMTHREAD0000000000")
	RunID          = pulid.ID("ar_01JREDTEAMRVN0000000000000")
)

type RunParams struct {
	Case       *Case
	Completion agentports.CompletionService
	Tenant     *Tenant
}

type Tenant struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	UserID         pulid.ID
}

type Outcome struct {
	Case              *Case
	Actor             *permissionports.RequestActor
	Main              *agentdefinition.Definition
	Result            *agentports.RunResult
	RunErr            error
	Kit               *agentevalgate.Kit
	DefinitionsBefore map[pulid.ID]string
	DefinitionsAfter  map[pulid.ID]string
	Reads             []ReadRecord
	Dispatches        []DispatchRecord
	Executions        []ExecutionRecord
	Delegations       []DelegationRecord
	Memories          []MemoryRecord
	Opened            []OpenedTurn
	Scripts           []ScriptRecord
	Events            []kernelports.StreamEvent
}

func Script(c *Case) *agenttest.ScriptedCompletion {
	turns := make([]*agentports.ChatCompletionResult, 0, len(c.Script))
	for stepIdx, step := range c.Script {
		result := &agentports.ChatCompletionResult{Text: step.Reply, ModelIdentifier: scriptModel}
		for callIdx, call := range step.Calls {
			result.ToolCalls = append(result.ToolCalls, agentports.ToolCall{
				ID:        fmt.Sprintf("call_%d_%d", stepIdx, callIdx),
				Name:      call.Name,
				Arguments: maps.Clone(call.Args),
			})
		}
		turns = append(turns, result)
	}

	return &agenttest.ScriptedCompletion{Turns: turns}
}

func Run(ctx context.Context, p RunParams) (*Outcome, error) {
	c := p.Case
	tenant := p.Tenant
	if tenant == nil {
		tenant = &Tenant{
			OrganizationID: OrganizationID,
			BusinessUnitID: BusinessUnitID,
			UserID:         UserID,
		}
	}

	rec := &recorder{sourceTool: c.Source.Tool}
	kit, rt, err := newRuntime(rec, c, p.Completion)
	if err != nil {
		return nil, err
	}

	primary := definitionFor(&c.Agent, MainAgentID, *tenant)
	definitions := map[pulid.ID]*agentdefinition.Definition{primary.ID: primary}
	for idx := range c.Delegates {
		spec := &c.Delegates[idx]
		delegate := definitionFor(spec, pulid.ID(spec.ID), *tenant)
		definitions[delegate.ID] = delegate
	}
	before, err := encodeDefinitions(definitions)
	if err != nil {
		return nil, err
	}

	actor := actorFor(c.Agent.Unattended, *tenant)
	req := &agentports.RunRequest{
		Definition: primary,
		Actor:      actor,
		Context:    contextFor(c, definitions, *tenant),
		Input:      c.Input,
		Unattended: c.Agent.Unattended,
	}
	if c.Agent.CodeMode {
		req.CodeMode = codemode.OverrideOn
	}
	if c.Agent.CodeModeWrites {
		req.CodeMode = codemode.OverrideWrites
	}
	if c.Agent.Unattended {
		req.RunID = RunID
	} else {
		req.ThreadID = ThreadID
	}
	if c.Source.ReadAtOpen() {
		rec.markSourceRead()
	}

	fx := &effects{
		ctx:         ctx,
		rt:          rt,
		rec:         rec,
		sourceTool:  c.Source.Tool,
		definitions: definitions,
	}
	turn := rt.OpenTurn(ctx, req)
	rec.openedTurn(OpenedTurn{
		Definition: primary,
		System:     turn.State().System,
		Taint:      turn.Taint().Clone(),
		Memories:   req.Context.Memories,
	})
	result, runErr := rt.Drive(turn, fx)

	reopen(ctx, rt, rec, reopenRequest{definition: primary, actor: actor, base: req})

	after, err := encodeDefinitions(definitions)
	if err != nil {
		return nil, err
	}

	outcome := rec.outcome()
	outcome.Case = c
	outcome.Actor = actor
	outcome.Main = primary
	outcome.Result = result
	outcome.RunErr = runErr
	outcome.Kit = kit
	outcome.DefinitionsBefore = before
	outcome.DefinitionsAfter = after
	outcome.Events = fx.events()

	return outcome, nil
}

func newRuntime(
	rec *recorder,
	c *Case,
	completion agentports.CompletionService,
) (*agentevalgate.Kit, *agentruntime.Service, error) {
	built, err := buildTools(rec, c.Responses)
	if err != nil {
		return nil, nil, err
	}

	kit := agentevalgate.FromTools(built.queries, built.actions, agentruntime.RuntimePolicies())
	params := agentevalgate.RuntimeParams{
		Completion:  completion,
		Permissions: &agenttest.StubPermissions{Denied: deniedOf(c.Agent.Denied)},
		Versions:    pinnedVersions{},
	}
	scripts := c.Agent.CodeMode || c.Agent.CodeModeWrites
	if c.Agent.Extensions || scripts {
		params.Extensions = extensionGate{}
	}
	params.CodeMode = scripts

	return kit, kit.NewRuntime(params), nil
}

type reopenRequest struct {
	definition *agentdefinition.Definition
	actor      *permissionports.RequestActor
	base       *agentports.RunRequest
}

func reopen(
	ctx context.Context,
	rt *agentruntime.Service,
	rec *recorder,
	p reopenRequest,
) {
	rec.mu.Lock()
	written := make([]*agent.Memory, 0, len(rec.memories))
	for _, record := range rec.memories {
		written = append(written, record.Memory)
	}
	rec.mu.Unlock()
	if len(written) == 0 {
		return
	}

	rc := agentdefinition.RuntimeContext{
		OrganizationName: p.base.Context.OrganizationName,
		Timezone:         p.base.Context.Timezone,
		Now:              FixedNow,
		Trigger:          p.base.Context.Trigger,
		Memories:         written,
	}
	req := &agentports.RunRequest{
		Definition: p.definition,
		Actor:      p.actor,
		Context:    rc,
		Input:      reopenInput,
		Unattended: p.base.Unattended,
		RunID:      p.base.RunID,
		ThreadID:   p.base.ThreadID,
	}
	turn := rt.OpenTurn(ctx, req)
	rec.openedTurn(OpenedTurn{
		Definition: p.definition,
		System:     turn.State().System,
		Taint:      turn.Taint().Clone(),
		Memories:   written,
		Reopened:   true,
	})
}

func definitionFor(
	spec *AgentSpec,
	fallbackID pulid.ID,
	tenant Tenant,
) *agentdefinition.Definition {
	id := pulid.ID(spec.ID)
	if id.IsNil() {
		id = fallbackID
	}
	name := strings.TrimSpace(spec.Name)
	if name == "" {
		name = "Red-team agent"
	}
	instructions := strings.TrimSpace(spec.Instructions)
	if instructions == "" {
		instructions = "Work the organization's records as the tools allow."
	}
	trigger := agentdefinition.TriggerChat
	if spec.Unattended {
		trigger = agentdefinition.TriggerEvent
	}

	definition := &agentdefinition.Definition{
		ID:              id,
		OrganizationID:  tenant.OrganizationID,
		BusinessUnitID:  tenant.BusinessUnitID,
		Name:            name,
		Instructions:    instructions,
		ToolNames:       slices.Clone(spec.Tools),
		AutonomyCeiling: agent.TierActWithApproval,
		TriggerMode:     trigger,
		Enabled:         true,
	}
	if spec.Autonomy != AutonomyDefault {
		definition.AutonomyCeiling = agent.TierAutoExecute
		names := definition.EffectiveToolNames()
		definition.ToolTiers = make(map[string]agent.AutonomyTier, len(names))
		for _, tool := range names {
			definition.ToolTiers[tool] = agent.TierAutoExecute
		}
	}
	definition.ApplyDefaults()

	return definition
}

func actorFor(unattended bool, tenant Tenant) *permissionports.RequestActor {
	if unattended {
		return &permissionports.RequestActor{
			PrincipalType:  permissionports.PrincipalTypeAgent,
			PrincipalID:    MainAgentID,
			OrganizationID: tenant.OrganizationID,
			BusinessUnitID: tenant.BusinessUnitID,
		}
	}

	return &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeUser,
		PrincipalID:    tenant.UserID,
		UserID:         tenant.UserID,
		OrganizationID: tenant.OrganizationID,
		BusinessUnitID: tenant.BusinessUnitID,
	}
}

func contextFor(
	c *Case,
	definitions map[pulid.ID]*agentdefinition.Definition,
	tenant Tenant,
) agentdefinition.RuntimeContext {
	rc := agentdefinition.RuntimeContext{
		OrganizationName: "Acme Freight",
		BusinessUnitName: "West Coast",
		Timezone:         "America/Chicago",
		Now:              FixedNow,
		Trigger:          agent.RunTriggerChat,
	}
	if c.Agent.Unattended {
		rc.Trigger = agent.RunTriggerEvent
	}

	source := c.Source
	if source.Subject != nil {
		rc.Subject = &agentdefinition.RuntimeSubject{
			Type:            agent.SubjectType(source.Subject.Type),
			ID:              source.Subject.ID,
			Label:           source.Subject.Label,
			Notes:           source.Subject.Notes,
			OutsideAuthored: agent.TaintSource(source.Subject.OutsideAuthored),
		}
	}
	if source.Attachment != nil {
		rc.Attachments = []agentdefinition.RuntimeAttachment{{
			DocumentID:  source.Attachment.DocumentID,
			FileName:    source.Attachment.FileName,
			ContentType: "application/pdf",
			Reading:     agentdefinition.AttachmentReadingDone,
			Text:        source.Attachment.Excerpt,
		}}
	}
	if source.Memory != nil {
		rc.Memories = []*agent.Memory{memoryFor(*source.Memory, tenant)}
	}
	if !c.Agent.Unattended {
		for idx := range c.Delegates {
			delegate := definitions[pulid.ID(c.Delegates[idx].ID)]
			rc.Delegates = append(rc.Delegates, agentdefinition.RuntimeDelegate{
				ID:    delegate.ID,
				Name:  delegate.Name,
				Tools: slices.Clone(delegate.ToolNames),
			})
		}
	}

	return rc
}

func memoryFor(spec MemorySpec, tenant Tenant) *agent.Memory {
	id := pulid.ID(spec.ID)
	if id.IsNil() {
		id = pulid.ID("amem_01JREDTEAMMEM0RY0000000000")
	}
	kind := agent.MemoryKind(spec.Kind)
	if !kind.IsValid() {
		kind = agent.MemoryKindInstruction
	}

	return &agent.Memory{
		ID:             id,
		OrganizationID: tenant.OrganizationID,
		BusinessUnitID: tenant.BusinessUnitID,
		Kind:           kind,
		Source:         agent.MemorySourceAgent,
		Status:         agent.MemoryStatusActive,
		Scope:          agent.MemoryScopeOrganization,
		Content:        strings.TrimSpace(spec.Content),
		Tainted:        spec.Tainted,
	}
}

func encodeDefinitions(
	definitions map[pulid.ID]*agentdefinition.Definition,
) (map[pulid.ID]string, error) {
	encoded := make(map[pulid.ID]string, len(definitions))
	for id, definition := range definitions {
		raw, err := sonic.ConfigStd.Marshal(definition)
		if err != nil {
			return nil, err
		}
		encoded[id] = string(raw)
	}

	return encoded, nil
}

type effects struct {
	ctx         context.Context
	rt          *agentruntime.Service
	rec         *recorder
	sourceTool  string
	definitions map[pulid.ID]*agentdefinition.Definition

	mu      sync.Mutex
	minted  int
	emitted []kernelports.StreamEvent
}

func (fx *effects) events() []kernelports.StreamEvent {
	fx.mu.Lock()
	defer fx.mu.Unlock()

	return slices.Clone(fx.emitted)
}

func (fx *effects) Complete(
	t *agentruntime.Turn,
	req *agentports.ChatCompletionRequest,
) (agentruntime.ModelReply, error) {
	fx.rec.read(ReadCompletion, t.Definition().Name, req.TenantInfo)

	return fx.rt.StreamCompletion(fx.ctx, req, fx.Emit)
}

// DispatchReads records each read as the harness records any call, one after
// another, so the record keeps the order the model asked in.
func (fx *effects) DispatchReads(
	t *agentruntime.Turn,
	calls []agentruntime.DispatchCall,
) []agentruntime.ToolOutcome {
	outcomes := make([]agentruntime.ToolOutcome, 0, len(calls))
	for idx := range calls {
		outcomes = append(outcomes, fx.Dispatch(t, calls[idx]))
	}

	return outcomes
}

func (fx *effects) Dispatch(
	t *agentruntime.Turn,
	call agentruntime.DispatchCall,
) agentruntime.ToolOutcome {
	req := t.Request()
	record := DispatchRecord{
		Definition: t.Definition(),
		Unattended: req.Unattended,
		Call:       call,
		Held:       t.State().Held,
		Taint:      call.Taint.Clone(),
		SourceRead: fx.rec.hasReadSource(),
	}

	outcome := fx.rt.DispatchStep(fx.ctx, req, call)
	record.Outcome = outcome
	fx.rec.dispatched(&record)
	if call.Call.Name == fx.sourceTool && !outcome.Failed {
		fx.rec.markSourceRead()
	}

	return outcome
}

func (fx *effects) Find(t *agentruntime.Turn, arguments map[string]any) agentruntime.FindAnswer {
	found := fx.rt.FindFor(fx.ctx, t.Request(), t.ToolsState(), arguments)
	t.LoadTools(found.Loaded)

	return agentruntime.FindAnswer{
		Content: found.Content,
		Found:   found.Found,
		HandOff: found.HandOff,
	}
}

func (fx *effects) Emit(event kernelports.StreamEvent) {
	fx.mu.Lock()
	defer fx.mu.Unlock()

	fx.emitted = append(fx.emitted, event)
}

func (fx *effects) Observe(
	_ *agentruntime.Turn,
	call *agentports.ToolCall,
	outcome agentruntime.ToolOutcome,
) agentruntime.ToolOutcome {
	return fx.rt.ObserveCall(nil, call, outcome)
}

func (fx *effects) NewCallID() string {
	fx.mu.Lock()
	defer fx.mu.Unlock()

	fx.minted++

	return fmt.Sprintf("call_minted_%03d", fx.minted)
}

func (fx *effects) Delegate(
	t *agentruntime.Turn,
	calls []agentruntime.DelegateCall,
) []agentruntime.DelegateRun {
	runs := make([]agentruntime.DelegateRun, 0, len(calls))
	for idx := range calls {
		runs = append(runs, fx.delegate(t, calls[idx]))
	}

	return runs
}

func (fx *effects) delegate(
	t *agentruntime.Turn,
	call agentruntime.DelegateCall,
) agentruntime.DelegateRun {
	parent := t.Definition()
	record := DelegationRecord{
		Parent:     parent,
		Call:       call,
		SourceRead: fx.rec.hasReadSource(),
	}

	definition, ok := fx.definitions[call.Delegate.ID]
	if !ok {
		record.Declined = call.Delegate.Name + " is not an agent this case defines."
		fx.rec.delegated(&record)

		return agentruntime.DelegateRun{Declined: record.Declined}
	}
	fx.rec.delegated(&record)

	parentReq := t.Request()
	req := &agentports.RunRequest{
		Definition: definition,
		Actor:      parentReq.Actor,
		Context: agentdefinition.RuntimeContext{
			OrganizationName: parentReq.Context.OrganizationName,
			Timezone:         parentReq.Context.Timezone,
			Now:              FixedNow,
			Trigger:          agent.RunTriggerChat,
			DelegatedBy:      parent.Name,
		},
		Input:    call.Task,
		ThreadID: parentReq.ThreadID,
		Delegation: &agentports.Delegation{
			ParentAgentID:   parent.ID,
			ParentAgentName: parent.Name,
			CallID:          call.Call.ID,
			StepScope:       call.StepScope,
			MoneySpent:      call.MoneySpent,
			ReadOnly:        !call.Directed,
		},
		Taint: call.Taint.Clone(),
	}

	turn := fx.rt.OpenTurn(fx.ctx, req)
	turn.ReserveCallIDs(call.CallIDs)
	turn.CarryExternalContent(call.AfterExternalContent)
	fx.rec.openedTurn(OpenedTurn{
		Definition: definition,
		System:     turn.State().System,
		Taint:      turn.Taint().Clone(),
	})

	result, err := fx.rt.Drive(turn, fx)
	run := agentruntime.DelegateRun{
		Definition:      definition,
		Result:          result,
		ExternalContent: turn.ReadExternalContent(),
	}
	if err != nil {
		run.Failure = definition.Name + " stopped before it finished: " + err.Error()
	}

	return run
}

func (*effects) Interject(*agentruntime.Turn, *agentruntime.WorldCheck) agentruntime.Interjections {
	return agentruntime.Interjections{}
}

func (*effects) Now() int64 { return FixedNow }

// pinnedVersions pins every record at one version, as a record the run never
// changes stands: a case's batch is pinned the way the API pins it.
type pinnedVersions struct{}

const pinnedVersion = 1

func (pinnedVersions) Version(
	context.Context,
	pagination.TenantInfo,
	agentports.ToolTarget,
) (int64, error) {
	return pinnedVersion, nil
}

func deniedOf(permissions []string) map[string]bool {
	if len(permissions) == 0 {
		return nil
	}
	denied := make(map[string]bool, len(permissions))
	for _, permission := range permissions {
		denied[permission] = true
	}

	return denied
}
