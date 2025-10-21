package expression

import (
	"fmt"
	"strconv"

	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/pkg/utils"
)

type Node interface {
	Evaluate(ctx *EvaluationContext) (any, error)
	Type() formulatypes.ValueType
	String() string
	Complexity() int
}

type NumberNode struct {
	Value float64
}

func (n *NumberNode) Evaluate(_ *EvaluationContext) (any, error) {
	return n.Value, nil
}

func (n *NumberNode) Type() formulatypes.ValueType {
	return formulatypes.ValueTypeNumber
}

func (n *NumberNode) String() string {
	return fmt.Sprintf("%g", n.Value)
}

func (n *NumberNode) Complexity() int {
	return 1
}

// StringNode represents a string literal
type StringNode struct {
	Value string
}

func (n *StringNode) Evaluate(_ *EvaluationContext) (any, error) {
	return n.Value, nil
}

func (n *StringNode) Type() formulatypes.ValueType {
	return formulatypes.ValueTypeString
}

func (n *StringNode) String() string {
	return fmt.Sprintf("%q", n.Value)
}

func (n *StringNode) Complexity() int {
	return 1
}

type BooleanNode struct {
	Value bool
}

func (n *BooleanNode) Evaluate(_ *EvaluationContext) (any, error) {
	return n.Value, nil
}

func (n *BooleanNode) Type() formulatypes.ValueType {
	return formulatypes.ValueTypeBoolean
}

func (n *BooleanNode) String() string {
	return strconv.FormatBool(n.Value)
}

func (n *BooleanNode) Complexity() int {
	return 1
}

type IdentifierNode struct {
	Name string
}

func (n *IdentifierNode) Evaluate(ctx *EvaluationContext) (any, error) {
	return ctx.ResolveVariable(n.Name)
}

func (n *IdentifierNode) Type() formulatypes.ValueType {
	return formulatypes.ValueTypeAny
}

func (n *IdentifierNode) String() string {
	return n.Name
}

func (n *IdentifierNode) Complexity() int {
	return 2
}

type BinaryOpNode struct {
	Left     Node
	Right    Node
	Operator TokenType
}

func (n *BinaryOpNode) Evaluate(ctx *EvaluationContext) (any, error) {
	if err := ctx.CheckLimits(); err != nil {
		return nil, err
	}

	left, err := n.Left.Evaluate(ctx)
	if err != nil {
		return nil, err
	}

	switch n.Operator { //nolint:exhaustive // all operators are covered
	case TokenAnd:
		if !toBool(left) {
			return false, nil
		}
	case TokenOr:
		if toBool(left) {
			return true, nil
		}
	}

	right, err := n.Right.Evaluate(ctx)
	if err != nil {
		return nil, err
	}

	return evaluateBinaryOp(n.Operator, left, right)
}

func (n *BinaryOpNode) Type() formulatypes.ValueType {
	switch n.Operator { //nolint:exhaustive // all operators are covered
	case TokenEqual,
		TokenNotEqual,
		TokenGreater,
		TokenLess,
		TokenGreaterEqual,
		TokenLessEqual,
		TokenAnd,
		TokenOr:
		return formulatypes.ValueTypeBoolean
	default:
		return formulatypes.ValueTypeNumber
	}
}

func (n *BinaryOpNode) String() string {
	return fmt.Sprintf("(%s %s %s)", n.Left.String(), n.Operator.String(), n.Right.String())
}

func (n *BinaryOpNode) Complexity() int {
	return n.Left.Complexity() + n.Right.Complexity() + 1
}

type UnaryOpNode struct {
	Operand  Node
	Operator TokenType
}

func (n *UnaryOpNode) Evaluate(ctx *EvaluationContext) (any, error) {
	if err := ctx.CheckLimits(); err != nil {
		return nil, err
	}

	operand, err := n.Operand.Evaluate(ctx)
	if err != nil {
		return nil, err
	}

	return evaluateUnaryOp(n.Operator, operand)
}

func (n *UnaryOpNode) Type() formulatypes.ValueType {
	switch n.Operator { //nolint:exhaustive // all operators are covered
	case TokenNot:
		return formulatypes.ValueTypeBoolean
	default:
		return n.Operand.Type()
	}
}

func (n *UnaryOpNode) String() string {
	return fmt.Sprintf("(%s%s)", n.Operator.String(), n.Operand.String())
}

func (n *UnaryOpNode) Complexity() int {
	return n.Operand.Complexity() + 1
}

type ConditionalNode struct {
	Condition Node
	TrueExpr  Node
	FalseExpr Node
}

func (n *ConditionalNode) Evaluate(ctx *EvaluationContext) (any, error) {
	if err := ctx.CheckLimits(); err != nil {
		return nil, err
	}

	condition, err := n.Condition.Evaluate(ctx)
	if err != nil {
		return nil, err
	}

	if toBool(condition) {
		return n.TrueExpr.Evaluate(ctx)
	}
	return n.FalseExpr.Evaluate(ctx)
}

func (n *ConditionalNode) Type() formulatypes.ValueType {
	return n.TrueExpr.Type()
}

func (n *ConditionalNode) String() string {
	return fmt.Sprintf(
		"(%s ? %s : %s)",
		n.Condition.String(),
		n.TrueExpr.String(),
		n.FalseExpr.String(),
	)
}

func (n *ConditionalNode) Complexity() int {
	trueComplexity := n.TrueExpr.Complexity()
	falseComplexity := n.FalseExpr.Complexity()
	maxBranch := max(falseComplexity, trueComplexity)
	return n.Condition.Complexity() + maxBranch + 1
}

type FunctionCallNode struct {
	Name      string
	Arguments []Node
}

func (n *FunctionCallNode) Evaluate(ctx *EvaluationContext) (any, error) {
	if err := ctx.CheckLimits(); err != nil {
		return nil, err
	}

	args := make([]any, len(n.Arguments))
	for i, arg := range n.Arguments {
		val, err := arg.Evaluate(ctx)
		if err != nil {
			return nil, err
		}
		args[i] = val
	}

	return ctx.CallFunction(n.Name, args...)
}

func (n *FunctionCallNode) Type() formulatypes.ValueType {
	return formulatypes.ValueTypeAny
}

func (n *FunctionCallNode) String() string {
	args := make([]string, len(n.Arguments))
	for i, arg := range n.Arguments {
		args[i] = arg.String()
	}
	return fmt.Sprintf("%s(%s)", n.Name, utils.JoinStringsSep(args, ", "))
}

func (n *FunctionCallNode) Complexity() int {
	complexity := 3
	for _, arg := range n.Arguments {
		complexity += arg.Complexity()
	}
	return complexity
}

type ArrayNode struct {
	Elements []Node
}

func (n *ArrayNode) Evaluate(ctx *EvaluationContext) (any, error) {
	if err := ctx.CheckLimits(); err != nil {
		return nil, err
	}

	values := make([]any, len(n.Elements))
	for i, elem := range n.Elements {
		val, err := elem.Evaluate(ctx)
		if err != nil {
			return nil, err
		}
		values[i] = val
	}

	return values, nil
}

func (n *ArrayNode) Type() formulatypes.ValueType {
	return formulatypes.ValueTypeArray
}

func (n *ArrayNode) String() string {
	elements := make([]string, len(n.Elements))
	for i, elem := range n.Elements {
		elements[i] = elem.String()
	}
	return fmt.Sprintf("[%s]", utils.JoinStringsSep(elements, ", "))
}

func (n *ArrayNode) Complexity() int {
	complexity := 1
	for _, elem := range n.Elements {
		complexity += elem.Complexity()
	}
	return complexity
}

type IndexNode struct {
	Array Node
	Index Node
}

func (n *IndexNode) Evaluate(ctx *EvaluationContext) (any, error) {
	if err := ctx.CheckLimits(); err != nil {
		return nil, err
	}

	arrayVal, err := n.Array.Evaluate(ctx)
	if err != nil {
		return nil, err
	}

	indexVal, err := n.Index.Evaluate(ctx)
	if err != nil {
		return nil, err
	}

	idx, ok := indexVal.(float64)
	if !ok {
		return nil, fmt.Errorf("array index must be a number, got %T", indexVal)
	}

	intIdx := int(idx)
	if float64(intIdx) != idx {
		return nil, fmt.Errorf("array index must be an integer, got %v", idx)
	}

	switch arr := arrayVal.(type) {
	case []any:
		if intIdx < 0 || intIdx >= len(arr) {
			return nil, fmt.Errorf("array index out of bounds: %d (array length: %d)", intIdx, len(arr))
		}
		return arr[intIdx], nil
	case string:
		runes := []rune(arr)
		if intIdx < 0 || intIdx >= len(runes) {
			return nil, fmt.Errorf("string index out of bounds: %d (string length: %d)", intIdx, len(runes))
		}
		return string(runes[intIdx]), nil
	default:
		return nil, fmt.Errorf("cannot index %T", arrayVal)
	}
}

func (n *IndexNode) Type() formulatypes.ValueType {
	return formulatypes.ValueTypeAny
}

func (n *IndexNode) String() string {
	return fmt.Sprintf("%s[%s]", n.Array.String(), n.Index.String())
}

func (n *IndexNode) Complexity() int {
	return n.Array.Complexity() + n.Index.Complexity() + 1
}

func toBool(v any) bool {
	switch val := v.(type) {
	case bool:
		return val
	case float64:
		return val != 0
	case string:
		return val != ""
	case []any:
		return len(val) > 0
	default:
		return v != nil
	}
}
