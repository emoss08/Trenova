# Formula Template User Guide

> **Note**: This documentation was generated with AI assistance and reviewed for accuracy. If you find any errors or have questions, please contact your system administrator.

---

## What Are Formula Templates?

Formula templates let you create custom billing calculations for your shipments. Instead of using fixed rates, you can build formulas that automatically calculate charges based on shipment details like distance, weight, number of stops, and special requirements.

**Example**: Instead of manually calculating "base rate + $2.50 per mile + hazmat fee if applicable", you create a formula template that does this automatically for every shipment.

---

## Getting Started

### Your First Formula

Let's start with a simple example. To charge a flat rate of $500 for every shipment:

```
500
```

That's it! The formula simply returns 500.

Now let's make it more useful. To charge $2.50 per mile:

```
2.50 * totalDistance
```

This formula multiplies your rate ($2.50) by the total distance of the shipment. If a shipment travels 200 miles, the charge is $500.

---

## Available Shipment Information

When you write a formula, you can use these shipment values:

### Distance & Stops

| Variable | Description | Example Value |
|----------|-------------|---------------|
| `totalDistance` | Total miles across all moves | `575.8` |
| `totalStops` | Number of pickup and delivery stops | `3` |

### Weight & Pieces

| Variable | Description | Example Value |
|----------|-------------|---------------|
| `totalWeight` | Total shipment weight in pounds | `45000` |
| `totalPieces` | Total number of pieces/units | `24` |
| `totalLinearFeet` | Total linear feet (pieces × linearFeetPerUnit) | `32.0` |

### Special Requirements

| Variable | Description | Example Value |
|----------|-------------|---------------|
| `hasHazmat` | Is hazardous material present? | `true` or `false` |
| `requiresTemperatureControl` | Does shipment need refrigeration? | `true` or `false` |
| `temperatureDifferential` | Difference between min and max temp (°F) | `4` |

### Existing Charge Amounts

| Variable | Description | Example Value |
|----------|-------------|---------------|
| `freightChargeAmount` | Current freight charge on shipment | `1250.00` |
| `otherChargeAmount` | Current other charges on shipment | `175.00` |
| `currentTotalCharge` | Current total charge on shipment | `1425.00` |

---

## User-Defined Variables

In addition to shipment information, you can define your own variables for rates and fees. These are values you set when creating the template that can be adjusted per customer or contract.

**Common user-defined variables:**

| Variable | Purpose | Example |
|----------|---------|---------|
| `baseRate` | Starting charge for every shipment | `75.00` |
| `ratePerMile` | Charge per mile traveled | `2.85` |
| `ratePerStop` | Charge per stop | `45.00` |
| `ratePerCWT` | Rate per hundred-weight (CWT) | `18.50` |
| `minimumCharge` | Lowest amount to charge | `250.00` |
| `hazmatFee` | Fee for hazardous materials | `175.00` |
| `reeferFee` | Fee for temperature control | `225.00` |
| `fuelSurchargePercent` | Fuel surcharge as percentage | `18.5` |

You define these when creating your template and can set default values.

---

## Writing Formulas

### Basic Math Operations

| Operation | Symbol | Example | Result |
|-----------|--------|---------|--------|
| Addition | `+` | `100 + 50` | `150` |
| Subtraction | `-` | `100 - 30` | `70` |
| Multiplication | `*` | `25 * 4` | `100` |
| Division | `/` | `100 / 4` | `25` |
| Parentheses | `( )` | `(100 + 50) * 2` | `300` |

### Conditional Logic (If/Then)

Use the `?` and `:` symbols to create if/then logic:

```
condition ? valueIfTrue : valueIfFalse
```

**Example**: Charge $150 hazmat fee only if shipment contains hazardous materials:

```
hasHazmat ? 150 : 0
```

This reads as: "If has hazmat, charge 150, otherwise charge 0"

**Example**: Apply different rates based on conditions:

```
hasHazmat ? hazmatRate : standardRate
```

---

## Common Formula Patterns

### 1. Flat Rate

Charge the same amount for every shipment:

```
baseRate
```

### 1b. Flat Fee (Use Existing Charges)

Use charges already entered on the shipment:

```
freightChargeAmount + otherChargeAmount
```

This reads the freight and other charge amounts that were manually entered on the shipment and adds them together. Useful when charges are pre-negotiated or manually set.

### 2. Per Mile

Charge based on distance traveled:

```
ratePerMile * totalDistance
```

### 3. Per Mile with Base Rate

Start with a base charge, then add per-mile:

```
baseRate + (ratePerMile * totalDistance)
```

### 4. Per Mile with Minimum

Ensure you always charge at least a minimum amount:

```
max(minimumCharge, ratePerMile * totalDistance)
```

### 5. Per Hundred-Weight (CWT)

Charge based on weight (per 100 lbs):

```
ratePerCWT * (totalWeight / 100)
```

### 6. Per Stop

Charge based on number of stops:

```
baseRate + (ratePerStop * totalStops)
```

### 7. With Fuel Surcharge

Add a percentage-based fuel surcharge:

```
(baseRate + ratePerMile * totalDistance) * (1 + fuelSurchargePercent / 100)
```

If base is $75, rate is $2.50/mile, distance is 200 miles, and fuel surcharge is 18%:

- Line haul: $75 + ($2.50 × 200) = $575
- With fuel: $575 × 1.18 = $678.50

### 8. Conditional Accessorial Fees

Add fees only when certain conditions apply:

```
baseRate + (ratePerMile * totalDistance) +
(hasHazmat ? hazmatFee : 0) +
(requiresTemperatureControl ? reeferFee : 0)
```

### 9. Tiered Pricing by Weight

Different rates for different weight ranges:

```
totalWeight < 10000 ? totalWeight * 0.15 :
totalWeight < 25000 ? 10000 * 0.15 + (totalWeight - 10000) * 0.12 :
10000 * 0.15 + 15000 * 0.12 + (totalWeight - 25000) * 0.09
```

### 10. Complete Freight Bill

A comprehensive formula combining multiple elements:

```
max(
    minimumCharge,
    (baseRate + ratePerMile * totalDistance + ratePerStop * totalStops) *
    (1 + fuelSurchargePercent / 100) +
    (hasHazmat ? hazmatFee : 0) +
    (requiresTemperatureControl ? reeferFee : 0)
)
```

---

## Available Functions

### Rounding Functions

| Function | Description | Example | Result |
|----------|-------------|---------|--------|
| `round(value)` | Round to nearest whole number | `round(3.7)` | `4` |
| `round(value, decimals)` | Round to specific decimal places | `round(3.14159, 2)` | `3.14` |
| `ceil(value)` | Round up to next whole number | `ceil(3.2)` | `4` |
| `floor(value)` | Round down to previous whole number | `floor(3.8)` | `3` |

**Tip**: Use `round(yourFormula, 2)` to round your final charge to cents.

### Comparison Functions

| Function | Description | Example | Result |
|----------|-------------|---------|--------|
| `min(a, b)` | Returns the smaller value | `min(100, 250)` | `100` |
| `max(a, b)` | Returns the larger value | `max(100, 250)` | `250` |
| `clamp(value, min, max)` | Keep value within a range | `clamp(300, 100, 250)` | `250` |

**Use case for `max`**: Ensuring a minimum charge:

```
max(minimumCharge, calculatedCharge)
```

**Use case for `clamp`**: Setting both minimum and maximum:

```
clamp(calculatedCharge, 250, 5000)
```

This ensures the charge is at least $250 but no more than $5,000.

### Math Functions

| Function | Description | Example | Result |
|----------|-------------|---------|--------|
| `abs(value)` | Absolute value (remove negative) | `abs(-50)` | `50` |
| `pow(base, exponent)` | Raise to a power | `pow(2, 3)` | `8` |
| `sqrt(value)` | Square root | `sqrt(16)` | `4` |

### Aggregation Functions

| Function | Description | Example | Result |
|----------|-------------|---------|--------|
| `sum(a, b, c, ...)` | Add multiple values | `sum(100, 50, 25)` | `175` |
| `avg(a, b, c, ...)` | Average of values | `avg(100, 50, 25)` | `58.33` |

### Null Handling

| Function | Description | Example |
|----------|-------------|---------|
| `coalesce(a, b, c, ...)` | Returns first non-empty value | `coalesce(customRate, defaultRate)` |

**Use case**: Use a custom rate if provided, otherwise fall back to default:

```
coalesce(customerSpecificRate, standardRate) * totalDistance
```

---

## Best Practices

### 1. Always Set a Minimum Charge

Protect against very short shipments that wouldn't cover costs:

```
max(minimumCharge, yourCalculation)
```

### 2. Round Your Final Result

Avoid charges like $547.8333333:

```
round(yourCalculation, 2)
```

### 3. Use Descriptive Variable Names

When defining your variables, use clear names:

- ✅ `ratePerMile`, `hazmatFee`, `minimumCharge`
- ❌ `r1`, `fee`, `min`

### 4. Test with Different Scenarios

Before using a formula, test it with:

- A very short shipment (low miles)
- A very long shipment (high miles)
- A heavy shipment
- A shipment with hazmat
- A shipment with temperature control
- A shipment with many stops

### 5. Use Parentheses for Clarity

Even when not required, parentheses make formulas easier to read:

```
(baseRate) + (ratePerMile * totalDistance) + (hasHazmat ? hazmatFee : 0)
```

---

## Troubleshooting

### "Unknown variable" Error

This means you used a variable name that doesn't exist. Check:

- Spelling (variables are case-sensitive)
- That you've defined the variable in your template
- That you're using the correct shipment variable name

### Unexpected Results

If your formula returns unexpected values:

1. Break it into smaller parts and test each
2. Check your order of operations (use parentheses)
3. Verify your variable values are correct

### Formula Won't Save

- Check for syntax errors (missing parentheses, operators)
- Ensure all variable names are valid
- Verify condition syntax uses `?` and `:`

---

## Quick Reference Card

### Shipment Variables

```
totalDistance          Total miles
totalStops             Number of stops
totalWeight            Weight in pounds
totalPieces            Number of pieces
totalLinearFeet        Total linear feet
hasHazmat              true/false
requiresTemperatureControl    true/false
temperatureDifferential       Temperature range in °F
freightChargeAmount    Existing freight charge
otherChargeAmount      Existing other charges
currentTotalCharge     Existing total charge
```

### Operators

```
+  -  *  /             Math operations
( )                    Grouping
? :                    If/then/else
```

### Functions

```
round(value, decimals) Round to decimal places
ceil(value)            Round up
floor(value)           Round down
min(a, b)              Smaller of two
max(a, b)              Larger of two
clamp(val, min, max)   Keep in range
abs(value)             Absolute value
sum(a, b, ...)         Add values
avg(a, b, ...)         Average values
coalesce(a, b, ...)    First non-null
```

---

## Example Templates Library

### Standard LTL Rate

```
round(
    max(
        minimumCharge,
        baseRate + (ratePerMile * totalDistance) + (ratePerStop * totalStops)
    ) * (1 + fuelSurchargePercent / 100),
    2
)
```

### Weight-Based with Accessorials

```
round(
    (ratePerCWT * ceil(totalWeight / 100)) +
    (hasHazmat ? hazmatFee : 0) +
    (requiresTemperatureControl ? reeferFee : 0),
    2
)
```

### Customer Contract Rate

```
round(
    max(
        contractMinimum,
        (contractRatePerMile * totalDistance) * (1 - discountPercent / 100)
    ),
    2
)
```

---

*Last updated: December 2024*
*Documentation generated with AI assistance*
