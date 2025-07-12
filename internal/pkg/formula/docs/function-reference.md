# Function Reference

This document provides a comprehensive reference for all built-in functions available in formula expressions.

## Table of Contents

- [Mathematical Functions](#mathematical-functions)
  - [Basic Math](#basic-math)
  - [Advanced Math](#advanced-math)
  - [Rounding](#rounding)
- [Type Conversion Functions](#type-conversion-functions)
- [Array Functions](#array-functions)
- [String Functions](#string-functions)
- [Conditional Functions](#conditional-functions)

## Mathematical Functions

### Basic Math

#### abs(x)
Returns the absolute value of a number.

**Parameters:**
- `x` (number): The input value

**Returns:** number

**Examples:**
```javascript
abs(-5)      // 5
abs(3.14)    // 3.14
abs(0)       // 0
```

#### min(...values)
Returns the smallest value from the arguments.

**Parameters:**
- `...values` (numbers): One or more numeric values

**Returns:** number

**Examples:**
```javascript
min(5, 3, 8)           // 3
min(10)                // 10
min(-5, -3, -8)        // -8
```

#### max(...values)
Returns the largest value from the arguments.

**Parameters:**
- `...values` (numbers): One or more numeric values

**Returns:** number

**Examples:**
```javascript
max(5, 3, 8)           // 8
max(10)                // 10
max(-5, -3, -8)        // -3
```

#### pow(base, exponent)
Returns base raised to the power of exponent.

**Parameters:**
- `base` (number): The base value
- `exponent` (number): The power to raise the base to

**Returns:** number

**Examples:**
```javascript
pow(2, 3)              // 8
pow(10, 2)             // 100
pow(5, 0)              // 1
pow(2, -1)             // 0.5
```

#### sqrt(x)
Returns the square root of a number.

**Parameters:**
- `x` (number): A non-negative number

**Returns:** number

**Errors:**
- Returns error if x is negative

**Examples:**
```javascript
sqrt(16)               // 4
sqrt(2)                // 1.414...
sqrt(0)                // 0
sqrt(-1)               // Error: cannot take square root of negative number
```

### Advanced Math

#### log(x, base?)
Returns the logarithm of x. If base is not provided, returns the natural logarithm.

**Parameters:**
- `x` (number): A positive number
- `base` (number, optional): The logarithm base (must be positive and not 1)

**Returns:** number

**Errors:**
- Returns error if x <= 0
- Returns error if base <= 0 or base == 1

**Examples:**
```javascript
log(10)                // 2.302... (natural log)
log(100, 10)           // 2 (log base 10)
log(8, 2)              // 3 (log base 2)
log(1)                 // 0
```

#### exp(x)
Returns e (Euler's number) raised to the power of x.

**Parameters:**
- `x` (number): The exponent

**Returns:** number

**Examples:**
```javascript
exp(0)                 // 1
exp(1)                 // 2.718... (e)
exp(2)                 // 7.389...
exp(-1)                // 0.368...
```

#### sin(x)
Returns the sine of x (x in radians).

**Parameters:**
- `x` (number): Angle in radians

**Returns:** number

**Examples:**
```javascript
sin(0)                 // 0
sin(PI/2)              // 1
sin(PI)                // 0 (approximately)
sin(3*PI/2)            // -1
```

#### cos(x)
Returns the cosine of x (x in radians).

**Parameters:**
- `x` (number): Angle in radians

**Returns:** number

**Examples:**
```javascript
cos(0)                 // 1
cos(PI/2)              // 0 (approximately)
cos(PI)                // -1
cos(2*PI)              // 1
```

#### tan(x)
Returns the tangent of x (x in radians).

**Parameters:**
- `x` (number): Angle in radians

**Returns:** number

**Examples:**
```javascript
tan(0)                 // 0
tan(PI/4)              // 1 (approximately)
tan(PI)                // 0 (approximately)
```

### Rounding

#### round(x, precision?)
Rounds a number to the specified decimal places.

**Parameters:**
- `x` (number): The number to round
- `precision` (number, optional): Number of decimal places (default: 0)

**Returns:** number

**Examples:**
```javascript
round(3.14159)         // 3
round(3.14159, 2)      // 3.14
round(3.14159, 4)      // 3.1416
round(1234.5)          // 1235
round(1234.5, -2)      // 1200
```

#### floor(x)
Returns the largest integer less than or equal to x.

**Parameters:**
- `x` (number): The input value

**Returns:** number

**Examples:**
```javascript
floor(3.9)             // 3
floor(3.1)             // 3
floor(-3.1)            // -4
floor(5)               // 5
```

#### ceil(x)
Returns the smallest integer greater than or equal to x.

**Parameters:**
- `x` (number): The input value

**Returns:** number

**Examples:**
```javascript
ceil(3.1)              // 4
ceil(3.9)              // 4
ceil(-3.1)             // -3
ceil(5)                // 5
```

## Type Conversion Functions

#### number(value)
Converts a value to a number.

**Parameters:**
- `value` (any): The value to convert

**Returns:** number

**Errors:**
- Returns error if value cannot be converted to a number

**Examples:**
```javascript
number("42")           // 42
number("3.14")         // 3.14
number(true)           // 1
number(false)          // 0
number("abc")          // Error: cannot convert string "abc" to number
```

#### string(value)
Converts a value to a string.

**Parameters:**
- `value` (any): The value to convert

**Returns:** string

**Examples:**
```javascript
string(42)             // "42"
string(3.14)           // "3.14"
string(true)           // "true"
string([1, 2, 3])      // "[1 2 3]"
```

#### bool(value)
Converts a value to a boolean.

**Parameters:**
- `value` (any): The value to convert

**Returns:** boolean

**Conversion Rules:**
- Numbers: 0 is false, all others are true
- Strings: empty string is false, all others are true
- Arrays: empty array is false, all others are true
- null/nil is false

**Examples:**
```javascript
bool(1)                // true
bool(0)                // false
bool("hello")          // true
bool("")               // false
bool([1, 2, 3])        // true
bool([])               // false
```

## Array Functions

#### len(value)
Returns the length of an array or string.

**Parameters:**
- `value` (array|string): An array or string

**Returns:** number

**Examples:**
```javascript
len([1, 2, 3])         // 3
len("hello")           // 5
len([])                // 0
len("")                // 0
```

#### sum(...values)
Returns the sum of all numeric values. Can accept individual numbers or arrays of numbers.

**Parameters:**
- `...values` (numbers|arrays): Numbers or arrays of numbers to sum

**Returns:** number

**Examples:**
```javascript
sum(1, 2, 3)           // 6
sum([1, 2, 3])         // 6
sum([1, 2], 3, [4, 5]) // 15
sum([])                // 0
```

#### avg(...values)
Returns the average of all numeric values. Can accept individual numbers or arrays of numbers.

**Parameters:**
- `...values` (numbers|arrays): Numbers or arrays of numbers to average

**Returns:** number

**Errors:**
- Returns error if no values provided or empty array

**Examples:**
```javascript
avg(1, 2, 3)           // 2
avg([1, 2, 3])         // 2
avg([1, 2], 3, [4, 5]) // 3
avg([10, 20, 30])      // 20
```

#### slice(array, start, end?)
Returns a portion of an array or string.

**Parameters:**
- `array` (array|string): The array or string to slice
- `start` (number): Starting index (negative counts from end)
- `end` (number, optional): Ending index (exclusive, negative counts from end)

**Returns:** array|string (same type as input)

**Examples:**
```javascript
slice([1, 2, 3, 4, 5], 1, 4)     // [2, 3, 4]
slice([1, 2, 3, 4, 5], -3)       // [3, 4, 5]
slice([1, 2, 3, 4, 5], 1, -1)    // [2, 3, 4]
slice("hello world", 0, 5)       // "hello"
slice("hello world", -5)         // "world"
```

#### concat(...values)
Concatenates arrays or strings together.

**Parameters:**
- `...values` (arrays|strings): Values to concatenate

**Returns:** array|string

**Behavior:**
- If all arguments are strings, returns concatenated string
- Otherwise, returns concatenated array
- Non-array values are treated as single-element arrays

**Examples:**
```javascript
concat([1, 2], [3, 4])           // [1, 2, 3, 4]
concat("hello", " ", "world")    // "hello world"
concat([1], 2, [3, 4])           // [1, 2, 3, 4]
concat([], [1], [], [2])         // [1, 2]
```

#### contains(collection, value)
Checks if an array contains a value or if a string contains a substring.

**Parameters:**
- `collection` (array|string): The array or string to search in
- `value` (any): The value or substring to search for

**Returns:** boolean

**Examples:**
```javascript
contains([1, 2, 3], 2)           // true
contains([1, 2, 3], 4)           // false
contains("hello world", "world") // true
contains("hello", "goodbye")     // false
contains(["a", "b", "c"], "b")   // true
```

#### indexOf(collection, value)
Returns the index of a value in an array or substring in a string. Returns -1 if not found.

**Parameters:**
- `collection` (array|string): The array or string to search in
- `value` (any): The value or substring to search for

**Returns:** number

**Examples:**
```javascript
indexOf([10, 20, 30], 20)        // 1
indexOf([10, 20, 30], 40)        // -1
indexOf("hello world", "world")  // 6
indexOf("hello", "goodbye")      // -1
indexOf(["a", "b", "c"], "b")    // 1
```

## String Functions

String manipulation is primarily done through array functions that also work with strings:

- `len(string)` - Returns string length
- `slice(string, start, end?)` - Extracts substring
- `concat(...strings)` - Concatenates strings
- `contains(string, substring)` - Checks for substring
- `indexOf(string, substring)` - Finds substring position
- `string[index]` - Access character at index

**Examples:**
```javascript
len("hello")                     // 5
slice("hello world", 6)          // "world"
concat("hello", " ", "world")    // "hello world"
contains("hello", "ell")         // true
indexOf("hello", "ll")           // 2
"hello"[1]                       // "e"
```

## Conditional Functions

#### if(condition, trueValue, falseValue)
Returns trueValue if condition is true, otherwise returns falseValue.

**Parameters:**
- `condition` (any): The condition to evaluate (converted to boolean)
- `trueValue` (any): Value to return if condition is true
- `falseValue` (any): Value to return if condition is false

**Returns:** any

**Examples:**
```javascript
if(true, "yes", "no")            // "yes"
if(false, "yes", "no")           // "no"
if(score > 90, "A", "B")         // "A" if score > 90
if(quantity > 0, price, 0)       // price if quantity > 0, else 0

// Nested conditions
if(score > 90, "A",
   if(score > 80, "B",
      if(score > 70, "C", "F")
   )
)
```

#### coalesce(...values)
Returns the first non-null, non-zero, non-empty value from the arguments.

**Parameters:**
- `...values` (any): Values to check

**Returns:** any

**Behavior:**
- Skips null/nil values
- Skips empty strings
- Skips zero numbers
- Returns first "truthy" value or null if all are falsy

**Examples:**
```javascript
coalesce(null, 0, "", "hello")   // "hello"
coalesce(null, 42)               // 42
coalesce(0, 10, 20)              // 10
coalesce("", "default")          // "default"
coalesce(false, true)            // false (booleans returned as-is)
```

## Function Composition

Functions can be nested and combined:

```javascript
// Mathematical composition
sqrt(pow(3, 2) + pow(4, 2))     // 5 (Pythagorean theorem)

// Array operations
sum(slice(prices, 0, -1))        // Sum all but last price
avg(concat(scores1, scores2))    // Average of combined scores

// Conditional logic
if(contains(tags, "urgent"), 
   min(deadline, 24),
   max(deadline, 72)
)

// Type conversion
number(slice("12345", 1, 3))     // 23
```

## Error Handling

Functions validate their inputs and return descriptive errors:

- **Type Errors**: When arguments are wrong type
- **Argument Count**: When too few or too many arguments
- **Range Errors**: When numeric inputs are out of valid range
- **Division by Zero**: When dividing by zero
- **Invalid Operations**: Like square root of negative number

Always test formulas with edge cases to ensure proper error handling.