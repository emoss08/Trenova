package main

import (
	"bufio"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type edit struct {
	off, end int
	text     string
}

type fileWork struct {
	path    string
	src     []byte
	file    *ast.File
	fset    *token.FileSet
	edits   []edit
	needImp map[string]string
	dropImp string
	keepImp bool
	newPath string
	pkgName string
}

var (
	root     = flag.String("root", ".", "module root (services/tms)")
	modPath  = flag.String("mod", "github.com/emoss08/trenova", "module path")
	srcRel   = flag.String("src", "", "source package dir relative to root")
	mapFile  = flag.String("map", "", "file<TAB>newpkg lines")
	mockRoot = flag.String("mocks", "internal/testutil/mocks", "mock root relative to root")
	extra    = flag.String("extra", "../../shared", "extra roots to scan, comma separated")
	dryRun   = flag.Bool("n", false, "dry run")
	symMap   = flag.String("symmap", "", "rewrite-only: lines Symbol<TAB>newImportPath; -src names the old package")
)

func die(f string, a ...any) { fmt.Fprintf(os.Stderr, f+"\n", a...); os.Exit(1) }

func main() {
	flag.Parse()
	if *symMap != "" {
		rewriteOnly()
		return
	}
	srcDir := filepath.Join(*root, *srcRel)
	oldName := filepath.Base(*srcRel)
	oldPath := *modPath + "/" + filepath.ToSlash(*srcRel)
	parent := filepath.Dir(*srcRel)

	groupOfFile := map[string]string{}
	mf, err := os.Open(*mapFile)
	if err != nil {
		die("%v", err)
	}
	sc := bufio.NewScanner(mf)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Fields(line)
		groupOfFile[parts[0]] = parts[1]
	}
	mf.Close()
	pathOf := func(group string) string {
		if group == oldName {
			return oldPath
		}
		return *modPath + "/" + filepath.ToSlash(filepath.Join(parent, group))
	}

	fset := token.NewFileSet()
	entries, _ := os.ReadDir(srcDir)
	type srcFile struct {
		fw    *fileWork
		group string
		ext   bool
	}
	var srcFiles []*srcFile
	declGroup := map[string]string{}
	declIsType := map[string]bool{}
	declIsFunc := map[string]bool{}
	edgeFiles := map[string][]string{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		p := filepath.Join(srcDir, e.Name())
		src, err := os.ReadFile(p)
		if err != nil {
			die("%v", err)
		}
		f, err := parser.ParseFile(fset, p, src, parser.ParseComments)
		if err != nil {
			die("%v", err)
		}
		g, ok := groupOfFile[e.Name()]
		if !ok {
			g = oldName
		}
		delete(groupOfFile, e.Name())
		ext := strings.HasSuffix(f.Name.Name, "_test")
		sf := &srcFile{fw: &fileWork{path: p, src: src, file: f, fset: fset, needImp: map[string]string{}}, group: g, ext: ext}
		srcFiles = append(srcFiles, sf)
		if ext {
			continue
		}
		for _, d := range f.Decls {
			switch d := d.(type) {
			case *ast.FuncDecl:
				if d.Recv == nil {
					declGroup[d.Name.Name] = g
					declIsFunc[d.Name.Name] = true
				}
			case *ast.GenDecl:
				for _, s := range d.Specs {
					switch s := s.(type) {
					case *ast.TypeSpec:
						declGroup[s.Name.Name] = g
						declIsType[s.Name.Name] = true
					case *ast.ValueSpec:
						for _, n := range s.Names {
							if n.Name != "_" {
								declGroup[n.Name] = g
							}
						}
					}
				}
			}
		}
	}
	for f := range groupOfFile {
		die("map names unknown file %s", f)
	}

	for _, sf := range srcFiles {
		if sf.ext {
			continue
		}
		for _, d := range sf.fw.file.Decls {
			if fd, ok := d.(*ast.FuncDecl); ok && fd.Recv != nil && len(fd.Recv.List) > 0 {
				t := recvType(fd.Recv.List[0].Type)
				if g, ok := declGroup[t]; ok && g != sf.group {
					die("%s: method %s.%s is in group %s but type is in %s", sf.fw.path, t, fd.Name.Name, sf.group, g)
				}
			}
		}
	}

	groupEdges := map[string]map[string]bool{}
	problems := 0

	for _, sf := range srcFiles {
		fw := sf.fw
		if sf.group != oldName {
			fw.newPath = filepath.Join(*root, parent, sf.group, filepath.Base(fw.path))
			newPkg := sf.group
			if sf.ext {
				newPkg += "_test"
			}
			fw.edits = append(fw.edits, edit{fset.Position(fw.file.Name.Pos()).Offset, fset.Position(fw.file.Name.End()).Offset, newPkg})
		}
		if sf.ext {
			continue
		}
		qualify := func(id *ast.Ident) {
			g, ok := declGroup[id.Name]
			if !ok || g == sf.group {
				return
			}
			if !ast.IsExported(id.Name) {
				fmt.Fprintf(os.Stderr, "UNEXPORTED %s: %s from group %s used in group %s\n", fset.Position(id.Pos()), id.Name, g, sf.group)
				problems++
				return
			}
			off := fset.Position(id.Pos()).Offset
			fw.edits = append(fw.edits, edit{off, off, g + "."})
			fw.needImp[g] = pathOf(g)
			if groupEdges[sf.group] == nil {
				groupEdges[sf.group] = map[string]bool{}
			}
			groupEdges[sf.group][g] = true
			edgeFiles[sf.group+">"+g] = append(edgeFiles[sf.group+">"+g], filepath.Base(fw.path)+":"+id.Name)
		}
		for _, id := range fw.file.Unresolved {
			qualify(id)
		}
		ast.Inspect(fw.file, func(n ast.Node) bool {
			if cl, ok := n.(*ast.CompositeLit); ok {
				for _, e := range cl.Elts {
					if kv, ok := e.(*ast.KeyValueExpr); ok {
						if id, ok := kv.Key.(*ast.Ident); ok && id.Obj == nil {
							_, isMap := cl.Type.(*ast.MapType)
							if g, ok := declGroup[id.Name]; ok && g != sf.group && (isMap || (!declIsType[id.Name] && !declIsFunc[id.Name])) {
								qualify(id)
							} else if g, ok := declGroup[id.Name]; ok && g != sf.group && !declIsType[id.Name] {
								fmt.Fprintf(os.Stderr, "CHECK key %s: %s (group %s) left unqualified\n", fset.Position(id.Pos()), id.Name, g)
							}
						}
					}
				}
			}
			return true
		})
	}
	if problems > 0 {
		die("%d unexported cross-group references", problems)
	}
	if cyc := findCycle(groupEdges); cyc != "" {
		for k, v := range edgeFiles {
			if len(v) > 8 {
				v = v[:8]
			}
			fmt.Fprintf(os.Stderr, "EDGE %s: %s\n", k, strings.Join(v, " "))
		}
		die("group cycle: %s", cyc)
	}

	mockOld := filepath.Join(*root, *mockRoot, oldName+"mock")
	mockOldPath := *modPath + "/" + filepath.ToSlash(filepath.Join(*mockRoot, oldName+"mock"))
	mockDeclGroup := map[string]string{}
	mockMoves := map[string]string{}
	if ents, err := os.ReadDir(mockOld); err == nil {
		for _, e := range ents {
			n := e.Name()
			if !strings.HasPrefix(n, "mock_") || !strings.HasSuffix(n, ".go") {
				continue
			}
			iface := strings.TrimSuffix(strings.TrimPrefix(n, "mock_"), ".go")
			g, ok := declGroup[iface]
			if !ok {
				fmt.Fprintf(os.Stderr, "WARN mock %s: interface not found\n", n)
				continue
			}
			if g == oldName {
				continue
			}
			mockMoves[filepath.Join(mockOld, n)] = g + "mock"
			src, _ := os.ReadFile(filepath.Join(mockOld, n))
			f, err := parser.ParseFile(token.NewFileSet(), n, src, 0)
			if err != nil {
				die("%v", err)
			}
			for _, d := range f.Decls {
				switch d := d.(type) {
				case *ast.FuncDecl:
					if d.Recv == nil {
						mockDeclGroup[d.Name.Name] = g + "mock"
					}
				case *ast.GenDecl:
					for _, s := range d.Specs {
						if ts, ok := s.(*ast.TypeSpec); ok {
							mockDeclGroup[ts.Name.Name] = g + "mock"
						}
					}
				}
			}
		}
	}
	mockPathOf := func(g string) string {
		return *modPath + "/" + filepath.ToSlash(filepath.Join(*mockRoot, g))
	}

	var all []*fileWork
	byPath := map[string]*fileWork{}
	for _, sf := range srcFiles {
		byPath[sf.fw.path] = sf.fw
		all = append(all, sf.fw)
	}
	roots := []string{*root}
	for _, r := range strings.Split(*extra, ",") {
		if r != "" {
			roots = append(roots, filepath.Join(*root, r))
		}
	}
	for _, r := range roots {
		filepath.WalkDir(r, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				n := d.Name()
				if p != r && (n == "node_modules" || n == "vendor" || strings.HasPrefix(n, ".")) {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(p, ".go") {
				return nil
			}
			src, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			if !strings.Contains(string(src), oldPath) && !strings.Contains(string(src), mockOldPath) {
				return nil
			}
			fw := byPath[p]
			if fw == nil {
				fs2 := token.NewFileSet()
				f, err := parser.ParseFile(fs2, p, src, parser.ParseComments)
				if err != nil {
					die("%v", err)
				}
				fw = &fileWork{path: p, src: src, file: f, fset: fs2, needImp: map[string]string{}}
				byPath[p] = fw
				all = append(all, fw)
			}
			if g, ok := mockMoves[p]; ok {
				fw.newPath = filepath.Join(*root, *mockRoot, g, filepath.Base(p))
				fw.edits = append(fw.edits, edit{fw.fset.Position(fw.file.Name.Pos()).Offset, fw.fset.Position(fw.file.Name.End()).Offset, g})
			}
			rewriteSelectors(fw, oldPath, oldName, declGroup, pathOf)
			rewriteSelectors(fw, mockOldPath, oldName+"mock", mockDeclGroup, mockPathOf)
			return nil
		})
	}

	changed := 0
	for _, fw := range all {
		if len(fw.edits) == 0 && len(fw.needImp) == 0 && fw.newPath == "" {
			continue
		}
		out := apply(fw)
		dst := fw.path
		if fw.newPath != "" {
			dst = fw.newPath
		}
		changed++
		if *dryRun {
			fmt.Println("would write", dst)
			continue
		}
		os.MkdirAll(filepath.Dir(dst), 0o755)
		if err := os.WriteFile(dst, out, 0o644); err != nil {
			die("%v", err)
		}
		if dst != fw.path {
			os.Remove(fw.path)
		}
		fmt.Println(dst)
	}
	fmt.Fprintf(os.Stderr, "changed %d files\n", changed)

	gs := make([]string, 0, len(groupEdges))
	for g := range groupEdges {
		gs = append(gs, g)
	}
	sort.Strings(gs)
	for _, g := range gs {
		ds := make([]string, 0)
		for d := range groupEdges[g] {
			ds = append(ds, d)
		}
		sort.Strings(ds)
		fmt.Fprintf(os.Stderr, "GROUP %s -> %s\n", g, strings.Join(ds, ","))
	}
}

func rewriteSelectors(fw *fileWork, impPath, defName string, declGroup map[string]string, pathOf func(string) string) {
	var spec *ast.ImportSpec
	for _, is := range fw.file.Imports {
		if strings.Trim(is.Path.Value, `"`) == impPath {
			spec = is
		}
	}
	if spec == nil {
		return
	}
	name := defName
	if spec.Name != nil {
		name = spec.Name.Name
	}
	remaining := 0
	ast.Inspect(fw.file, func(n ast.Node) bool {
		se, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		id, ok := se.X.(*ast.Ident)
		if !ok || id.Name != name || id.Obj != nil {
			return true
		}
		g, ok := declGroup[se.Sel.Name]
		if !ok || g == defName {
			remaining++
			return true
		}
		fw.edits = append(fw.edits, edit{fw.fset.Position(id.Pos()).Offset, fw.fset.Position(id.End()).Offset, g})
		fw.needImp[g] = pathOf(g)
		return true
	})
	if remaining == 0 && len(fw.needImp) > 0 {
		start := fw.fset.Position(spec.Pos()).Offset
		end := fw.fset.Position(spec.End()).Offset

		for start > 0 && fw.src[start-1] != '\n' {
			start--
		}
		for end < len(fw.src) && fw.src[end] != '\n' {
			end++
		}
		if end < len(fw.src) {
			end++
		}
		fw.edits = append(fw.edits, edit{start, end, ""})
	}
}

func apply(fw *fileWork) []byte {

	own := ""
	if fw.newPath != "" {
		own = filepath.Base(filepath.Dir(fw.newPath))
	} else {
		own = filepath.Base(filepath.Dir(fw.path))
	}
	existing := map[string]bool{}
	for _, is := range fw.file.Imports {
		existing[strings.Trim(is.Path.Value, `"`)] = true
	}
	var lines []string
	for g, p := range fw.needImp {
		if existing[p] || (g == own && !strings.HasSuffix(fw.file.Name.Name, "_test")) {
			continue
		}
		lines = append(lines, "\t\""+p+"\"\n")
	}
	sort.Strings(lines)
	if len(lines) > 0 {
		var off int
		var text string
		var block *ast.GenDecl
		for _, d := range fw.file.Decls {
			if gd, ok := d.(*ast.GenDecl); ok && gd.Tok == token.IMPORT && gd.Lparen.IsValid() {
				block = gd
				break
			}
		}
		if block != nil {
			off = fw.fset.Position(block.Lparen).Offset + 1
			text = "\n" + strings.TrimSuffix(strings.Join(lines, ""), "\n")
		} else {
			off = fw.fset.Position(fw.file.Name.End()).Offset
			text = "\n\nimport (\n" + strings.Join(lines, "") + ")"
		}
		fw.edits = append(fw.edits, edit{off, off, text})
	}
	sort.SliceStable(fw.edits, func(i, j int) bool { return fw.edits[i].off > fw.edits[j].off })
	out := append([]byte(nil), fw.src...)
	for _, e := range fw.edits {
		out = append(out[:e.off], append([]byte(e.text), out[e.end:]...)...)
	}
	return out
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

func findCycle(g map[string]map[string]bool) string {
	state := map[string]int{}
	var stack []string
	var res string
	var visit func(string) bool
	visit = func(n string) bool {
		state[n] = 1
		stack = append(stack, n)
		for m := range g[n] {
			if state[m] == 1 {
				res = strings.Join(append(stack, m), " -> ")
				return true
			}
			if state[m] == 0 && visit(m) {
				return true
			}
		}
		stack = stack[:len(stack)-1]
		state[n] = 2
		return false
	}
	for n := range g {
		if state[n] == 0 && visit(n) {
			return res
		}
	}
	return ""
}

func rewriteOnly() {
	oldPath := *modPath + "/" + filepath.ToSlash(*srcRel)
	oldName := filepath.Base(*srcRel)
	declGroup := map[string]string{}
	paths := map[string]string{}
	data, err := os.ReadFile(*symMap)
	if err != nil {
		die("%v", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.Fields(line)
		if len(parts) != 2 {
			continue
		}
		g := filepath.Base(parts[1])
		if g == oldName {
			continue
		}
		declGroup[parts[0]] = g
		paths[g] = parts[1]
	}
	pathOf := func(g string) string { return paths[g] }
	roots := []string{*root}
	for _, r := range strings.Split(*extra, ",") {
		if r != "" {
			roots = append(roots, filepath.Join(*root, r))
		}
	}
	changed := 0
	for _, r := range roots {
		filepath.WalkDir(r, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				n := d.Name()
				if p != r && (n == "node_modules" || n == "vendor" || strings.HasPrefix(n, ".")) {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(p, ".go") {
				return nil
			}
			src, err := os.ReadFile(p)
			if err != nil || !strings.Contains(string(src), oldPath) {
				return err
			}
			fs2 := token.NewFileSet()
			f, err := parser.ParseFile(fs2, p, src, parser.ParseComments)
			if err != nil {
				die("%v", err)
			}
			fw := &fileWork{path: p, src: src, file: f, fset: fs2, needImp: map[string]string{}}
			rewriteSelectors(fw, oldPath, oldName, declGroup, pathOf)
			if len(fw.edits) == 0 {
				return nil
			}
			out := apply(fw)
			changed++
			if *dryRun {
				fmt.Println("would write", p)
				return nil
			}
			if err := os.WriteFile(p, out, 0o644); err != nil {
				die("%v", err)
			}
			fmt.Println(p)
			return nil
		})
	}
	fmt.Fprintf(os.Stderr, "rewrote %d files\n", changed)
}
