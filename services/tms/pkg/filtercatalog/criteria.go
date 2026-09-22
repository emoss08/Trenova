package filtercatalog

import (
	"fmt"
	"strings"
)

// Criteria accumulates the filters a compilation actually applied, in the words
// a person would use, so they can be read back.
//
// A caller handed a result and no terms cannot tell "nothing matched what you
// asked for" from "this system holds no such records", and it will guess. It
// guessed wrong in production: a driver search that matched nobody became
// "there are no driver records currently in the Trenova system", which is a
// confident claim about a customer's database made from a single empty result.
//
// Naming the terms back turns the same answer into something to act on — widen
// the search, or say plainly that nothing matched these terms.
type Criteria struct {
	Entity string
	// Clock is the frame the criteria's dates were read in. It rides here
	// because every compilation step already receives the criteria, and a
	// window built on the wrong day is a criterion nobody stated.
	Clock Clock

	terms []string
}

func NewCriteria(entity string) *Criteria {
	return &Criteria{Entity: entity, terms: make([]string, 0, 4)}
}

// At sets the frame the criteria read dates in.
func (c *Criteria) At(clk Clock) *Criteria {
	c.Clock = clk

	return c
}

// Text records a free-text term. An empty value records nothing, which is how
// "no filter" stays distinguishable from "filtered on the empty string".
func (c *Criteria) Text(value string) {
	if strings.TrimSpace(value) == "" {
		return
	}

	c.terms = append(c.terms, fmt.Sprintf("text matching %q", value))
}

// Field records one applied narrowing.
func (c *Criteria) Field(label, value string) {
	if strings.TrimSpace(value) == "" {
		return
	}

	c.terms = append(c.terms, fmt.Sprintf("%s %s", label, value))
}

// Terms is what was applied, with no filler for the unfiltered case.
func (c *Criteria) Terms() []string {
	return c.terms
}

// Describe renders the applied filters, including the case where there were
// none — "every" is the honest description of an unfiltered list and stops a
// caller reporting a short page as the whole population.
func (c *Criteria) Describe() []string {
	if len(c.terms) == 0 {
		return []string{"no filters: the most recent " + c.Entity}
	}

	return c.terms
}

// EmptyNote is what to say when nothing matched.
func (c *Criteria) EmptyNote() string {
	if len(c.terms) == 0 {
		return fmt.Sprintf(
			"No %s are visible to you in this organization. "+
				"This is an unfiltered list, so the result is the whole set, not a near miss.",
			c.Entity,
		)
	}

	return fmt.Sprintf(
		"No %s matched %s. Other %s may exist; only these filters were applied.",
		c.Entity, strings.Join(c.terms, " and "), c.Entity,
	)
}
