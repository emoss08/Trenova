package environment

import (
	"sort"

	"github.com/emoss08/trenova/pkg/formulatypes"
)

type Sources map[string]formulatypes.ValueSource

func (s Sources) Mark(paths []string, source formulatypes.ValueSource) {
	for _, path := range paths {
		s[path] = source
	}
}

func ForSchema(definition *formulatypes.Definition) Sources {
	sources := make(Sources, len(definition.FieldSources))
	for path, source := range definition.FieldSources {
		switch {
		case source != nil && source.Provided:
			sources[path] = formulatypes.ValueSourceProvided
		case source != nil && source.Computed:
			sources[path] = formulatypes.ValueSourceComputed
		default:
			sources[path] = formulatypes.ValueSourceField
		}
	}
	return sources
}

func ReceiptVariables(env map[string]any, sources Sources) []formulatypes.VariableProvenance {
	flat := make([]formulatypes.VariableProvenance, 0, len(env))
	flatten(env, "", sources, &flat)
	sort.Slice(flat, func(i, j int) bool { return flat[i].Name < flat[j].Name })
	return flat
}

func flatten(
	env map[string]any,
	prefix string,
	sources Sources,
	out *[]formulatypes.VariableProvenance,
) {
	for key, value := range env {
		path := key
		if prefix != "" {
			path = prefix + "." + key
		}
		if nested, ok := value.(map[string]any); ok {
			flatten(nested, path, sources, out)
			continue
		}
		source := sources[path]
		if source == "" {
			source = sourceForPrefix(path, sources)
		}
		*out = append(
			*out,
			formulatypes.VariableProvenance{Name: path, Value: value, Source: source},
		)
	}
}

func sourceForPrefix(path string, sources Sources) formulatypes.ValueSource {
	for i := len(path) - 1; i > 0; i-- {
		if path[i] == '.' {
			if source, ok := sources[path[:i]]; ok {
				return source
			}
		}
	}
	return formulatypes.ValueSourceSample
}
