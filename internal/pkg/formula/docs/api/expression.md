# Expression Package API

The expression package provides the core expression parsing and evaluation engine.

## Core Types

### Token

Represents a lexical token in the expression.

```go
type Token struct {
    Value    string    // Token value
    Position int       // Position in input
    Type     TokenType // Type of token
    Line     uint16    // Line number
    Column   uint16    // Column number
}
```

### Node

Interface for all AST nodes.

```go
type Node interface {
    Evaluate(ctx *EvaluationContext) (any, error)
    Type() formula.ValueType
    String() string
    Complexity() int
}
```

### EvaluationContext

Context for expression evaluation with security limits.

```go
type EvaluationContext struct {
    Context        context.Context
    Variables      map[string]any
    VariableCtx    variables.VariableContext
    MemoryLimit    int64
    memoryUsed     int64
    variableCache  map[string]any
}
```

## Main Components

### Tokenizer

Performs lexical analysis on expression strings.

```go
// Create a new tokenizer
tokenizer := NewTokenizer(expression)

// Enable debug mode
tokenizer.EnableDebug()

// Tokenize the expression
tokens, err := tokenizer.Tokenize()
if err != nil {
    // Handle tokenization error
}
```

**Security Limits:**

- Maximum expression length: 10,000 characters
- Maximum token count: 1,000 tokens

### Parser

Builds an Abstract Syntax Tree (AST) from tokens.

```go
// Create parser from tokens
parser := NewParser(tokens)

// Parse into AST
node, err := parser.Parse()
if err != nil {
    // Handle parsing error
}
```

**Features:**

- Operator precedence handling
- Error accumulation
- Support for all expression types

### Evaluator

Evaluates expressions with caching support.

```go
// Create evaluator
evaluator := NewEvaluator()

// Simple evaluation
result, err := evaluator.Evaluate(ctx, expression, variables)

// With variable context
evalCtx := NewEvaluationContext(ctx, variableCtx)
result, err := evaluator.EvaluateWithContext(evalCtx, expression)

// Batch evaluation
results, err := evaluator.EvaluateBatch(ctx, expression, variablesList)
```

**Features:**

- Expression compilation and caching
- Batch evaluation support
- Variable extraction
- Timeout and memory limit enforcement

## Node Types

### Literal Nodes

```go
// Number
node := &NumberNode{Value: 42.5}

// String
node := &StringNode{Value: "hello"}

// Boolean
node := &BooleanNode{Value: true}

// Array
node := &ArrayNode{Elements: []Node{...}}
```

### Operator Nodes

```go
// Binary operation
node := &BinaryOpNode{
    Left:     leftNode,
    Right:    rightNode,
    Operator: TokenPlus,
}

// Unary operation
node := &UnaryOpNode{
    Operand:  operandNode,
    Operator: TokenNot,
}
```

### Other Nodes

```go
// Variable reference
node := &IdentifierNode{Name: "variable_name"}

// Function call
node := &FunctionCallNode{
    Name: "max",
    Args: []Node{...},
}

// Array indexing
node := &IndexNode{
    Array: arrayNode,
    Index: indexNode,
}

// Conditional
node := &ConditionalNode{
    Condition: conditionNode,
    TrueExpr:  trueNode,
    FalseExpr: falseNode,
}
```

## Functions

### Function Interface

```go
type Function interface {
    Name() string
    MinArgs() int
    MaxArgs() int  // -1 for variadic
    Call(ctx *EvaluationContext, args ...any) (any, error)
}
```

### Function Registry

```go
// Get default registry
registry := DefaultFunctionRegistry()

// Check if function exists
if fn, exists := registry["sqrt"]; exists {
    result, err := fn.Call(ctx, 16.0)
}
```

## Error Types

The expression package uses specific error types from the errors package:

```go
// Parse error with position
err := &ParseError{
    Message:  "unexpected token",
    Position: 42,
    Line:     2,
    Column:   10,
}

// Evaluation error with context
err := &EvaluationError{
    Expression: "x / 0",
    Variable:   "x",
    Cause:      "division by zero",
}

// Type error
err := &TypeError{
    Expected: "number",
    Actual:   "string",
    Operation: "addition",
}
```

## Usage Examples

### Basic Expression Evaluation

```go
ctx := context.Background()
evaluator := NewEvaluator()

// Simple arithmetic
result, err := evaluator.Evaluate(ctx, "2 + 3 * 4", nil)
// result: 14.0

// With variables
vars := map[string]any{
    "price": 10.50,
    "quantity": 3,
}
result, err := evaluator.Evaluate(ctx, "price * quantity", vars)
// result: 31.5
```

### Advanced Features

```go
// Array operations
expr := "sum(prices) / len(prices)"
vars := map[string]any{
    "prices": []any{10.0, 20.0, 30.0},
}
result, err := evaluator.Evaluate(ctx, expr, vars)
// result: 20.0

// Conditional logic
expr = "if(temperature > 100, 'hot', if(temperature < 32, 'cold', 'mild'))"
vars = map[string]any{
    "temperature": 75,
}
result, err := evaluator.Evaluate(ctx, expr, vars)
// result: "mild"
```

### With Variable Context

```go
// Implement variable context
type MyVariableContext struct {
    data map[string]any
}

func (c *MyVariableContext) ResolveVariable(name string) (any, error) {
    if val, ok := c.data[name]; ok {
        return val, nil
    }
    return nil, fmt.Errorf("unknown variable: %s", name)
}

func (c *MyVariableContext) GetFieldSources() map[string]any {
    return c.data
}

// Use with evaluator
varCtx := &MyVariableContext{
    data: map[string]any{
        "base_rate": 2.50,
        "distance": 100,
    },
}

evalCtx := NewEvaluationContext(ctx, varCtx)
result, err := evaluator.EvaluateWithContext(evalCtx, "base_rate * distance")
```

### Performance Optimization

```go
// Pre-compile expressions
compiled, err := evaluator.Compile("complex_expression")

// Reuse compiled expression
for _, vars := range dataSet {
    result, err := evaluator.EvaluateCompiled(ctx, compiled, vars)
    // Process result
}

// Extract variables for optimization
variables := compiled.ExtractVariables()
// Preload only needed variables
```

## Best Practices

1. **Reuse Evaluators**: Create once and reuse for multiple evaluations
2. **Pre-compile Complex Expressions**: Use `Compile()` for repeated evaluations
3. **Set Appropriate Timeouts**: Use context with timeout for untrusted expressions
4. **Handle Errors Properly**: Check error types for specific handling
5. **Validate Before Execution**: Use `Compile()` to validate syntax early
6. **Monitor Cache Performance**: Expressions are cached by default
7. **Use Batch Evaluation**: For bulk operations with the same expression

## Thread Safety

- Tokenizer: Not thread-safe, create per goroutine
- Parser: Not thread-safe, create per goroutine  
- Evaluator: Thread-safe, can be shared
- Node types: Immutable and thread-safe
- Function registry: Thread-safe for reads
