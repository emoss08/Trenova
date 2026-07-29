package templateengine

import (
	"fmt"
	"maps"
	"slices"
	"sort"
	"strings"
	"text/template/parse"
)

// The analyzer walks the parse tree instead of waiting for execution.
//
// The old invoice-email renderer discovered unknown variables by rendering and
// noticing what did not substitute, which means a typo was only detectable on a
// real invoice with real data. Walking the tree catches the same class of mistake
// while the author is still in the editor, and catches it in branches that the
// sample data happens not to exercise.

// analysis is what a tree walk found.
type analysis struct {
	// paths are the dotted field chains the template reads, e.g. "BillTo.Name".
	paths map[string]struct{}
	// functions are the identifiers it calls.
	functions map[string]struct{}
	// rangeVars tracks variables bound by range/with so their field accesses are
	// resolved against the collection element, not the root context.
	nestedTemplates []string
}

func newAnalysis() *analysis {
	return &analysis{
		paths:     make(map[string]struct{}, 16),
		functions: make(map[string]struct{}, 8),
	}
}

// Paths returns the referenced field chains, sorted.
func (a *analysis) Paths() []string {
	out := make([]string, 0, len(a.paths))
	for p := range a.paths {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

// analyzeTree walks a parsed template and collects everything it references.
func analyzeTree(tree *parse.Tree) *analysis {
	a := newAnalysis()
	if tree == nil || tree.Root == nil {
		return a
	}
	a.walk(tree.Root, nil)
	return a
}

// scope tracks the variables bound by enclosing range/with blocks.
//
// Inside {{range .ChargeRows}}, a bare {{.Description}} refers to an element of
// ChargeRows, so it must be recorded as "ChargeRows.Description" for the catalog
// check to mean anything.
type scope struct {
	// dotPrefix is the path the current dot resolves to, empty at the root.
	dotPrefix string
	// vars maps a $name to the path it was bound to.
	vars map[string]string
}

func (s *scope) child(dotPrefix string) *scope {
	vars := make(map[string]string, len(s.vars)+1)
	maps.Copy(vars, s.vars)
	return &scope{dotPrefix: dotPrefix, vars: vars}
}

func (s *scope) resolve(ident []string) string {
	joined := strings.Join(ident, ".")
	if s == nil || s.dotPrefix == "" {
		return joined
	}
	if joined == "" {
		return s.dotPrefix
	}
	return s.dotPrefix + "." + joined
}

// A parse-tree walk is inherently one branch per node type; splitting the switch
// would scatter it without simplifying anything.
func (a *analysis) walk(node parse.Node, sc *scope) {
	if node == nil {
		return
	}
	if sc == nil {
		sc = &scope{vars: map[string]string{}}
	}

	switch n := node.(type) {
	case *parse.ListNode:
		if n == nil {
			return
		}
		for _, child := range n.Nodes {
			a.walk(child, sc)
		}

	case *parse.ActionNode:
		a.walkPipe(n.Pipe, sc)

	case *parse.IfNode:
		a.walkBranch(&n.BranchNode, sc, sc)

	case *parse.WithNode:
		// with rebinds dot to the pipeline's value. walkBranch reads the pipeline.
		inner := sc.child(a.pipePath(n.Pipe, sc))
		a.walkBranch(&n.BranchNode, inner, sc)

	case *parse.RangeNode:
		// range rebinds dot to an element of the collection. The collection path
		// doubles as the element path: a catalog declares "ChargeRows" once with
		// the element's fields underneath it.
		inner := sc.child(a.pipePath(n.Pipe, sc))
		a.bindPipeVars(n.Pipe, inner)
		a.walkBranch(&n.BranchNode, inner, sc)

	case *parse.TemplateNode:
		// Recorded, then rejected by the caller: a version is one template, and
		// {{template}} would let one reach outside itself.
		a.nestedTemplates = append(a.nestedTemplates, n.Name)

	case *parse.PipeNode:
		a.walkPipe(n, sc)

	case *parse.TextNode, *parse.CommentNode:

	default:
	}
}

// walkBranch walks a branch's condition and both arms.
//
// The condition is walked with the outer scope: {{ if .Truncated }} reads a field
// whether or not the body does, and a template whose only use of a variable is a
// condition still depends on it. Missing that made "is this path referenced?" answer
// no for every field used purely to decide whether a section appears.
//
// The body gets the inner scope and the else the outer one, since else sits outside
// whatever the branch bound.
func (a *analysis) walkBranch(branch *parse.BranchNode, inner, outer *scope) {
	a.walkPipe(branch.Pipe, outer)

	if branch.List != nil {
		a.walk(branch.List, inner)
	}
	if branch.ElseList != nil {
		a.walk(branch.ElseList, outer)
	}
}

func (a *analysis) walkPipe(pipe *parse.PipeNode, sc *scope) {
	if pipe == nil {
		return
	}

	for _, cmd := range pipe.Cmds {
		for _, arg := range cmd.Args {
			a.walkArg(arg, sc)
		}
	}
}

func (a *analysis) walkArg(arg parse.Node, sc *scope) {
	switch v := arg.(type) {
	case *parse.FieldNode:
		a.record(sc.resolve(v.Ident))

	case *parse.DotNode:
		if sc.dotPrefix != "" {
			a.record(sc.dotPrefix)
		}

	case *parse.VariableNode:
		// $x.Field resolves through whatever $x was bound to.
		if len(v.Ident) == 0 {
			return
		}
		base, ok := sc.vars[v.Ident[0]]
		if !ok {
			// $ is the root context.
			if v.Ident[0] == "$" && len(v.Ident) > 1 {
				a.record(strings.Join(v.Ident[1:], "."))
			}
			return
		}
		if len(v.Ident) == 1 {
			a.record(base)
			return
		}
		a.record(base + "." + strings.Join(v.Ident[1:], "."))

	case *parse.IdentifierNode:
		a.functions[v.Ident] = struct{}{}

	case *parse.PipeNode:
		a.walkPipe(v, sc)

	case *parse.ChainNode:
		// (expr).Field.Field
		a.walkArg(v.Node, sc)

	default:
	}
}

// pipePath returns the path a range/with pipeline resolves to, when it is a plain
// field reference. A computed pipeline has no static path, so nested accesses in
// that block cannot be checked and are left alone rather than guessed at.
func (a *analysis) pipePath(pipe *parse.PipeNode, sc *scope) string {
	if pipe == nil || len(pipe.Cmds) != 1 {
		return ""
	}
	cmd := pipe.Cmds[0]
	if len(cmd.Args) != 1 {
		return ""
	}

	switch v := cmd.Args[0].(type) {
	case *parse.FieldNode:
		return sc.resolve(v.Ident)
	case *parse.VariableNode:
		if len(v.Ident) == 0 {
			return ""
		}
		if base, ok := sc.vars[v.Ident[0]]; ok {
			if len(v.Ident) == 1 {
				return base
			}
			return base + "." + strings.Join(v.Ident[1:], ".")
		}
		return ""
	default:
		return ""
	}
}

// bindPipeVars records the loop variables of a range so `{{$i, $row := ...}}`
// followed by `{{$row.Field}}` resolves.
func (a *analysis) bindPipeVars(pipe *parse.PipeNode, sc *scope) {
	if pipe == nil {
		return
	}
	switch len(pipe.Decl) {
	case 1:
		// {{range $v := .Items}} — $v is the element.
		sc.vars[pipe.Decl[0].Ident[0]] = sc.dotPrefix
	case 2:
		// {{range $i, $v := .Items}} — $i is the index, $v the element.
		sc.vars[pipe.Decl[1].Ident[0]] = sc.dotPrefix
	default:
	}
}

func (a *analysis) record(path string) {
	if path == "" {
		return
	}
	a.paths[path] = struct{}{}
}

// checkPaths compares referenced paths against what the kind publishes.
//
// Unknown paths are a warning while drafting and an error when publishing. The
// asymmetry is deliberate: an author mid-edit should not be blocked by a variable
// they are halfway through typing, but a template that reaches a customer with a
// path that resolves to nothing has a blank where a number belongs.
func checkPaths(
	field string,
	referenced []string,
	allowed []string,
	strict bool,
) Diagnostics {
	if len(allowed) == 0 {
		return nil
	}

	allowedSet := make(map[string]struct{}, len(allowed))
	for _, p := range allowed {
		allowedSet[p] = struct{}{}
	}

	severity := SeverityWarning
	if strict {
		severity = SeverityError
	}

	var diags Diagnostics
	for _, path := range referenced {
		if _, ok := allowedSet[path]; ok {
			continue
		}
		// A path is also acceptable when it descends into a declared one: a
		// catalog entry for "BillTo" covers "BillTo.Name" without having to
		// enumerate every leaf.
		if hasDeclaredPrefix(path, allowedSet) {
			continue
		}

		diags = append(diags, Diagnostic{
			Severity: severity,
			Code:     CodeUnknownVariable,
			Field:    field,
			Message: fmt.Sprintf(
				"%q is not available on this template. %s",
				path, suggestClosest(path, allowed),
			),
		})
	}

	return diags
}

func hasDeclaredPrefix(path string, allowed map[string]struct{}) bool {
	for i := len(path) - 1; i > 0; i-- {
		if path[i] != '.' {
			continue
		}
		if _, ok := allowed[path[:i]]; ok {
			return true
		}
	}
	return false
}

// checkRequiredPaths enforces the paths a kind declares mandatory.
//
// This is what stops an organization from publishing an invoice that no longer
// prints its total or its remit-to address. Those fields are not styling; leaving
// one out produces a document a customer can refuse to pay against.
func checkRequiredPaths(field string, referenced, required []string) Diagnostics {
	if len(required) == 0 {
		return nil
	}

	var diags Diagnostics
	for _, want := range required {
		if slices.Contains(referenced, want) {
			continue
		}
		// A reference to a child counts: printing BillTo.Name satisfies BillTo.
		found := false
		for _, got := range referenced {
			if strings.HasPrefix(got, want+".") {
				found = true
				break
			}
		}
		if !found {
			diags = append(diags, Diagnostic{
				Severity: SeverityError,
				Code:     CodeRequiredVariableMissing,
				Field:    field,
				Message: fmt.Sprintf(
					"%q is required on this template and is no longer referenced", want,
				),
			})
		}
	}

	return diags
}

// checkFunctions rejects calls to anything outside the allowed set.
//
// This is not redundant with the parse-time "function not defined" failure. Go
// installs html, js, urlquery, print, printf, and friends as *builtins*, and a
// FuncMap cannot remove them — {{ .Memo | html }} parses whether we want it to or
// not. Those names are harmless in that none of them produce unescaped output,
// but they let an author pick an escaping context, which is the one decision
// contextual auto-escaping needs to own. So the allowlist is enforced here, on
// the parsed tree, where builtins are visible.
func checkFunctions(field string, called map[string]struct{}) Diagnostics {
	allowed := allowedFunctionSet()

	names := make([]string, 0, len(called))
	for name := range called {
		names = append(names, name)
	}
	sort.Strings(names)

	var diags Diagnostics
	for _, name := range names {
		if _, ok := allowed[name]; ok {
			continue
		}
		diags = append(diags, errorDiag(CodeUnknownFunction, field, fmt.Sprintf(
			"%q is not a function you can call in a template. Available functions: %s",
			name, strings.Join(AllowedFunctionNames(), ", "),
		)))
	}

	return diags
}

// maxSuggestionDistance bounds how far a typo can be from a real path before we
// stop guessing at what was meant.
const maxSuggestionDistance = 3

// suggestClosest names the nearest available path, because "Totl is not
// available" is far less useful than "did you mean Total?".
func suggestClosest(path string, allowed []string) string {
	best, bestDistance := "", maxSuggestionDistance+1

	target := strings.ToLower(path)
	for _, candidate := range allowed {
		d := levenshtein(target, strings.ToLower(candidate))
		if d < bestDistance {
			best, bestDistance = candidate, d
		}
	}

	if best == "" {
		return "Use the variable list to see what this template can reference."
	}
	return fmt.Sprintf("Did you mean %q?", best)
}

// levenshtein is the standard two-row edit distance.
func levenshtein(a, b string) int {
	if a == b {
		return 0
	}
	if a == "" {
		return len(b)
	}
	if b == "" {
		return len(a)
	}

	prev := make([]int, len(b)+1)
	curr := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}

	for i := 1; i <= len(a); i++ {
		curr[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min(curr[j-1]+1, prev[j]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}

	return prev[len(b)]
}
