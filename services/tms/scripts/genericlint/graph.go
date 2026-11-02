package main

import (
	"go/types"
)

type hit struct {
	instance string
	origin   string
	methods  int
}

type scan struct {
	self *types.TypeName
	hits map[string]hit
	next []*types.TypeName
	seen map[*types.TypeName]bool
}

func newScan(self *types.TypeName) *scan {
	return &scan{self: self, seen: make(map[*types.TypeName]bool)}
}

func (s *scan) follow(obj *types.TypeName) {
	if obj == s.self || s.seen[obj] {
		return
	}
	s.seen[obj] = true
	s.next = append(s.next, obj)
}

type node struct {
	index   int
	low     int
	onStack bool
	done    bool
	direct  map[string]hit
	next    []*node
	reach   map[string]hit
}

type graph struct {
	maxMethods int
	linted     map[string]bool
	nodes      map[*types.TypeName]*node
	stack      []*node
	counter    int
	methods    map[*types.TypeName]int
}

func newGraph(linted map[string]bool, maxMethods int) *graph {
	return &graph{
		maxMethods: maxMethods,
		linted:     linted,
		nodes:      make(map[*types.TypeName]*node),
		methods:    make(map[*types.TypeName]int),
	}
}

func (g *graph) walkThrough(obj *types.TypeName) bool {
	pkg := obj.Pkg()
	if pkg == nil {
		return false
	}

	return !g.linted[pkg.Path()] || !obj.Exported()
}

func (g *graph) scanType(t types.Type, s *scan) {
	switch t := t.(type) {
	case *types.Alias:
		g.scanType(types.Unalias(t), s)
	case *types.Named:
		g.scanNamed(t, s)
	case *types.Pointer:
		g.scanType(t.Elem(), s)
	case *types.Slice:
		g.scanType(t.Elem(), s)
	case *types.Array:
		g.scanType(t.Elem(), s)
	case *types.Chan:
		g.scanType(t.Elem(), s)
	case *types.Map:
		g.scanType(t.Key(), s)
		g.scanType(t.Elem(), s)
	case *types.Signature:
		g.scanTypeParams(t.TypeParams(), s)
		g.scanType(t.Params(), s)
		g.scanType(t.Results(), s)
	case *types.Tuple:
		for i := range t.Len() {
			g.scanType(t.At(i).Type(), s)
		}
	case *types.Struct:
		for i := range t.NumFields() {
			g.scanType(t.Field(i).Type(), s)
		}
	case *types.Interface:
		for i := range t.NumExplicitMethods() {
			g.scanType(t.ExplicitMethod(i).Type(), s)
		}
		for i := range t.NumEmbeddeds() {
			g.scanType(t.EmbeddedType(i), s)
		}
	case *types.Union:
		for i := range t.Len() {
			g.scanType(t.Term(i).Type(), s)
		}
	}
}

func (g *graph) scanTypeParams(list *types.TypeParamList, s *scan) {
	for i := range list.Len() {
		g.scanType(list.At(i).Constraint(), s)
	}
}

func (g *graph) scanNamed(t *types.Named, s *scan) {
	origin := t.Origin()
	obj := origin.Obj()
	if obj.Pkg() == nil || obj == s.self {
		return
	}
	if args := t.TypeArgs(); args.Len() > 0 {
		g.record(t, origin, s)
		for i := range args.Len() {
			g.scanType(args.At(i), s)
		}
	}
	if g.walkThrough(obj) {
		s.follow(obj)
	}
}

func (g *graph) record(inst, origin *types.Named, s *scan) {
	if _, ok := origin.Underlying().(*types.Interface); ok {
		return
	}
	count := g.methodCount(origin)
	if count <= g.maxMethods {
		return
	}
	key := qualifiedName(origin.Obj())
	if _, ok := s.hits[key]; ok {
		return
	}
	if s.hits == nil {
		s.hits = make(map[string]hit)
	}
	s.hits[key] = hit{
		instance: types.TypeString(inst, packageName),
		origin:   key,
		methods:  count,
	}
}

func (g *graph) methodCount(origin *types.Named) int {
	obj := origin.Obj()
	if count, ok := g.methods[obj]; ok {
		return count
	}
	count := types.NewMethodSet(types.NewPointer(origin)).Len()
	g.methods[obj] = count

	return count
}

func (g *graph) declScan(obj *types.TypeName) *scan {
	s := newScan(obj)
	named, ok := obj.Type().(*types.Named)
	if !ok {
		g.scanType(obj.Type(), s)
		return s
	}
	g.scanTypeParams(named.TypeParams(), s)
	g.scanType(named.Underlying(), s)
	for i := range named.NumMethods() {
		g.scanType(named.Method(i).Signature(), s)
	}

	return s
}

func (g *graph) reach(obj *types.TypeName) map[string]hit {
	n := g.nodes[obj]
	if n == nil {
		n = g.connect(obj)
	}

	return n.reach
}

func (g *graph) connect(obj *types.TypeName) *node {
	n := &node{index: g.counter, low: g.counter, onStack: true}
	g.counter++
	g.nodes[obj] = n
	g.stack = append(g.stack, n)

	s := g.declScan(obj)
	n.direct = s.hits
	for _, nextObj := range s.next {
		next := g.nodes[nextObj]
		switch {
		case next == nil:
			next = g.connect(nextObj)
			n.low = min(n.low, next.low)
		case next.onStack:
			n.low = min(n.low, next.index)
		}
		n.next = append(n.next, next)
	}
	if n.low == n.index {
		g.closeComponent(n)
	}

	return n
}

func (g *graph) closeComponent(root *node) {
	var members []*node
	for {
		top := g.stack[len(g.stack)-1]
		g.stack = g.stack[:len(g.stack)-1]
		top.onStack = false
		members = append(members, top)
		if top == root {
			break
		}
	}

	var reach map[string]hit
	for _, m := range members {
		reach = mergeHits(reach, m.direct)
		for _, next := range m.next {
			if next.done {
				reach = mergeHits(reach, next.reach)
			}
		}
	}
	for _, m := range members {
		m.reach = reach
		m.done = true
		m.direct = nil
		m.next = nil
	}
}

func mergeHits(dst, src map[string]hit) map[string]hit {
	for key, h := range src {
		if _, ok := dst[key]; ok {
			continue
		}
		if dst == nil {
			dst = make(map[string]hit, len(src))
		}
		dst[key] = h
	}

	return dst
}

func qualifiedName(obj *types.TypeName) string {
	return obj.Pkg().Path() + "." + obj.Name()
}

func displayName(obj *types.TypeName) string {
	return obj.Pkg().Name() + "." + obj.Name()
}

func packageName(pkg *types.Package) string {
	return pkg.Name()
}
