package agentquerytoolservice

import (
	"fmt"
	"reflect"
	"strings"
)

// searchOutcome is what a search tool returns instead of a bare slice.
//
// A model handed `[]` cannot tell "nothing matched what you asked for" from
// "this system holds no such records", and it will guess. It guessed wrong in
// production: a driver search that matched nobody became "there are no driver
// records currently in the Trenova system", which is a confident claim about a
// customer's database made from a single empty result.
//
// Naming the terms back turns the same answer into something the model can act
// on — it can widen the search, or say plainly that nothing matched these terms.
type searchOutcome struct {
	Count       int      `json:"count"`
	SearchedFor []string `json:"searchedFor"`
	Items       any      `json:"items"`
	// Columns is the row projection's fields in the order it declares them.
	//
	// The Desk draws a list result as a table beside the conversation, and a
	// table needs its columns in an order somebody chose. JSON objects have
	// none: by the time the result reaches the pane it is a map, and a map
	// scatters "pro number, customer, status" into whatever order it hashes
	// to. So the order is taken here, where the rows are still a type.
	//
	// It is also the cheapest possible answer to "what can I ask about these",
	// which the model otherwise infers from whichever fields happened to be
	// populated in the first row.
	Columns []string `json:"columns,omitempty"`
	// Note is set only when nothing matched, because a full result speaks for
	// itself and an extra sentence in front of it is noise in a context window.
	Note string `json:"note,omitempty"`
}

// searchCriteria accumulates the filters a tool actually applied, in the words
// a person would use, so they can be read back in the outcome.
type searchCriteria struct {
	entityPlural string
	terms        []string
	// clock is the frame the search's dates were read in. It rides with the
	// criteria because every filter builder already receives them, and a
	// window built on the wrong day is a criterion nobody stated.
	clock clock
}

func newSearchCriteria(entityPlural string) *searchCriteria {
	return &searchCriteria{entityPlural: entityPlural, terms: make([]string, 0, 4)}
}

// at sets the frame the search reads dates in.
func (c *searchCriteria) at(clk clock) *searchCriteria {
	c.clock = clk

	return c
}

// text records a free-text term. An empty value records nothing, which is how
// "no filter" stays distinguishable from "filtered on the empty string".
func (c *searchCriteria) text(value string) {
	if strings.TrimSpace(value) == "" {
		return
	}

	c.terms = append(c.terms, fmt.Sprintf("text matching %q", value))
}

func (c *searchCriteria) field(label, value string) {
	if strings.TrimSpace(value) == "" {
		return
	}

	c.terms = append(c.terms, fmt.Sprintf("%s %s", label, value))
}

// describe renders the applied filters for the model, including the case where
// there were none — "every" is the honest description of an unfiltered list and
// stops the model reporting a short page as the whole population.
func (c *searchCriteria) describe() []string {
	if len(c.terms) == 0 {
		return []string{"no filters: the most recent " + c.entityPlural}
	}

	return c.terms
}

func (c *searchCriteria) emptyNote() string {
	if len(c.terms) == 0 {
		return fmt.Sprintf(
			"No %s are visible to you in this organization. "+
				"This is an unfiltered list, so the result is the whole set, not a near miss.",
			c.entityPlural,
		)
	}

	return fmt.Sprintf(
		"No %s matched %s. Other %s may exist; only these filters were applied.",
		c.entityPlural, strings.Join(c.terms, " and "), c.entityPlural,
	)
}

// result wraps what a repository returned. count is taken from the caller
// rather than derived by reflection so a tool stays in charge of what "how
// many" means for its own shape.
func (c *searchCriteria) result(items any, count int) searchOutcome {
	outcome := searchOutcome{
		Count:       count,
		SearchedFor: c.describe(),
		Items:       items,
		Columns:     columnsOf(items),
	}
	if count == 0 {
		outcome.Note = c.emptyNote()
	}

	return outcome
}

// columnsOf reads a row projection's JSON field names in declaration order.
//
// It works off the first row because every row in one result is the same
// projection. A result with no rows has no columns to report, which is
// correct: there is no table to draw.
func columnsOf(items any) []string {
	list := reflect.ValueOf(items)
	if list.Kind() != reflect.Slice || list.Len() == 0 {
		return nil
	}

	rowType := reflect.Indirect(reflect.ValueOf(list.Index(0).Interface())).Type()
	if rowType.Kind() != reflect.Struct {
		return nil
	}

	names := make([]string, 0, rowType.NumField())

	return appendColumns(names, rowType)
}

// appendColumns walks one struct, descending into embedded structs so a row
// built from a shared base reports its inherited fields where they appear
// rather than as a nested object the grid cannot draw.
func appendColumns(names []string, rowType reflect.Type) []string {
	for i := range rowType.NumField() {
		field := rowType.Field(i)

		tag, ok := field.Tag.Lookup("json")
		name, _, _ := strings.Cut(tag, ",")
		if name == "-" {
			continue
		}

		// An embedded struct's fields are promoted onto the row, and that
		// holds whether the embedded type is exported or not — which is why
		// this runs before the exported check rather than after it.
		if field.Anonymous && name == "" {
			embedded := field.Type
			if embedded.Kind() == reflect.Ptr {
				embedded = embedded.Elem()
			}
			if embedded.Kind() == reflect.Struct {
				names = appendColumns(names, embedded)

				continue
			}
		}

		if !field.IsExported() {
			continue
		}
		if !ok || name == "" {
			name = field.Name
		}
		names = append(names, name)
	}

	return names
}
