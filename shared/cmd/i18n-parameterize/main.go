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
}

var skipDirs = map[string]struct{}{
	".git":         {},
	"node_modules": {},
	"testdata":     {},
}

type edit struct {
	start int
	end   int
	text  string
}

type rewrite struct {
	message string
	args    []string
}

type failure struct {
	where  string
	reason string
}

func main() {
	write := flag.Bool("write", false, "apply the rewrites instead of reporting them")
	flag.Parse()

	roots := flag.Args()
	if len(roots) == 0 {
		roots = []string{"."}
	}

	var files, changed, sites int
	var failures []failure

	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if _, skip := skipDirs[d.Name()]; skip {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}

			files++
			n, bad, rewriteErr := processFile(path, *write)
			if rewriteErr != nil {
				return rewriteErr
			}
			failures = append(failures, bad...)
			if n > 0 {
				changed++
				sites += n
			}
			return nil
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	sort.Slice(failures, func(i, j int) bool { return failures[i].where < failures[j].where })
	for _, f := range failures {
		fmt.Printf("%s\t%s\n", f.where, f.reason)
	}

	verb := "would rewrite"
	if *write {
		verb = "rewrote"
	}
	fmt.Printf("i18n-parameterize: %s %d sites in %d/%d files, %d left for hand review\n",
		verb, sites, changed, files, len(failures))
}

func processFile(path string, write bool) (int, []failure, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return 0, nil, err
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	if err != nil {
		return 0, nil, nil
	}

	base := fset.File(file.Pos()).Base()
	source := string(src)
	offset := func(pos token.Pos) int { return int(pos) - base }

	var edits []edit
	var failures []failure

	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		idx, known := messageArgs[calleeName(call.Fun)]
		if !known || idx >= len(call.Args) || call.Ellipsis.IsValid() {
			return true
		}

		arg := call.Args[idx]
		if _, isLiteral := stringLiteral(arg); isLiteral {
			return true
		}

		where := fmt.Sprintf("%s:%d", filepath.ToSlash(path), fset.Position(call.Pos()).Line)

		built, reason := build(arg, source, offset)
		if built == nil {
			if reason != "" {
				failures = append(failures, failure{where: where, reason: reason})
			}
			return true
		}

		last := call.Args[len(call.Args)-1]
		edits = append(edits,
			edit{start: offset(arg.Pos()), end: offset(arg.End()), text: strconv.Quote(built.message)},
			edit{start: offset(last.End()), end: offset(last.End()), text: ", " + strings.Join(built.args, ", ")},
		)
		return true
	})

	if len(edits) == 0 {
		return 0, failures, nil
	}

	if write {
		if err := os.WriteFile(path, []byte(splice(source, edits)), 0o644); err != nil {
			return 0, failures, err
		}
	}

	return len(edits) / 2, failures, nil
}

func build(expr ast.Expr, source string, offset func(token.Pos) int) (*rewrite, string) {
	switch v := expr.(type) {
	case *ast.CallExpr:
		if calleeName(v.Fun) != "Sprintf" || len(v.Args) == 0 {
			return nil, ""
		}
		format, ok := stringLiteral(v.Args[0])
		if !ok {
			return nil, ""
		}
		return fromFormat(format, v.Args[1:], source, offset)
	case *ast.BinaryExpr:
		return fromConcat(v, source, offset)
	default:
		return nil, ""
	}
}

func fromFormat(
	format string,
	args []ast.Expr,
	source string,
	offset func(token.Pos) int,
) (*rewrite, string) {
	if !hasLetters(format) {
		return nil, ""
	}
	if strings.ContainsAny(format, "{}") {
		return nil, "format string already contains braces"
	}

	var message strings.Builder
	out := make([]string, 0, len(args))

	for i := 0; i < len(format); i++ {
		if format[i] != '%' {
			message.WriteByte(format[i])
			continue
		}

		end := i + 1
		for end < len(format) && strings.IndexByte("+-# 0123456789.*", format[end]) >= 0 {
			end++
		}
		if end >= len(format) {
			return nil, "format string ends in an incomplete verb"
		}

		verb := format[i : end+1]
		i = end

		if verb == "%%" {
			message.WriteByte('%')
			continue
		}
		if len(out) >= len(args) {
			return nil, "format string has more verbs than arguments"
		}

		text := strings.TrimSpace(source[offset(args[len(out)].Pos()):offset(args[len(out)].End())])

		switch verb {
		case "%s", "%v", "%d":
			message.WriteString(placeholder(len(out)))
			out = append(out, text)
		case "%q":
			message.WriteString(`"` + placeholder(len(out)) + `"`)
			out = append(out, text)
		default:
			message.WriteString(placeholder(len(out)))
			out = append(out, fmt.Sprintf("fmt.Sprintf(%q, %s)", verb, text))
		}
	}

	if len(out) != len(args) {
		return nil, "format string consumes fewer arguments than it is given"
	}
	if len(out) == 0 {
		return nil, ""
	}

	return &rewrite{message: message.String(), args: out}, ""
}

func fromConcat(expr *ast.BinaryExpr, source string, offset func(token.Pos) int) (*rewrite, string) {
	parts, ok := flatten(expr)
	if !ok {
		return nil, ""
	}

	var message strings.Builder
	out := make([]string, 0, len(parts))
	prose := false

	for _, part := range parts {
		if literal, isLiteral := stringLiteral(part); isLiteral {
			if strings.ContainsAny(literal, "{}") {
				return nil, "concatenated literal already contains braces"
			}
			if hasLetters(literal) {
				prose = true
			}
			message.WriteString(literal)
			continue
		}
		message.WriteString(placeholder(len(out)))
		out = append(out, strings.TrimSpace(source[offset(part.Pos()):offset(part.End())]))
	}

	if !prose || len(out) == 0 {
		return nil, ""
	}

	return &rewrite{message: message.String(), args: out}, ""
}

func flatten(expr ast.Expr) ([]ast.Expr, bool) {
	binary, ok := expr.(*ast.BinaryExpr)
	if !ok {
		return []ast.Expr{expr}, true
	}
	if binary.Op != token.ADD {
		return nil, false
	}

	left, ok := flatten(binary.X)
	if !ok {
		return nil, false
	}
	right, ok := flatten(binary.Y)
	if !ok {
		return nil, false
	}

	return append(left, right...), true
}

func placeholder(index int) string {
	return "{" + strconv.Itoa(index) + "}"
}

func splice(source string, edits []edit) string {
	sort.SliceStable(edits, func(i, j int) bool { return edits[i].start < edits[j].start })

	var out strings.Builder
	out.Grow(len(source) + 64*len(edits))

	cursor := 0
	for _, e := range edits {
		if e.start < cursor {
			continue
		}
		out.WriteString(source[cursor:e.start])
		out.WriteString(e.text)
		cursor = e.end
	}
	out.WriteString(source[cursor:])

	return out.String()
}

func calleeName(fun ast.Expr) string {
	switch v := fun.(type) {
	case *ast.Ident:
		return v.Name
	case *ast.SelectorExpr:
		return v.Sel.Name
	default:
		return ""
	}
}

func stringLiteral(expr ast.Expr) (string, bool) {
	basic, ok := expr.(*ast.BasicLit)
	if !ok || basic.Kind != token.STRING {
		return "", false
	}
	value, err := strconv.Unquote(basic.Value)
	if err != nil {
		return "", false
	}
	return value, true
}

func hasLetters(s string) bool {
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			return true
		}
	}
	return false
}
