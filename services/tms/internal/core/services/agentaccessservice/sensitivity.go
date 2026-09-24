package agentaccessservice

import (
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
)

// HeldTool is a tool an agent holds, with the grant it needs of the person
// using it and where its work can go. A tool that needs no grant names no
// resource.
type HeldTool struct {
	Name      string
	Resource  permission.Resource
	Operation permission.Operation
	Egress    []agent.EgressClass
}

// SensitiveToolRule reports whether a held tool makes an agent open to
// everyone worth a second look.
type SensitiveToolRule func(tool HeldTool) bool

// AnySensitive is sensitive when any of the rules says so.
func AnySensitive(rules ...SensitiveToolRule) SensitiveToolRule {
	return func(tool HeldTool) bool {
		for _, rule := range rules {
			if rule != nil && rule(tool) {
				return true
			}
		}

		return false
	}
}

// RestrictedResourceRule marks a tool whose resource is restricted or
// confidential by default.
func RestrictedResourceRule(registry *permission.Registry) SensitiveToolRule {
	return func(tool HeldTool) bool {
		if registry == nil || tool.Resource == "" {
			return false
		}

		definition, ok := registry.Get(tool.Resource.String())
		if !ok {
			definition, ok = registry.Get(registry.GetEffectiveResource(tool.Resource.String()))
		}
		if !ok || definition == nil {
			return false
		}

		return definition.DefaultSensitivity.Level() >= permission.SensitivityRestricted.Level()
	}
}

// LeavesOrganizationRule marks a tool whose work can leave the organization.
func LeavesOrganizationRule() SensitiveToolRule {
	return func(tool HeldTool) bool {
		return slices.ContainsFunc(tool.Egress, agent.EgressClass.Leaves)
	}
}

// DefaultSensitiveRule is the rule the audience suggestion reads. Another
// reason a tool is sensitive is ORed in here.
func DefaultSensitiveRule(registry *permission.Registry) SensitiveToolRule {
	return AnySensitive(RestrictedResourceRule(registry), LeavesOrganizationRule())
}

// SensitiveTools names the held tools the rule marks, in the order held.
func SensitiveTools(held []HeldTool, rule SensitiveToolRule) []string {
	names := make([]string, 0, len(held))
	if rule == nil {
		return names
	}

	for _, tool := range held {
		if rule(tool) {
			names = append(names, tool.Name)
		}
	}

	return names
}
