package agentdefinition

import (
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentcapability"
	"github.com/emoss08/trenova/pkg/errortypes"
)

const (
	MaxPinnedTools  = 12
	MaxScopeEntries = 512
	AllAreas        = "*"
)

type Scope struct {
	Areas            []string `json:"areas,omitempty"`
	Capabilities     []string `json:"capabilities,omitempty"`
	DenyAreas        []string `json:"denyAreas,omitempty"`
	DenyCapabilities []string `json:"denyCapabilities,omitempty"`
}

func (s *Scope) Everything() bool {
	return len(s.Areas) == 0 && len(s.Capabilities) == 0
}

func (s *Scope) Allows(tool string) bool {
	if tool == "" {
		return false
	}
	if IsCoreTool(tool) {
		return true
	}
	id := agentcapability.CapabilityIDForTool(tool)
	area := toolArea(tool)
	if slices.Contains(s.DenyCapabilities, tool) || slices.Contains(s.DenyCapabilities, id) ||
		slices.Contains(s.DenyAreas, area) || slices.Contains(s.DenyAreas, AllAreas) {
		return false
	}
	if s.Everything() {
		return true
	}

	return slices.Contains(s.Areas, area) ||
		slices.Contains(s.Capabilities, tool) ||
		slices.Contains(s.Capabilities, id)
}

func (s *Scope) AllowsEntity(entity string, verb agentcapability.Verb) bool {
	described, ok := agentcapability.EntityNamed(entity)
	if !ok {
		return false
	}
	id := agentcapability.EntityCapabilityID(entity, verb)
	if slices.Contains(s.DenyCapabilities, id) || slices.Contains(s.DenyAreas, described.Area) ||
		slices.Contains(s.DenyAreas, AllAreas) {
		return false
	}
	if s.Everything() {
		return true
	}

	return slices.Contains(s.Areas, described.Area) || slices.Contains(s.Capabilities, id)
}

func (s *Scope) Deny(tool string) {
	id := agentcapability.CapabilityIDForTool(tool)
	s.Capabilities = slices.DeleteFunc(slices.Clone(s.Capabilities), func(held string) bool {
		return held == tool || held == id
	})
	if !slices.Contains(s.DenyCapabilities, id) {
		s.DenyCapabilities = append(slices.Clone(s.DenyCapabilities), id)
	}
}

func (s *Scope) Allow(tool string) {
	id := agentcapability.CapabilityIDForTool(tool)
	s.DenyCapabilities = slices.DeleteFunc(slices.Clone(s.DenyCapabilities), func(off string) bool {
		return off == tool || off == id
	})
	if !s.Allows(tool) {
		s.Capabilities = append(slices.Clone(s.Capabilities), id)
	}
}

func (s *Scope) Denied() []string {
	return slices.Clone(s.DenyCapabilities)
}

func (s *Scope) validate(multiErr *errortypes.MultiError) {
	lists := []struct {
		field  string
		values []string
	}{
		{"scope.areas", s.Areas},
		{"scope.capabilities", s.Capabilities},
		{"scope.denyAreas", s.DenyAreas},
		{"scope.denyCapabilities", s.DenyCapabilities},
	}
	for _, list := range lists {
		if len(list.values) > MaxScopeEntries {
			multiErr.Add(list.field, errortypes.ErrInvalid,
				fmt.Sprintf("A scope list cannot hold more than %d entries", MaxScopeEntries))
		}
		seen := make(map[string]struct{}, len(list.values))
		for idx, value := range list.values {
			field := fmt.Sprintf("%s[%d]", list.field, idx)
			if strings.TrimSpace(value) == "" {
				multiErr.Add(field, errortypes.ErrInvalid, "A scope entry cannot be empty")
				continue
			}
			if _, duplicate := seen[value]; duplicate {
				multiErr.Add(field, errortypes.ErrDuplicate, "Entry is listed more than once")
			}
			seen[value] = struct{}{}
		}
	}
	for idx, area := range s.Areas {
		if slices.Contains(s.DenyAreas, area) {
			multiErr.Add(fmt.Sprintf("scope.areas[%d]", idx), errortypes.ErrInvalid,
				"An area cannot be both allowed and denied")
		}
	}
	for idx, capability := range s.Capabilities {
		if slices.Contains(s.DenyCapabilities, capability) {
			multiErr.Add(fmt.Sprintf("scope.capabilities[%d]", idx), errortypes.ErrInvalid,
				"A capability cannot be both allowed and denied")
		}
	}
}

func toolArea(tool string) string {
	if entity, _, ok := agentcapability.EntityBacking(tool); ok {
		return entity.Area
	}

	return agentcapability.AreaOf(tool)
}

func (d *Definition) Holds(catalog []string) []string {
	held := make([]string, 0, len(catalog)+len(coreTools))
	held = append(held, coreTools[:]...)
	for _, name := range catalog {
		if !slices.Contains(held, name) && d.AllowsTool(name) {
			held = append(held, name)
		}
	}

	return held
}

func (d *Definition) IsUniversal() bool {
	return d.Scope.Everything() && len(d.Scope.DenyAreas) == 0 && len(d.Scope.DenyCapabilities) == 0
}

func (d *Definition) ScopeEntries() []string {
	entries := make([]string, 0, len(d.Scope.Areas)+len(d.Scope.Capabilities)+
		len(d.Scope.DenyAreas)+len(d.Scope.DenyCapabilities))
	for _, area := range d.Scope.Areas {
		entries = append(entries, "area:"+area)
	}
	entries = append(entries, d.Scope.Capabilities...)
	for _, area := range d.Scope.DenyAreas {
		entries = append(entries, "-area:"+area)
	}
	for _, capability := range d.Scope.DenyCapabilities {
		entries = append(entries, "-"+capability)
	}

	return entries
}

func (d *Definition) StarterTools() []string {
	if d.Scope.Everything() {
		return nil
	}
	tools := make([]string, 0, len(d.Scope.Capabilities)+len(d.PinnedTools))
	tools = append(tools, d.PinnedTools...)
	for _, capability := range d.Scope.Capabilities {
		if !slices.Contains(tools, capability) {
			tools = append(tools, capability)
		}
	}

	return tools
}
