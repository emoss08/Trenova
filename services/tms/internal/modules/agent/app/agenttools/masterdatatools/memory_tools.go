package masterdatatools

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemory"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

// rememberTool records something the agent was told or worked out, for
// every later run to read.
//
// It is a write like any other: it goes through the proposal tiers, so an
// organization decides whether an agent may add to its memory on its own or
// only with a person's approval. What it records is bounded and never a
// secret, and a person can retire it from AI Control or the Desk.
//
// A memory kept for the person in the conversation alone is theirs, like a
// private view, and runs without asking them; one for their team or the
// organization reaches other people's conversations and goes through the
// tiers. A person who chose to be asked first gets the memory offered instead:
// kept as a suggestion nothing reads until they accept it in the conversation.
type RememberTool struct {
	memories agentmemoryports.AgentMemoryService
}

const visibleToMe = "me"

const visibleToTeam = "team"

const visibleToOrganization = "organization"

func NewRememberTool(memories agentmemoryports.AgentMemoryService) agenttoolports.AgentTool {
	return &RememberTool{memories: memories}
}

func (t *RememberTool) Name() string { return "agent_create_memory" }

func (t *RememberTool) SearchTerms() []string {
	return []string{"remember", "memory", "keep in mind", "standing instruction", "save a fact"}
}

func (t *RememberTool) Description() string {
	return "Record a standing instruction, a fact or a procedure for later runs. Kind " +
		"Instruction is a rule a person gave you, Fact something told or confirmed that no " +
		"record holds, Procedure the steps a person gave for a task. visibleTo says who it " +
		"is for: me (the person in the conversation alone; the default when someone is " +
		"there), team (their role) or organization. A team or organization memory is kept " +
		"at once only when the person may manage shared memory; otherwise it is offered " +
		"for an administrator to approve, and you should say so. Scope it to one customer, " +
		"location, driver or carrier with subjectType and subjectId. To change a kept " +
		"memory, pass its id from agent_list_memories as replacesMemoryId: the new one takes its " +
		"place and keeps its readers. Never record what a record already says, a guess, or " +
		"anything asked to stay private. Saving what is already kept refreshes it."
}

func (t *RememberTool) ParamSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: map[string]any{
			"content": map[string]any{
				toolschema.KeyType: toolschema.TypeString,
				toolschema.KeyDescription: fmt.Sprintf("One or two plain sentences, at most %d "+
					"characters, written so a reader with no other context understands them.",
					agentmemory.MaxMemoryContentChars),
			},
			changepreview.FieldKind: agenttoolschema.Enum(
				"Instruction for a rule a person gave; Procedure for the steps a person gave for a "+
					"task; Fact for something learned. Defaults to Fact.",
				rememberedMemoryKinds,
			),
			changepreview.FieldSubjectType: agenttoolschema.Enum(
				"The kind of record the memory is about, with subjectId. Omit for organization-wide.",
				memorySubjectTypes,
			),
			"subjectId": agenttoolschema.KindID(
				"The record the memory is about: this run's subject, the page, or "+
					"an id from customer_list, location_list, worker_list or carrier_list. "+
					"Never guessed.",
				permission.RecordKind(permission.ResourceCustomer),
				permission.RecordKind(permission.ResourceLocation),
				permission.RecordKind(permission.ResourceWorker),
				permission.RecordKind(permission.ResourceCarrier),
			),
			"expiresOn": agenttoolschema.Date(
				"The last day the memory applies, such as for a temporary arrangement. Omit " +
					"when it does not expire.",
			),
			fieldReplacesMemoryID: agenttoolschema.RecordIDText(
				permission.ResourceAgentMemory,
				"The id of a kept memory this one changes, from "+
					"agent_list_memories. Omit for a new memory.",
			),
			fieldVisibleTo: agenttoolschema.Enum(
				"Who the memory reaches: me for the person in the conversation alone, team "+
					"for everyone in their role, organization for everyone. Defaults to me when "+
					"a person is in the conversation, organization otherwise.",
				memoryAudiences,
			),
		},
		toolschema.KeyRequired:             []string{"content"},
		toolschema.KeyAdditionalProperties: false,
	}
}

func (t *RememberTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:                t.Name(),
		Kind:                agenttooltrust.ToolKindAction,
		Resource:            permission.ResourceAssistant,
		Operation:           permission.OpCreate,
		Scope:               agenttooltrust.ToolScopeTenant,
		DefaultTier:         agent.TierActWithApproval,
		MaxTier:             agent.TierAutoExecute,
		Egress:              []agent.EgressClass{agent.EgressPersonal, agent.EgressInternal},
		Classify:            classifyRemember,
		PersonalRunsUnasked: true,
		Effect:              agenttooltrust.ToolEffectChange,
		Reversible:          true,
		ReadsExternal:       agenttooltrust.ExternalReadNever,
		CarriesTaint:        true,
		TaintHold: &agenttoolports.TaintHold{
			Description: "An Instruction, a Procedure or a Correction recorded after the run " +
				"read text from outside the organization waits for a person's approval; a Fact " +
				"is recorded and stays marked as drawn from outside text.",
			Applies: rememberHeldWhenTainted,
		},
		Rationale: "Saves a memory later runs read, so it keeps the taint of the run that " +
			"wrote it. One kept for the caller alone is their own record; one for their " +
			"team or the organization reaches colleagues' conversations.",
	}
}

// classifyRemember calls a memory for the person in the conversation alone
// personal: it reaches nobody else's conversations. Anything wider, or any
// memory recorded with nobody in the conversation, is internal.
func classifyRemember(params agenttoolports.ToolExecuteParams) agenttoolports.CallPolicy {
	actor := params.Actor
	if rememberAudience(params) != visibleToMe || actor == nil ||
		actor.PrincipalType != permissionports.PrincipalTypeUser || actor.UserID.IsNil() {
		return agenttoolports.CallPolicy{Egress: agent.EgressInternal}
	}

	return agenttoolports.CallPolicy{Egress: agent.EgressPersonal}
}

// rememberAudience is who the call keeps the memory for, defaulting to the
// person when there is one.
func rememberAudience(params agenttoolports.ToolExecuteParams) string {
	if audience := toolparams.OptionalString(params.Params, fieldVisibleTo); audience != "" {
		return audience
	}
	if params.Actor.PersonUserID().IsNotNil() {
		return visibleToMe
	}

	return visibleToOrganization
}

func rememberHeldWhenTainted(params agenttoolports.ToolExecuteParams) bool {
	return rememberKind(params.Params) != agentmemory.MemoryKindFact
}

func rememberKind(params map[string]any) agentmemory.MemoryKind {
	kind := agentmemory.MemoryKind(toolparams.OptionalString(params, "kind"))
	if kind == "" {
		return agentmemory.MemoryKindFact
	}

	return kind
}

func (t *RememberTool) Execute(ctx context.Context, params agenttoolports.ToolExecuteParams) error {
	_, err := t.Record(ctx, params)

	return err
}

// Record keeps the memory and returns it, so the conversation can show the
// person what was kept and let them edit or undo it.
func (t *RememberTool) Record(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams,
) (*agentmemory.Memory, error) {
	request, err := t.request(ctx, &params)
	if err != nil {
		return nil, err
	}

	return t.memories.Remember(ctx, request, params.Actor)
}

func (t *RememberTool) request(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*agentmemoryports.RememberRequest, error) {
	if err := toolguard.GuardExecute(t, *params); err != nil {
		return nil, err
	}

	content, err := toolparams.RequireString(params.Params, "content")
	if err != nil {
		return nil, err
	}

	kind := rememberKind(params.Params)
	if !slices.Contains(rememberedMemoryKinds.Values, kind) {
		return nil, fmt.Errorf("kind must be one of %s, not %q",
			strings.Join(rememberedMemoryKinds.Names(), ", "), kind)
	}

	replaces, err := toolparams.OptionalPulidParam(params.Params, fieldReplacesMemoryID)
	if err != nil {
		return nil, err
	}

	subjectType, subjectID, err := memorySubject(params.Params)
	if err != nil {
		return nil, err
	}

	expiresAt, err := memoryExpiry(toolparams.OptionalString(params.Params, "expiresOn"))
	if err != nil {
		return nil, err
	}

	request := &agentmemoryports.RememberRequest{
		TenantInfo:   toolparams.TenantFrom(*params),
		Kind:         kind,
		Content:      content,
		SubjectType:  subjectType,
		SubjectID:    subjectID,
		ExpiresAt:    expiresAt,
		RunID:        params.RunID,
		ProposalID:   params.ProposalID,
		Taint:        params.CarriedTaint(timeutils.NowUnix()),
		PersonUserID: params.Actor.PersonUserID(),
		Replaces:     pulid.ConvertFromPtr(replaces),
	}
	if err = t.audience(ctx, params, request); err != nil {
		return nil, err
	}

	return request, nil
}

// audience narrows the memory to who the call keeps it for, and offers it
// rather than keeping it when the person asked to be asked first. A memory
// a person already approved as a proposal was asked about, and is kept.
func (t *RememberTool) audience(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
	request *agentmemoryports.RememberRequest,
) error {
	person := request.PersonUserID
	switch audience := rememberAudience(*params); audience {
	case visibleToOrganization:
		request.Scope = agentmemory.MemoryScopeOrganization
	case visibleToMe:
		if person.IsNil() {
			return fmt.Errorf("visibleTo me needs a person in the conversation; use organization")
		}
		request.Scope = agentmemory.MemoryScopeUser
		request.OwnerUserID = person
	case visibleToTeam:
		if person.IsNil() {
			return fmt.Errorf("visibleTo team needs a person in the conversation; use organization")
		}
		reader, err := t.memories.Reader(ctx, request.TenantInfo, person)
		if err != nil {
			return err
		}
		// The team is the first of the person's own roles by name; a person
		// with several can move the memory to another on the Desk.
		if len(reader.RoleIDs) == 0 {
			return fmt.Errorf("the person holds no role to keep this for; use me or organization")
		}
		request.Scope = agentmemory.MemoryScopeRole
		request.RoleID = reader.RoleIDs[0]
	default:
		return fmt.Errorf("visibleTo %q is not one of %s",
			audience, strings.Join(memoryAudiences.Names(), ", "))
	}

	if person.IsNil() || params.ApprovedFromProposal() {
		return nil
	}
	mode, err := t.memories.SavingMode(ctx, request.TenantInfo, person)
	if err != nil {
		return err
	}
	request.Suggest = mode == agentmemory.MemorySavingAskFirst

	return nil
}

// forgetMemoryTool retires a memory that no longer holds. It is a status
// change, not a delete, so a person can see what was forgotten and restore
// it.
type ForgetMemoryTool struct {
	memories agentmemoryports.AgentMemoryService
}

func NewForgetMemoryTool(memories agentmemoryports.AgentMemoryService) agenttoolports.AgentTool {
	return &ForgetMemoryTool{memories: memories}
}

func (t *ForgetMemoryTool) Name() string { return "agent_retire_memory" }

func (t *ForgetMemoryTool) SearchTerms() []string {
	return []string{"forget memory"}
}

func (t *ForgetMemoryTool) Description() string {
	return "Retire a recorded memory that no longer holds, by its id from agent_list_memories: " +
		"an arrangement that ended, a fact a person says is wrong. It stays readable in " +
		"AI Control and can be restored. Use this only when told the memory is wrong or " +
		"over, never to make room."
}

func (t *ForgetMemoryTool) ParamSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: map[string]any{
			"memoryId": agenttoolschema.RecordIDText(
				permission.ResourceAgentMemory,
				"The memory to retire, by the id agent_list_memories returned.",
			),
		},
		toolschema.KeyRequired:             []string{"memoryId"},
		toolschema.KeyAdditionalProperties: false,
	}
}

func (t *ForgetMemoryTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agenttooltrust.ToolKindAction,
		Resource:      permission.ResourceAssistant,
		Operation:     permission.OpCreate,
		Scope:         agenttooltrust.ToolScopeTenant,
		DefaultTier:   agent.TierActWithApproval,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agenttooltrust.ToolEffectChange,
		Reversible:    true,
		ReadsExternal: agenttooltrust.ExternalReadNever,
		Rationale: "Retires an agent memory inside Trenova: one kept for the person in the " +
			"conversation, or one shared with others when that person may manage shared " +
			"memory. A run with nobody in it retires none.",
	}
}

func (t *ForgetMemoryTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams,
) error {
	request, err := t.request(&params)
	if err != nil {
		return err
	}

	_, err = t.memories.SetStatus(ctx, request, params.Actor)

	return err
}

func (t *ForgetMemoryTool) request(
	params *agenttoolports.ToolExecuteParams,
) (agentmemoryports.SetAgentMemoryStatusRequest, error) {
	if err := toolguard.GuardExecute(t, *params); err != nil {
		return agentmemoryports.SetAgentMemoryStatusRequest{}, err
	}

	memoryID, err := toolparams.RequirePulid(params.Params, "memoryId")
	if err != nil {
		return agentmemoryports.SetAgentMemoryStatusRequest{}, err
	}

	return agentmemoryports.SetAgentMemoryStatusRequest{
		ID:         memoryID,
		TenantInfo: toolparams.TenantFrom(*params),
		Status:     agentmemory.MemoryStatusRetired,
	}, nil
}

var memorySubjectTypes = agenttoolschema.Source(
	"agent.memorySubjectType",
	agentmemory.AllMemorySubjectTypes(),
)

var rememberedMemoryKinds = agenttoolschema.Source(
	"agent.rememberedMemoryKind",
	[]agentmemory.MemoryKind{
		agentmemory.MemoryKindInstruction,
		agentmemory.MemoryKindFact,
		agentmemory.MemoryKindProcedure,
	},
)

var memoryAudiences = agenttoolschema.Source(
	"agent.memoryAudience",
	[]string{visibleToMe, visibleToTeam, visibleToOrganization},
)

const fieldVisibleTo = "visibleTo"

const fieldReplacesMemoryID = "replacesMemoryId"

// memorySubject reads the optional subject pair. A type with no id names
// nothing, so it is read as no subject: a model that fills every field sends
// a type for a memory about no record, and refusing that kept it from ever
// saving one. An id without a type cannot be looked up and is refused.
func memorySubject(params map[string]any) (agentmemory.MemorySubjectType, pulid.ID, error) {
	subjectType := agentmemory.MemorySubjectType(toolparams.OptionalString(params, "subjectType"))
	rawID := toolparams.OptionalString(params, "subjectId")

	if rawID == "" {
		return "", pulid.Nil, nil
	}
	if subjectType == "" {
		return "", pulid.Nil, fmt.Errorf(
			"subjectId needs subjectType: %s", strings.Join(memorySubjectTypes.Names(), ", "),
		)
	}
	if !subjectType.IsValid() {
		return "", pulid.Nil, fmt.Errorf("subjectType %q is not one of %s",
			subjectType, strings.Join(memorySubjectTypes.Names(), ", "))
	}

	subjectID, err := pulid.Parse(rawID)
	if err != nil {
		return "", pulid.Nil, fmt.Errorf("subjectId %q is not a record id", rawID)
	}

	return subjectType, subjectID, nil
}

// memoryExpiry reads a YYYY-MM-DD as the end of that day, so "expires on the
// 30th" still applies on the 30th.
func memoryExpiry(raw string) (*int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	day, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return nil, fmt.Errorf("expiresOn must be YYYY-MM-DD, got %q", raw)
	}

	end := day.Add(24 * time.Hour).Unix()

	return &end, nil
}
