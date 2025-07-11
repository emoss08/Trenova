package expression

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/pkg/formula/variables"
)

// * TraceStep represents a single step in the evaluation trace
type TraceStep struct {
	Step        string         `json:"step"`
	Description string         `json:"description"`
	Result      string         `json:"result"`
	Value       any            `json:"value,omitempty"`
	NodeType    string         `json:"nodeType,omitempty"`
	Children    []TraceStep    `json:"children,omitempty"`
}

// * TracingEvaluator wraps an evaluator to capture evaluation steps
type TracingEvaluator struct {
	evaluator *Evaluator
}

// * NewTracingEvaluator creates a new tracing evaluator
func NewTracingEvaluator(vars *variables.Registry) *TracingEvaluator {
	return &TracingEvaluator{
		evaluator: NewEvaluator(vars),
	}
}

// * EvaluateWithTrace evaluates an expression and returns both the result and trace
func (te *TracingEvaluator) EvaluateWithTrace(
	ctx context.Context,
	expr string,
	varCtx variables.VariableContext,
) (float64, []TraceStep, error) {
	// Create trace for this evaluation
	trace := make([]TraceStep, 0)

	// Step 1: Tokenization
	trace = append(trace, TraceStep{
		Step:        "Tokenization",
		Description: fmt.Sprintf("Tokenizing expression: %s", expr),
		Result:      "In Progress",
	})

	tokenizer := NewTokenizer(expr)
	tokens, err := tokenizer.Tokenize()
	if err != nil {
		trace[len(trace)-1].Result = fmt.Sprintf("Failed: %v", err)
		return 0, trace, fmt.Errorf("tokenization error: %w", err)
	}

	trace[len(trace)-1].Result = fmt.Sprintf("Success: %d tokens", len(tokens))
	trace[len(trace)-1].Value = tokensToString(tokens)

	// Step 2: Parsing
	trace = append(trace, TraceStep{
		Step:        "Parsing",
		Description: "Building Abstract Syntax Tree (AST)",
		Result:      "In Progress",
	})

	parser := NewParser(tokens)
	ast, err := parser.Parse()
	if err != nil {
		trace[len(trace)-1].Result = fmt.Sprintf("Failed: %v", err)
		return 0, trace, fmt.Errorf("parse error: %w", err)
	}

	trace[len(trace)-1].Result = "Success"
	trace[len(trace)-1].Value = fmt.Sprintf("AST complexity: %d", ast.Complexity())

	// Step 3: Variable extraction
	vars := te.evaluator.extractVariables(ast)
	trace = append(trace, TraceStep{
		Step:        "Variable Extraction",
		Description: "Identifying variables used in expression",
		Result:      fmt.Sprintf("Found %d variables", len(vars)),
		Value:       vars,
	})

	// Step 4: Variable resolution
	varResolutionStep := TraceStep{
		Step:        "Variable Resolution",
		Description: "Loading variable values from context",
		Result:      "In Progress",
		Children:    make([]TraceStep, 0),
	}

	for _, varName := range vars {
		varStep := TraceStep{
			Step:        fmt.Sprintf("Resolve '%s'", varName),
			Description: fmt.Sprintf("Looking up variable '%s'", varName),
		}

		if variable, err := te.evaluator.variables.Get(varName); err == nil {
			if value, err := variable.Resolve(varCtx); err == nil {
				varStep.Result = "Success"
				varStep.Value = value
			} else {
				varStep.Result = fmt.Sprintf("Failed: %v", err)
			}
		} else {
			varStep.Result = "Not found in registry"
		}

		varResolutionStep.Children = append(varResolutionStep.Children, varStep)
	}

	varResolutionStep.Result = "Complete"
	trace = append(trace, varResolutionStep)

	// Step 5: Evaluation
	evalStep := TraceStep{
		Step:        "Expression Evaluation",
		Description: "Evaluating the AST with resolved variables",
		Result:      "In Progress",
	}

	// Create tracing context
	evalCtx := &tracingEvaluationContext{
		EvaluationContext: NewEvaluationContext(ctx, varCtx).
			WithFunctions(te.evaluator.functions).
			WithVariableRegistry(te.evaluator.variables),
		trace:            &evalStep,
	}

	result, err := te.evaluateNode(ast, evalCtx)
	if err != nil {
		evalStep.Result = fmt.Sprintf("Failed: %v", err)
		trace = append(trace, evalStep)
		return 0, trace, fmt.Errorf("evaluation error: %w", err)
	}

	evalStep.Result = "Success"
	evalStep.Value = result
	trace = append(trace, evalStep)

	// Step 6: Type conversion
	trace = append(trace, TraceStep{
		Step:        "Type Conversion",
		Description: "Converting result to float64",
		Result:      "Success",
		Value:       result,
	})

	// Convert result
	switch v := result.(type) {
	case float64:
		return v, trace, nil
	case int:
		return float64(v), trace, nil
	case bool:
		if v {
			return 1, trace, nil
		}
		return 0, trace, nil
	default:
		trace[len(trace)-1].Result = fmt.Sprintf("Failed: unexpected type %T", result)
		return 0, trace, fmt.Errorf("expression must return a numeric value, got %T", result)
	}
}

// * tracingEvaluationContext wraps EvaluationContext to capture evaluation steps
type tracingEvaluationContext struct {
	*EvaluationContext
	trace *TraceStep
}

// * evaluateNode evaluates a node and captures the trace
func (te *TracingEvaluator) evaluateNode(node Node, ctx *tracingEvaluationContext) (any, error) {
	nodeTrace := TraceStep{
		NodeType: fmt.Sprintf("%T", node),
		Children: make([]TraceStep, 0),
	}

	var result any
	var err error

	switch n := node.(type) {
	case *NumberNode:
		nodeTrace.Step = "Number"
		nodeTrace.Description = fmt.Sprintf("Literal number: %v", n.Value)
		result = n.Value
		nodeTrace.Result = "Success"
		nodeTrace.Value = result

	case *IdentifierNode:
		nodeTrace.Step = "Variable"
		nodeTrace.Description = fmt.Sprintf("Resolving variable: %s", n.Name)
		result, err = n.Evaluate(ctx.EvaluationContext)
		if err != nil {
			nodeTrace.Result = fmt.Sprintf("Failed: %v", err)
		} else {
			nodeTrace.Result = "Success"
			nodeTrace.Value = result
		}

	case *BinaryOpNode:
		nodeTrace.Step = fmt.Sprintf("Binary Operation: %s", n.Operator)
		nodeTrace.Description = fmt.Sprintf("Evaluating %s operation", n.Operator)

		// Evaluate left operand
		leftCtx := &tracingEvaluationContext{
			EvaluationContext: ctx.EvaluationContext,
			trace:            &TraceStep{},
		}
		leftVal, leftErr := te.evaluateNode(n.Left, leftCtx)
		nodeTrace.Children = append(nodeTrace.Children, *leftCtx.trace)

		if leftErr != nil {
			nodeTrace.Result = fmt.Sprintf("Failed: left operand error: %v", leftErr)
			err = leftErr
			break
		}

		// Evaluate right operand
		rightCtx := &tracingEvaluationContext{
			EvaluationContext: ctx.EvaluationContext,
			trace:            &TraceStep{},
		}
		rightVal, rightErr := te.evaluateNode(n.Right, rightCtx)
		nodeTrace.Children = append(nodeTrace.Children, *rightCtx.trace)

		if rightErr != nil {
			nodeTrace.Result = fmt.Sprintf("Failed: right operand error: %v", rightErr)
			err = rightErr
			break
		}

		// Perform the operation directly
		result, err = evaluateBinaryOp(n.Operator, leftVal, rightVal)
		if err != nil {
			nodeTrace.Result = fmt.Sprintf("Failed: %v", err)
		} else {
			nodeTrace.Result = fmt.Sprintf("Success: %v %s %v = %v", leftVal, n.Operator, rightVal, result)
			nodeTrace.Value = result
		}

	case *FunctionCallNode:
		nodeTrace.Step = fmt.Sprintf("Function Call: %s", n.Name)
		nodeTrace.Description = fmt.Sprintf("Calling function %s with %d arguments", n.Name, len(n.Arguments))

		// Evaluate arguments
		for i, arg := range n.Arguments {
			argCtx := &tracingEvaluationContext{
				EvaluationContext: ctx.EvaluationContext,
				trace:            &TraceStep{},
			}
			_, argErr := te.evaluateNode(arg, argCtx)
			argCtx.trace.Step = fmt.Sprintf("Argument %d", i+1)
			nodeTrace.Children = append(nodeTrace.Children, *argCtx.trace)
			if argErr != nil {
				nodeTrace.Result = fmt.Sprintf("Failed: argument %d error: %v", i+1, argErr)
				return nil, argErr
			}
		}

		result, err = n.Evaluate(ctx.EvaluationContext)
		if err != nil {
			nodeTrace.Result = fmt.Sprintf("Failed: %v", err)
		} else {
			nodeTrace.Result = "Success"
			nodeTrace.Value = result
		}

	case *UnaryOpNode:
		nodeTrace.Step = fmt.Sprintf("Unary Operation: %s", n.Operator)
		nodeTrace.Description = fmt.Sprintf("Evaluating unary %s operation", n.Operator)

		// Evaluate operand
		operandCtx := &tracingEvaluationContext{
			EvaluationContext: ctx.EvaluationContext,
			trace:            &TraceStep{},
		}
		operandVal, operandErr := te.evaluateNode(n.Operand, operandCtx)
		nodeTrace.Children = append(nodeTrace.Children, *operandCtx.trace)

		if operandErr != nil {
			nodeTrace.Result = fmt.Sprintf("Failed: operand error: %v", operandErr)
			err = operandErr
			break
		}

		// Perform the operation
		result, err = evaluateUnaryOp(n.Operator, operandVal)
		if err != nil {
			nodeTrace.Result = fmt.Sprintf("Failed: %v", err)
		} else {
			nodeTrace.Result = fmt.Sprintf("Success: %s%v = %v", n.Operator, operandVal, result)
			nodeTrace.Value = result
		}

	case *ConditionalNode:
		nodeTrace.Step = "Conditional Expression"
		nodeTrace.Description = "Evaluating ternary conditional"

		// Evaluate condition
		condCtx := &tracingEvaluationContext{
			EvaluationContext: ctx.EvaluationContext,
			trace:            &TraceStep{Step: "Condition"},
		}
		condVal, condErr := te.evaluateNode(n.Condition, condCtx)
		nodeTrace.Children = append(nodeTrace.Children, *condCtx.trace)

		if condErr != nil {
			nodeTrace.Result = fmt.Sprintf("Failed: condition error: %v", condErr)
			err = condErr
			break
		}

		// Evaluate the appropriate branch
		condTrue := toBool(condVal)
		var branchNode Node
		var branchName string
		if condTrue {
			branchNode = n.TrueExpr
			branchName = "True Branch"
		} else {
			branchNode = n.FalseExpr
			branchName = "False Branch"
		}

		branchCtx := &tracingEvaluationContext{
			EvaluationContext: ctx.EvaluationContext,
			trace:            &TraceStep{Step: branchName},
		}
		result, err = te.evaluateNode(branchNode, branchCtx)
		nodeTrace.Children = append(nodeTrace.Children, *branchCtx.trace)

		if err != nil {
			nodeTrace.Result = fmt.Sprintf("Failed: %s error: %v", branchName, err)
		} else {
			nodeTrace.Result = fmt.Sprintf("Success: condition was %v, returned %v", condTrue, result)
			nodeTrace.Value = result
		}

	case *ArrayNode:
		nodeTrace.Step = "Array"
		nodeTrace.Description = fmt.Sprintf("Evaluating array with %d elements", len(n.Elements))

		elements := make([]any, 0, len(n.Elements))
		for i, elem := range n.Elements {
			elemCtx := &tracingEvaluationContext{
				EvaluationContext: ctx.EvaluationContext,
				trace:            &TraceStep{Step: fmt.Sprintf("Element %d", i)},
			}
			elemVal, elemErr := te.evaluateNode(elem, elemCtx)
			nodeTrace.Children = append(nodeTrace.Children, *elemCtx.trace)

			if elemErr != nil {
				nodeTrace.Result = fmt.Sprintf("Failed: element %d error: %v", i, elemErr)
				err = elemErr
				break
			}
			elements = append(elements, elemVal)
		}

		if err == nil {
			result = elements
			nodeTrace.Result = "Success"
			nodeTrace.Value = result
		}

	case *BooleanNode:
		nodeTrace.Step = "Boolean"
		nodeTrace.Description = fmt.Sprintf("Literal boolean: %v", n.Value)
		result = n.Value
		nodeTrace.Result = "Success"
		nodeTrace.Value = result

	case *StringNode:
		nodeTrace.Step = "String"
		nodeTrace.Description = fmt.Sprintf("Literal string: %q", n.Value)
		result = n.Value
		nodeTrace.Result = "Success"
		nodeTrace.Value = result

	default:
		// For any other node types, just evaluate normally
		result, err = node.Evaluate(ctx.EvaluationContext)
		nodeTrace.Step = fmt.Sprintf("Evaluate %T", node)
		if err != nil {
			nodeTrace.Result = fmt.Sprintf("Failed: %v", err)
		} else {
			nodeTrace.Result = "Success"
			nodeTrace.Value = result
		}
	}

	// Add this node's trace to the parent
	if ctx.trace.Children == nil {
		ctx.trace.Children = make([]TraceStep, 0)
	}
	ctx.trace.Children = append(ctx.trace.Children, nodeTrace)

	return result, err
}

// * tokensToString converts tokens to a readable string
func tokensToString(tokens []Token) string {
	result := ""
	for i, token := range tokens {
		if i > 0 {
			result += " "
		}
		result += fmt.Sprintf("[%s:%s]", token.Type, token.Value)
	}
	return result
}