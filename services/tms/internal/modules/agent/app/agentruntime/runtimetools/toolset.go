package runtimetools

import (
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/toolschema"
)

const (
	FindToolsName = "find_tools"
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

// FindToolsSpec is the one tool a disclosed turn always carries.
//
// It is answered by the runtime rather than a registry, because its effect is on
// the turn itself: it adds tools to the set the model may call next. A registry
// tool cannot do that — it returns data and the tool list stays as it was.
func FindToolsSpec() kernelports.ToolSpec {
	return kernelports.ToolSpec{
		Name:        FindToolsName,
		Description: findToolsDescription,
		Searches:    true,
		Parameters: map[string]any{
			toolschema.KeyType: toolschema.TypeObject,
			toolschema.KeyProperties: map[string]any{
				"need": map[string]any{
					toolschema.KeyType:        toolschema.TypeString,
					toolschema.KeyDescription: "What you are trying to do, in the operator's words.",
				},
			},
			toolschema.KeyRequired:             []string{"need"},
			toolschema.KeyAdditionalProperties: false,
		},
	}
}
