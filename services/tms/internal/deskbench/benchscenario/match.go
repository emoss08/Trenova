package benchscenario

import (
	"slices"
	"strings"
	"sync"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetools"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
)

const (
	alternativeSplit = "|"
)

func Alternatives(spec string) []string {
	parts := strings.Split(spec, alternativeSplit)
	out := make([]string, 0, len(parts))
	for idx := 0; idx < len(parts); idx++ {
		part := strings.TrimSpace(parts[idx])
		if strings.HasPrefix(part, "/") && !closesPattern(part) {
			for idx+1 < len(parts) && !closesPattern(part) {
				idx++
				part += alternativeSplit + parts[idx]
			}
			part = strings.TrimSpace(part)
		}
		if part != "" {
			out = append(out, part)
		}
	}

	return out
}

func closesPattern(part string) bool {
	return len(part) > 1 && strings.HasSuffix(part, "/")
}

var runtimeToolNames = sync.OnceValue(func() []string {
	policies := runtimetools.Policies()
	names := make([]string, 0, len(policies))
	for idx := range policies {
		names = append(names, policies[idx].Name)
	}

	return names
})

func Plumbing(name string) bool {
	return agentdefinition.IsCoreTool(name) || slices.Contains(runtimeToolNames(), name)
}

func Labelled(name string, labels []string) bool {
	return slices.ContainsFunc(labels, func(spec string) bool {
		return slices.Contains(Alternatives(spec), name)
	})
}
