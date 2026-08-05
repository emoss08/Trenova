package mutate

import (
	"fmt"
	"sort"
	"strings"
)

// schemaSite is one form placed in a file, carrying the runtime id that selects
// it.
type schemaSite struct {
	form *schemaForm
	id   int
	kids []*schemaSite
}

// renderSchemata rewrites a file so that every site becomes a runtime switch.
//
// Sites nest rather than sit side by side: the range of `a + b` in `a + b < c`
// lies inside the range of the comparison. Flat splicing would corrupt the outer
// rewrite, so the sites are first arranged into a containment forest and then
// rendered recursively — each form asks for its own sub-ranges back, already
// rewritten.
func renderSchemata(source []byte, sites []*schemaSite) ([]byte, error) {
	roots, err := containmentForest(sites)
	if err != nil {
		return nil, err
	}

	r := renderer{source: source}
	rendered, err := r.span(schemaSpan{from: 0, to: len(source)}, roots, make([]bool, len(roots)))
	if err != nil {
		return nil, err
	}

	return []byte(rendered), nil
}

// containmentForest nests each site inside the innermost site that contains it.
//
// Two sites can share a byte range exactly: `if a > b` carries a boundary swap and
// both branch forcings, all three rewriting the same bytes. Equal ranges therefore
// nest rather than conflict, but the order is not free. A form that re-renders its
// whole range can host anything; one that re-renders only its operands, like the
// boundary helper, has nowhere to put a sibling that spans the operator too. So
// whole-range forms are placed outermost, and only then by id for determinism.
//
// Partial overlap cannot be rendered at all and is refused, so the caller falls
// back rather than emitting something that will not compile.
func containmentForest(sites []*schemaSite) ([]*schemaSite, error) {
	ordered := make([]*schemaSite, len(sites))
	copy(ordered, sites)
	sort.SliceStable(ordered, func(i, j int) bool {
		a, b := ordered[i].form, ordered[j].form
		if a.from != b.from {
			return a.from < b.from
		}
		if a.to != b.to {
			return a.to > b.to
		}
		if a.wholeRange() != b.wholeRange() {
			return a.wholeRange()
		}

		return ordered[i].id < ordered[j].id
	})

	var roots, stack []*schemaSite
	for _, site := range ordered {
		site.kids = nil

		for len(stack) > 0 {
			top := stack[len(stack)-1]
			if site.form.from >= top.form.from && site.form.to <= top.form.to {
				break
			}
			if site.form.from < top.form.to {
				return nil, fmt.Errorf(
					"mutation sites %d..%d and %d..%d overlap without nesting",
					top.form.from, top.form.to, site.form.from, site.form.to)
			}
			stack = stack[:len(stack)-1]
		}

		if len(stack) == 0 {
			roots = append(roots, site)
		} else {
			top := stack[len(stack)-1]
			top.kids = append(top.kids, site)
		}
		stack = append(stack, site)
	}

	return roots, nil
}

type renderer struct {
	source []byte
}

// span copies a byte range, substituting the rendered form of every child that
// falls inside it and marking that child as placed.
func (r renderer) span(sp schemaSpan, kids []*schemaSite, used []bool) (string, error) {
	if sp.from < 0 || sp.to > len(r.source) || sp.from > sp.to {
		return "", fmt.Errorf("mutation span %d..%d is outside the file", sp.from, sp.to)
	}

	var b strings.Builder
	cursor := sp.from

	for i, kid := range kids {
		if kid.form.from < sp.from || kid.form.to > sp.to {
			continue
		}
		if kid.form.from < cursor {
			return "", fmt.Errorf("mutation site %d..%d starts inside an already rendered site",
				kid.form.from, kid.form.to)
		}

		b.Write(r.source[cursor:kid.form.from])
		text, err := r.site(kid)
		if err != nil {
			return "", err
		}
		b.WriteString(text)

		cursor = kid.form.to
		used[i] = true
	}

	b.Write(r.source[cursor:sp.to])

	return b.String(), nil
}

// site renders one form, handing it each of its requested sub-ranges with the
// site's own children already substituted in.
func (r renderer) site(s *schemaSite) (string, error) {
	used := make([]bool, len(s.kids))
	parts := make([]string, 0, len(s.form.spans))

	for _, sp := range s.form.spans {
		part, err := r.span(sp, s.kids, used)
		if err != nil {
			return "", err
		}
		parts = append(parts, part)
	}

	// A child outside every requested span would be silently dropped, and a mutant
	// that quietly stops existing would be scored against a binary that never
	// contained it.
	for i, placed := range used {
		if !placed {
			return "", fmt.Errorf("mutation site %d..%d falls outside every span of its parent %d..%d",
				s.kids[i].form.from, s.kids[i].form.to, s.form.from, s.form.to)
		}
	}

	return s.form.render(s.id, parts), nil
}
