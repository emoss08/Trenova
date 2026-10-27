package scriptports

import (
	"context"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/domain/analysis"
	"github.com/emoss08/trenova/internal/modules/agent/domain/codemode"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type Status string

const (
	Succeeded = Status("Succeeded")
	Errored   = Status("Errored")
	TimedOut  = Status("TimedOut")
	Stopped   = Status("Stopped")
	Failed    = Status("Failed")
)

// DatasetColumn is one column of a dataset as the script reads it: by its
// name, with values of its kind.
type DatasetColumn struct {
	Name     string             `json:"name"`
	Kind     analysis.ValueKind `json:"kind"`
	Currency string             `json:"currency,omitempty"`
}

// Dataset is a table the caller resolved and may read: a report run's rows or
// an attached file's. Rows hold numbers, text, flags and nothing, with times
// as ISO 8601 text the SDK reads back as dates. A text file that is no table
// is Text instead, and has no columns or rows.
type Dataset struct {
	Name     string
	Source   analysis.DatasetSource
	SourceID pulid.ID
	Label    string
	Columns  []DatasetColumn
	Rows     [][]any
	Text     *string
}

type Request struct {
	TenantInfo pagination.TenantInfo
	Title      string
	Purpose    string
	Code       string
	// Earlier is the code of the script this one continues, run quietly
	// before it.
	Earlier  string
	Inputs   map[string]any
	Datasets []Dataset
	Timezone string
	Broker   codemode.Broker
	StepKey  string
}

type Outcome struct {
	Status          Status
	Output          string
	OutputTruncated bool
	Result          sonic.NoCopyRawMessage
	Error           *codemode.ScriptError
	Limit           codemode.Limit
	Failure         string
	Limits          codemode.Limits
	DurationMs      int64
	// Shown is what the script showed the person, in the order it showed it,
	// and NotShown how many figures, tables and charts it showed past what a
	// run may keep.
	Shown    analysis.Display
	NotShown int
}

type Runner interface {
	Run(ctx context.Context, req *Request) (*Outcome, error)
}
