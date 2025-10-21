<!--
Copyright 2023-2025 Eric Moss
Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md-->
# Expression Syntax Guide

This guide covers the complete syntax for formula expressions in the Trenova system. The expression engine provides a safe, sandboxed environment for evaluating mathematical and logical expressions with support for variables, functions, and complex business logic.

## Table of Contents

- [Basic Syntax](#basic-syntax)
- [Data Types](#data-types)
- [Operators](#operators)
- [Variables](#variables)
- [Functions](#functions)
- [Arrays](#arrays)
- [Conditionals](#conditionals)
- [Type System](#type-system)
- [Performance Features](#performance-features)
- [Error Handling](#error-handling)
- [Best Practices](#best-practices)
- [Examples](#examples)
- [Limitations](#limitations)

## Basic Syntax

Formula expressions follow a familiar syntax similar to JavaScript or Excel formulas:

```javascript
// Simple arithmetic
price * quantity

// With parentheses for precedence
(price + tax) * quantity

// Function calls
round(total, 2)

// Conditionals
if(quantity > 100, price * 0.9, price)
```

## Data Types

### Numbers

Numbers can be integers or floating-point values:

```javascript
42          // Integer
3.14159     // Decimal
1.23e-4     // Scientific notation
-100        // Negative numbers
```

### Strings

Strings are enclosed in double quotes:

```javascript
"Hello, World!"
"Line 1\nLine 2"    // Escaped newline
"Quote: \"text\""   // Escaped quotes
```

### Booleans

Boolean values are `true` or `false`:

```javascript
true
false
```

### Arrays

Arrays are ordered collections enclosed in square brackets:

```javascript
[1, 2, 3]                    // Array of numbers
["a", "b", "c"]              // Array of strings
[1, "mixed", true]           // Mixed types
[[1, 2], [3, 4]]            // Nested arrays
[]                          // Empty array
```

## Operators

### Arithmetic Operators

| Operator | Description | Example | Result |
|----------|-------------|---------|--------|
| `+` | Addition | `5 + 3` | `8` |
| `-` | Subtraction | `10 - 4` | `6` |
| `*` | Multiplication | `3 * 4` | `12` |
| `/` | Division | `15 / 3` | `5` |
| `%` | Modulo | `10 % 3` | `1` |
| `^` | Power | `2 ^ 3` | `8` |

### Comparison Operators

| Operator | Description | Example | Result |
|----------|-------------|---------|--------|
| `==` | Equal | `5 == 5` | `true` |
| `!=` | Not equal | `5 != 3` | `true` |
| `>` | Greater than | `10 > 5` | `true` |
| `<` | Less than | `3 < 7` | `true` |
| `>=` | Greater or equal | `5 >= 5` | `true` |
| `<=` | Less or equal | `4 <= 6` | `true` |

### Logical Operators

| Operator | Description | Example | Result |
|----------|-------------|---------|--------|
| `&&` | Logical AND | `true && false` | `false` |
| `\|\|` | Logical OR | `true \|\| false` | `true` |
| `!` | Logical NOT | `!true` | `false` |

### String Operators

The `+` operator concatenates strings:

```javascript
"Hello" + " " + "World"    // "Hello World"
"Value: " + 42             // "Value: 42"
```

### Array Indexing

Use square brackets to access array elements:

```javascript
arr[0]              // First element
arr[len(arr) - 1]   // Last element
matrix[i][j]        // Nested array access
"hello"[1]          // String indexing returns "e"
```

## Operator Precedence

Operators are evaluated in this order (highest to lowest):

1. Parentheses `()`
2. Array indexing `[]`
3. Unary operators `!`, `-`
4. Power `^`
5. Multiplication, Division, Modulo `*`, `/`, `%`
6. Addition, Subtraction `+`, `-`
7. Comparison `>`, `<`, `>=`, `<=`
8. Equality `==`, `!=`
9. Logical AND `&&`
10. Logical OR `||`
11. Ternary conditional `? :`

## Variables

Variables reference values from the evaluation context:

```javascript
// Simple variables
base_rate
distance
has_hazmat

// Variables from entities
shipment.weight
customer.discount_rate
equipment.capacity

// Computed fields
temperature_differential
total_stops
```

### Variable Naming Rules

- Must start with a letter or underscore
- Can contain letters, numbers, and underscores
- Case-sensitive
- Cannot be reserved keywords

## Functions

Functions are called with parentheses and comma-separated arguments:

```javascript
// No arguments
random()

// Single argument
abs(-5)

// Multiple arguments
pow(2, 8)
round(3.14159, 2)

// Nested function calls
max(abs(x), abs(y))
```

### Built-in Functions

#### Math Functions

| Function | Description | Example | Result |
|----------|-------------|---------|--------|
| `abs(x)` | Absolute value | `abs(-5)` | `5` |
| `min(...args)` | Minimum value | `min(3, 1, 4)` | `1` |
| `max(...args)` | Maximum value | `max(3, 1, 4)` | `4` |
| `round(x, precision?)` | Round to precision | `round(3.14159, 2)` | `3.14` |
| `floor(x)` | Round down | `floor(3.7)` | `3` |
| `ceil(x)` | Round up | `ceil(3.2)` | `4` |
| `sqrt(x)` | Square root | `sqrt(16)` | `4` |
| `pow(x, y)` | Power | `pow(2, 3)` | `8` |
| `log(x, base?)` | Logarithm | `log(100, 10)` | `2` |
| `exp(x)` | Exponential | `exp(1)` | `2.718` |
| `sin(x)` | Sine (radians) | `sin(1.571)` | `~1` |
| `cos(x)` | Cosine (radians) | `cos(0)` | `1` |
| `tan(x)` | Tangent (radians) | `tan(0.785)` | `~1` |

#### Type Conversion Functions

| Function | Description | Example | Result |
|----------|-------------|---------|--------|
| `number(x)` | Convert to number | `number("42")` | `42` |
| `string(x)` | Convert to string | `string(42)` | `"42"` |
| `bool(x)` | Convert to boolean | `bool("true")` | `true` |

#### Array Functions

| Function | Description | Example | Result |
|----------|-------------|---------|--------|
| `len(array)` | Array length | `len([1, 2, 3])` | `3` |
| `sum(...values)` | Sum of elements | `sum([1, 2, 3])` | `6` |
| `avg(...values)` | Average of elements | `avg([1, 2, 3])` | `2` |
| `contains(array, value)` | Check if contains | `contains([1, 2, 3], 2)` | `true` |
| `indexOf(array, value)` | Find index | `indexOf(["a", "b"], "b")` | `1` |
| `slice(array, start, end?)` | Extract subarray | `slice([1, 2, 3, 4], 1, 3)` | `[2, 3]` |
| `concat(...arrays)` | Concatenate arrays | `concat([1, 2], [3, 4])` | `[1, 2, 3, 4]` |

#### String Functions

String operations use the same array functions:

| Function | Description | Example | Result |
|----------|-------------|---------|--------|
| `len(string)` | String length | `len("hello")` | `5` |
| `slice(string, start, end?)` | Extract substring | `slice("hello", 1, 3)` | `"el"` |
| `concat(...strings)` | Concatenate strings | `concat("hello", " ", "world")` | `"hello world"` |
| `contains(string, substr)` | Check for substring | `contains("hello", "ell")` | `true` |
| `indexOf(string, substr)` | Find substring index | `indexOf("hello", "ll")` | `2` |

#### Conditional Functions

| Function | Description | Example | Result |
|----------|-------------|---------|--------|
| `if(condition, true_val, false_val)` | Conditional | `if(x > 0, "pos", "neg")` | `"pos"` or `"neg"` |
| `coalesce(...values)` | First non-null/non-empty | `coalesce(null, "", "default")` | `"default"` |

## Array Operations

### Array Literals

Create arrays using square brackets:

```javascript
[1, 2, 3, 4, 5]
["red", "green", "blue"]
[true, false, true]
```

### Array Operations

```javascript
// Length
len([1, 2, 3])              // 3

// Sum and average
sum([10, 20, 30])           // 60
avg([10, 20, 30])           // 20

// Slicing
slice([1, 2, 3, 4, 5], 1, 4)   // [2, 3, 4]
slice(arr, -3, -1)              // Last 3 elements except the last

// Concatenation
concat([1, 2], [3, 4])          // [1, 2, 3, 4]

// Search
contains([1, 2, 3], 2)          // true
indexOf([10, 20, 30], 20)       // 1
```

## Conditionals

### Ternary Operator

The `? :` operator provides inline conditionals:

```javascript
condition ? true_value : false_value

// Examples
distance > 500 ? "long_haul" : "short_haul"
has_hazmat ? base_rate * 1.25 : base_rate
```

### If Function

The `if()` function provides the same functionality:

```javascript
if(condition, true_value, false_value)

// Nested conditions
if(weight < 1000,
   "light",
   if(weight < 5000, "medium", "heavy")
)
```

### Comparison Examples

```javascript
// Numeric comparison
temperature > 32 && temperature < 80

// String comparison  
status == "active" || status == "pending"

// Array membership
contains(["A", "B", "C"], grade)

// Complex conditions
distance > 500 && (has_hazmat || requires_escort) && !is_weekend
```

## Comments

Formula expressions do not support comments. Use descriptive variable names and break complex expressions into multiple formulas for clarity.

## Type System

### Type Hierarchy

The expression engine uses a flexible type system:

1. **Number**: Integer or floating-point values
2. **String**: Text values
3. **Boolean**: True/false values
4. **Array**: Ordered collections of values
5. **Null**: Absence of value
6. **Any**: Can hold any type (used in functions)

### Type Checking

The engine performs type checking at evaluation time:

```javascript
// Type-safe operations
5 + 3              // OK: number + number
"hello" + "world"  // OK: string + string

// Type errors
5 + "hello"        // Error: cannot add number and string
[1, 2] * 3         // Error: cannot multiply array
```

### Null Handling

Null values propagate through most operations:

```javascript
null + 5           // null
null * 10          // null
null && true       // false (null is falsy)
null || "default"  // "default"
coalesce(null, 10) // 10
```

## Type Coercion

The expression engine performs automatic type conversion in certain contexts:

### Numeric Contexts

Values are converted to numbers when needed:

```javascript
"42" * 1        // 42 (string to number)
true + 0        // 1 (boolean to number)
false + 0       // 0 (boolean to number)
```

### String Contexts

Values are converted to strings with the `+` operator when one operand is a string:

```javascript
"Value: " + 42          // "Value: 42"
"Flag: " + true         // "Flag: true"
```

### Boolean Contexts

Values are converted to booleans in conditions:

- Numbers: `0` is `false`, all others are `true`
- Strings: empty string is `false`, others are `true`  
- Arrays: empty array is `false`, others are `true`
- `null` is `false`

### Explicit Conversion

Use conversion functions for explicit type conversion:

```javascript
number("42")     // 42
string(42)       // "42"
bool(1)          // true
```

## Performance Features

### Expression Caching

The engine caches compiled expressions using an LRU cache:
- Default cache size: 1000 expressions
- Cache hit rate typically > 95% in production
- Automatic eviction of least recently used expressions

### Arena Allocation

The engine uses arena allocation to reduce garbage collection pressure:
- Pre-allocated memory blocks
- Object pooling for AST nodes
- String interning for common values
- Minimal heap allocations during evaluation

### Optimization Techniques

1. **Constant Folding**: `2 + 3` is optimized to `5` at compile time
2. **Dead Code Elimination**: Unreachable code is removed
3. **Short-Circuit Evaluation**: `false && expensive()` doesn't evaluate `expensive()`
4. **Lazy Evaluation**: Array operations only process needed elements

## Best Practices

### 1. Use Parentheses for Clarity

```javascript
// Less clear
price * quantity + tax * quantity

// More clear
(price + tax) * quantity
```

### 2. Break Complex Formulas

Instead of one complex formula, use multiple simpler ones:

```javascript
// Complex
base_rate * distance * if(has_hazmat, 1.25, 1) * if(is_expedited, 1.5, 1) * (1 + fuel_surcharge/100)

// Better: create multiple formula templates or use nested expressions
base_rate * distance * 
  if(has_hazmat, 1.25, 1) * 
  if(is_expedited, 1.5, 1) * 
  (1 + fuel_surcharge/100)
```

### 3. Validate Input Ranges

```javascript
// Ensure percentage is valid
if(discount_percent < 0, 0, if(discount_percent > 100, 100, discount_percent))

// Or use min/max
max(0, min(100, discount_percent))
```

### 4. Handle Division by Zero

```javascript
// Risky
total / count

// Safer
if(count > 0, total / count, 0)
```

### 5. Use Meaningful Variable Names

```javascript
// Poor
br * d * (1 + fs)

// Better
base_rate * distance * (1 + fuel_surcharge)
```

## Examples

### Tiered Pricing

```javascript
// Weight-based tiers with clear breakpoints
if(weight <= 100,
   weight * 5.00,
   if(weight <= 500,
      100 * 5.00 + (weight - 100) * 4.00,
      100 * 5.00 + 400 * 4.00 + (weight - 500) * 3.00
   )
)
```

### Multi-Factor Pricing

```javascript
// Complex pricing with multiple factors
// All multipliers calculated inline
distance * base_rate *
  (hasHazmat ? 1.25 : 1.0) *
  (requiresTemperatureControl ? 1.15 : 1.0) *
  (isExpedited ? 1.50 : 1.0) *
  (1 + (fuel_surcharge / 100))
```

### Zone-Based Pricing

```javascript
// Zone-based pricing using array indexing
// Zone rates: [100, 150, 200, 250, 300]
// Calculate zone index based on distance and apply rate
[100, 150, 200, 250, 300][min(floor(distance / 100), 4)] + (weight * 0.05)
```

### Accessorial Charges

```javascript
// Calculate total accessorial charges
(needs_liftgate ? 75 : 0) +
(is_inside_delivery ? 50 : 0) +
(is_residential ? 35 : 0) +
(delivery_hour < 8 || delivery_hour > 17 ? 100 : 0)
```

### Dynamic Discount Calculation

```javascript
// Apply best discount between volume and loyalty
base_amount * (1 - max(
  // Volume discount
  if(monthly_volume > 50000, 0.15,
     if(monthly_volume > 25000, 0.10,
        if(monthly_volume > 10000, 0.05, 0))),
  // Loyalty discount (capped at 10%)
  min((current_date - customer_since_date) / 365 * 0.01, 0.10)
))
```

### Complex Business Rules

```javascript
// Apply rates based on service criteria
// Service level determined by conditions
base_rate * (
  hasHazmat && distance > 500 ? 2.5 :                            // specialized
  requiresTemperatureControl && temperatureDifferential > 50 ? 2.0 : // refrigerated
  weight > 10000 || pieces > 50 ? 1.5 :                          // ltl
  isExpedited ? 1.75 :                                           // expedited
  1.0                                                            // standard
)
```

### Array Operations Example

```javascript
// Calculate insurance based on total commodity value
// Insurance rate: 0.2% for high value, 0.1% for standard
// Minimum charge: $25
max(
  total_commodity_value * 
  (total_commodity_value > 10000 ? 0.002 : 0.001),
  25
)
```

## Limitations

### Resource Limits

| Limit | Value | Description |
|-------|-------|-------------|
| Expression Length | 10,000 chars | Maximum characters in an expression |
| Token Count | 1,000 tokens | Maximum number of lexical tokens |
| Expression Depth | 50 levels | Maximum nesting depth |
| Evaluation Timeout | 100ms | Maximum evaluation time (configurable) |
| Memory Limit | 1MB | Maximum memory per evaluation |
| Array Size | 10,000 elements | Maximum array length |
| String Length | 100,000 chars | Maximum string length |

### Language Limitations

- **No Variable Assignment**: Cannot create or assign values to variables (e.g., `x = 5` is not allowed)
- **No Loops**: No `for`, `while`, or `do-while` constructs
- **No Recursion**: Functions cannot call themselves
- **No User Functions**: Cannot define custom functions in expressions
- **No Side Effects**: Expressions are pure and cannot modify external state
- **No I/O Operations**: No file, network, or system access
- **No Code Execution**: Cannot execute arbitrary code

**Important**: All calculations must be done inline within a single expression. If you need to reuse a complex calculation, consider:
1. Creating separate formula templates for different components
2. Using nested function calls or conditional expressions
3. Leveraging the built-in functions like `map`, `filter`, and `reduce` for complex operations

### Security Restrictions

- Sandboxed execution environment
- No access to system resources
- No import or require statements
- No eval or dynamic code execution
- Input validation and sanitization
- Protected against injection attacks

## Error Handling

The expression engine provides detailed error messages with location information:

### Error Types

| Error Type | Description | Example |
|------------|-------------|----------|
| **Syntax Error** | Invalid expression syntax | `"Unexpected token ')'"` |
| **Type Error** | Invalid operation for data type | `"Cannot add number and string"` |
| **Division by Zero** | Attempting to divide by zero | `"Division by zero at position 15"` |
| **Index Out of Bounds** | Array index beyond array length | `"Index 5 out of bounds for array of length 3"` |
| **Unknown Variable** | Reference to undefined variable | `"Variable 'foo' not found"` |
| **Unknown Function** | Call to undefined function | `"Function 'bar' not defined"` |
| **Argument Count** | Wrong number of function arguments | `"Function 'round' expects 2 arguments, got 1"` |
| **Resource Limit** | Expression too complex | `"Expression depth limit exceeded"` |
| **Timeout** | Evaluation took too long | `"Expression evaluation timed out after 100ms"` |

### Error Format

Errors include position information:

```javascript
// Expression: base_rate * (1 + tax_rate / 100
// Error: "Unexpected end of expression, expected ')' at position 28"
```

### Safe Error Recovery

The engine ensures safe error recovery:
- No panic or crashes
- Clean error propagation
- Resource cleanup on errors
- Detailed error context

Always test formulas with representative data and edge cases before using in production.
