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
	"strconv"
	"strings"

	"github.com/bytedance/sonic"
)

var messageArgs = map[string]int{
	"Add":                            2,
	"AddWithPriority":                2,
	"Advise":                         2,
	"NewAdvisory":                    2,
	"NewValidationError":             2,
	"NewValidationErrorWithPriority": 2,
	"NewBusinessError":               0,
	"NewDatabaseError":               0,
	"NewAuthenticationError":         0,
	"NewAuthorizationError":          0,
	"NewNotFoundError":               0,
	"NewNotImplementedError":         0,
	"NewConflictError":               0,
	"NewRateLimitError":              1,
	"Error":                          0,
}

var messageFields = map[string]struct{}{
	"Title":       {},
	"Message":     {},
	"Detail":      {},
	"Description": {},
	"Label":       {},
	"Summary":     {},
	"Subject":     {},
}

var labelMethods = map[string]struct{}{
	"Label":       {},
	"DisplayName": {},
}

var noMessage = map[string]struct{}{
	"NewMultiError":                   {},
	"NewMultiErrorWithLimit":          {},
	"NewError":                        {},
	"NewRequestTimeoutError":          {},
	"NewRequestTooLargeError":         {},
	"NewResolveError":                 {},
	"NewTransformError":               {},
	"NewComputeError":                 {},
	"NewVariableError":                {},
	"NewSchemaError":                  {},
	"NewMissingFieldError":            {},
	"NewSeedError":                    {},
	"NewMissingDependencyError":       {},
	"NewCircularDependencyError":      {},
	"NewApplicationError":             {},
	"NewRetryableError":               {},
	"NewRetryableErrorWithDelay":      {},
	"NewNonRetryableError":            {},
	"NewNonRetryableApplicationError": {},
	"NewTemplateInvalidError":         {},
	"NewInvalidInputError":            {},
	"NewResourceNotFoundError":        {},
	"NewPermissionDeniedError":        {},
	"NewDataIntegrityError":           {},
	"NewThrottleError":                {},
	"NewConcurrentAccessError":        {},
}

func isErrortypesConstructor(name string) bool {
	return strings.HasPrefix(name, "New") && strings.HasSuffix(name, "Error")
}

type site struct {
	Callee string
	Where  string
	Expr   string
}

type entry struct {
	Message string `json:"message"`
	File    string `json:"file"`
	Line    int    `json:"line"`
	Callee  string `json:"callee"`
}

type extractor struct {
	repoRoot string
	fset     *token.FileSet
	entries  []entry
	unknown  map[string][]string
	dynamic  []site
}

func (e *extractor) walkDir(root string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case "node_modules", "testdata", ".git", "vendor", "mocks", "seeds":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		if strings.HasSuffix(path, "_test.go") || strings.HasSuffix(path, "_gen.go") {
			return nil
		}
		return e.parseFile(path)
	})
}

func (e *extractor) parseFile(path string) error {
	file, err := parser.ParseFile(e.fset, path, nil, 0)
	if err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}

	discarded := discardedCalls(file)

	rel, err := filepath.Rel(e.repoRoot, path)
	if err != nil {
		rel = path
	}

	ast.Inspect(file, func(n ast.Node) bool {
		if fn, isFn := n.(*ast.FuncDecl); isFn {
			e.recordLabelMethod(fn, rel)
			return true
		}

		if kv, isKV := n.(*ast.KeyValueExpr); isKV {
			e.recordStructField(kv, rel)
			return true
		}

		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		name := calleeName(call.Fun)
		if name == "" {
			return true
		}

		idx, known := messageArgs[name]
		if !known {
			if _, ignored := noMessage[name]; ignored {
				return true
			}
			if isErrortypesConstructor(name) {
				e.unknown[name] = append(e.unknown[name], fmt.Sprintf("%s:%d", rel, e.fset.Position(call.Pos()).Line))
			}
			return true
		}

		if idx >= len(call.Args) {
			return true
		}

		if name == "Error" && discarded[call] {
			return true
		}

		msg, ok := stringLiteral(call.Args[idx])
		if !ok {
			if constructsProse(call.Args[idx]) {
				e.dynamic = append(e.dynamic, site{
					Callee: name,
					Where:  fmt.Sprintf("%s:%d", rel, e.fset.Position(call.Pos()).Line),
					Expr:   exprText(call.Args[idx]),
				})
			}
			return true
		}
		if strings.TrimSpace(msg) == "" {
			return true
		}

		e.entries = append(e.entries, entry{
			Message: msg,
			File:    filepath.ToSlash(rel),
			Line:    e.fset.Position(call.Pos()).Line,
			Callee:  name,
		})
		return true
	})

	return nil
}

func discardedCalls(file *ast.File) map[*ast.CallExpr]bool {
	discarded := map[*ast.CallExpr]bool{}
	ast.Inspect(file, func(n ast.Node) bool {
		stmt, ok := n.(*ast.ExprStmt)
		if !ok {
			return true
		}
		if call, isCall := stmt.X.(*ast.CallExpr); isCall {
			discarded[call] = true
		}
		return true
	})
	return discarded
}

func (e *extractor) recordLabelMethod(fn *ast.FuncDecl, rel string) {
	if fn.Recv == nil || fn.Body == nil || fn.Name == nil {
		return
	}
	if strings.Contains(rel, "pkg/dbdialect") {
		return
	}
	if _, wanted := labelMethods[fn.Name.Name]; !wanted {
		return
	}
	if fn.Type.Results == nil || len(fn.Type.Results.List) != 1 {
		return
	}
	if ident, ok := fn.Type.Results.List[0].Type.(*ast.Ident); !ok || ident.Name != "string" {
		return
	}

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		ret, ok := n.(*ast.ReturnStmt)
		if !ok || len(ret.Results) != 1 {
			return true
		}

		msg, ok := stringLiteral(ret.Results[0])
		if !ok || strings.TrimSpace(msg) == "" {
			return true
		}

		e.entries = append(e.entries, entry{
			Message: msg,
			File:    filepath.ToSlash(rel),
			Line:    e.fset.Position(ret.Pos()).Line,
			Callee:  "label:" + fn.Name.Name,
		})
		return true
	})
}

func (e *extractor) recordStructField(kv *ast.KeyValueExpr, rel string) {
	key, ok := kv.Key.(*ast.Ident)
	if !ok {
		return
	}
	if _, wanted := messageFields[key.Name]; !wanted {
		return
	}

	msg, ok := stringLiteral(kv.Value)
	if !ok || strings.TrimSpace(msg) == "" {
		return
	}

	e.entries = append(e.entries, entry{
		Message: msg,
		File:    filepath.ToSlash(rel),
		Line:    e.fset.Position(kv.Pos()).Line,
		Callee:  "field:" + key.Name,
	})
}

func calleeName(fun ast.Expr) string {
	switch f := fun.(type) {
	case *ast.Ident:
		return f.Name
	case *ast.SelectorExpr:
		return f.Sel.Name
	default:
		return ""
	}
}

func stringLiteral(expr ast.Expr) (string, bool) {
	switch v := expr.(type) {
	case *ast.BasicLit:
		if v.Kind != token.STRING {
			return "", false
		}
		s, err := strconv.Unquote(v.Value)
		if err != nil {
			return "", false
		}
		return s, true
	case *ast.BinaryExpr:
		if v.Op != token.ADD {
			return "", false
		}
		left, ok := stringLiteral(v.X)
		if !ok {
			return "", false
		}
		right, ok := stringLiteral(v.Y)
		if !ok {
			return "", false
		}
		return left + right, true
	default:
		return "", false
	}
}

func constructsProse(expr ast.Expr) bool {
	switch v := expr.(type) {
	case *ast.CallExpr:
		if calleeName(v.Fun) != "Sprintf" {
			return false
		}
		format, ok := stringLiteral(v.Args[0])
		return ok && strings.ContainsAny(format, "abcdefghijklmnopqrstuvwxyz")
	case *ast.BinaryExpr:
		if v.Op != token.ADD {
			return false
		}
		_, leftLiteral := stringLiteral(v.X)
		_, rightLiteral := stringLiteral(v.Y)
		return leftLiteral != rightLiteral
	default:
		return false
	}
}

func exprText(expr ast.Expr) string {
	switch v := expr.(type) {
	case *ast.CallExpr:
		return calleeName(v.Fun) + "(...)"
	case *ast.Ident:
		return v.Name
	case *ast.SelectorExpr:
		return exprText(v.X) + "." + v.Sel.Name
	case *ast.BinaryExpr:
		return exprText(v.X) + " " + v.Op.String() + " " + exprText(v.Y)
	case *ast.BasicLit:
		return v.Value
	default:
		return fmt.Sprintf("%T", expr)
	}
}

func main() {
	var repoRoot string
	var outPath string
	flag.StringVar(&repoRoot, "root", ".", "repository root")
	flag.StringVar(&outPath, "out", "", "output JSON path (default stdout)")
	var reportDynamic bool
	flag.BoolVar(&reportDynamic, "dynamic", false,
		"report message arguments that are not string literals instead of writing entries")
	flag.Parse()

	absRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "i18n-extract: %v\n", err)
		os.Exit(1)
	}

	e := &extractor{
		repoRoot: absRoot,
		fset:     token.NewFileSet(),
		unknown:  map[string][]string{},
	}

	for _, dir := range []string{"services/tms", "services/gtc", "shared"} {
		target := filepath.Join(absRoot, dir)
		if _, statErr := os.Stat(target); statErr != nil {
			continue
		}
		if walkErr := e.walkDir(target); walkErr != nil {
			fmt.Fprintf(os.Stderr, "i18n-extract: %v\n", walkErr)
			os.Exit(1)
		}
	}

	if reportDynamic {
		sort.Slice(e.dynamic, func(i, j int) bool { return e.dynamic[i].Where < e.dynamic[j].Where })
		for _, d := range e.dynamic {
			fmt.Printf("%s\t%s\t%s\n", d.Where, d.Callee, d.Expr)
		}
		fmt.Fprintf(os.Stderr, "i18n-extract: %d dynamic message arguments\n", len(e.dynamic))
		return
	}

	if len(e.unknown) > 0 {
		names := make([]string, 0, len(e.unknown))
		for name := range e.unknown {
			names = append(names, name)
		}
		sort.Strings(names)
		fmt.Fprintf(os.Stderr,
			"i18n-extract: unrecognised error constructors. Add each to messageArgs (with the\n"+
				"index of its message argument) or to noMessage (with a reason) in\n"+
				"shared/cmd/i18n-extract/main.go. Leaving one unlisted would drop a whole class\n"+
				"of messages from the catalog with nothing failing:\n")
		for _, name := range names {
			sites := e.unknown[name]
			sample := sites[0]
			fmt.Fprintf(os.Stderr, "  %s (%d call sites, e.g. %s)\n", name, len(sites), sample)
		}
		os.Exit(1)
	}

	sort.Slice(e.entries, func(i, j int) bool {
		if e.entries[i].File != e.entries[j].File {
			return e.entries[i].File < e.entries[j].File
		}
		return e.entries[i].Line < e.entries[j].Line
	})

	payload, err := sonic.ConfigStd.MarshalIndent(map[string]any{"entries": e.entries}, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "i18n-extract: %v\n", err)
		os.Exit(1)
	}

	if outPath == "" {
		os.Stdout.Write(payload)
		return
	}
	if err := os.WriteFile(outPath, append(payload, '\n'), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "i18n-extract: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "i18n-extract: %d message sites -> %s\n", len(e.entries), outPath)
}
