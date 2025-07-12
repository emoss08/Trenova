# Conversion Package API

The conversion package provides safe type conversion utilities used throughout the formula system.

## Core Functions

### ToFloat64

Converts various types to float64 with comprehensive type support.

```go
func ToFloat64(v any) (float64, bool)
```

**Supported Types:**

- All numeric types (int, int8-64, uint, uint8-64, float32, float64)
- Decimal types (shopspring/decimal)
- String parsing
- Boolean (true=1, false=0)
- Pointer dereferencing

**Examples:**

```go
// Basic conversions
f, ok := ToFloat64(42)           // 42.0, true
f, ok := ToFloat64(3.14)         // 3.14, true
f, ok := ToFloat64("123.45")     // 123.45, true
f, ok := ToFloat64(true)         // 1.0, true
f, ok := ToFloat64(false)        // 0.0, true

// Decimal conversion
d := decimal.NewFromFloat(99.99)
f, ok := ToFloat64(d)            // 99.99, true

// Pointer handling
val := 42.0
f, ok := ToFloat64(&val)         // 42.0, true

// Invalid conversions
f, ok := ToFloat64("abc")        // 0, false
f, ok := ToFloat64(nil)          // 0, false
f, ok := ToFloat64([]int{1,2})   // 0, false
```

### ToInt64

Converts various types to int64 with proper rounding.

```go
func ToInt64(v any) (int64, bool)
```

**Supported Types:**

- All numeric types
- Decimal types
- String parsing
- Boolean (true=1, false=0)
- Pointer dereferencing

**Examples:**

```go
// Basic conversions
i, ok := ToInt64(42.7)           // 43, true (rounds)
i, ok := ToInt64("123")          // 123, true
i, ok := ToInt64(true)           // 1, true

// Overflow handling
i, ok := ToInt64(1e20)           // 0, false (overflow)

// Decimal conversion
d := decimal.NewFromFloat(99.99)
i, ok := ToInt64(d)              // 100, true (rounds)
```

### ToBool

Converts various types to boolean following truthy/falsy rules.

```go
func ToBool(v any) (bool, bool)
```

**Conversion Rules:**

- Numbers: 0 is false, all others are true
- Strings: empty string is false, all others are true
- Booleans: direct conversion
- nil: false
- Pointers: dereferences then applies rules

**Examples:**

```go
// Numeric conversions
b, ok := ToBool(0)               // false, true
b, ok := ToBool(1)               // true, true
b, ok := ToBool(-1)              // true, true
b, ok := ToBool(0.0)             // false, true

// String conversions
b, ok := ToBool("")              // false, true
b, ok := ToBool("hello")         // true, true
b, ok := ToBool("false")         // true, true (non-empty)

// Direct boolean
b, ok := ToBool(true)            // true, true
b, ok := ToBool(false)           // false, true

// Nil handling
b, ok := ToBool(nil)             // false, true

// Pointer handling
val := false
b, ok := ToBool(&val)            // false, true
```

## Usage Patterns

### Safe Numeric Operations

```go
// Safe addition with type checking
func safeAdd(a, b any) (float64, error) {
    aFloat, ok := ToFloat64(a)
    if !ok {
        return 0, fmt.Errorf("cannot convert %v to number", a)
    }
    
    bFloat, ok := ToFloat64(b)
    if !ok {
        return 0, fmt.Errorf("cannot convert %v to number", b)
    }
    
    return aFloat + bFloat, nil
}
```

### Type Validation

```go
// Validate and convert function arguments
func validateNumericArgs(args []any) ([]float64, error) {
    result := make([]float64, len(args))
    
    for i, arg := range args {
        val, ok := ToFloat64(arg)
        if !ok {
            return nil, fmt.Errorf("argument %d must be numeric, got %T", i+1, arg)
        }
        result[i] = val
    }
    
    return result, nil
}
```

### Decimal Handling

```go
// Work with decimal types safely
func processDecimalValue(v any) (decimal.Decimal, error) {
    // Try direct decimal assertion first
    if d, ok := v.(decimal.Decimal); ok {
        return d, nil
    }
    
    // Convert to float64 then to decimal
    f, ok := ToFloat64(v)
    if !ok {
        return decimal.Zero, fmt.Errorf("cannot convert %v to decimal", v)
    }
    
    return decimal.NewFromFloat(f), nil
}
```

### Boolean Logic

```go
// Implement truthy/falsy logic
func isTruthy(v any) bool {
    b, _ := ToBool(v)
    return b
}

// Conditional evaluation
func evaluateCondition(condition any) bool {
    // ToBool always succeeds, returning false for invalid types
    result, _ := ToBool(condition)
    return result
}
```

## Integration with Formula System

### In Operators

```go
// Used in arithmetic operations
func evaluateAddition(left, right any) (any, error) {
    leftNum, leftOk := ToFloat64(left)
    rightNum, rightOk := ToFloat64(right)
    
    if leftOk && rightOk {
        return leftNum + rightNum, nil
    }
    
    // Fall back to string concatenation
    return fmt.Sprintf("%v%v", left, right), nil
}
```

### In Functions

```go
// Math function implementation
func funcAbs(args ...any) (any, error) {
    if len(args) != 1 {
        return nil, errors.New("abs requires exactly 1 argument")
    }
    
    val, ok := ToFloat64(args[0])
    if !ok {
        return nil, fmt.Errorf("abs: argument must be numeric")
    }
    
    return math.Abs(val), nil
}
```

### In Comparisons

```go
// Numeric comparison with type coercion
func compareNumbers(left, right any) (int, bool) {
    leftNum, leftOk := ToFloat64(left)
    rightNum, rightOk := ToFloat64(right)
    
    if !leftOk || !rightOk {
        return 0, false
    }
    
    if leftNum < rightNum {
        return -1, true
    } else if leftNum > rightNum {
        return 1, true
    }
    return 0, true
}
```

## Performance Considerations

1. **Type Assertions First**: Direct type assertions are fastest
2. **Avoid Repeated Conversions**: Cache conversion results
3. **Pointer Handling**: Automatic but has overhead
4. **String Parsing**: Most expensive operation

### Optimization Example

```go
// Optimized batch conversion
type ConversionCache struct {
    cache map[any]float64
    mu    sync.RWMutex
}

func (c *ConversionCache) ToFloat64(v any) (float64, bool) {
    // Check cache first
    c.mu.RLock()
    if val, ok := c.cache[v]; ok {
        c.mu.RUnlock()
        return val, true
    }
    c.mu.RUnlock()
    
    // Perform conversion
    val, ok := ToFloat64(v)
    if ok {
        c.mu.Lock()
        c.cache[v] = val
        c.mu.Unlock()
    }
    
    return val, ok
}
```

## Error Handling

```go
// Comprehensive error reporting
func convertWithContext(v any, context string) (float64, error) {
    result, ok := ToFloat64(v)
    if !ok {
        return 0, fmt.Errorf(
            "cannot convert %T to number in %s: value = %v",
            v, context, v,
        )
    }
    return result, nil
}

// Validation with multiple types
func validateInput(v any) error {
    switch v := v.(type) {
    case int, int8, int16, int32, int64,
         uint, uint8, uint16, uint32, uint64,
         float32, float64:
        return nil
    case string:
        _, ok := ToFloat64(v)
        if !ok {
            return fmt.Errorf("string %q is not a valid number", v)
        }
        return nil
    case decimal.Decimal:
        return nil
    default:
        return fmt.Errorf("unsupported type %T", v)
    }
}
```

## Testing Utilities

```go
// Test helper for conversion testing
func TestConversionMatrix(t *testing.T) {
    tests := []struct {
        input    any
        wantF64  float64
        wantI64  int64
        wantBool bool
        wantOk   bool
    }{
        {42, 42.0, 42, true, true},
        {3.14, 3.14, 3, true, true},
        {"123", 123.0, 123, true, true},
        {true, 1.0, 1, true, true},
        {false, 0.0, 0, false, true},
        {"", 0.0, 0, false, false},
        {nil, 0.0, 0, false, false},
    }
    
    for _, tt := range tests {
        t.Run(fmt.Sprintf("%v", tt.input), func(t *testing.T) {
            gotF64, ok := ToFloat64(tt.input)
            if ok != tt.wantOk {
                t.Errorf("ToFloat64() ok = %v, want %v", ok, tt.wantOk)
            }
            if ok && gotF64 != tt.wantF64 {
                t.Errorf("ToFloat64() = %v, want %v", gotF64, tt.wantF64)
            }
        })
    }
}
```

## Best Practices

1. **Check Conversion Success**: Always check the boolean return value
2. **Handle Edge Cases**: Consider nil, empty strings, and overflows
3. **Use Appropriate Function**: Choose the right conversion for your needs
4. **Document Assumptions**: Make conversion rules clear in your code
5. **Validate Early**: Convert and validate inputs at system boundaries
6. **Consider Precision**: Be aware of floating-point precision limits
7. **Cache When Possible**: Conversions can be expensive for strings

## Thread Safety

All conversion functions are thread-safe and can be called concurrently. They don't modify any shared state and work with immutable inputs.
