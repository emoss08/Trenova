package agentquerytoolservice

import (
	"fmt"
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
	// Note is set only when nothing matched, because a full result speaks for
	// itself and an extra sentence in front of it is noise in a context window.
	Note string `json:"note,omitempty"`
}

// searchCriteria accumulates the filters a tool actually applied, in the words
// a person would use, so they can be read back in the outcome.
type searchCriteria struct {
	entityPlural string
	terms        []string
}

func newSearchCriteria(entityPlural string) *searchCriteria {
	return &searchCriteria{entityPlural: entityPlural, terms: make([]string, 0, 4)}
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
	}
	if count == 0 {
		outcome.Note = c.emptyNote()
	}

	return outcome
}
