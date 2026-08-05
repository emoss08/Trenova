package mutate

import (
	"go/ast"
	"go/token"
	"go/types"
)

type Kind string

const (
	KindArithmetic Kind = "arithmetic"
	KindBoundary   Kind = "boundary"
	KindEquality   Kind = "equality"
	KindConnector  Kind = "connector"
	KindBoolean    Kind = "boolean"
	KindBranch     Kind = "branch"
	KindIncDec     Kind = "incdec"
)

func Kinds() []Kind {
	out := make([]Kind, 0, len(mutators))
	for _, m := range mutators {
		out = append(out, m.Kind())
	}

	return out
}

// edit is a byte-range replacement in the original source. Mutants are produced
// by splicing text rather than reprinting the AST, so comments, formatting and
// //go: directives survive exactly — losing a //go:embed line would not merely
// reformat the mutant, it would stop it compiling.
type edit struct {
	Offset int
	Length int
	Text   string
}

type candidate struct {
	Kind        Kind
	Pos         token.Pos
	Original    string
	Replacement string
	Edits       []edit
}

type mutator interface {
	Kind() Kind
	candidates(ctx *fileContext, node ast.Node) []candidate
}

type fileContext struct {
	fset   *token.FileSet
	info   *types.Info
	source []byte
}

func (c *fileContext) offset(pos token.Pos) int {
	return c.fset.Position(pos).Offset
}

func (c *fileContext) text(from, to token.Pos) string {
	start, end := c.offset(from), c.offset(to)
	if start < 0 || end > len(c.source) || start >= end {
		return ""
	}

	return string(c.source[start:end])
}

var mutators = []mutator{
	arithmeticMutator{},
	boundaryMutator{},
	equalityMutator{},
	connectorMutator{},
	booleanMutator{},
	branchMutator{},
	incDecMutator{},
}

func replaceToken(kind Kind, ctx *fileContext, pos token.Pos, from, to token.Token) candidate {
	return candidate{
		Kind:        kind,
		Pos:         pos,
		Original:    from.String(),
		Replacement: to.String(),
		Edits: []edit{{
			Offset: ctx.offset(pos),
			Length: len(from.String()),
			Text:   to.String(),
		}},
	}
}

type arithmeticMutator struct{}

func (arithmeticMutator) Kind() Kind { return KindArithmetic }

var arithmeticSwaps = map[token.Token]token.Token{
	token.ADD: token.SUB,
	token.SUB: token.ADD,
	token.MUL: token.QUO,
	token.QUO: token.MUL,
	token.REM: token.MUL,
}

func (m arithmeticMutator) candidates(ctx *fileContext, node ast.Node) []candidate {
	expr, ok := node.(*ast.BinaryExpr)
	if !ok {
		return nil
	}

	to, ok := arithmeticSwaps[expr.Op]
	if !ok {
		return nil
	}

	// `+` is also string concatenation, so swapping it for `-` on strings would not
	// compile. Only provably numeric operands qualify.
	if !isNumeric(ctx.info, expr.X) || !isNumeric(ctx.info, expr.Y) {
		return nil
	}

	return []candidate{replaceToken(m.Kind(), ctx, expr.OpPos, expr.Op, to)}
}

type boundaryMutator struct{}

func (boundaryMutator) Kind() Kind { return KindBoundary }

var boundarySwaps = map[token.Token]token.Token{
	token.LSS: token.LEQ,
	token.LEQ: token.LSS,
	token.GTR: token.GEQ,
	token.GEQ: token.GTR,
}

func (m boundaryMutator) candidates(ctx *fileContext, node ast.Node) []candidate {
	expr, ok := node.(*ast.BinaryExpr)
	if !ok {
		return nil
	}

	to, ok := boundarySwaps[expr.Op]
	if !ok {
		return nil
	}

	return []candidate{replaceToken(m.Kind(), ctx, expr.OpPos, expr.Op, to)}
}

type equalityMutator struct{}

func (equalityMutator) Kind() Kind { return KindEquality }

func (m equalityMutator) candidates(ctx *fileContext, node ast.Node) []candidate {
	expr, ok := node.(*ast.BinaryExpr)
	if !ok {
		return nil
	}

	switch expr.Op {
	case token.EQL:
		return []candidate{replaceToken(m.Kind(), ctx, expr.OpPos, token.EQL, token.NEQ)}
	case token.NEQ:
		return []candidate{replaceToken(m.Kind(), ctx, expr.OpPos, token.NEQ, token.EQL)}
	default:
		return nil
	}
}

type connectorMutator struct{}

func (connectorMutator) Kind() Kind { return KindConnector }

func (m connectorMutator) candidates(ctx *fileContext, node ast.Node) []candidate {
	expr, ok := node.(*ast.BinaryExpr)
	if !ok {
		return nil
	}

	switch expr.Op {
	case token.LAND:
		return []candidate{replaceToken(m.Kind(), ctx, expr.OpPos, token.LAND, token.LOR)}
	case token.LOR:
		return []candidate{replaceToken(m.Kind(), ctx, expr.OpPos, token.LOR, token.LAND)}
	default:
		return nil
	}
}

type booleanMutator struct{}

func (booleanMutator) Kind() Kind { return KindBoolean }

func (m booleanMutator) candidates(ctx *fileContext, node ast.Node) []candidate {
	ident, ok := node.(*ast.Ident)
	if !ok {
		return nil
	}

	var to string
	switch ident.Name {
	case "true":
		to = "false"
	case "false":
		to = "true"
	default:
		return nil
	}

	// true and false are ordinary identifiers in Go and can be shadowed; only the
	// universe constants may be flipped.
	if !isUniverseBool(ctx.info, ident) {
		return nil
	}

	return []candidate{{
		Kind:        m.Kind(),
		Pos:         ident.NamePos,
		Original:    ident.Name,
		Replacement: to,
		Edits: []edit{{
			Offset: ctx.offset(ident.NamePos),
			Length: len(ident.Name),
			Text:   to,
		}},
	}}
}

type branchMutator struct{}

func (branchMutator) Kind() Kind { return KindBranch }

// candidates forces a branch never or always taken. Deleting the branch would be
// the obvious mutation, but removing `if err != nil { return err }` routinely
// leaves `err` unused and the mutant fails to compile. Rewriting the condition
// keeps every identifier live and generalises to any branch.
func (m branchMutator) candidates(ctx *fileContext, node ast.Node) []candidate {
	stmt, ok := node.(*ast.IfStmt)
	if !ok || stmt.Cond == nil {
		return nil
	}
	if !isBoolean(ctx.info, stmt.Cond) {
		return nil
	}

	condText := ctx.text(stmt.Cond.Pos(), stmt.Cond.End())
	if condText == "" {
		return nil
	}

	start := ctx.offset(stmt.Cond.Pos())
	length := ctx.offset(stmt.Cond.End()) - start

	force := func(suffix string) candidate {
		return candidate{
			Kind:        m.Kind(),
			Pos:         stmt.Cond.Pos(),
			Original:    condText,
			Replacement: "(" + condText + ")" + suffix,
			Edits: []edit{{
				Offset: start,
				Length: length,
				Text:   "(" + condText + ")" + suffix,
			}},
		}
	}

	return []candidate{force(" && false"), force(" || true")}
}

type incDecMutator struct{}

func (incDecMutator) Kind() Kind { return KindIncDec }

func (m incDecMutator) candidates(ctx *fileContext, node ast.Node) []candidate {
	stmt, ok := node.(*ast.IncDecStmt)
	if !ok {
		return nil
	}
	if !isNumeric(ctx.info, stmt.X) {
		return nil
	}

	to := token.INC
	if stmt.Tok == token.INC {
		to = token.DEC
	}

	return []candidate{replaceToken(m.Kind(), ctx, stmt.TokPos, stmt.Tok, to)}
}

func isNumeric(info *types.Info, expr ast.Expr) bool {
	basic, ok := underlyingBasic(info, expr)
	if !ok {
		return false
	}

	return basic.Info()&(types.IsInteger|types.IsFloat|types.IsComplex) != 0
}

func isBoolean(info *types.Info, expr ast.Expr) bool {
	basic, ok := underlyingBasic(info, expr)
	if !ok {
		return false
	}

	return basic.Info()&types.IsBoolean != 0
}

func underlyingBasic(info *types.Info, expr ast.Expr) (*types.Basic, bool) {
	if info == nil {
		return nil, false
	}
	typ := info.TypeOf(expr)
	if typ == nil {
		return nil, false
	}
	basic, ok := typ.Underlying().(*types.Basic)

	return basic, ok
}

func isUniverseBool(info *types.Info, ident *ast.Ident) bool {
	if info == nil {
		return false
	}
	obj := info.ObjectOf(ident)
	if obj == nil {
		return false
	}

	return obj.Parent() == types.Universe
}
