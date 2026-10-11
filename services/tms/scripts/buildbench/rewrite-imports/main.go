package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/tools/imports"
)

var errUsage = errors.New(
	"usage: rewrite-imports [-moves moves.tsv] [-symbols symbol-moves.tsv] <dir> [dir ...]",
)

type symbolTarget struct {
	path string
	name string
}

type ledger struct {
	packages map[string]string
	symbols  map[string]map[string]symbolTarget
}

type edit struct {
	start, end int
	text       string
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "rewrite-imports:", err)
		os.Exit(1)
	}
}

func run() error {
	movesPath := flag.String("moves", "docs/modules/moves.tsv", "package moves, old<TAB>new")
	symbolsPath := flag.String(
		"symbols",
		"docs/modules/symbol-moves.tsv",
		"symbol moves, old.Sym<TAB>new.Sym",
	)
	flag.Parse()
	if flag.NArg() == 0 {
		return errUsage
	}

	l, err := loadLedger(*movesPath, *symbolsPath)
	if err != nil {
		return err
	}

	changed := 0
	for _, dir := range flag.Args() {
		err = filepath.WalkDir(dir, func(p string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() || !strings.HasSuffix(p, ".go") {
				return nil
			}
			ok, rerr := rewriteFile(p, l)
			if ok {
				changed++
			}
			return rerr
		})
		if err != nil {
			return err
		}
	}
	fmt.Printf("rewrote %d files\n", changed)

	return nil
}

func loadLedger(movesPath, symbolsPath string) (*ledger, error) {
	l := &ledger{packages: map[string]string{}, symbols: map[string]map[string]symbolTarget{}}
	err := readPairs(movesPath, func(oldPath, newPath string) {
		l.packages[oldPath] = newPath
	})
	if err != nil {
		return nil, err
	}
	err = readPairs(symbolsPath, func(oldSym, newSym string) {
		oldPkg, oldName := splitSymbol(oldSym)
		newPkg, newName := splitSymbol(newSym)
		if l.symbols[oldPkg] == nil {
			l.symbols[oldPkg] = map[string]symbolTarget{}
		}
		l.symbols[oldPkg][oldName] = symbolTarget{path: newPkg, name: newName}
	})
	if err != nil {
		return nil, err
	}

	return l, nil
}

func readPairs(p string, fn func(a, b string)) error {
	f, err := os.Open(p)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("open %s: %w", p, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		a, b, ok := strings.Cut(line, "\t")
		if !ok {
			return fmt.Errorf("%s: malformed line %q", p, line)
		}
		fn(a, b)
	}
	if err = scanner.Err(); err != nil {
		return fmt.Errorf("read %s: %w", p, err)
	}

	return nil
}

func (l *ledger) resolvePackage(p string) string {
	for range len(l.packages) + 1 {
		next, moved := l.packages[p]
		if !moved {
			return p
		}
		p = next
	}

	return p
}

func splitSymbol(s string) (pkg, name string) {
	i := strings.LastIndex(s, ".")
	return s[:i], s[i+1:]
}

func rewriteFile(p string, l *ledger) (bool, error) {
	data, err := os.ReadFile(p)
	if err != nil {
		return false, fmt.Errorf("read %s: %w", p, err)
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, p, data, parser.ParseComments)
	if err != nil {
		return false, fmt.Errorf("parse %s: %w", p, err)
	}

	aliases := map[string]string{}
	var edits []edit
	for _, im := range file.Imports {
		importPath, _ := strconv.Unquote(im.Path.Value)
		name := path.Base(importPath)
		if im.Name != nil {
			name = im.Name.Name
		}
		if _, moved := l.symbols[importPath]; moved {
			aliases[name] = importPath
		}
		if newPath := l.resolvePackage(importPath); newPath != importPath {
			edits = append(edits, edit{
				start: fset.Position(im.Path.Pos()).Offset,
				end:   fset.Position(im.Path.End()).Offset,
				text:  strconv.Quote(newPath),
			})
		}
	}

	added := map[string]bool{}
	ast.Inspect(file, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		id, ok := sel.X.(*ast.Ident)
		if !ok || id.Obj != nil {
			return true
		}
		target, ok := l.symbols[aliases[id.Name]][sel.Sel.Name]
		if !ok {
			return true
		}
		added[target.path] = true
		edits = append(edits, edit{
			start: fset.Position(id.Pos()).Offset,
			end:   fset.Position(sel.End()).Offset,
			text:  path.Base(target.path) + "." + target.name,
		})
		return true
	})
	if len(edits) == 0 {
		return false, nil
	}

	out := applyEdits(data, edits)
	resolved := make(map[string]bool, len(added))
	for p := range added {
		resolved[l.resolvePackage(p)] = true
	}
	out = addImports(out, resolved)
	formatted, err := imports.Process(p, out, nil)
	if err != nil {
		return false, fmt.Errorf("format %s: %w", p, err)
	}
	//nolint:gosec // G703: the file is rewritten in place under a directory the operator named
	if err = os.WriteFile(p, formatted, 0o600); err != nil {
		return false, fmt.Errorf("write %s: %w", p, err)
	}

	return true, nil
}

func applyEdits(data []byte, edits []edit) []byte {
	slices.SortFunc(edits, func(a, b edit) int { return b.start - a.start })
	out := slices.Clone(data)
	for _, e := range edits {
		out = slices.Concat(out[:e.start], []byte(e.text), out[e.end:])
	}

	return out
}

func addImports(data []byte, paths map[string]bool) []byte {
	if len(paths) == 0 {
		return data
	}
	sorted := make([]string, 0, len(paths))
	for p := range paths {
		if !strings.Contains(string(data), strconv.Quote(p)) {
			sorted = append(sorted, "\t"+strconv.Quote(p)+"\n")
		}
	}
	slices.Sort(sorted)
	block := strings.Join(sorted, "")
	s := string(data)
	if i := strings.Index(s, "\nimport (\n"); i >= 0 {
		j := i + len("\nimport (\n")
		return []byte(s[:j] + block + s[j:])
	}
	if i := strings.Index(s, "\nimport "); i >= 0 {
		return []byte(s[:i] + "\nimport (\n" + block + ")\n" + s[i:])
	}

	return data
}
