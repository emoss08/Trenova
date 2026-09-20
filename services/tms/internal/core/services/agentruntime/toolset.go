package agentruntime

import (
	"fmt"
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
}

func (s *Service) newToolSet(
	definition *agentdefinition.Definition,
	input string,
) *toolSet {
	configured := s.configuredSpecs(definition)

	set := &toolSet{
		loaded:  make(map[string]struct{}, len(configured)),
		allowed: definition.ToolNames,
	}

	if len(configured) <= disclosureThreshold || s.catalog == nil {
		set.specs = configured
		for _, spec := range configured {
			set.loaded[spec.Name] = struct{}{}
		}

		return set
	}

	set.disclosed = true
	for _, descriptor := range s.catalog.Rank(definition.ToolNames, input, preselectedTools) {
		set.add(toSpec(descriptor))
	}
	set.specs = append(set.specs, findToolsSpec())

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
		return fmt.Sprintf(
			"Nothing further matched %q, and everything that did is already loaded. "+
				"Use what you have, or tell the person this system does not track it.",
			need,
		)
	}

	return fmt.Sprintf("These tools are now callable:\n%s", b.String())
}

func (s *Service) configuredSpecs(
	definition *agentdefinition.Definition,
) []serviceports.ToolSpec {
	specs := make([]serviceports.ToolSpec, 0, len(definition.ToolNames))

	for _, name := range definition.ToolNames {
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
	if index := strings.IndexByte(text, '.'); index > 0 {
		return text[:index+1]
	}

	return text
}
