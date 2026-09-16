package agentdefinitionservice

import (
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
)

// validateToolSelection is the enforcement behind the narrowing rule. The domain
// checks the shape of a tool list; this checks its membership, because only the
// live registry knows what a tool name actually resolves to.
//
// Three things are refused: a tool that does not exist, a tool whose resource
// falls outside the template's bound, and any tool at all on a read-only
// template. Together they mean an organization can only ever hand an agent a
// subset of what its template already permits.
func validateToolSelection(
	definition *agentdefinition.Definition,
	registry serviceports.AgentToolRegistry,
	multiErr *errortypes.MultiError,
) {
	if !definition.Kind.IsValid() || len(definition.ToolNames) == 0 {
		return
	}

	if !definition.Kind.MutatingAllowed() {
		// The domain already reports this; repeating the membership walk would
		// produce a second, noisier error for the same cause.
		return
	}

	allowed := make(map[permission.Resource]struct{})
	for _, resource := range definition.Kind.AllowedResources() {
		allowed[resource] = struct{}{}
	}

	for idx, name := range definition.ToolNames {
		field := fmt.Sprintf("toolNames[%d]", idx)

		tool, ok := registry.Get(name)
		if !ok {
			multiErr.Add(
				field,
				errortypes.ErrInvalid,
				fmt.Sprintf("%q is not a tool this system provides", name),
			)
			continue
		}

		if _, permitted := allowed[tool.PermissionResource()]; !permitted {
			multiErr.Add(
				field,
				errortypes.ErrInvalid,
				fmt.Sprintf(
					"A %s cannot be given %q, which acts on %s",
					definition.Kind.Label(),
					name,
					tool.PermissionResource().String(),
				),
			)
		}
	}
}

// AvailableTools lists the tools an organization may choose for a template, so
// the configuration UI offers exactly the permitted set rather than the whole
// registry.
func AvailableTools(
	kind agentdefinition.Kind,
	registry serviceports.AgentToolRegistry,
) []serviceports.AgentToolDescriptor {
	if !kind.IsValid() || !kind.MutatingAllowed() {
		return []serviceports.AgentToolDescriptor{}
	}

	allowed := make(map[permission.Resource]struct{})
	for _, resource := range kind.AllowedResources() {
		allowed[resource] = struct{}{}
	}

	tools := registry.All()
	descriptors := make([]serviceports.AgentToolDescriptor, 0, len(tools))
	for _, tool := range tools {
		if _, permitted := allowed[tool.PermissionResource()]; !permitted {
			continue
		}
		descriptors = append(descriptors, serviceports.AgentToolDescriptor{
			Name:         tool.Name(),
			Description:  tool.Description(),
			Parameters:   tool.ParamSchema(),
			AutonomyTier: tool.DefaultAutonomyTier(),
		})
	}

	return descriptors
}
