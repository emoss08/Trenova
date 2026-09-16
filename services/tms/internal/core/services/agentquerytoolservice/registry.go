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
	"strings"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
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
	default:
		return fallback
	}
}
