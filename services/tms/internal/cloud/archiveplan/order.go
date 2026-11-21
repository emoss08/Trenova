package archiveplan

import (
	"fmt"
	"slices"
	"strings"
)

type edge struct {
	fk     *ForeignKey
	child  string
	parent string
}

func (b *builder) order() error {
	edges := make([]edge, 0, len(b.plan.ForeignKeys))
	for _, fk := range b.plan.ForeignKeys {
		if fk.SelfReferencing() {
			continue
		}
		edges = append(edges, edge{fk: fk, child: fk.Child, parent: fk.Parent})
	}
	names := make([]string, 0, len(b.plan.Tables))
	for _, t := range b.plan.Tables {
		names = append(names, t.Name)
	}
	slices.Sort(names)

	kept := make([]edge, 0, len(edges))
	for _, component := range components(names, edges) {
		if len(component) < 2 {
			continue
		}
		members := make(map[string]struct{}, len(component))
		for _, name := range component {
			members[name] = struct{}{}
		}
		for _, e := range edges {
			_, childIn := members[e.child]
			_, parentIn := members[e.parent]
			if childIn && parentIn && b.breakable(e.fk) {
				b.deferEdge(e.fk)
			}
		}
	}
	for _, e := range edges {
		child := b.plan.byName[e.child]
		if deferredEdge(child, e.fk) {
			continue
		}
		kept = append(kept, e)
	}

	ordered, err := topological(names, kept)
	if err != nil {
		return err
	}
	tables := make([]*PlannedTable, 0, len(ordered))
	for i, name := range ordered {
		t := b.plan.byName[name]
		t.Position = i + 1
		tables = append(tables, t)
	}
	b.plan.Tables = tables
	return nil
}

func (b *builder) breakable(fk *ForeignKey) bool {
	own := fk.ownColumns()
	if len(own) == 0 {
		return false
	}
	table := b.in.Catalog.Tables[fk.Child]
	for _, c := range own {
		column, ok := table.Column(c)
		if !ok || !column.Nullable {
			return false
		}
	}
	return true
}

func (b *builder) deferEdge(fk *ForeignKey) {
	child := b.plan.byName[fk.Child]
	for _, c := range fk.ownColumns() {
		if !slices.Contains(child.Deferred, c) {
			child.Deferred = append(child.Deferred, c)
		}
	}
	slices.Sort(child.Deferred)
}

func deferredEdge(child *PlannedTable, fk *ForeignKey) bool {
	own := fk.ownColumns()
	if len(own) == 0 {
		return false
	}
	for _, c := range own {
		if slices.Contains(child.Deferred, c) {
			return true
		}
	}
	return false
}

func topological(names []string, edges []edge) ([]string, error) {
	parents := make(map[string]map[string]struct{}, len(names))
	children := make(map[string][]string, len(names))
	for _, e := range edges {
		set := parents[e.child]
		if set == nil {
			set = make(map[string]struct{})
			parents[e.child] = set
		}
		if _, seen := set[e.parent]; seen {
			continue
		}
		set[e.parent] = struct{}{}
		children[e.parent] = append(children[e.parent], e.child)
	}
	pending := make(map[string]int, len(names))
	ready := make([]string, 0, len(names))
	for _, name := range names {
		pending[name] = len(parents[name])
		if pending[name] == 0 {
			ready = append(ready, name)
		}
	}
	ordered := make([]string, 0, len(names))
	for len(ready) > 0 {
		slices.Sort(ready)
		name := ready[0]
		ready = ready[1:]
		ordered = append(ordered, name)
		for _, child := range children[name] {
			pending[child]--
			if pending[child] == 0 {
				ready = append(ready, child)
			}
		}
	}
	if len(ordered) == len(names) {
		return ordered, nil
	}
	stuck := make([]string, 0)
	for _, name := range names {
		if pending[name] > 0 {
			stuck = append(stuck, name)
		}
	}
	return nil, fmt.Errorf(
		"these tables refer to each other through columns that cannot be left empty while they load: %s",
		strings.Join(stuck, ", "),
	)
}

func components(names []string, edges []edge) [][]string {
	next := make(map[string][]string, len(names))
	for _, e := range edges {
		next[e.child] = append(next[e.child], e.parent)
	}
	t := &tarjan{
		next:  next,
		index: make(map[string]int, len(names)),
		low:   make(map[string]int, len(names)),
		on:    make(map[string]bool, len(names)),
	}
	for _, name := range names {
		if _, seen := t.index[name]; !seen {
			t.visit(name)
		}
	}
	return t.out
}

type tarjan struct {
	next    map[string][]string
	index   map[string]int
	low     map[string]int
	on      map[string]bool
	stack   []string
	counter int
	out     [][]string
}

func (t *tarjan) visit(v string) {
	t.index[v] = t.counter
	t.low[v] = t.counter
	t.counter++
	t.stack = append(t.stack, v)
	t.on[v] = true
	for _, w := range t.next[v] {
		if _, seen := t.index[w]; !seen {
			t.visit(w)
			t.low[v] = min(t.low[v], t.low[w])
		} else if t.on[w] {
			t.low[v] = min(t.low[v], t.index[w])
		}
	}
	if t.low[v] != t.index[v] {
		return
	}
	component := make([]string, 0)
	for {
		w := t.stack[len(t.stack)-1]
		t.stack = t.stack[:len(t.stack)-1]
		t.on[w] = false
		component = append(component, w)
		if w == v {
			break
		}
	}
	slices.Sort(component)
	t.out = append(t.out, component)
}
