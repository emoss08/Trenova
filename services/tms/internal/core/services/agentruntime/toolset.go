package agentruntime

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.uber.org/zap"
)

const (
	// disclosureThreshold is the point past which a turn stops being given every
	// tool the agent holds.
	//
	// Below it the whole set is sent and nothing changes, which keeps today's
	// agents behaving exactly as they do. Above it the schemas cost more
	// attention than they buy: thirteen tools is already ~3,900 tokens, and a
	// model picking from thirty overlapping descriptions reaches for the first
	// plausible one rather than the right one.
	disclosureThreshold = 12

	// preselectedTools is how many the turn opens with once disclosure is on.
	preselectedTools = 8

	// foundToolsLimit bounds one find_tools answer. A search that returns
	// fifteen tools has re-created the problem it was added to solve.
	foundToolsLimit = 6

	findToolsName = "find_tools"
)

// findToolsDescription is a constant rather than a literal in the spec below
// because the i18n extractor harvests any Description: field it finds
// (shared/cmd/i18n-extract). This text is addressed to a model; translating it
// would change what the assistant is told based on the operator's locale.
const findToolsDescription = "Load more tools. You start a turn with the tools that " +
	"best fit the request, not all of them. When nothing you can see does the job, " +
	"describe what you need in plain words — \"driver medical card expiry\", " +
	"\"unbilled shipments\" — and the matching tools become callable. Do this instead " +
	"of answering from a tool that only nearly fits, and instead of telling the person " +
	"it cannot be done."

// findToolsSpec is the one tool a disclosed turn always carries.
//
// It is answered by the runtime rather than a registry, because its effect is on
// the turn itself: it adds tools to the set the model may call next. A registry
// tool cannot do that — it returns data and the tool list stays as it was.
func findToolsSpec() serviceports.ToolSpec {
	return serviceports.ToolSpec{
		Name:        findToolsName,
		Description: findToolsDescription,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"need": map[string]any{
					"type":        "string",
					"description": "What you are trying to do, in the operator's words.",
				},
			},
			"required":             []string{"need"},
			"additionalProperties": false,
		},
	}
}

// maxFindCalls bounds find_tools per turn. A search is not charged against
// the tool budget — looking for the right tool is not doing work — so a model
// that searches in a loop has to be stopped by something else.
const maxFindCalls = 4

// toolSetRequest is what a turn's tool set is built from.
type toolSetRequest struct {
	definition *agentdefinition.Definition
	// held is every tool the agent holds, when the caller has already worked
	// it out; otherwise it is worked out from the definition.
	held  []string
	actor *serviceports.RequestActor
	input string
	// history is the replayed conversation. The tools the model used or
	// loaded in its recent turns are loaded again, so a follow-up like "yes,
	// do it" does not reopen with a toolbox that has forgotten the work.
	history    []conversation.Message
	unattended bool
	// delegated says the turn is working on a task another agent handed it.
	// Nobody reads it but that agent, so it has nobody to ask.
	delegated bool
	// publishes says the turn has somewhere to put a document: a
	// conversation with artifacts beside it.
	publishes bool
	// delegates are the agents the turn may hand a task to.
	delegates []agentdefinition.RuntimeDelegate
}

// toolSet is the live set of tools a turn may call. It starts from the agent's
// configuration and grows when the model asks for more.
type toolSet struct {
	specs      []serviceports.ToolSpec
	loaded     map[string]struct{}
	allowed    []string
	disclosed  bool
	unattended bool
	findCalls  int
	// usable answers whether the person may use a tool at all, for naming
	// what exists beyond the agent's configuration.
	usable func(name string) bool
	// delegates are the agents the turn may ask, for naming one that holds
	// what a search could not load.
	delegates []agentdefinition.RuntimeDelegate
}

func (s *Service) newToolSet(ctx context.Context, req toolSetRequest) *toolSet {
	// The set a turn may call is the agent's configuration narrowed to what
	// the person driving it may do. A tool they cannot use was still shown,
	// ranked and offered, and denied only when called, which taught the
	// model that the system refuses rather than that this person lacks the
	// right; the denial also named the resource they lacked.
	held := req.held
	if held == nil {
		held = s.heldTools(req.definition)
	}
	allowed := s.permittedTools(ctx, req.actor, held)
	if req.unattended {
		allowed = s.withoutSelfScoped(allowed)
	}
	selected := agentdefinition.WithoutCoreTools(allowed)

	set := &toolSet{
		loaded:     make(map[string]struct{}, len(allowed)),
		allowed:    allowed,
		unattended: req.unattended,
		usable: func(name string) bool {
			return len(s.permittedTools(ctx, req.actor, []string{name})) == 1
		},
		delegates: req.delegates,
	}

	// The core tools ride on every turn and do not count toward narrowing: a
	// turn that opened with eight slots would otherwise spend half of them on
	// memory and escalation before reaching the work it was asked to do.
	for _, name := range coreOf(allowed) {
		s.load(set, name)
	}

	if len(selected) <= disclosureThreshold || s.catalog == nil {
		for _, name := range selected {
			s.load(set, name)
		}
		set.addConversational(req)

		return set
	}

	set.disclosed = true
	for _, descriptor := range s.catalog.Rank(
		selected,
		rankingText(req.input, req.history),
		preselectedTools,
	) {
		s.load(set, descriptor.Name)
	}
	s.carryOver(set, req.history)
	set.add(findToolsSpec())
	set.addConversational(req)

	return set
}

// addConversational adds the tools that only mean something with a person
// reading. A background run has nobody to ask: offering the question tool
// anyway let an event-driven agent end its run on a question no one would
// see, recorded as complete. Nor has it anywhere to publish a document.
func (t *toolSet) addConversational(req toolSetRequest) {
	if req.unattended {
		return
	}
	if !req.delegated {
		t.add(askUserSpec())
	}
	if req.publishes {
		t.add(publishArtifactSpec())
	}
	if len(req.delegates) > 0 && !req.delegated {
		t.add(delegateTaskSpec(req.delegates))
	}
}

func (t *toolSet) add(spec serviceports.ToolSpec) bool {
	if _, ok := t.loaded[spec.Name]; ok {
		return false
	}

	t.loaded[spec.Name] = struct{}{}
	t.specs = append(t.specs, spec)

	return true
}

// offers reports whether the turn's request carries a tool.
func (t *toolSet) offers(name string) bool {
	_, ok := t.loaded[name]

	return ok
}

// load makes a tool the agent holds callable, with the tools its arguments
// come from. It reports whether the tool itself was newly loaded.
//
// Prerequisites travel with the tool because a model handed create_dashboard
// without list_reports does not go looking for a report id; it writes one.
func (s *Service) load(set *toolSet, name string) bool {
	if !slices.Contains(set.allowed, name) {
		return false
	}
	spec, ok := s.specFor(name)
	if !ok || !set.add(spec) {
		return false
	}

	if s.catalog != nil {
		for _, prerequisite := range s.catalog.Prerequisites(name) {
			if !slices.Contains(set.allowed, prerequisite) {
				continue
			}
			if dependency, found := s.specFor(prerequisite); found {
				set.add(dependency)
			}
		}
	}

	return true
}

// carryOver reloads what the model worked with in its recent turns: the tools
// it called, and what its searches found, by running the same search again.
// The catalog is fixed for the life of the process, so the same need finds
// the same tools, and nothing about the loaded set has to be stored.
func (s *Service) carryOver(set *toolSet, history []conversation.Message) {
	turns := 0
	for idx := len(history) - 1; idx >= 0 && turns < recentToolTurns; idx-- {
		message := history[idx]
		if message.Role == conversation.RoleUser {
			turns++
			continue
		}
		for _, call := range message.ToolCalls {
			switch call.Name {
			case askUserName, publishArtifactName, delegateTaskName:
			case findToolsName:
				need, _ := call.Arguments["need"].(string)
				if strings.TrimSpace(need) == "" {
					continue
				}
				for _, descriptor := range s.catalog.Find(set.allowed, need, foundToolsLimit) {
					s.load(set, descriptor.Name)
				}
			default:
				s.load(set, call.Name)
			}
		}
	}
}

// rankingText is what preselection ranks against: the request, and the one
// before it. "Yes, do that" names nothing; the message it answers does.
func rankingText(input string, history []conversation.Message) string {
	for idx := len(history) - 1; idx >= 0; idx-- {
		if history[idx].Role == conversation.RoleUser {
			return input + "\n" + history[idx].Content
		}
	}

	return input
}

// resolveFind answers a find_tools call and reports what it loaded.
//
// The answer names the tools in the same words the model will see them in, so a
// model that cannot infer from a bare list still has the description in front of
// it on the next turn.
func (s *Service) resolveFind(set *toolSet, arguments map[string]any) string {
	need, _ := arguments["need"].(string)
	if strings.TrimSpace(need) == "" {
		return "Say what you need in a few words — the tools are matched against it."
	}
	if s.catalog == nil {
		return "No other tools are available. Use the ones you have."
	}
	// A turn that was sent everything has nothing left to load; the answer
	// can still say what exists beyond the agent.
	if !set.disclosed {
		return s.nothingLoaded(set, need)
	}

	found := s.catalog.Find(set.allowed, need, foundToolsLimit)

	var b strings.Builder
	added := 0
	for _, descriptor := range found {
		if !s.load(set, descriptor.Name) {
			continue
		}
		added++
		fmt.Fprintf(&b, "- %s: %s\n", descriptor.Name, firstSentence(descriptor.Description))
	}

	if added == 0 {
		return s.nothingLoaded(set, need)
	}

	return fmt.Sprintf("These tools are now callable:\n%s", b.String())
}

// unheldRefusal answers a call to a tool the agent does not hold.
//
// The old answer told the model to call find_tools on turns where find_tools
// was not offered, and find_tools could not have loaded the tool anyway. This
// says which of the two it is — not enabled, or no such tool — and hands over
// the nearest tools the agent does hold, loaded, so the next call can work.
func (s *Service) unheldRefusal(set *toolSet, name string) string {
	var b strings.Builder
	if _, _, exists := s.toolGate(name); exists {
		fmt.Fprintf(&b, "%q is not enabled for this agent, so it was not run. An "+
			"administrator can add it to the agent in AI Control.", name)
	} else {
		fmt.Fprintf(&b, "There is no tool named %q.", name)
	}

	var nearest []serviceports.AgentToolDescriptor
	if s.catalog != nil {
		nearest = s.catalog.Find(set.allowed, strings.ReplaceAll(name, "_", " "), 3)
	}
	if len(nearest) > 0 {
		b.WriteString(" Tools this agent holds that may do the job, now callable:\n")
		for _, descriptor := range nearest {
			s.load(set, descriptor.Name)
			fmt.Fprintf(&b, "- %s: %s\n", descriptor.Name, firstSentence(descriptor.Description))
		}
		return b.String()
	}

	if set.offers(findToolsName) {
		b.WriteString(" Call find_tools with what you are trying to do.")
	} else {
		b.WriteString(" Use one of the tools you have, or tell the person what is missing.")
	}

	return b.String()
}

// heldTools is every tool the agent holds: its core and selected tools, and
// the reads its tools take their arguments from. A dependency is granted only
// when it is a read — the agent holding create_dashboard may look reports up,
// but nothing it holds grants it another write.
func (s *Service) heldTools(definition *agentdefinition.Definition) []string {
	names := definition.EffectiveToolNames()
	if s.catalog == nil {
		return names
	}

	held := len(names)
	for idx := range held {
		for _, prerequisite := range s.catalog.Prerequisites(names[idx]) {
			if s.catalog.IsQuery(prerequisite) && !slices.Contains(names, prerequisite) {
				names = append(names, prerequisite)
			}
		}
	}

	return names
}

func (s *Service) holds(definition *agentdefinition.Definition, name string) bool {
	return slices.Contains(s.heldTools(definition), name)
}

// nothingLoaded explains an empty search without overstating it.
//
// Searching only what the agent holds is correct — find_tools must never reach
// past its configuration — but the old answer turned that into "this system
// does not track it", and a model repeated it as fact. Asked four times whether
// a reports tool existed, it said no four times, growing more certain each
// time, while list_reports sat in the catalog unassigned to that agent.
//
// So the catalog is searched a second time with no allowlist, and the answer
// distinguishes the two cases it was conflating: nothing like this exists, or
// it exists and this agent was not given it. The second names the tools, because
// "ask an administrator to enable something" is not actionable without the name
// — the person ended up supplying it from their side of the conversation.
//
// Naming them widens nothing. The specs are not loaded and the guard still
// refuses a call to anything outside the allowlist.
func (s *Service) nothingLoaded(set *toolSet, need string) string {
	elsewhere := make([]serviceports.AgentToolDescriptor, 0, foundToolsLimit)
	unheld := make([]string, 0, foundToolsLimit)
	for _, descriptor := range s.catalog.Find(nil, need, foundToolsLimit) {
		if _, held := set.loaded[descriptor.Name]; held {
			continue
		}
		if slices.Contains(set.allowed, descriptor.Name) {
			continue
		}
		unheld = append(unheld, descriptor.Name)
		// A tool the person may not use is not worth naming: an
		// administrator adding it to the agent would change nothing for them.
		if set.usable != nil && !set.usable(descriptor.Name) {
			continue
		}
		elsewhere = append(elsewhere, descriptor)
	}

	if note := delegatedSearchNote(set.delegates, unheld); note != "" {
		return note
	}

	if len(elsewhere) == 0 {
		return fmt.Sprintf(
			"Nothing further matched %q, and everything that did is already loaded. "+
				"Use what you have. If this system genuinely does not track it, say so "+
				"— but only about the data, not about tools you cannot see.",
			need,
		)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Nothing you can call matched %q. These exist in this system but "+
		"are not enabled for this agent:\n", need)
	for _, descriptor := range elsewhere {
		fmt.Fprintf(&b, "- %s: %s\n", descriptor.Name, firstSentence(descriptor.Description))
	}
	b.WriteString("\nYou cannot call them. Tell the person these exist and that an " +
		"administrator can add them to this agent in Agent Control, naming them exactly " +
		"as above. Do not say the system has no such capability.")

	return b.String()
}

// permittedTools keeps the names whose tool the actor may use: read for a
// query tool, the tool's own operation for a write. One check per distinct
// resource and operation, since many tools share both.
func (s *Service) permittedTools(
	ctx context.Context,
	actor *serviceports.RequestActor,
	names []string,
) []string {
	// Never nil: the catalog reads nil as "everything", and a person who may
	// use nothing must be offered nothing.
	permitted := make([]string, 0, len(names))
	if s.permissions == nil || actor == nil {
		return permitted
	}

	verdicts := make(map[string]bool, len(names))
	for _, name := range names {
		// A self-scoped tool touches only the person's own records, which any
		// signed-in person may arrange; it needs no grant, and nobody but a
		// person has records of that kind.
		if serviceports.IsSelfScoped(s.toolNamed(name)) {
			if actor.PrincipalType == serviceports.PrincipalTypeUser {
				permitted = append(permitted, name)
			}
			continue
		}

		resource, operation, ok := s.toolGate(name)
		if !ok {
			continue
		}
		key := resource.String() + ":" + string(operation)
		allowed, checked := verdicts[key]
		if !checked {
			result, err := s.permissions.Check(ctx, &serviceports.PermissionCheckRequest{
				PrincipalType:  actor.PrincipalType,
				PrincipalID:    actor.PrincipalID,
				UserID:         actor.UserID,
				APIKeyID:       actor.APIKeyID,
				BusinessUnitID: actor.BusinessUnitID,
				OrganizationID: actor.OrganizationID,
				Resource:       resource.String(),
				Operation:      operation,
			})
			allowed = err == nil && result != nil && result.Allowed
			if err != nil {
				s.logger.Warn("could not check whether a tool may be offered; withholding it",
					zap.String("tool", name), zap.Error(err))
			}
			verdicts[key] = allowed
		}
		if allowed {
			permitted = append(permitted, name)
		}
	}

	return permitted
}

// toolNamed is the registered tool behind a name, read or write, or nil.
func (s *Service) toolNamed(name string) any {
	if tool, ok := s.queryTools.Get(name); ok {
		return tool
	}
	if tool, ok := s.actionTools.Get(name); ok {
		return tool
	}

	return nil
}

// toolGate is the permission a tool is used under.
func (s *Service) toolGate(name string) (permission.Resource, permission.Operation, bool) {
	if tool, ok := s.queryTools.Get(name); ok {
		return tool.PermissionResource(), permission.OpRead, true
	}
	if tool, ok := s.actionTools.Get(name); ok {
		return tool.PermissionResource(), tool.PermissionOperation(), true
	}

	return "", "", false
}

func (s *Service) specFor(name string) (serviceports.ToolSpec, bool) {
	if tool, ok := s.queryTools.Get(name); ok {
		return serviceports.ToolSpec{
			Name:        tool.Name(),
			Description: tool.Description(),
			Parameters:  tool.ParamSchema(),
		}, true
	}
	if tool, ok := s.actionTools.Get(name); ok {
		return serviceports.ToolSpec{
			Name:        tool.Name(),
			Description: tool.Description(),
			Parameters:  tool.ParamSchema(),
		}, true
	}

	return serviceports.ToolSpec{}, false
}

func coreOf(names []string) []string {
	core := make([]string, 0, len(names))
	for _, name := range names {
		if agentdefinition.IsCoreTool(name) {
			core = append(core, name)
		}
	}

	return core
}

func toSpec(descriptor serviceports.AgentToolDescriptor) serviceports.ToolSpec {
	return serviceports.ToolSpec{
		Name:        descriptor.Name,
		Description: descriptor.Description,
		Parameters:  descriptor.Parameters,
	}
}

func firstSentence(text string) string {
	return stringutils.FirstSentence(text)
}

// withoutSelfScoped drops the tools that act on a person's own records. A run
// nobody is watching has no person whose records they would be.
func (s *Service) withoutSelfScoped(names []string) []string {
	kept := make([]string, 0, len(names))
	for _, name := range names {
		if !serviceports.IsSelfScoped(s.toolNamed(name)) {
			kept = append(kept, name)
		}
	}

	return kept
}
