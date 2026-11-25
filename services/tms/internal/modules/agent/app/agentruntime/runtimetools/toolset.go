package runtimetools

import (
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/toolschema"
)

const SearchToolsName = "search_tools"

// searchToolsDescription is a constant rather than a literal in the spec below
// because the i18n extractor harvests any Description: field it finds
// (shared/cmd/i18n-extract). This text is addressed to a model.
const searchToolsDescription = "Search every capability Trenova has for what you need, in plain " +
	"words: the record, the action and anything it must match (\"unpaid invoices for Acme\", " +
	"\"assign a driver\"). Each result says whether it is loaded, available (call it by name " +
	"and it loads), not_permitted (the person lacks the permission named) or not_in_this_agent " +
	"(outside this agent's scope: offer a hand-off, do not work around it). Run this before " +
	"saying something cannot be done; on low confidence rephrase once with the entity, " +
	"action and fields, and only then report the gap, naming the searches tried."

// SearchToolsSpec is the one tool a turn that can reach beyond what it was
// sent always carries. It is answered by the runtime rather than a registry,
// because its effect is on the turn itself: it adds capabilities to the set the
// model may call next.
func SearchToolsSpec() kernelports.ToolSpec {
	return kernelports.ToolSpec{
		Name:        SearchToolsName,
		Description: searchToolsDescription,
		Searches:    true,
		Parameters: map[string]any{
			toolschema.KeyType: toolschema.TypeObject,
			toolschema.KeyProperties: map[string]any{
				"query": map[string]any{
					toolschema.KeyType:        toolschema.TypeString,
					toolschema.KeyDescription: "What you need, in the person's words: the record, the action and the fields.",
				},
			},
			toolschema.KeyRequired:             []string{"query"},
			toolschema.KeyAdditionalProperties: false,
		},
	}
}
