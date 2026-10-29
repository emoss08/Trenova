package toolaccess

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/config/aiconfig"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/askuser"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolcall"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/turnhistory"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolcatalog"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreset"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentturn"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aianalytics"
	"github.com/emoss08/trenova/internal/modules/agent/domain/codemode"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/retrievalports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const (
	// DisclosureThreshold is the point past which a turn stops being given every
	// tool the agent holds.
	//
	// Below it the whole set is sent and nothing changes, which keeps today's
	// agents behaving exactly as they do. Above it the schemas cost more
	// attention than they buy: thirteen tools is already ~3,900 tokens, and a
	// model picking from thirty overlapping descriptions reaches for the first
	// plausible one rather than the right one.
	DisclosureThreshold = aiconfig.DefaultToolDisclosureThreshold

	// PreselectedTools is how many the turn opens with once disclosure is on.
	PreselectedTools = 8

	// foundToolsLimit bounds one find_tools answer. A search that returns
	// fifteen tools has re-created the problem it was added to solve.
	foundToolsLimit = 6
)

// MaxFindCalls bounds find_tools per turn. A search is not charged against
// the tool budget — looking for the right tool is not doing work — so a model
// that searches in a loop has to be stopped by something else.
const MaxFindCalls = 4

const EscalationReserve = 2

// SearchesLeft is said once a turn has searched for tools more than twice:
// gpt-6-luna re-worded the same need five times rather than call the tools
// its first search had loaded, and ran out of searches before it answered.
func SearchesLeft(calls int) string {
	if calls < MaxFindCalls-1 {
		return ""
	}
	left := MaxFindCalls - calls
	if left == 0 {
		return "\n\nThat was your last tool search this turn. Call the tools you hold, " +
			"or answer with what you have."
	}

	return fmt.Sprintf("\n\nThat was tool search %d of %d this turn, and what it found is "+
		"loaded now. Call one of those tools rather than searching again.", calls, MaxFindCalls)
}

// ToolSetRequest is what a turn's tool set is built from.
type ToolSetRequest struct {
	Definition *agentdefinition.Definition
	// Held is every tool the agent holds, when the caller has already worked
	// it out; otherwise it is worked out from the definition.
	Held   []string
	Beyond map[string]agent.AutonomyTier
	Actor  *permissionports.RequestActor
	Input  string
	Need   string
	Query  retrievalports.QueryVectorRequest
	// History is the replayed conversation. The tools the model used or
	// loaded in its recent turns are loaded again, so a follow-up like "yes,
	// do it" does not reopen with a toolbox that has forgotten the work.
	History    []conversation.Message
	Unattended bool
	// Delegated says the turn is working on a task another agent handed it.
	// Nobody reads it but that agent, so it has nobody to ask.
	Delegated bool
	// Publishes says the turn has somewhere to put a document: a
	// conversation with artifacts beside it.
	Publishes bool
	// Delegates are the agents the turn may hand a task to.
	Delegates []agentturn.RuntimeDelegate
	Decisions bool
	// Extensions is what the turn already read of the organization's
	// extensions; nil reads them again.
	Extensions  *agentextension.Active
	Attachments bool
	Command     agentextension.Command
}

// ToolSet is the set of tools a turn may call. It is declared once, when the
// turn opens: the tools it opens with, sent in full, and every other tool it
// may load, declared as deferred. Loading one later never changes what is
// declared; it marks the tool loaded and names it on the next tool result, so
// the request's tools and everything cached after them stay the same bytes
// from the first call of the turn to the last.
type ToolSet struct {
	Specs      []kernelports.ToolSpec
	Deferred   []string
	frozen     bool
	Pending    []string
	Loaded     map[string]struct{}
	Allowed    []string
	Beyond     []string
	Disclosed  bool
	unattended bool
	FindCalls  int
	// Usable answers whether the person may use a tool at all, for naming
	// what exists beyond the agent's configuration.
	Usable func(name string) bool
	Query  retrievalports.QueryVector
	// Self is the agent the turn runs as, left out when naming another of
	// the person's agents that holds a tool.
	Self        pulid.ID
	held        int
	Preselected []string
	Pinned      bool
}

func (s *Access) NewToolSet(ctx context.Context, req ToolSetRequest) *ToolSet {
	set := s.openToolSet(ctx, &req)
	if req.Command != "" {
		set.Pinned = s.pinCommand(set, req.Command)
	}
	s.freeze(set)

	return set
}

func (s *Access) openToolSet(ctx context.Context, req *ToolSetRequest) *ToolSet {
	// The set a turn may call is the agent's configuration narrowed to what
	// the person driving it may do. A tool they cannot use was still shown,
	// ranked and offered, and denied only when called, which taught the
	// model that the system refuses rather than that this person lacks the
	// right; the denial also named the resource they lacked.
	held := req.Held
	if held == nil {
		held = s.HeldTools(req.Definition)
	}
	allowed := s.PermittedWith(ctx, req.Actor, held, req.Extensions)
	if req.Unattended {
		allowed = s.withoutSelfScoped(allowed)
	}
	selected := agentdefinition.WithoutCoreTools(allowed)

	set := &ToolSet{
		Loaded:     make(map[string]struct{}, len(allowed)),
		Allowed:    allowed,
		Beyond:     s.PermittedWith(ctx, req.Actor, BeyondNames(req.Beyond), req.Extensions),
		unattended: req.Unattended,
		Usable: func(name string) bool {
			return len(s.PermittedTools(ctx, req.Actor, []string{name})) == 1
		},
		Self: req.Definition.ID,
	}

	disclosed := len(selected) > s.toolThreshold() && s.catalog != nil
	searchable := disclosed || (len(set.Beyond) > 0 && s.catalog != nil)
	text := rankingText(req.Input, req.History)
	if req.Need != "" {
		text = req.Need + "\n" + req.Input
	}

	// The core tools ride on every turn and do not count toward narrowing: a
	// turn that opened with eight slots would otherwise spend half of them on
	// memory and escalation before reaching the work it was asked to do. On a
	// turn that can search, the ones few turns call wait, deferred, unless the
	// request names what they do.
	waiting := make([]string, 0, len(deferredCoreTools))
	for _, name := range coreOf(allowed) {
		if name == agentdefinition.CoreToolReadAttachment && !req.Attachments {
			continue
		}
		if searchable && defersCoreTool(name, req.Unattended) {
			waiting = append(waiting, name)
			continue
		}
		s.Load(set, name)
	}
	if len(waiting) > 0 {
		found := s.catalog.FindHybrid(agenttoolcatalog.Query{
			Allowed:    waiting,
			Text:       text,
			Limit:      len(waiting),
			StrongOnly: true,
		})
		for idx := range found {
			s.Load(set, found[idx].Name)
		}
	}

	if !disclosed {
		for _, name := range selected {
			s.Load(set, name)
		}
		if searchable {
			s.carryOver(set, req.History)
			set.add(runtimetools.FindToolsSpec())
		}
		set.addConversational(*req)
		s.declareWithdrawal(set, req.Delegated)
		s.declareNamePlan(set, req.Delegated)

		return set
	}

	set.Disclosed = true
	set.held = len(selected)
	set.Query = s.Vectorize(ctx, &req.Query)
	for _, name := range agentpreset.StandingTools(req.Definition.Template) {
		s.Load(set, name)
	}
	for _, descriptor := range s.catalog.RankHybrid(agenttoolcatalog.Query{
		Allowed:  slices.DeleteFunc(slices.Clone(selected), set.Offers),
		Text:     text,
		Limit:    PreselectedTools,
		Semantic: s.ToolSemantic(ctx, req.Query.TenantInfo, set.Query),
	}) {
		set.Preselected = append(set.Preselected, descriptor.Name)
		s.Load(set, descriptor.Name)
	}
	// Whether a question needs many reads is the model's to judge once it
	// sees the first result, which is after the turn's tools are fixed, so
	// the script tool rides on every turn that may run one rather than
	// waiting on a search the model would not think to make.
	s.Load(set, codemode.ToolRunScript)
	s.carryOver(set, req.History)
	set.add(runtimetools.FindToolsSpec())
	set.addConversational(*req)
	s.declareWithdrawal(set, req.Delegated)
	s.declareNamePlan(set, req.Delegated)

	return set
}

var deferredCoreTools = []string{
	agentdefinition.CoreToolCancelWait,
	agentdefinition.CoreToolGuideProfile,
	agentdefinition.CoreToolRaiseException,
	agentdefinition.CoreToolSchedule,
	agentdefinition.CoreToolWaitUntil,
}

func defersCoreTool(name string, unattended bool) bool {
	if unattended && name == agentdefinition.CoreToolRaiseException {
		return false
	}

	return slices.Contains(deferredCoreTools, name)
}

// addConversational adds the tools that only mean something with a person
// reading. A background run has nobody to ask: offering the question tool
// anyway let an event-driven agent end its run on a question no one would
// see, recorded as complete. Nor has it anywhere to publish a document.
func (t *ToolSet) addConversational(req ToolSetRequest) {
	if req.Unattended {
		return
	}
	if !req.Delegated {
		t.add(askuser.Spec())
	}
	if req.Publishes {
		t.add(runtimetools.PublishArtifactSpec())
		t.add(runtimetools.ShowBoardSpec())
	}
	if len(req.Delegates) > 0 && !req.Delegated {
		t.add(runtimetools.DelegateTaskSpec(req.Delegates))
	}
	if req.Decisions && req.Publishes && !req.Delegated {
		t.add(runtimetools.RequestDecisionSpec())
	}
}

func (t *ToolSet) add(spec kernelports.ToolSpec) bool {
	if _, ok := t.Loaded[spec.Name]; ok || t.frozen {
		return false
	}

	t.Loaded[spec.Name] = struct{}{}
	t.Specs = append(t.Specs, spec)

	return true
}

func (t *ToolSet) reaches(name string) bool {
	return slices.Contains(t.Allowed, name) || slices.Contains(t.Beyond, name)
}

func (t *ToolSet) Searchable() bool {
	return t.Disclosed || len(t.Beyond) > 0
}

// Offers reports whether the turn's request carries a tool.
func (t *ToolSet) Offers(name string) bool {
	_, ok := t.Loaded[name]

	return ok
}

// Load makes a tool the agent holds callable, with the tools its arguments
// come from. It reports whether the tool itself was newly loaded.
//
// Prerequisites travel with the tool because a model handed report_create_dashboard
// without report_list does not go looking for a report id; it writes one.
func (s *Access) Load(set *ToolSet, name string) bool {
	if !s.loadOne(set, name) {
		return false
	}

	if s.catalog != nil {
		for _, prerequisite := range s.catalog.Prerequisites(name) {
			if slices.Contains(set.Allowed, prerequisite) {
				s.loadOne(set, prerequisite)
			}
		}
		for _, companion := range s.catalog.Companions(name) {
			s.Load(set, companion)
		}
		if fed := soleFed(set.Allowed, s.catalog.Feeds(name)); fed != "" {
			s.Load(set, fed)
		}
	}

	return true
}

func (s *Access) loadOne(set *ToolSet, name string) bool {
	if !set.reaches(name) || set.Offers(name) {
		return false
	}
	if set.frozen {
		if !slices.Contains(set.Deferred, name) {
			return false
		}
		set.Loaded[name] = struct{}{}
		set.Pending = append(set.Pending, name)

		return true
	}
	spec, ok := s.SpecFor(name)

	return ok && set.add(spec)
}

func (s *Access) freeze(set *ToolSet) {
	if set.frozen {
		return
	}
	slices.SortFunc(set.Specs, func(a, b kernelports.ToolSpec) int {
		return strings.Compare(a.Name, b.Name)
	})
	if set.Offers(runtimetools.FindToolsName) {
		reachable := slices.Concat(set.Allowed, set.Beyond)
		deferred := make([]string, 0, len(reachable))
		for _, name := range reachable {
			if set.Offers(name) || slices.Contains(deferred, name) {
				continue
			}
			if _, ok := s.SpecFor(name); ok {
				deferred = append(deferred, name)
			}
		}
		slices.Sort(deferred)
		set.Deferred = deferred
	}
	set.frozen = true
}

func (t *ToolSet) TakeLoaded() []string {
	loaded := t.Pending
	t.Pending = nil

	return loaded
}

// soleFed is the one held tool a lookup is a step of the recipe of, when it
// is one. A lookup that exists to hand one write its arguments brings the
// write: shipment_match_records was loaded on its own, its answer said to
// call shipment_create next, and the model went looking for shipment_create
// with find_tools first. A lookup many recipes start from (shipment_list,
// shipment_get) brings none of them.
func soleFed(allowed, fed []string) string {
	held := ""
	for _, name := range fed {
		if !slices.Contains(allowed, name) {
			continue
		}
		if held != "" {
			return ""
		}
		held = name
	}

	return held
}

// carryOver reloads what the model worked with in its recent turns: the tools
// it called, and what its searches found. A search's result names what it
// found, so the same tools come back however the ranking has moved since; a
// result saved before the names were kept runs the search again by keyword.
func (s *Access) carryOver(set *ToolSet, history []conversation.Message) {
	found := foundByCall(history)
	turns := 0
	for idx := len(history) - 1; idx >= 0 && turns < turnhistory.RecentToolTurns; idx-- {
		message := history[idx]
		if message.Role == conversation.RoleUser {
			turns++
			continue
		}
		for _, call := range message.ToolCalls {
			switch call.Name {
			case askuser.Name,
				runtimetools.PublishArtifactName,
				runtimetools.ShowBoardName,
				runtimetools.DelegateTaskName,
				runtimetools.RequestDecisionName,
				runtimetools.WithdrawProposalName,
				runtimetools.NamePlanName:
			case runtimetools.FindToolsName:
				if names, stored := found[call.ID]; stored {
					for _, name := range names {
						s.Load(set, name)
					}
					continue
				}
				need, _ := call.Arguments["need"].(string)
				if strings.TrimSpace(need) == "" {
					continue
				}
				for _, descriptor := range s.catalog.Find(set.Allowed, need, foundToolsLimit) {
					s.Load(set, descriptor.Name)
				}
			default:
				s.Load(set, call.Name)
			}
		}
	}
}

func FrameNeed(req *agentruntimeports.RunRequest) string {
	if req.Context.Frame == nil {
		return ""
	}

	return strings.TrimSpace(req.Context.Frame.Need)
}

func TurnQuery(req *agentruntimeports.RunRequest) retrievalports.QueryVectorRequest {
	query := agentruntimeports.TurnQueryRequest(req)
	if need := FrameNeed(req); need != "" && req.Actor != nil {
		query.Text = need + "\n" + req.Input
	}

	return query
}

// rankingText is what preselection ranks against: the request, and the one
// before it. "Yes, do that" names nothing; the message it answers does.
func rankingText(input string, history []conversation.Message) string {
	return retrievalports.TurnQueryText(input, history)
}

// ResolveFind answers a find_tools call and reports what it loaded.
//
// The answer names the tools in the same words the model will see them in, so a
// model that cannot infer from a bare list still has the description in front of
// it on the next turn.
func (s *Access) ResolveFind(
	ctx context.Context,
	set *ToolSet,
	actor *permissionports.RequestActor,
	arguments map[string]any,
) FindAnswer {
	need, _ := arguments["need"].(string)
	if strings.TrimSpace(need) == "" {
		return FindAnswer{
			Content: "Say what you need in a few words — the tools are matched against it.",
		}
	}
	if s.catalog == nil {
		return FindAnswer{Content: "No other tools are available. Use the ones you have."}
	}
	// A turn that was sent everything has nothing left to load; the answer
	// can still say what exists beyond the agent.
	if !set.Searchable() {
		content, handOff := s.nothingLoaded(ctx, set, actor, need)

		return FindAnswer{Content: content, HandOff: handOff}
	}

	found := s.withNamedTools(set.Allowed, need, s.catalog.FindHybrid(agenttoolcatalog.Query{
		Allowed:      set.Allowed,
		Text:         need,
		Limit:        foundToolsLimit,
		Semantic:     s.ToolSemantic(ctx, actorTenant(actor), set.Query),
		SkipSemantic: set.Loaded,
	}))
	names := make([]string, 0, len(found))
	callable := make([]agenttoolports.AgentToolDescriptor, 0, len(found))
	added := make([]agenttoolports.AgentToolDescriptor, 0, len(found))
	alongside := make([]string, 0, 4)
	for _, descriptor := range found {
		names = append(names, descriptor.Name)
		if set.Offers(descriptor.Name) {
			callable = append(callable, descriptor)
			continue
		}
		before := len(set.Pending)
		if s.Load(set, descriptor.Name) {
			added = append(added, descriptor)
			for _, name := range set.Pending[before:] {
				if name != descriptor.Name {
					alongside = append(alongside, name)
				}
			}
		}
	}

	reached := s.reachBeyond(ctx, set, actor, need, s.strongestMatch(need, callable, added))
	for idx := range reached {
		names = append(names, reached[idx].Name)
	}

	if len(added) == 0 && len(callable) == 0 && len(reached) == 0 {
		content, handOff := s.nothingLoaded(ctx, set, actor, need)

		return FindAnswer{Content: content, Found: names, HandOff: handOff}
	}

	var b strings.Builder
	if len(callable) > 0 {
		b.WriteString("These tools you already hold match, are already loaded and can be " +
			"called now:\n")
		writeToolLines(&b, callable)
	}
	if len(added) > 0 {
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		b.WriteString("These tools are now callable:\n")
		writeToolLines(&b, added)
		if len(alongside) > 0 {
			fmt.Fprintf(&b, "Loaded with them, because their ids come from these: %s.\n",
				strings.Join(alongside, ", "))
		}
	}
	if len(reached) > 0 {
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(BeyondLead)
		writeToolLines(&b, reached)
	}
	if len(added) == 0 && len(reached) == 0 {
		fmt.Fprintf(&b, "\nNothing new matched %q beyond those.", need)
	}

	return FindAnswer{Content: strings.TrimRight(b.String(), "\n"), Found: names}
}

const BeyondLead = "These come from the desks of the person's other agents and are now " +
	"callable here, under the same approvals as on those desks:\n"

func (s *Access) reachBeyond(
	ctx context.Context,
	set *ToolSet,
	actor *permissionports.RequestActor,
	need string,
	ownScore int,
) []agenttoolports.AgentToolDescriptor {
	if len(set.Beyond) == 0 {
		return nil
	}
	found := s.withNamedTools(set.Beyond, need, s.catalog.FindHybrid(agenttoolcatalog.Query{
		Allowed:    set.Beyond,
		Text:       need,
		Limit:      foundToolsLimit,
		Semantic:   s.ToolSemantic(ctx, actorTenant(actor), set.Query),
		StrongOnly: true,
	}))
	reached := make([]agenttoolports.AgentToolDescriptor, 0, len(found))
	for idx := range found {
		if ownScore > 0 && s.catalog.StrongScore(found[idx].Name, need) <= ownScore {
			continue
		}
		if set.Offers(found[idx].Name) || s.Load(set, found[idx].Name) {
			reached = append(reached, found[idx])
		}
	}

	return reached
}

func (s *Access) strongestMatch(
	need string,
	matches ...[]agenttoolports.AgentToolDescriptor,
) int {
	strongest := 0
	for _, descriptors := range matches {
		for idx := range descriptors {
			strongest = max(strongest, s.catalog.StrongScore(descriptors[idx].Name, need))
		}
	}

	return strongest
}

// withNamedTools puts first the tools a need names outright. The prompt lists
// what find_tools can load by name, and a model that asked for "worker_list"
// by name was handed six other tools ranked above it and searched again.
func (s *Access) withNamedTools(
	allowed []string,
	need string,
	found []agenttoolports.AgentToolDescriptor,
) []agenttoolports.AgentToolDescriptor {
	named := make([]agenttoolports.AgentToolDescriptor, 0, 4)
	for _, word := range strings.FieldsFunc(need, notToolNameRune) {
		if !slices.Contains(allowed, word) ||
			slices.ContainsFunc(named, func(d agenttoolports.AgentToolDescriptor) bool {
				return d.Name == word
			}) {
			continue
		}
		if descriptor, ok := s.catalog.Descriptor(word); ok {
			named = append(named, descriptor)
		}
	}
	if len(named) == 0 {
		return found
	}

	limit := max(foundToolsLimit, len(named))
	for _, descriptor := range found {
		if len(named) >= limit {
			break
		}
		if !slices.ContainsFunc(named, func(d agenttoolports.AgentToolDescriptor) bool {
			return d.Name == descriptor.Name
		}) {
			named = append(named, descriptor)
		}
	}

	return named
}

func notToolNameRune(r rune) bool {
	return (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '_'
}

// writeToolLines names each tool by its first sentence, and for one done
// as a sequence, the order of the tools around it, so the model does not
// search again for the lookup it needs next.
func writeToolLines(b *strings.Builder, descriptors []agenttoolports.AgentToolDescriptor) {
	for _, descriptor := range descriptors {
		fmt.Fprintf(
			b,
			"- %s: %s\n",
			descriptor.Name,
			stringutils.FirstSentence(descriptor.Description),
		)
		if len(descriptor.Recipe) > 0 {
			fmt.Fprintf(b, "  In order: %s\n", strings.Join(descriptor.Recipe, " → "))
		}
	}
}

func (s *Access) unheldStrongMatches(set *ToolSet, need string) []string {
	unheld := make([]string, 0, foundToolsLimit)
	for _, descriptor := range s.catalog.FindHybrid(agenttoolcatalog.Query{
		Text:       need,
		Limit:      foundToolsLimit,
		StrongOnly: true,
	}) {
		if _, loaded := set.Loaded[descriptor.Name]; loaded {
			continue
		}
		if slices.Contains(set.Allowed, descriptor.Name) {
			continue
		}
		unheld = append(unheld, descriptor.Name)
	}

	return unheld
}

// unheldRefusal answers a call to a tool the agent does not hold.
//
// The old answer told the model to call find_tools on turns where find_tools
// was not offered, and find_tools could not have loaded the tool anyway. This
// says which of the two it is — not enabled, or no such tool — and hands over
// the nearest tools the agent does hold, loaded, so the next call can work.
func (s *Access) unheldRefusal(set *ToolSet, name string, exists bool) string {
	var b strings.Builder
	if exists {
		fmt.Fprintf(&b, "%q is not enabled for this agent, so it was not run. An "+
			"administrator can add it to the agent in AI Control.", name)
	} else {
		fmt.Fprintf(&b, "There is no tool named %q.", name)
	}

	var nearest []agenttoolports.AgentToolDescriptor
	if s.catalog != nil {
		nearest = s.catalog.Find(set.Allowed, strings.ReplaceAll(name, "_", " "), 3)
	}
	if len(nearest) > 0 {
		b.WriteString(" Tools this agent holds that may do the job, now callable:\n")
		for _, descriptor := range nearest {
			s.Load(set, descriptor.Name)
			fmt.Fprintf(
				&b,
				"- %s: %s\n",
				descriptor.Name,
				stringutils.FirstSentence(descriptor.Description),
			)
		}
		return b.String()
	}

	if set.Offers(runtimetools.FindToolsName) {
		b.WriteString(" Call find_tools with what you are trying to do.")
	} else {
		b.WriteString(" Use one of the tools you have, or tell the person what is missing.")
	}

	return b.String()
}

func (s *Access) UnheldOutcome(set *ToolSet, name string) toolcall.Result {
	if _, _, exists := s.ToolGate(name); exists {
		return toolcall.Refused(aitrace.OutcomeDenied, aianalytics.ReasonNotHeld,
			"%s", s.unheldRefusal(set, name, true))
	}

	return toolcall.Refused(aitrace.OutcomeInvalid, aianalytics.ReasonUnregistered,
		"%s", s.unheldRefusal(set, name, false))
}

// HeldTools is every tool the agent holds: its core and selected tools, the
// writes a selected tool names as its companions, and the reads its tools take
// their arguments from. A prerequisite is granted only when it is a read — the
// agent holding report_create_dashboard may look reports up, but that grants it no
// write. A companion is the one write a tool may bring, because its job cannot
// be finished without it: shipment_create brings the customer, location and
// commodity it may find missing. Each still answers to its own policy, so it
// is proposed and approved like any write, and one an administrator switched
// off for the agent stays off. An agent that holds a tool for its work also
// holds the tool that runs a script over its tools, unless an administrator switched it off
// for the agent; whether a turn is shown it is the code-mode gate's to decide.
func (s *Access) HeldTools(definition *agentdefinition.Definition) []string {
	names := definition.EffectiveToolNames()
	if s.catalog == nil {
		return s.withScripts(definition, names)
	}

	selected := len(names)
	for idx := range selected {
		for _, companion := range s.catalog.Companions(names[idx]) {
			if !slices.Contains(names, companion) &&
				!slices.Contains(definition.DisabledToolNames, companion) {
				names = append(names, companion)
			}
		}
	}
	held := len(names)
	for idx := range held {
		for _, prerequisite := range s.catalog.Prerequisites(names[idx]) {
			if s.catalog.IsQuery(prerequisite) && !slices.Contains(names, prerequisite) {
				names = append(names, prerequisite)
			}
		}
	}

	return s.withScripts(definition, names)
}

func (s *Access) withScripts(definition *agentdefinition.Definition, names []string) []string {
	if !s.LooksUp(codemode.ToolRunScript) ||
		slices.Contains(names, codemode.ToolRunScript) ||
		slices.Contains(definition.DisabledToolNames, codemode.ToolRunScript) ||
		!slices.ContainsFunc(names, scriptable) {
		return names
	}

	return append(names, codemode.ToolRunScript)
}

// scriptable reports whether a held tool is one a script could call: a tool
// the agent was given for its work, not one of the core tools every agent
// holds.
func scriptable(name string) bool {
	return !agentdefinition.IsCoreTool(name)
}

func (s *Access) holds(definition *agentdefinition.Definition, name string) bool {
	return slices.Contains(s.HeldTools(definition), name)
}

// nothingLoaded explains an empty search without overstating it.
//
// Searching only what the agent holds is correct — find_tools must never reach
// past its configuration — but the old answer turned that into "this system
// does not track it", and a model repeated it as fact. Asked four times whether
// a reports tool existed, it said no four times, growing more certain each
// time, while report_list sat in the catalog unassigned to that agent.
//
// So the catalog is searched a second time with no allowlist, and the answer
// distinguishes the two cases it was conflating: nothing like this exists, or
// it exists and this agent was not given it. The second names the tools, because
// "ask an administrator to enable something" is not actionable without the name
// — the person ended up supplying it from their side of the conversation.
//
// Naming them widens nothing. The specs are not loaded and the guard still
// refuses a call to anything outside the allowlist. The person's other agents
// that hold them are returned beside the answer, for the hand-off menu.
func (s *Access) nothingLoaded(
	ctx context.Context,
	set *ToolSet,
	actor *permissionports.RequestActor,
	need string,
) (string, []pulid.ID) {
	unheld := s.unheldStrongMatches(set, need)
	elsewhere := make([]agenttoolports.AgentToolDescriptor, 0, len(unheld))
	for _, name := range unheld {
		if set.Usable != nil && !set.Usable(name) {
			continue
		}
		if descriptor, ok := s.catalog.Descriptor(name); ok {
			elsewhere = append(elsewhere, descriptor)
		}
	}

	if len(elsewhere) == 0 {
		return fmt.Sprintf(
			"Nothing further matched %q, and everything that did is already loaded. "+
				"Use what you have. If this system genuinely does not track it, say so "+
				"— but only about the data, not about tools you cannot see.",
			need,
		), nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Nothing you can call matched %q. These exist in this system but "+
		"are not enabled for this agent:\n", need)
	writeToolLines(&b, elsewhere)
	names := make([]string, 0, len(elsewhere))
	for idx := range elsewhere {
		names = append(names, elsewhere[idx].Name)
	}
	if note, holders := s.handOffNote(ctx, actor, set.Self, names); note != "" {
		b.WriteString("\nYou cannot call them. " + note +
			" Do not say the system has no such capability.")

		return b.String(), holders
	}
	b.WriteString("\nYou cannot call them. Tell the person these exist and that an " +
		"administrator can add them to this agent in AI Control, naming them exactly " +
		"as above. Do not say the system has no such capability.")

	return b.String(), nil
}

// PermittedTools keeps the names whose tool the actor may use: read for a
// query tool, the tool's own operation for a write. One check per distinct
// resource and operation, since many tools share both.
func (s *Access) PermittedTools(
	ctx context.Context,
	actor *permissionports.RequestActor,
	names []string,
) []string {
	return s.PermittedWith(ctx, actor, names, nil)
}

// maxConcurrentToolChecks bounds how many permission checks one turn asks at
// once, so an agent with a wide toolbox does not take the cache's whole pool.
const maxConcurrentToolChecks = 8

// PermittedWith is permittedTools with the organization's extensions already
// read, when the caller has them; nil reads them.
func (s *Access) PermittedWith(
	ctx context.Context,
	actor *permissionports.RequestActor,
	names []string,
	extensions *agentextension.Active,
) []string {
	// Never nil: the catalog reads nil as "everything", and a person who may
	// use nothing must be offered nothing.
	permitted := make([]string, 0, len(names))
	if s.permissions == nil || actor == nil {
		return permitted
	}
	// An extension's tools exist only while the organization has it on and
	// set up. Offering one it turned off would fail the call; naming one it
	// never bought would read as a feature it has.
	if touchesExtensions(names) {
		if extensions == nil {
			active := s.ActiveExtensions(ctx, actor)
			extensions = &active
		}
		names = onlyOffered(names, *extensions)
	}

	verdicts := s.toolVerdicts(ctx, actor, names)
	for _, name := range names {
		// A self-scoped tool touches only the person's own records, which any
		// signed-in person may arrange; it needs no grant, and nobody but a
		// person has records of that kind.
		if agenttoolports.IsSelfScoped(s.ToolNamed(name)) {
			if actor.PrincipalType == permissionports.PrincipalTypeUser {
				permitted = append(permitted, name)
			}
			continue
		}

		resource, operation, ok := s.ToolGate(name)
		if !ok {
			continue
		}
		if verdicts[resource.String()+":"+string(operation)] {
			permitted = append(permitted, name)
		}
	}

	return permitted
}

// toolGateCheck is one distinct permission the tools ask for.
type toolGateCheck struct {
	key       string
	tool      string
	resource  permission.Resource
	operation permission.Operation
}

// toolVerdicts asks each distinct permission the named tools need once, side
// by side: an agent with a wide toolbox holds dozens, and one after another
// they were as many round trips before the model was called. A check that
// fails withholds the tools behind it.
func (s *Access) toolVerdicts(
	ctx context.Context,
	actor *permissionports.RequestActor,
	names []string,
) map[string]bool {
	checks := make([]toolGateCheck, 0, len(names))
	seen := make(map[string]struct{}, len(names))
	for _, name := range names {
		if agenttoolports.IsSelfScoped(s.ToolNamed(name)) {
			continue
		}
		resource, operation, ok := s.ToolGate(name)
		if !ok {
			continue
		}
		key := resource.String() + ":" + string(operation)
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		checks = append(checks, toolGateCheck{
			key:       key,
			tool:      name,
			resource:  resource,
			operation: operation,
		})
	}

	allowed := make([]bool, len(checks))
	var group errgroup.Group
	group.SetLimit(maxConcurrentToolChecks)
	for i := range checks {
		check := checks[i]
		group.Go(func() error {
			result, err := s.permissions.Check(ctx, &permissionports.PermissionCheckRequest{
				PrincipalType:  actor.PrincipalType,
				PrincipalID:    actor.PrincipalID,
				UserID:         actor.UserID,
				APIKeyID:       actor.APIKeyID,
				BusinessUnitID: actor.BusinessUnitID,
				OrganizationID: actor.OrganizationID,
				Resource:       check.resource.String(),
				Operation:      check.operation,
			})
			if err != nil {
				s.logger.Warn("could not check whether a tool may be offered; withholding it",
					zap.String("tool", check.tool), zap.Error(err))

				return nil
			}
			allowed[i] = result != nil && result.Allowed

			return nil
		})
	}
	_ = group.Wait()

	verdicts := make(map[string]bool, len(checks))
	for i, check := range checks {
		verdicts[check.key] = allowed[i]
	}

	return verdicts
}

// ToolNamed is the registered tool behind a name, read or write, or nil.
func (s *Access) ToolNamed(name string) any {
	if tool, ok := s.queryTools.Get(name); ok {
		return tool
	}
	if tool, ok := s.actionTools.Get(name); ok {
		return tool
	}

	return nil
}

// ToolGate is the permission a tool is used under.
func (s *Access) ToolGate(name string) (permission.Resource, permission.Operation, bool) {
	if tool, ok := s.queryTools.Get(name); ok {
		return tool.Policy().Resource, permission.OpRead, true
	}
	if tool, ok := s.actionTools.Get(name); ok {
		policy := tool.Policy()

		return policy.Resource, policy.Operation, true
	}

	return "", "", false
}

func (s *Access) SpecFor(name string) (kernelports.ToolSpec, bool) {
	if tool, ok := s.queryTools.Get(name); ok {
		return kernelports.ToolSpec{
			Name:        tool.Name(),
			Description: tool.Description(),
			Parameters:  tool.ParamSchema(),
		}, true
	}
	if tool, ok := s.actionTools.Get(name); ok {
		return kernelports.ToolSpec{
			Name:        tool.Name(),
			Description: tool.Description(),
			Parameters:  tool.ParamSchema(),
		}, true
	}

	return kernelports.ToolSpec{}, false
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

func toSpec(descriptor agenttoolports.AgentToolDescriptor) kernelports.ToolSpec {
	return kernelports.ToolSpec{
		Name:        descriptor.Name,
		Description: descriptor.Description,
		Parameters:  descriptor.Parameters,
	}
}

// withoutSelfScoped drops the tools that act on a person's own records. A run
// nobody is watching has no person whose records they would be.
func (s *Access) withoutSelfScoped(names []string) []string {
	kept := make([]string, 0, len(names))
	for _, name := range names {
		if !agenttoolports.IsSelfScoped(s.ToolNamed(name)) {
			kept = append(kept, name)
		}
	}

	return kept
}

func ConfiguredThreshold(cfg *aiconfig.Config) *int {
	if cfg == nil {
		return nil
	}
	threshold := cfg.GetToolDisclosureThreshold()

	return &threshold
}

func (s *Access) toolThreshold() int {
	if s.threshold == nil {
		return DisclosureThreshold
	}

	return *s.threshold
}

func (s *Access) NotePreselected(
	ctx context.Context,
	req *agentruntimeports.RunRequest,
	set *ToolSet,
) {
	if s.disclosures == nil || !set.Disclosed {
		return
	}

	s.disclosures.Preselected(ctx, &agenttoolports.ToolDisclosure{
		Attribution: TurnAttribution(req),
		Disclosed:   true,
		Held:        set.held,
		Ranked:      slices.Clone(set.Preselected),
	})
}

func (s *Access) NoteFound(
	ctx context.Context,
	req *agentruntimeports.RunRequest,
	set *ToolSet,
	arguments map[string]any,
	answer *FindAnswer,
) {
	if s.disclosures == nil {
		return
	}
	need, _ := arguments["need"].(string)

	s.disclosures.Found(ctx, &agenttoolports.ToolDisclosure{
		Attribution: TurnAttribution(req),
		Disclosed:   set.Disclosed,
		Held:        set.held,
		Need:        need,
		Ranked:      slices.Clone(answer.Found),
	})
}
