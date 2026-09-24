package services

import (
	"context"

	"github.com/emoss08/trenova/shared/pulid"
)

// QueryToolParams carries one lookup. The tenant identifiers come from the
// authenticated actor rather than from the model, so a tool cannot be argued
// into reading another organization's data.
type QueryToolParams struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	Actor          *RequestActor
	// Timezone is the organization's, and it is what "today", "the next 30
	// days" and a bare YYYY-MM-DD mean in a call. Empty is UTC.
	Timezone string
	Params   map[string]any
	// AgentDefinitionID is the agent the call is made for, which decides
	// what it may recall of what was kept for one agent alone.
	AgentDefinitionID pulid.ID
}

// AgentQueryTool reads data and returns it.
//
// It is a separate interface from AgentTool rather than an extra method on it,
// and the distinction is structural rather than stylistic: there is no Execute
// here and no way to add one without changing this file, so a query tool cannot
// mutate anything. That is what makes it safe for the assistant to run one
// without asking a human first, while anything that writes still goes through
// AgentTool's proposal and approval path.
type AgentQueryTool interface {
	Name() string
	Description() string
	ParamSchema() map[string]any
	Policy() ToolPolicy
	// Query returns data for the model to reason over. The result is serialized
	// and handed back as untrusted content, since records carry customer-authored
	// text.
	Query(ctx context.Context, params *QueryToolParams) (any, error)
}

// AgentQueryToolRegistry resolves query tools by name.
type AgentQueryToolRegistry interface {
	Get(name string) (AgentQueryTool, bool)
	All() []AgentQueryTool
	Descriptors() []AgentToolDescriptor
}
