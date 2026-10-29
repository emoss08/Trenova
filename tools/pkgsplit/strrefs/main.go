package main

import (
	"bufio"
	"flag"
	"fmt"
	"go/scanner"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var (
	root   = flag.String("root", ".", "module root (services/tms)")
	old    = flag.String("old", "", "old package name as it appears in string literals and comments")
	symMap = flag.String("symmap", "", "lines Symbol<TAB>newImportPath, as for pkgmove -symmap")
	extra  = flag.String("extra", "../../shared", "extra roots to scan, comma separated")
	dryRun = flag.Bool("n", false, "dry run")
)

type edit struct {
	start, end int
	text       string
}

func die(f string, a ...any) { fmt.Fprintf(os.Stderr, f+"\n", a...); os.Exit(1) }

func isIdent(b byte) bool {
	return b == '_' || b >= '0' && b <= '9' || b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z'
}

func loadMap() map[string]string {
	f, err := os.Open(*symMap)
	if err != nil {
		die("%v", err)
	}
	defer f.Close()
	m := map[string]string{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		parts := strings.Fields(sc.Text())
		if len(parts) != 2 {
			continue
		}
		if g := filepath.Base(parts[1]); g != *old {
			m[parts[0]] = g
		}
	}
	return m
}

func literalEdits(lit string, off int, newPkg map[string]string) []edit {
	var out []edit
	prefix := *old + "."
	for i := 0; ; {
		j := strings.Index(lit[i:], prefix)
		if j < 0 {
			return out
		}
		s := i + j
		i = s + len(prefix)
		if s > 0 && isIdent(lit[s-1]) {
			continue
		}
		e := i
		for e < len(lit) && isIdent(lit[e]) {
			e++
		}
		if g, ok := newPkg[lit[i:e]]; ok {
			out = append(out, edit{off + s, off + s + len(*old), g})
		}
	}
}

func rewrite(path string, src []byte, newPkg map[string]string) ([]byte, bool) {
	fset := token.NewFileSet()
	file := fset.AddFile(path, -1, len(src))
	var s scanner.Scanner
	s.Init(file, src, nil, scanner.ScanComments)
	var edits []edit
	for {
		pos, tok, lit := s.Scan()
		if tok == token.EOF {
			break
		}
		if tok == token.STRING || tok == token.COMMENT {
			edits = append(edits, literalEdits(lit, file.Offset(pos), newPkg)...)
		}
	}
	if len(edits) == 0 {
		return nil, false
	}
	sort.Slice(edits, func(a, b int) bool { return edits[a].start > edits[b].start })
	out := append([]byte(nil), src...)
	for _, e := range edits {
		out = append(out[:e.start], append([]byte(e.text), out[e.end:]...)...)
	}
	return out, true
}

func main() {
	flag.Parse()
	if *old == "" || *symMap == "" {
		die("usage: strrefs -root services/tms -old <pkg> -symmap <file>")
	}
	newPkg := loadMap()
	roots := []string{*root}
	for _, r := range strings.Split(*extra, ",") {
		if r != "" {
			roots = append(roots, filepath.Join(*root, r))
		}
	}
	needle := []byte(*old + ".")
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
			if err != nil {
				return err
			}
			if !strings.Contains(string(src), string(needle)) {
				return nil
			}
			out, ok := rewrite(p, src, newPkg)
			if !ok {
				return nil
			}
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
