package agenttoolports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/codemode"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/shared/pulid"
)

// QueryToolParams carries one lookup. The tenant identifiers come from the
// authenticated actor rather than from the model, so a tool cannot be argued
// into reading another organization's data.
type QueryToolParams struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	Actor          *permissionports.RequestActor
	// Timezone is the organization's, and it is what "today", "the next 30
	// days" and a bare YYYY-MM-DD mean in a call. Empty is UTC.
	Timezone string
	Params   map[string]any
	// AgentDefinitionID is the agent the call is made for, which decides
	// what it may recall of what was kept for one agent alone.
	AgentDefinitionID pulid.ID
	DataAccessCeiling permission.FieldSensitivity
	ThreadID          pulid.ID
	RunID             pulid.ID
	// Broker is set only for the tool that runs a script: every read the
	// script makes goes through it, as the run's actor.
	Broker codemode.Broker
	// StepKey is the step the call is claimed under, for the tool that runs a
	// script to name its sandbox.
	StepKey string
	// CallID is the model's id for the call, which a script's answer gives
	// back as the script a later call may continue.
	CallID string
	// Continues is the arguments of the script of the turn this one
	// continues, set only once the runtime has found it may be continued.
	Continues map[string]any
	// IncludeTotal asks a list for how many records it holds in all: set for
	// a model's first page and for a person's table when it counts.
	IncludeTotal bool
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

// PageCutter is a page of a list that can end early and still say where the
// list goes on. The runtime cuts a page too long for the model to read whole
// to the rows that fit, and the next page then starts at the first row it
// left out. Cutting the encoded result instead kept the cursor the tool made
// for the whole page, so the rows between the cut and the page's end were
// never read by anyone: fifty workers asked for, thirteen shown, and the next
// page began at the fifty-first.
type PageCutter interface {
	// PageRows is how many rows the page holds.
	PageRows() int
	// CutPage is the page ending after its first rows rows, saying there is
	// more and where the next page starts. It is false when the page cannot
	// say where a shorter page would go on, such as rows ranked or merged
	// out of the list's own order.
	CutPage(rows int) (any, bool)
}

// ListFieldsParam names the columns of each row of a list a model reads. The
// runtime narrows the model's copy of the rows to them; the list itself, the
// table beside the conversation and its later pages keep every column.
const ListFieldsParam = "fields"
