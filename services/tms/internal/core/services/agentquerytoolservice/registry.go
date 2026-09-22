// Package agentquerytoolservice holds the read-only tools the assistant may run
// without asking a human first.
//
// Everything here is read-only by construction rather than by convention: these
// types implement AgentQueryTool, which has no Execute method. A tool that needs
// to change something implements AgentTool instead and goes through the proposal
// and approval path.
package agentquerytoolservice

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

type RegistryParams struct {
	fx.In

	Tools []serviceports.AgentQueryTool `group:"agent_query_tools"`
}

type registry struct {
	byName  map[string]serviceports.AgentQueryTool
	ordered []serviceports.AgentQueryTool
}

// legacyToolNames maps a name an agent may still have saved to the tool that
// now carries it.
//
// A tool name is persisted — in agent_definitions.tool_names and as a key in
// tool_tiers — so renaming one is a data change. The migration rewrites both
// columns, but a definition written by an API client that hard-coded the old
// name would otherwise resolve to nothing, and the failure mode is silent: the
// agent simply stops being able to look drivers up. Resolving the old name
// costs one map entry.
//
// These are deliberately absent from Descriptors, so the model is never offered
// a name we no longer want it to learn.
var legacyToolNames = map[string]string{
	"search_workers": "search_worker",
}

func NewRegistry(p RegistryParams) serviceports.AgentQueryToolRegistry {
	byName := make(map[string]serviceports.AgentQueryTool, len(p.Tools))
	ordered := make([]serviceports.AgentQueryTool, 0, len(p.Tools))

	for _, tool := range p.Tools {
		if _, exists := byName[tool.Name()]; exists {
			continue
		}
		byName[tool.Name()] = tool
		ordered = append(ordered, tool)
	}

	for legacy, current := range legacyToolNames {
		if _, taken := byName[legacy]; taken {
			continue
		}
		if tool, ok := byName[current]; ok {
			byName[legacy] = tool
		}
	}

	return &registry{byName: byName, ordered: ordered}
}

func (r *registry) Get(name string) (serviceports.AgentQueryTool, bool) {
	tool, ok := r.byName[name]
	return tool, ok
}

func (r *registry) All() []serviceports.AgentQueryTool { return r.ordered }

func (r *registry) Descriptors() []serviceports.AgentToolDescriptor {
	descriptors := make([]serviceports.AgentToolDescriptor, 0, len(r.ordered))

	for _, tool := range r.ordered {
		descriptors = append(descriptors, serviceports.AgentToolDescriptor{
			Name:        tool.Name(),
			Description: tool.Description(),
			Parameters:  tool.ParamSchema(),
		})
	}

	return descriptors
}

var (
	ErrMissingActor   = errors.New("query tool requires an actor")
	ErrTenantMismatch = errors.New("query tool parameters do not match the actor tenant")
)

// guardQuery repeats the tenant check the write tools make. The tenant comes
// from the actor rather than from the model either way, but asserting it here
// means a future caller that builds params by hand cannot skip it silently.
func guardQuery(params serviceports.QueryToolParams) error {
	if params.Actor == nil {
		return ErrMissingActor
	}

	if params.Actor.OrganizationID != params.OrganizationID ||
		params.Actor.BusinessUnitID != params.BusinessUnitID {
		return ErrTenantMismatch
	}

	return nil
}

func requireString(params map[string]any, key string) (string, error) {
	raw, ok := params[key]
	if !ok {
		return "", fmt.Errorf("missing required parameter %q", key)
	}

	value, ok := raw.(string)
	if !ok || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("parameter %q must be a non-empty string", key)
	}

	return strings.TrimSpace(value), nil
}

// optionalString reads a string the caller may leave out.
//
// Absent, null and the empty string all mean "not given", which is the same
// thing to every filter that takes one: a model that omits an argument and one
// that sends "" are asking for the same unfiltered result.
func optionalString(params map[string]any, key string) string {
	raw, ok := params[key]
	if !ok || raw == nil {
		return ""
	}

	value, ok := raw.(string)
	if !ok {
		return ""
	}

	return strings.TrimSpace(value)
}

// optionalStrings reads a list of strings the caller may leave out, accepting
// every shape a model reaches for: the array, a lone string, a comma-separated
// string, and a single-key object wrapping one of those. asList already knows
// them, so the coercion lives in one place rather than per tool.
func optionalStrings(params map[string]any, key string) []string {
	raw, ok := params[key]
	if !ok || raw == nil {
		return nil
	}

	list, ok := asList(raw).([]any)
	if !ok {
		return nil
	}

	values := make([]string, 0, len(list))
	for _, entry := range list {
		text, isText := entry.(string)
		if text = strings.TrimSpace(text); isText && text != "" {
			values = append(values, text)
		}
	}
	if len(values) == 0 {
		return nil
	}

	return values
}

func requirePulid(params map[string]any, key string) (pulid.ID, error) {
	value, err := requireString(params, key)
	if err != nil {
		return pulid.Nil, err
	}

	id, err := pulid.Parse(value)
	if err != nil {
		return pulid.Nil, fmt.Errorf("parameter %q is not a valid id: %w", key, err)
	}

	return id, nil
}

// optionalBool reads a flag the caller may leave out. Anything that is not a
// literal true is false, including a model that helpfully sends the string
// "true" — a flag that silently widens a compliance query on a type confusion
// is worse than one that stays narrow.
func optionalBool(params map[string]any, key string) bool {
	value, _ := params[key].(bool)

	return value
}

func optionalInt(params map[string]any, key string, fallback int) int {
	raw, ok := params[key]
	if !ok {
		return fallback
	}

	switch value := raw.(type) {
	case float64:
		return int(value)
	case int:
		return value
	case int64:
		return int(value)
	case string:
		// "30" is 30. Models send numbers as strings often enough that
		// refusing the string form turned a valid request into "needs a
		// whole number of days", which the model then retried identically.
		parsed, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil {
			return fallback
		}

		return parsed
	default:
		return fallback
	}
}

// optionalObject reads a nested object the caller may leave out. A model that
// sends null rather than omitting the key means the same thing, and a tool that
// treats the two differently fails on a distinction the model cannot see.
func optionalObject(params map[string]any, key string) map[string]any {
	value, _ := params[key].(map[string]any)

	return value
}

// decodeParam reads a structured argument into a typed value. A model that
// sends a list of objects gets each one checked against the struct rather
// than picked apart by hand at every tool.
func decodeParam(params map[string]any, key string, out any) error {
	raw, ok := params[key]
	if !ok {
		return fmt.Errorf("missing required parameter %q", key)
	}

	if err := jsonutils.Convert(raw, out); err != nil {
		return fmt.Errorf("parameter %q: %w", key, err)
	}

	return nil
}
