package main

import (
	"fmt"
	"go/token"
	"go/types"
	"maps"
	"path/filepath"
	"slices"

	"golang.org/x/tools/go/packages"
)

type finding struct {
	pkgPath  string
	ident    string
	member   string
	position string
	via      string
	hit      hit
}

func allowKey(pkgPath, ident, origin string) string {
	return pkgPath + " " + ident + " " + origin
}

func (f finding) key() string {
	return allowKey(f.pkgPath, f.ident, f.hit.origin)
}

func (f finding) String() string {
	subject := f.ident
	if f.member != "" {
		subject += " " + f.member
	}
	text := fmt.Sprintf(
		"%s: %s names %s, whose %d methods every importer compiles again",
		f.position,
		subject,
		f.hit.instance,
		f.hit.methods,
	)
	if f.via != "" {
		text += " (through " + f.via + ")"
	}

	return text + " [" + f.key() + "]"
}

type collector struct {
	g        *graph
	pkgPath  string
	fset     *token.FileSet
	dir      string
	seen     map[string]bool
	findings []finding
}

func check(pkgs []*packages.Package, maxMethods int, dir string) []finding {
	linted := make(map[string]bool, len(pkgs))
	for _, pkg := range pkgs {
		linted[pkg.PkgPath] = true
	}
	g := newGraph(linted, maxMethods)

	var findings []finding
	for _, pkg := range pkgs {
		c := &collector{
			g:       g,
			pkgPath: pkg.PkgPath,
			fset:    pkg.Fset,
			dir:     dir,
			seen:    make(map[string]bool),
		}
		scope := pkg.Types.Scope()
		for _, name := range scope.Names() {
			if obj := scope.Lookup(name); obj.Exported() {
				c.object(obj)
			}
		}
		findings = append(findings, c.findings...)
	}

	return findings
}

func (c *collector) object(obj types.Object) {
	switch obj := obj.(type) {
	case *types.TypeName:
		c.typeName(obj)
	case *types.Func:
		c.member(obj.Name(), "", obj.Pos(), nil, obj.Signature())
	case *types.Var:
		c.member(obj.Name(), "", obj.Pos(), nil, obj.Type())
	}
}

func (c *collector) typeName(obj *types.TypeName) {
	name := obj.Name()
	named, ok := obj.Type().(*types.Named)
	if obj.IsAlias() || !ok {
		c.member(name, "", obj.Pos(), nil, obj.Type())
		return
	}

	tparams := named.TypeParams()
	for i := range tparams.Len() {
		tp := tparams.At(i)
		c.member(name, "type parameter "+tp.Obj().Name(), obj.Pos(), obj, tp.Constraint())
	}
	switch u := named.Underlying().(type) {
	case *types.Struct:
		for i := range u.NumFields() {
			f := u.Field(i)
			c.member(name, "field "+f.Name(), f.Pos(), obj, f.Type())
		}
	case *types.Interface:
		for i := range u.NumExplicitMethods() {
			m := u.ExplicitMethod(i)
			c.member(name, "method "+m.Name(), m.Pos(), obj, m.Type())
		}
		for i := range u.NumEmbeddeds() {
			c.member(name, "", obj.Pos(), obj, u.EmbeddedType(i))
		}
	default:
		c.member(name, "", obj.Pos(), obj, u)
	}
	for i := range named.NumMethods() {
		m := named.Method(i)
		c.member(name, "method "+m.Name(), m.Pos(), obj, m.Signature())
	}
}

func (c *collector) member(
	ident, label string,
	pos token.Pos,
	self *types.TypeName,
	t types.Type,
) {
	s := newScan(self)
	c.g.scanType(t, s)
	c.add(ident, label, pos, "", s.hits)
	for _, next := range s.next {
		c.add(ident, label, pos, displayName(next), c.g.reach(next))
	}
}

func (c *collector) add(ident, label string, pos token.Pos, via string, hits map[string]hit) {
	for _, origin := range slices.Sorted(maps.Keys(hits)) {
		key := allowKey(c.pkgPath, ident, origin)
		if c.seen[key] {
			continue
		}
		c.seen[key] = true
		c.findings = append(c.findings, finding{
			pkgPath:  c.pkgPath,
			ident:    ident,
			member:   label,
			position: c.position(pos),
			via:      via,
			hit:      hits[origin],
		})
	}
}

func (c *collector) position(pos token.Pos) string {
	p := c.fset.Position(pos)
	name := p.Filename
	if rel, err := filepath.Rel(c.dir, name); err == nil {
		name = rel
	}

	return fmt.Sprintf("%s:%d", filepath.ToSlash(name), p.Line)
}
