package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type cut struct {
	start, end int
}

type srcFile struct {
	path string
	src  []byte
	file *ast.File
	cuts []cut
}

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

func main() {
	dir := flag.String("dir", "", "package dir")
	out := flag.String("out", "", "target file name in dir")
	flag.Parse()
	want := map[string]bool{}
	for _, a := range flag.Args() {
		want[strings.TrimSuffix(a, ".*")] = true
	}
	fset := token.NewFileSet()
	entries, _ := os.ReadDir(*dir)
	var files []*srcFile
	target := filepath.Join(*dir, *out)
	var imports []string
	seenImp := map[string]bool{}
	var pkgName string
	for _, e := range entries {
		n := e.Name()
		if e.IsDir() || !strings.HasSuffix(n, ".go") || strings.HasSuffix(n, "_test.go") {
			continue
		}
		p := filepath.Join(*dir, n)
		src, err := os.ReadFile(p)
		if err != nil {
			panic(err)
		}
		f, err := parser.ParseFile(fset, p, src, parser.ParseComments)
		if err != nil {
			panic(err)
		}
		pkgName = f.Name.Name
		files = append(files, &srcFile{path: p, src: src, file: f})
	}
	found := map[string]bool{}
	var moved []string
	var movedConsts, movedVars []string
	off := func(p token.Pos) int { return fset.Position(p).Offset }
	lineStart := func(src []byte, o int) int {
		for o > 0 && src[o-1] != '\n' {
			o--
		}
		return o
	}
	lineEnd := func(src []byte, o int) int {
		for o < len(src) && src[o] != '\n' {
			o++
		}
		if o < len(src) {
			o++
		}
		return o
	}
	for _, sf := range files {
		used := false
		for _, d := range sf.file.Decls {
			switch d := d.(type) {
			case *ast.FuncDecl:
				name := d.Name.Name
				key := name
				if d.Recv != nil && len(d.Recv.List) > 0 {
					t := recvType(d.Recv.List[0].Type)
					if !want[t] && !want[t+"."+name] {
						continue
					}
					key = t + "." + name
					found[t] = true
				} else if !want[name] {
					continue
				}
				found[key] = true
				start := d.Pos()
				if d.Doc != nil {
					start = d.Doc.Pos()
				}
				s, e := lineStart(sf.src, off(start)), lineEnd(sf.src, off(d.End()))
				sf.cuts = append(sf.cuts, cut{s, e})
				moved = append(moved, string(sf.src[s:e]))
				used = true
			case *ast.GenDecl:
				if d.Tok == token.IMPORT {
					continue
				}
				var sel []ast.Spec
				for _, s := range d.Specs {
					switch s := s.(type) {
					case *ast.TypeSpec:
						if want[s.Name.Name] {
							sel = append(sel, s)
							found[s.Name.Name] = true
						}
					case *ast.ValueSpec:
						hit := false
						for _, n := range s.Names {
							if want[n.Name] {
								hit = true
								found[n.Name] = true
							}
						}
						if hit {
							if len(s.Names) > 1 {
								for _, n := range s.Names {
									if !want[n.Name] {
										fmt.Fprintf(os.Stderr, "multi-name spec %s needs all names\n", n.Name)
										os.Exit(1)
									}
								}
							}
							sel = append(sel, s)
						}
					}
				}
				if len(sel) == 0 {
					continue
				}
				used = true
				if len(sel) == len(d.Specs) {
					start := d.Pos()
					if d.Doc != nil {
						start = d.Doc.Pos()
					}
					s, e := lineStart(sf.src, off(start)), lineEnd(sf.src, off(d.End()))
					sf.cuts = append(sf.cuts, cut{s, e})
					moved = append(moved, string(sf.src[s:e]))
					continue
				}
				if !d.Lparen.IsValid() {
					panic("partial single spec")
				}
				for _, s := range sel {
					start := s.Pos()
					var doc *ast.CommentGroup
					switch s := s.(type) {
					case *ast.ValueSpec:
						doc = s.Doc
					case *ast.TypeSpec:
						doc = s.Doc
					}
					if doc != nil {
						start = doc.Pos()
					}
					a, b := lineStart(sf.src, off(start)), lineEnd(sf.src, off(s.End()))
					sf.cuts = append(sf.cuts, cut{a, b})
					text := string(sf.src[a:b])
					switch d.Tok {
					case token.CONST:
						if vs, ok := s.(*ast.ValueSpec); ok && vs.Type == nil && vs.Values == nil {
							fmt.Fprintf(os.Stderr, "iota-continued const %s cannot move alone\n", vs.Names[0].Name)
							os.Exit(1)
						}
						movedConsts = append(movedConsts, text)
					case token.VAR:
						movedVars = append(movedVars, text)
					case token.TYPE:
						moved = append(moved, "type "+strings.TrimLeft(text, "\t"))
					}
				}
			}
		}
		if used {
			for _, is := range sf.file.Imports {
				line := "\t" + is.Path.Value + "\n"
				if is.Name != nil {
					line = "\t" + is.Name.Name + " " + is.Path.Value + "\n"
				}
				if !seenImp[line] {
					seenImp[line] = true
					imports = append(imports, line)
				}
			}
		}
	}
	for w := range want {
		if !found[w] {
			fmt.Fprintf(os.Stderr, "not found: %s\n", w)
			os.Exit(1)
		}
	}
	var b strings.Builder
	existing, err := os.ReadFile(target)
	if err == nil {
		b.Write(existing)
		b.WriteString("\n")
	} else {
		b.WriteString("package " + pkgName + "\n\n")
		if len(imports) > 0 {
			sort.Strings(imports)
			b.WriteString("import (\n" + strings.Join(imports, "") + ")\n\n")
		}
	}
	if len(movedConsts) > 0 {
		b.WriteString("const (\n" + strings.Join(movedConsts, "") + ")\n\n")
	}
	if len(movedVars) > 0 {
		b.WriteString("var (\n" + strings.Join(movedVars, "") + ")\n\n")
	}
	b.WriteString(strings.Join(moved, "\n"))
	for _, sf := range files {
		if len(sf.cuts) == 0 {
			continue
		}
		sort.Slice(sf.cuts, func(i, j int) bool { return sf.cuts[i].start > sf.cuts[j].start })
		src := sf.src
		for _, c := range sf.cuts {
			src = append(src[:c.start:c.start], src[c.end:]...)
		}
		if err := os.WriteFile(sf.path, src, 0o644); err != nil {
			panic(err)
		}
	}
	if err := os.WriteFile(target, []byte(b.String()), 0o644); err != nil {
		panic(err)
	}
	fmt.Fprintf(os.Stderr, "moved %d decls, %d consts, %d vars into %s (existing=%v)\n", len(moved), len(movedConsts), len(movedVars), *out, err == nil)
}
