// Command i18n-extract walks the Go services and reports every user-facing message
// literal it finds, so the translation catalogs are derived from the source rather than
// maintained by hand.
//
// Messages are recognised by call site, not by heuristics on the string itself: a literal
// in the message position of a known constructor is user-facing by construction, and a
// literal anywhere else is not. That keeps SQL fragments, log lines, and struct tags out
// of the catalog without needing a denylist on this side.
//
// The recognised shapes are declared in messageArgs. An errortypes constructor that is not
// listed there is reported as an error rather than skipped, because a silently ignored
// constructor means a whole class of messages never reaches a translator and nothing fails.
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

// messageArgs maps a callee name to the argument index holding its user-facing message.
// Index is zero-based and counts only the call's own arguments. Indices are taken from the
// real signatures in pkg/errortypes/errors.go — note NewRateLimitError puts the message
// second, behind a field name.
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
	// ozzo-validation attaches a custom message to any rule via .Error(msg). The same
	// selector is how zap logs (logger.Error("failed to ...")), so this entry alone would
	// pull several thousand internal log lines into the catalog. discardedCalls below
	// separates them: a log call throws its value away, an ozzo rule is built to be passed
	// into validation.Field.
	"Error": 0,
}

// noMessage lists New*Error constructors that carry nothing to translate, so that the
// unrecognised-constructor gate below stays meaningful. Three reasons appear here:
//
//   - containers and non-errors that only match the New*Error name shape
//     (NewMultiError, the Prometheus metrics NewError);
//   - constructors that take structured identifiers and compose their own text
//     (the formula and seeder errors, NewRequestTooLargeError) — the composed text is
//     reported separately as a dynamic message rather than captured here;
//   - Temporal control-plane errors, which drive workflow retry decisions and reach
//     operators through job history and logs, never a translated end-user surface.
//
// Moving one of these into messageArgs is the single edit needed if that judgement changes.
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

// isErrortypesConstructor matches the constructor naming convention so an unlisted one can
// be reported instead of quietly dropped.
func isErrortypesConstructor(name string) bool {
	return strings.HasPrefix(name, "New") && strings.HasSuffix(name, "Error")
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
}

func (e *extractor) walkDir(root string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case "node_modules", "testdata", ".git", "vendor", "mocks":
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

		// A .Error(msg) whose result is dropped is a log statement, not a validation rule.
		if name == "Error" && discarded[call] {
			return true
		}

		msg, ok := stringLiteral(call.Args[idx])
		if !ok {
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

// discardedCalls collects calls whose return value is thrown away — the bare
// `logger.Error("...")` statement shape. Every ozzo rule, by contrast, is constructed to be
// handed to validation.Field, so it always appears in a value position.
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

// calleeName returns the final identifier of a call target: both `NewBusinessError(...)`
// and `errortypes.NewBusinessError(...)` and `multiErr.Add(...)` reduce to one name, which
// is all messageArgs needs to key on.
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

// stringLiteral unquotes a plain string literal, and concatenations of them, so a message
// split across source lines for width is still captured whole. Anything involving a
// variable or a call yields false: those are dynamic and need an explicit parameterised
// message instead.
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

func main() {
	var repoRoot string
	var outPath string
	flag.StringVar(&repoRoot, "root", ".", "repository root")
	flag.StringVar(&outPath, "out", "", "output JSON path (default stdout)")
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
