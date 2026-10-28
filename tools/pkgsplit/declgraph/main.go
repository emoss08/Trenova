package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func recvType(e ast.Expr) string {
	for {
		switch t := e.(type) {
		case *ast.StarExpr:
			e = t.X
		case *ast.IndexExpr:
			e = t.X
		case *ast.IndexListExpr:
			e = t.X
		case *ast.Ident:
			return t.Name
		default:
			return ""
		}
	}
}

type node struct {
	name  string
	file  string
	lines int
	body  []ast.Node
}

func main() {
	dir := os.Args[1]
	fset := token.NewFileSet()
	entries, _ := os.ReadDir(dir)
	nodes := map[string]*node{}
	get := func(n, f string) *node {
		if nodes[n] == nil {
			nodes[n] = &node{name: n, file: f}
		}
		return nodes[n]
	}
	type pending struct {
		owner string
		n     ast.Node
		file  string
		lines int
	}
	var all []pending
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, parser.SkipObjectResolution)
		if err != nil {
			panic(err)
		}
		for _, d := range f.Decls {
			lines := fset.Position(d.End()).Line - fset.Position(d.Pos()).Line + 1
			switch d := d.(type) {
			case *ast.FuncDecl:
				owner := d.Name.Name
				if d.Recv != nil && len(d.Recv.List) > 0 {
					owner = recvType(d.Recv.List[0].Type)
				}
				all = append(all, pending{owner, d, name, lines})
			case *ast.GenDecl:
				if d.Tok == token.IMPORT {
					continue
				}
				for _, s := range d.Specs {
					sl := fset.Position(s.End()).Line - fset.Position(s.Pos()).Line + 1
					switch s := s.(type) {
					case *ast.TypeSpec:
						all = append(all, pending{s.Name.Name, s, name, sl})
					case *ast.ValueSpec:
						for _, n := range s.Names {
							if n.Name != "_" {
								all = append(all, pending{n.Name, s, name, sl})
							}
						}
					}
				}
			}
		}
	}
	for _, p := range all {
		if fd, ok := p.n.(*ast.FuncDecl); ok && fd.Recv != nil {
			continue
		}
		get(p.owner, p.file)
	}
	for _, p := range all {
		nd := nodes[p.owner]
		if nd == nil {
			nd = get(p.owner, p.file)
		}
		nd.lines += p.lines
		nd.body = append(nd.body, p.n)
	}
	names := make([]string, 0, len(nodes))
	for n := range nodes {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		nd := nodes[n]
		fmt.Printf("N\t%s\t%s\t%d\n", n, nd.file, nd.lines)
	}
	for _, n := range names {
		nd := nodes[n]
		refs := map[string]bool{}
		for _, b := range nd.body {
			ast.Inspect(b, func(x ast.Node) bool {
				switch x := x.(type) {
				case *ast.SelectorExpr:
					ast.Inspect(x.X, func(m ast.Node) bool {
						if id, ok := m.(*ast.Ident); ok && nodes[id.Name] != nil && id.Name != n {
							refs[id.Name] = true
						}
						return true
					})
					return false
				case *ast.Ident:
					if nodes[x.Name] != nil && x.Name != n {
						refs[x.Name] = true
					}
				}
				return true
			})
		}
		rs := make([]string, 0, len(refs))
		for r := range refs {
			rs = append(rs, r)
		}
		sort.Strings(rs)
		for _, r := range rs {
			fmt.Printf("E\t%s\t%s\n", n, r)
		}
	}
}
