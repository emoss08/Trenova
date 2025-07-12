# Errors Package API

The errors package provides specialized error types for the formula system with rich context information.

## Error Types

### BaseError

Base type for all formula errors with context information.

```go
type BaseError struct {
    Message string
    Context map[string]any
    Cause   error
}

// Methods
func (e *BaseError) Error() string
func (e *BaseError) Unwrap() error
func (e *BaseError) WithContext(key string, value any) *BaseError
```

### ParseError

Error during expression parsing with position information.

```go
type ParseError struct {
    BaseError
    Position int
    Line     int
    Column   int
}

// Usage
err := &ParseError{
    BaseError: BaseError{
        Message: "unexpected token",
        Context: map[string]any{
            "token": "&&",
            "expected": "operand",
        },
    },
    Position: 42,
    Line:     3,
    Column:   15,
}
```

### EvaluationError

Error during expression evaluation.

```go
type EvaluationError struct {
    BaseError
    Expression string
    Variable   string
}

// Usage
err := &EvaluationError{
    BaseError: BaseError{
        Message: "division by zero",
        Context: map[string]any{
            "numerator": 100,
            "denominator": 0,
        },
    },
    Expression: "total / count",
    Variable:   "count",
}
```

### TypeError

Type mismatch error during operations.

```go
type TypeError struct {
    BaseError
    Expected  string
    Actual    string
    Operation string
}

// Usage
err := &TypeError{
    BaseError: BaseError{
        Message: "type mismatch",
    },
    Expected:  "number",
    Actual:    "string",
    Operation: "multiplication",
}
```

### ValidationError

Schema or input validation error.

```go
type ValidationError struct {
    BaseError
    Field       string
    Value       any
    Constraint  string
}

// Usage
err := &ValidationError{
    BaseError: BaseError{
        Message: "validation failed",
    },
    Field:      "weight",
    Value:      -10,
    Constraint: "minimum: 0",
}
```

## Error Creation Helpers

```go
// Create parse error with position
func NewParseError(msg string, pos, line, col int) *ParseError {
    return &ParseError{
        BaseError: BaseError{Message: msg},
        Position:  pos,
        Line:      line,
        Column:    col,
    }
}

// Create evaluation error
func NewEvaluationError(msg, expr string) *EvaluationError {
    return &EvaluationError{
        BaseError:  BaseError{Message: msg},
        Expression: expr,
    }
}

// Create type error
func NewTypeError(expected, actual, operation string) *TypeError {
    return &TypeError{
        BaseError: BaseError{
            Message: fmt.Sprintf("expected %s but got %s for %s",
                expected, actual, operation),
        },
        Expected:  expected,
        Actual:    actual,
        Operation: operation,
    }
}
```

## Error Handling Patterns

### Error Wrapping

```go
// Wrap with context
func processFormula(formula string) error {
    result, err := evaluate(formula)
    if err != nil {
        return &EvaluationError{
            BaseError: BaseError{
                Message: "formula processing failed",
                Cause:   err,
                Context: map[string]any{
                    "formula": formula,
                    "timestamp": time.Now(),
                },
            },
            Expression: formula,
        }
    }
    return nil
}
```

### Error Type Checking

```go
// Check specific error types
err := evaluator.Evaluate(expr, vars)
if err != nil {
    switch e := err.(type) {
    case *ParseError:
        // Handle parse error
        log.Printf("Parse error at line %d, column %d: %s",
            e.Line, e.Column, e.Message)
        
    case *TypeError:
        // Handle type error
        log.Printf("Type error: expected %s but got %s",
            e.Expected, e.Actual)
        
    case *EvaluationError:
        // Handle evaluation error
        if e.Variable != "" {
            log.Printf("Variable %s caused error: %s",
                e.Variable, e.Message)
        }
        
    default:
        // Handle unknown error
        log.Printf("Unknown error: %v", err)
    }
}
```

### Error Unwrapping

```go
// Unwrap to find root cause
func getRootCause(err error) error {
    for {
        unwrapped := errors.Unwrap(err)
        if unwrapped == nil {
            return err
        }
        err = unwrapped
    }
}

// Check for specific cause
if errors.Is(err, context.DeadlineExceeded) {
    // Handle timeout
}
```

### Error Context Enhancement

```go
// Add context as error propagates
func calculateShipmentRate(shipmentID string) error {
    shipment, err := loadShipment(shipmentID)
    if err != nil {
        return &EvaluationError{
            BaseError: BaseError{
                Message: "failed to calculate rate",
                Cause:   err,
                Context: map[string]any{
                    "shipment_id": shipmentID,
                    "step": "load_shipment",
                },
            },
        }
    }
    
    rate, err := evaluateFormula(shipment)
    if err != nil {
        evalErr := &EvaluationError{
            BaseError: BaseError{
                Message: "formula evaluation failed",
                Cause:   err,
            },
        }
        evalErr.WithContext("shipment_id", shipmentID)
        evalErr.WithContext("step", "evaluate_formula")
        evalErr.WithContext("weight", shipment.Weight)
        return evalErr
    }
    
    return nil
}
```

## Error Formatting

### Custom Error Messages

```go
// Implement custom formatting
func (e *ParseError) Error() string {
    if e.Line > 0 && e.Column > 0 {
        return fmt.Sprintf("%s at line %d, column %d",
            e.Message, e.Line, e.Column)
    }
    return e.Message
}

func (e *TypeError) Error() string {
    return fmt.Sprintf("type error in %s: expected %s but got %s",
        e.Operation, e.Expected, e.Actual)
}
```

### Detailed Error Output

```go
// Format error with full context
func formatError(err error) string {
    var buf strings.Builder
    
    // Write main message
    buf.WriteString(err.Error())
    
    // Add context if available
    if ctxErr, ok := err.(*BaseError); ok && len(ctxErr.Context) > 0 {
        buf.WriteString("\nContext:")
        
        // Sort keys for consistent output
        keys := make([]string, 0, len(ctxErr.Context))
        for k := range ctxErr.Context {
            keys = append(keys, k)
        }
        sort.Strings(keys)
        
        for _, k := range keys {
            buf.WriteString(fmt.Sprintf("\n  %s: %v", k, ctxErr.Context[k]))
        }
    }
    
    // Add cause chain
    if cause := errors.Unwrap(err); cause != nil {
        buf.WriteString("\nCaused by: ")
        buf.WriteString(formatError(cause))
    }
    
    return buf.String()
}
```

## Integration Examples

### With Expression Package

```go
// Parse error handling
tokens, err := tokenizer.Tokenize()
if err != nil {
    if parseErr, ok := err.(*ParseError); ok {
        // Show error in context
        showErrorInExpression(expression, parseErr.Position)
    }
    return err
}
```

### With Validation

```go
// Validation error collection
type ValidationResult struct {
    Errors []*ValidationError
}

func (v *ValidationResult) AddError(field string, value any, constraint string) {
    v.Errors = append(v.Errors, &ValidationError{
        BaseError: BaseError{
            Message: fmt.Sprintf("field %s failed constraint %s", field, constraint),
        },
        Field:      field,
        Value:      value,
        Constraint: constraint,
    })
}

func (v *ValidationResult) HasErrors() bool {
    return len(v.Errors) > 0
}

func (v *ValidationResult) Error() string {
    if !v.HasErrors() {
        return ""
    }
    
    messages := make([]string, len(v.Errors))
    for i, err := range v.Errors {
        messages[i] = err.Error()
    }
    
    return strings.Join(messages, "; ")
}
```

## Best Practices

1. **Always Include Context**: Add relevant information to help debugging
2. **Use Appropriate Error Types**: Choose the right error type for the situation
3. **Wrap Lower-Level Errors**: Preserve the error chain for debugging
4. **Provide Actionable Messages**: Help users understand how to fix the issue
5. **Include Position Information**: For parse errors, always include location
6. **Type Safety**: Use type assertions to handle specific error types
7. **Avoid Sensitive Data**: Don't include passwords or keys in error context
8. **Consistent Formatting**: Use consistent error message formats

## Error Recovery

```go
// Graceful degradation with error recovery
func evaluateWithFallback(primary, fallback string, vars map[string]any) (any, error) {
    // Try primary expression
    result, err := evaluator.Evaluate(context.Background(), primary, vars)
    if err == nil {
        return result, nil
    }
    
    // Log primary error
    log.Printf("Primary expression failed: %v", err)
    
    // Try fallback
    result, fallbackErr := evaluator.Evaluate(context.Background(), fallback, vars)
    if fallbackErr != nil {
        // Return composite error
        return nil, &EvaluationError{
            BaseError: BaseError{
                Message: "both primary and fallback expressions failed",
                Context: map[string]any{
                    "primary_error":  err.Error(),
                    "fallback_error": fallbackErr.Error(),
                },
            },
            Expression: primary,
        }
    }
    
    return result, nil
}
```