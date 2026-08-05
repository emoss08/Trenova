package mutate

import (
	"context"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/zeebo/blake3"
	"golang.org/x/tools/go/packages"
)

const loadMode = packages.NeedName |
	packages.NeedFiles |
	packages.NeedSyntax |
	packages.NeedTypes |
	packages.NeedTypesInfo

type Mutant struct {
	ID          string   `json:"id"`
	Kind        Kind     `json:"kind"`
	Package     string   `json:"package"`
	File        string   `json:"file"`
	Line        int      `json:"line"`
	Function    string   `json:"function"`
	Original    string   `json:"original"`
	Replacement string   `json:"replacement"`
	source      []byte   `json:"-"`
	edits       []edit   `json:"-"`
	tests       testPlan `json:"-"`
}

func (m Mutant) Location() string {
	return fmt.Sprintf("%s:%d", filepath.Base(m.File), m.Line)
}

// testPlan is the set of tests that cover the mutated line, grouped by the test
// package that owns them.
type testPlan map[string][]string

func (p testPlan) empty() bool {
	for _, tests := range p {
		if len(tests) > 0 {
			return false
		}
	}

	return true
}

func (p testPlan) count() int {
	total := 0
	for _, tests := range p {
		total += len(tests)
	}

	return total
}

// LineFilter answers whether a mutation site is eligible. Sites outside it are
// never generated, so an uncovered or unchanged line costs nothing.
type LineFilter func(absPath string, line int) bool

type GenerateOptions struct {
	Root     string
	Package  string
	Tags     []string
	Env      []string
	Eligible LineFilter
	Tests    func(absPath string, line int) testPlan
}

func Generate(ctx context.Context, opts GenerateOptions) ([]Mutant, error) {
	cfg := &packages.Config{
		Mode:      loadMode,
		Context:   ctx,
		Dir:       opts.Root,
		Env:       opts.Env,
		ParseFile: parseWithComments,
	}
	if len(opts.Tags) > 0 {
		cfg.BuildFlags = []string{"-tags", strings.Join(opts.Tags, ",")}
	}

	loaded, err := packages.Load(cfg, opts.Package)
	if err != nil {
		return nil, fmt.Errorf("load %s for mutation: %w", opts.Package, err)
	}
	if len(loaded) == 0 {
		return nil, fmt.Errorf("package %s not found", opts.Package)
	}

	pkg := loaded[0]
	if len(pkg.Errors) > 0 {
		return nil, fmt.Errorf("package %s has errors, refusing to mutate: %v", opts.Package, pkg.Errors[0])
	}

	var out []Mutant
	for i, file := range pkg.Syntax {
		path := syntaxPath(pkg, i)
		if path == "" || strings.HasSuffix(path, "_test.go") {
			continue
		}

		source, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, fmt.Errorf("read %s: %w", path, readErr)
		}

		out = append(out, mutantsInFile(pkg, file, path, source, opts)...)
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].File != out[j].File {
			return out[i].File < out[j].File
		}
		if out[i].Line != out[j].Line {
			return out[i].Line < out[j].Line
		}
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}

		return out[i].Replacement < out[j].Replacement
	})

	return out, nil
}

func parseWithComments(fset *token.FileSet, filename string, src []byte) (*ast.File, error) {
	return parser.ParseFile(fset, filename, src, parser.ParseComments|parser.AllErrors)
}

func syntaxPath(pkg *packages.Package, i int) string {
	if i < len(pkg.CompiledGoFiles) && !strings.Contains(pkg.CompiledGoFiles[i], "go-build") {
		return pkg.CompiledGoFiles[i]
	}
	if i < len(pkg.GoFiles) {
		return pkg.GoFiles[i]
	}

	return ""
}

func mutantsInFile(
	pkg *packages.Package,
	file *ast.File,
	path string,
	source []byte,
	opts GenerateOptions,
) []Mutant {
	ctx := &fileContext{fset: pkg.Fset, info: pkg.TypesInfo, source: source}
	functions := functionSpans(pkg.Fset, file)

	seen := make(map[string]int)
	var out []Mutant

	ast.Inspect(file, func(node ast.Node) bool {
		if node == nil {
			return false
		}

		for _, m := range mutators {
			for _, cand := range m.candidates(ctx, node) {
				line := pkg.Fset.Position(cand.Pos).Line
				if opts.Eligible != nil && !opts.Eligible(path, line) {
					continue
				}

				function := enclosingFunction(functions, line)

				// The occurrence counter disambiguates identical mutations in the same
				// function without pinning the ID to a line number, which shifts on
				// every unrelated edit above it.
				fingerprint := strings.Join([]string{
					function, string(cand.Kind), cand.Original, cand.Replacement,
				}, "\x00")
				occurrence := seen[fingerprint]
				seen[fingerprint]++

				mutant := Mutant{
					Kind:        cand.Kind,
					Package:     pkg.PkgPath,
					File:        path,
					Line:        line,
					Function:    function,
					Original:    cand.Original,
					Replacement: cand.Replacement,
					source:      source,
					edits:       cand.Edits,
				}
				mutant.ID = mutantID(opts.Root, mutant, occurrence)

				if opts.Tests != nil {
					mutant.tests = opts.Tests(path, line)
				}

				out = append(out, mutant)
			}
		}

		return true
	})

	return out
}

func mutantID(root string, m Mutant, occurrence int) string {
	rel, err := filepath.Rel(root, m.File)
	if err != nil {
		rel = m.File
	}

	h := blake3.New()
	for _, field := range []string{
		"assay-mutant-v1",
		filepath.ToSlash(rel),
		m.Function,
		string(m.Kind),
		m.Original,
		m.Replacement,
		strconv.Itoa(occurrence),
	} {
		h.Write([]byte(field))
		h.Write([]byte{0})
	}

	return hex.EncodeToString(h.Sum(nil)[:12])
}

type functionSpan struct {
	Name  string
	Start int
	End   int
}

func functionSpans(fset *token.FileSet, file *ast.File) []functionSpan {
	var spans []functionSpan

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		spans = append(spans, functionSpan{
			Name:  functionName(fn),
			Start: fset.Position(fn.Pos()).Line,
			End:   fset.Position(fn.End()).Line,
		})
	}

	return spans
}

func functionName(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return fn.Name.Name
	}

	return receiverName(fn.Recv.List[0].Type) + "." + fn.Name.Name
}

func receiverName(expr ast.Expr) string {
	switch typ := expr.(type) {
	case *ast.StarExpr:
		return "*" + receiverName(typ.X)
	case *ast.Ident:
		return typ.Name
	case *ast.IndexExpr:
		return receiverName(typ.X)
	case *ast.IndexListExpr:
		return receiverName(typ.X)
	default:
		return "?"
	}
}

func enclosingFunction(spans []functionSpan, line int) string {
	for _, span := range spans {
		if line >= span.Start && line <= span.End {
			return span.Name
		}
	}

	return ""
}

// Source renders the mutant by splicing its edits into the original bytes.
func (m Mutant) Source() []byte {
	ordered := append([]edit(nil), m.edits...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Offset > ordered[j].Offset })

	out := append([]byte(nil), m.source...)
	for _, e := range ordered {
		if e.Offset < 0 || e.Offset+e.Length > len(out) {
			continue
		}
		out = append(out[:e.Offset], append([]byte(e.Text), out[e.Offset+e.Length:]...)...)
	}

	return out
}

func (m Mutant) Tests() testPlan { return m.tests }
