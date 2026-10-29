package readspec

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/filtercatalog"
)

// List is what a person writes for a list tool that reads through
// GraphQL: its name and the words a model reads. Everything else, from the
// operation to every parameter, is generated from the schema and the
// operation the screen sends (graphqlreads_gen.go).
type List struct {
	Name         string
	EntityPlural string
	Summary      string
	SearchTerms  []string
	QueryNote    string
	// notes add what the schema does not say to a derived parameter's
	// description, keyed by the parameter's name.
	Notes map[string]string
	// references name, for each id field, the resource its ids belong to
	// and the tool that supplies one. An id field with none is offered only
	// as its has<Thing> parameter: a model has no honest way to get an id no
	// tool lists.
	References map[string]Reference
	// rank orders a page that matched a free-text query when the call chose
	// no order, as the screen's search cannot (closest name first).
	Rank func(query string, rows []map[string]any) []map[string]any
	// shape applies to each row what the screen's resolver does not, given
	// the fields withheld from it and the call's clock.
	Shape func(row map[string]any, withheld map[string]struct{}, clk filtercatalog.Clock)
	// derived are the fields shape works out for every row (a worker's
	// dispatchable and why not), which fields may name beside the columns:
	// the model sees them on every row, and naming one was refused, twice in
	// a row, as not a column.
	Derived []string
	// defaults are parameter values a call that omits them reads with (the
	// active courses, not the retired ones), named in their notes.
	Defaults map[string]any
	// defaultsAt are defaults that depend on the day (time off from today),
	// applied together only when a call gives none of them.
	DefaultsAt func(clk filtercatalog.Clock) map[string]any
	// unscoped are defaults a call that names none of its references reads
	// with (the open safety events across the organization, where one
	// worker's are read whole).
	Unscoped map[string]any
	// inputs are input fields every call sends, as the screen sets them for
	// its own rendering (each time-off request's worker).
	Inputs map[string]any
	// alongside are the reads a list makes beside its own: for records of
	// another kind its next call needs, or one keyed by the call that says
	// what its rows mean (the tolerance a drift is judged by).
	Alongside []Alongside
	// combine applies to a page what the records read alongside it say
	// about each row (a drift within the tolerance the overview carries).
	Combine func(items []map[string]any, beside map[string]any)
	// instead are the reads a list makes in place of its own when a call
	// names their own parameters (the sync state of documents named by id).
	Instead []Alongside
	// page is how many rows a call reads unless it asks, and the most it may
	// ask for, where the list's rows are larger than a table row (a capture
	// stack with every document it was split into). Zero is the lists' own.
	Page [2]int
	// names are the names the tool takes derived parameters by, keyed by
	// the derived name (integrationType is system).
	Names map[string]string
	// presents are the row fields shown under the name people call them by,
	// keyed by the field's own name (a tractor's code is its unitNumber).
	Presents map[string]string
	// instants are the Timestamp fields a call bounds to the minute rather
	// than the day (when an EDI message was created).
	Instants []string
	// unoffered are the record's fields no one narrows the list by (the
	// template an EDI message was rendered from, the path it was dropped
	// at), left out of the parameters so the ones people use stay readable.
	Unoffered []string
	// matches are the row fields, by dotted path, an unpaged list whose root
	// takes no search argument matches a free-text query against in the rows
	// it read (program.code), as its repository tool did; queryNote then
	// describes that query.
	Matches []string
	// searches are record fields a free-text query matches beyond the ones
	// the record's search configuration names: a bespoke filter's search
	// (an accounting sync record's numbers), which no configuration
	// describes. A query is refused, as a filter on them is, when the caller
	// may not be shown one of the fields it matches.
	Searches []string
	Resource permission.Resource
	// reads is what the tool's records carry from outside the organization,
	// declared as every read tool declares it (readSpec).
	Reads querytool.ReadSpec
	// outside are the fields whose text someone outside the organization
	// wrote (a driver's own description); a record that shows one is marked
	// as read from outside, as reads Marked declares.
	Outside []string
	// amounts shows each amount the record holds in minor units as the
	// decimal a person reads, and filters it by that decimal (graphql_money.go).
	Amounts bool
	Marks   func(row map[string]any) (agentrecord.RecordRef, bool)

	Conditions []Condition
}

type Condition struct {
	Name        string
	Description string
	When        []domaintypes.FieldFilter
	Unless      []domaintypes.FieldFilter
}

func (c *Condition) Compile(yes bool) (filters, groups []map[string]any) {
	if yes {
		return querytool.FieldFilterMaps(c.When), nil
	}

	return nil, []map[string]any{
		{querytool.FilterGroupFilters: querytool.FieldFilterMaps(c.Unless)},
	}
}

// Reference is where a list's id parameter points: the resource, what the
// id is, and the tool a model gets one from. An id of several kinds (a sync
// record's document) names the kinds instead of a resource, and a list of
// ids names the most one call takes.
type Reference struct {
	Resource permission.Resource
	Kinds    []permission.RecordKind
	What     string
	Source   string
	Most     int
}

// schema is the parameter an id or a list of ids becomes.
func (ref *Reference) Schema(list bool) map[string]any {
	description := agenttoolschema.IDDescription(ref.What, ref.Source)
	if ref.Most == 0 {
		ref.Most = querytool.UnpagedMost
	}
	switch {
	case list && len(ref.Kinds) > 0:
		return agenttoolschema.KindIDs(description, ref.Most, ref.Kinds...)
	case list:
		return agenttoolschema.RecordIDs(ref.Resource, description, ref.Most)
	case len(ref.Kinds) > 0:
		return agenttoolschema.KindID(description, ref.Kinds...)
	}

	return agenttoolschema.RecordID(ref.Resource, ref.What, ref.Source)
}
