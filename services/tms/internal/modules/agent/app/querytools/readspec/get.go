package readspec

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/typeutils"
)

/*
The expand step.

A list answers "which ones", and then someone asks about one of them. These
return the whole record rather than a projection, because that is the question:
a list already decided what was worth summarising, and a model that has narrowed
to a single row needs the fields the summary left out.

Each names the list tool that yields its id, in the summary and again in
idSource on the parameter itself. Without that, a model holding a customer's
name and no id has nowhere to go but guessing one.
*/
type Get struct {
	Name        string
	Entity      string
	Summary     string
	SearchTerms []string
	Resource    permission.Resource
	// scope narrows the policy to the caller's own records (the person's own
	// home page).
	Scope     agenttooltrust.ToolScope
	Kinds     []permission.RecordKind
	ParamName string
	IDSource  string
	Reads     agenttooltrust.ExternalRead
	Source    agent.TaintSource
	Rationale string
	Fetch     func(ctx context.Context, id pulid.ID, tenant pagination.TenantInfo) (any, error)
	// A get that reads through GraphQL takes its operation's other
	// variables as parameters, named, typed and noted as a list's are.
	Names      map[string]string
	References map[string]Reference
	Notes      map[string]string
	// alongside are the reads it makes beside the record, keyed by the
	// call (a sync record's tries by its id).
	Alongside []Alongside
	// shape applies to a record read through GraphQL what the card's
	// resolver does not, given the fields withheld from it and the call
	// (what a sync record's status means; a NonCompliant worker cannot be
	// assigned; a driver's clocks in minutes; a customer that takes no
	// routine notices).
	Shape func(record map[string]any, withheld map[string]struct{}, clk filtercatalog.Clock)
	// absent is what a get whose root returns nothing answers, where nothing
	// is itself the finding (no ELD has reported a driver's hours) rather
	// than a missing record.
	Absent func(id pulid.ID) map[string]any
	// keyResource is the resource of the record a get is keyed by when that
	// is not its own (a driver's earnings, read by the worker's id).
	KeyResource permission.Resource
	// amounts shows each amount the record holds in minor units as its
	// decimal (graphql_money.go), and minorUnits are fields that hold minor
	// units under a name the schema does not mark with Minor.
	Amounts    bool
	MinorUnits []string
	// batch makes the get's parameter a list of at most this many ids,
	// each read on its own, and shapeEach applies to each record of a batch
	// what one read alone does not (fewer of an invoice's lines).
	Batch     int
	ShapeEach func(record map[string]any)
	// outside are the fields whose text someone outside the organization
	// wrote (a driver's own description); a record that shows one is marked
	// as read from outside, as reads Marked declares.
	Outside []string
	// pages marks a get whose operation reads one page of a list the record
	// holds, from an offset the call sends (a rate matrix's cells): the
	// result says whether more follow and the offset to ask for next.
	Pages *Pages
}

// Pages names the offset variable a get's operation pages a held list by,
// the list, and the field that counts all of it.
type Pages struct {
	Offset string
	Items  string
	Total  string
}

func (s Get) IDDescription() string {
	if s.IDSource == "" {
		return fmt.Sprintf("The %s's id.", s.Entity)
	}

	return fmt.Sprintf("The %s's id, %s.", s.Entity, s.IDSource)
}

// mark says whether the held list continues past the page the variables
// read, and where the next page starts.
func (p *Pages) Mark(record, variables map[string]any) {
	offset, _ := typeutils.IntOf(variables[p.Offset])
	items, _ := record[p.Items].([]any)
	total, counted := typeutils.IntOf(record[p.Total])
	more := counted && offset+len(items) < total
	record[p.Items+"HasMore"] = more
	if more {
		record["next"+strings.ToUpper(p.Offset[:1])+p.Offset[1:]] = offset + len(items)
	}
}
