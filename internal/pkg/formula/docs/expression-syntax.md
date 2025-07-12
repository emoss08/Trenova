# Expression Syntax Guide

This guide covers the complete syntax for formula expressions in the Trenova system.

## Table of Contents

- [Basic Syntax](#basic-syntax)
- [Data Types](#data-types)
- [Operators](#operators)
- [Variables](#variables)
- [Functions](#functions)
- [Arrays](#arrays)
- [Conditionals](#conditionals)
- [Best Practices](#best-practices)

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
| `||` | Logical OR | `true || false` | `true` |
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

See the [Function Reference](function-reference.md) for a complete list of available functions.

## Arrays

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

## Type Coercion

The expression engine performs automatic type conversion in certain contexts:

### Numeric Contexts

Values are converted to numbers when needed:

```javascript
"42" + 0        // 42 (string to number)
true + 0        // 1 (boolean to number)
false + 0       // 0 (boolean to number)
```

### String Contexts

Values are converted to strings with the `+` operator:

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

// Better: use intermediate variables
hazmat_multiplier = if(has_hazmat, 1.25, 1)
expedited_multiplier = if(is_expedited, 1.5, 1)
fuel_factor = 1 + fuel_surcharge/100
base_rate * distance * hazmat_multiplier * expedited_multiplier * fuel_factor
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
// Weight-based tiers
if(weight <= 100,
   weight * 5.00,
   if(weight <= 500,
      100 * 5.00 + (weight - 100) * 4.00,
      100 * 5.00 + 400 * 4.00 + (weight - 500) * 3.00
   )
)
```

### Distance Calculation

```javascript
// Haversine formula (simplified)
radius = 3959  // Earth radius in miles
lat_diff = abs(dest_lat - origin_lat)
lon_diff = abs(dest_lon - origin_lon)
a = sin(lat_diff/2)^2 + cos(origin_lat) * cos(dest_lat) * sin(lon_diff/2)^2
c = 2 * atan2(sqrt(a), sqrt(1-a))
radius * c
```

### Business Rules

```javascript
// Delivery window calculation
is_expedited && distance < 100 ? 4 : 
is_expedited && distance < 500 ? 8 :
is_expedited ? 24 :
distance < 100 ? 24 :
distance < 500 ? 48 :
72
```

## Limitations

- Maximum expression length: 10,000 characters
- Maximum token count: 1,000 tokens
- Maximum expression complexity: 1,000
- No loops or recursion
- No user-defined functions
- No variable assignment within expressions

## Error Handling

Common expression errors:

- **Syntax Error**: Invalid expression syntax
- **Type Error**: Invalid operation for data type
- **Division by Zero**: Attempting to divide by zero
- **Index Out of Bounds**: Array index beyond array length
- **Unknown Variable**: Reference to undefined variable
- **Unknown Function**: Call to undefined function
- **Argument Count**: Wrong number of function arguments

Always test formulas with representative data before using in production.
