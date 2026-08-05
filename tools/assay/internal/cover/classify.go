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

// ExecutableRanges returns the line spans inside function bodies — the only lines
// coverage can attribute, and therefore the only ones worth mutating.
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

	return functionBodyRanges(fset, file), nil
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
