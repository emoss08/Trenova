package main

import (
	"fmt"
	"strings"

	"github.com/99designs/gqlgen/codegen/config"
)

// generatedModels is gqlgen's model package; a type bound there is a shape
// for the wire, not the entity the resolver filters.
const generatedModels = "gqlmodel"

// goType is the Go type a GraphQL object binds to, as gqlgen's binder sees
// it: its package path and name.
type goType struct {
	Path string
	Name string
}

// goTypes reads, from gqlgen's own configuration, the Go type each bound
// tool's node binds to, so the tool can ask that entity for the filter
// configuration the server compiles its filters with. An unpaged list
// filters by its root field's arguments alone and a get reads one record by
// its id, so neither needs one: a card (JournalEntryCard) is no entity.
func goTypes(configPath string, bindings []*binding) (map[string]goType, error) {
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("load %s: %w", configPath, err)
	}
	if err = cfg.LoadSchema(); err != nil {
		return nil, fmt.Errorf("load schema: %w", err)
	}
	if err = cfg.Init(); err != nil {
		return nil, fmt.Errorf("bind schema: %w", err)
	}

	out := make(map[string]goType, len(bindings))
	var problems []string
	for _, b := range bindings {
		if b.unpaged() || b.Verb == verbGet {
			continue
		}
		full := b.Entity
		if full == "" {
			model, ok := cfg.Models[b.Node]
			if !ok || len(model.Model) == 0 {
				problems = append(problems, b.Tool+": "+b.Node+" binds to no Go type; "+
					"name the entity in reads.yml")
				continue
			}
			full = model.Model[0]
		}
		idx := strings.LastIndex(full, ".")
		if idx < 0 {
			problems = append(problems, b.Tool+": "+b.Node+" binds to "+full)
			continue
		}
		if b.Entity == "" && strings.HasSuffix(full[:idx], "/"+generatedModels) {
			problems = append(problems, b.Tool+": "+b.Node+" binds to the generated "+full+
				", which has no filter configuration; name the entity in reads.yml")
			continue
		}
		out[b.Node] = goType{Path: full[:idx], Name: full[idx+1:]}
	}
	if len(problems) > 0 {
		return nil, fmt.Errorf("%s", strings.Join(problems, "\n"))
	}

	return out, nil
}
