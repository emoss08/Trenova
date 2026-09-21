package agentruntime

import (
	"context"
	"fmt"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.uber.org/zap"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
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

// toolSet is the live set of tools a turn may call. It starts from the agent's
// configuration and grows when the model asks for more.
type toolSet struct {
	specs     []serviceports.ToolSpec
	loaded    map[string]struct{}
	allowed   []string
	disclosed bool
	// usable answers whether the person may use a tool at all, for naming
	// what exists beyond the agent's configuration.
	usable func(name string) bool
}

func (s *Service) newToolSet(
	ctx context.Context,
	definition *agentdefinition.Definition,
	actor *serviceports.RequestActor,
	input string,
	unattended bool,
) *toolSet {
	// The set a turn may call is the agent's configuration narrowed to what
	// the person driving it may do. A tool they cannot use was still shown,
	// ranked and offered, and denied only when called, which taught the
	// model that the system refuses rather than that this person lacks the
	// right; the denial also named the resource they lacked.
	allowed := s.permittedTools(ctx, actor, definition.ToolNames)
	configured := s.configuredSpecs(allowed)

	set := &toolSet{
		loaded:  make(map[string]struct{}, len(configured)),
		allowed: allowed,
		usable: func(name string) bool {
			return len(s.permittedTools(ctx, actor, []string{name})) == 1
		},
	}

	if len(configured) <= disclosureThreshold || s.catalog == nil {
		set.specs = configured
		for _, spec := range configured {
			set.loaded[spec.Name] = struct{}{}
		}
		if !unattended {
			set.add(askUserSpec())
		}

		return set
	}

	set.disclosed = true
	for _, descriptor := range s.catalog.Rank(allowed, input, preselectedTools) {
		set.add(toSpec(descriptor))
	}
	set.specs = append(set.specs, findToolsSpec())
	// A background run has nobody to ask. Offering the question tool anyway
	// let an event-driven agent end its run on a question no one would see,
	// recorded as complete.
	if !unattended {
		set.add(askUserSpec())
	}

	return set
}

func (t *toolSet) add(spec serviceports.ToolSpec) bool {
	if _, ok := t.loaded[spec.Name]; ok {
		return false
	}

	t.loaded[spec.Name] = struct{}{}
	t.specs = append(t.specs, spec)

	return true
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

	found := s.catalog.Find(set.allowed, need, foundToolsLimit)

	var b strings.Builder
	added := 0
	for _, descriptor := range found {
		if !set.add(toSpec(descriptor)) {
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
	for _, descriptor := range s.catalog.Find(nil, need, foundToolsLimit) {
		if _, held := set.loaded[descriptor.Name]; held {
			continue
		}
		if slices.Contains(set.allowed, descriptor.Name) {
			continue
		}
		// A tool the person may not use is not worth naming: an
		// administrator adding it to the agent would change nothing for them.
		if set.usable != nil && !set.usable(descriptor.Name) {
			continue
		}
		elsewhere = append(elsewhere, descriptor)
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

func (s *Service) configuredSpecs(names []string) []serviceports.ToolSpec {
	specs := make([]serviceports.ToolSpec, 0, len(names))

	for _, name := range names {
		if tool, ok := s.queryTools.Get(name); ok {
			specs = append(specs, serviceports.ToolSpec{
				Name:        tool.Name(),
				Description: tool.Description(),
				Parameters:  tool.ParamSchema(),
			})
			continue
		}

		tool, ok := s.actionTools.Get(name)
		if !ok {
			continue
		}

		specs = append(specs, serviceports.ToolSpec{
			Name:        tool.Name(),
			Description: tool.Description(),
			Parameters:  tool.ParamSchema(),
		})
	}

	return specs
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
