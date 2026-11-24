package agentcapability

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

var ErrUnknownBacking = errors.New("an entity names a reader no tool provides")

type Registry struct {
	ordered []Capability
	byID    map[string]int
	byName  map[string][]int
}

func Build(facts []Facts) (*Registry, error) {
	present := make(map[string]struct{}, len(facts))
	for idx := range facts {
		present[facts[idx].Name] = struct{}{}
	}
	var missing []string
	for _, backing := range Backings() {
		if _, ok := present[backing]; !ok {
			missing = append(missing, backing)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("%w: %s", ErrUnknownBacking, strings.Join(missing, ", "))
	}

	registry := &Registry{
		ordered: make([]Capability, 0, len(facts)+len(entities)*2),
		byID:    make(map[string]int, len(facts)+len(entities)*2),
		byName:  make(map[string][]int, len(facts)),
	}
	for idx := range facts {
		if IsBacking(facts[idx].Name) {
			continue
		}
		registry.add(Classify(&facts[idx]))
	}
	for idx := range entities {
		for _, capability := range EntityReads(&entities[idx]) {
			registry.add(capability)
		}
	}

	return registry, nil
}

func (r *Registry) add(capability Capability) {
	if _, taken := r.byID[capability.ID]; taken {
		return
	}
	r.byID[capability.ID] = len(r.ordered)
	r.byName[capability.Name] = append(r.byName[capability.Name], len(r.ordered))
	r.ordered = append(r.ordered, capability)
}

func (r *Registry) All() []Capability {
	return slices.Clone(r.ordered)
}

func (r *Registry) Get(id string) (Capability, bool) {
	idx, ok := r.byID[id]
	if !ok {
		return Capability{}, false
	}

	return r.ordered[idx], true
}

func (r *Registry) Tools() []string {
	names := make([]string, 0, len(r.byName))
	for name := range r.byName {
		names = append(names, name)
	}
	slices.Sort(names)

	return names
}

func (r *Registry) ForTool(name string) []Capability {
	indexes := r.byName[name]
	out := make([]Capability, 0, len(indexes))
	for _, idx := range indexes {
		out = append(out, r.ordered[idx])
	}

	return out
}

func (r *Registry) Areas() []string {
	seen := make(map[string]struct{}, 32)
	out := make([]string, 0, 32)
	for idx := range r.ordered {
		area := r.ordered[idx].Area
		if _, ok := seen[area]; ok {
			continue
		}
		seen[area] = struct{}{}
		out = append(out, area)
	}
	slices.Sort(out)

	return out
}

func CapabilityIDForTool(name string) string {
	if entity, verb, ok := EntityBacking(name); ok {
		if verb == VerbInclude {
			return EntityCapabilityID(entity.Name, VerbGet)
		}

		return EntityCapabilityID(entity.Name, verb)
	}

	return name
}
