package cover

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
)

type LineKind int

const (
	LineIgnorable LineKind = iota
	LineInFunction
	LineDeclaration
)

func (k LineKind) String() string {
	switch k {
	case LineIgnorable:
		return "ignorable"
	case LineInFunction:
		return "in-function"
	default:
		return "declaration"
	}
}

type FileClassification struct {
	Path        string
	Executable  []int
	Declaration []int
}

func (c FileClassification) Narrowable() bool {
	return len(c.Declaration) == 0
}

// ExecutableRanges returns the lines that carry statements, which are the lines a
// coverage profile can attribute and therefore the only ones worth mutating.
//
// This deliberately differs from the spans narrowing uses. Narrowing has to treat
// the line a function's opening brace sits on as a declaration, because a signature
// change lands there and can break callers that no test of this line covers.
// Mutation has no such exposure — it rewrites expressions in place — so it wants
// every attributable line, including the body of `func F() int { return a + b }`,
// which coverage reports against the function's own line. Sharing one definition
// would either make narrowing unsound or leave every one-line function unmutated.
func ExecutableRanges(absPath string) ([]Block, error) {
	source, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", absPath, err)
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, absPath, source, parser.SkipObjectResolution)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", absPath, err)
	}

	return statementRanges(fset, file), nil
}

// statementRanges spans each statement of every function body. Nested statements
// need no separate entry: a block statement's own range already encloses them.
func statementRanges(fset *token.FileSet, file *ast.File) []Block {
	var blocks []Block

	add := func(body *ast.BlockStmt) {
		if body == nil {
			return
		}
		for _, stmt := range body.List {
			blocks = append(blocks, Block{
				StartLine: fset.Position(stmt.Pos()).Line,
				EndLine:   fset.Position(stmt.End()).Line,
			})
		}
	}

	ast.Inspect(file, func(node ast.Node) bool {
		switch decl := node.(type) {
		case *ast.FuncDecl:
			add(decl.Body)
		case *ast.FuncLit:
			add(decl.Body)
		}

		return true
	})

	return mergeBlocksOrEmpty(blocks)
}

// TestFunction is a top-level Test, Benchmark, Example or Fuzz declaration and
// the line span it occupies, signature included: an edit anywhere in the
// declaration affects that test and only that test.
type TestFunction struct {
	Name  string
	Start int
	End   int
}

// TestFunctions lists the test declarations in a _test.go file. Watch mode uses
// the spans to run exactly the tests an edit touched — an edit outside every
// span (a helper, an import, a package-level variable) can affect any test in
// the package and disqualifies the refinement.
func TestFunctions(absPath string) ([]TestFunction, error) {
	source, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", absPath, err)
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, absPath, source, parser.SkipObjectResolution)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", absPath, err)
	}

	var out []TestFunction
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv != nil || !isTestFuncName(fn.Name.Name) {
			continue
		}
		out = append(out, TestFunction{
			Name:  fn.Name.Name,
			Start: fset.Position(fn.Pos()).Line,
			End:   fset.Position(fn.End()).Line,
		})
	}

	return out, nil
}

func isTestFuncName(name string) bool {
	for _, prefix := range [...]string{"Test", "Benchmark", "Example", "Fuzz"} {
		if rest, ok := strings.CutPrefix(name, prefix); ok {
			// The character after the prefix must not be lowercase, mirroring
			// `go test`'s own rule: Testify is a helper, TestIfy is a test.
			if rest == "" || !isLowerASCII(rest[0]) {
				return true
			}
		}
	}

	return false
}

func isLowerASCII(c byte) bool {
	return c >= 'a' && c <= 'z'
}

func ClassifyFile(absPath string, lines []int) (FileClassification, error) {
	source, err := os.ReadFile(absPath)
	if err != nil {
		return FileClassification{}, fmt.Errorf("read %s: %w", absPath, err)
	}

	return classifySource(absPath, source, lines)
}

func classifySource(absPath string, source []byte, lines []int) (FileClassification, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, absPath, source, parser.SkipObjectResolution)
	if err != nil {
		return FileClassification{}, fmt.Errorf("parse %s: %w", absPath, err)
	}

	bodies := functionBodyRanges(fset, file)
	text := splitLines(source)

	out := FileClassification{Path: absPath}
	for _, line := range lines {
		switch classifyLine(line, text, bodies) {
		case LineIgnorable:
		case LineInFunction:
			out.Executable = append(out.Executable, line)
		default:
			out.Declaration = append(out.Declaration, line)
		}
	}

	return out, nil
}

func classifyLine(line int, text []string, bodies []Block) LineKind {
	if line < 1 {
		return LineDeclaration
	}

	if line <= len(text) && isIgnorableText(text[line-1]) {
		return LineIgnorable
	}

	for _, body := range bodies {
		if body.Contains(line) {
			return LineInFunction
		}
	}

	return LineDeclaration
}

func isIgnorableText(raw string) bool {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return true
	}
	if !strings.HasPrefix(trimmed, "//") {
		return false
	}

	return !strings.HasPrefix(trimmed, "//go:")
}

func functionBodyRanges(fset *token.FileSet, file *ast.File) []Block {
	var bodies []Block

	add := func(body *ast.BlockStmt) {
		if body == nil {
			return
		}
		open := fset.Position(body.Lbrace).Line
		end := fset.Position(body.Rbrace).Line
		if end-open < 2 {
			return
		}
		bodies = append(bodies, Block{StartLine: open + 1, EndLine: end - 1})
	}

	ast.Inspect(file, func(node ast.Node) bool {
		switch decl := node.(type) {
		case *ast.FuncDecl:
			add(decl.Body)
		case *ast.FuncLit:
			add(decl.Body)
		}

		return true
	})

	return mergeBlocksOrEmpty(bodies)
}

func mergeBlocksOrEmpty(blocks []Block) []Block {
	if len(blocks) == 0 {
		return nil
	}

	return mergeBlocks(blocks)
}

func splitLines(source []byte) []string {
	raw := bytes.Split(source, []byte("\n"))
	out := make([]string, len(raw))
	for i, line := range raw {
		out[i] = string(bytes.TrimSuffix(line, []byte("\r")))
	}

	return out
}
