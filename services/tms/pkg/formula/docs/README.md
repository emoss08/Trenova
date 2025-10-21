<!--
Copyright 2023-2025 Eric Moss
Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md-->
# Formula Documentation

This directory contains comprehensive documentation for the Trenova Formula package.

## Documentation Structure

### Core Documentation

- **[Expression Syntax Guide](expression-syntax.md)** - Complete syntax reference for formula expressions
- **[Function Reference](function-reference.md)** - Detailed documentation of all built-in functions
- **[Integration Guide](integration-guide.md)** - How to integrate formulas into Trenova applications

### API Documentation

The `api/` subdirectory contains technical documentation for developers:

- **[API Overview](api/README.md)** - Technical API overview
- **[Type Conversion](api/conversion.md)** - Type system and conversion rules
- **[Error Handling](api/errors.md)** - Error types and handling
- **[Expression Engine](api/expression.md)** - Expression parsing and evaluation
- **[Schema System](api/schema.md)** - JSON Schema integration
- **[Variable System](api/variables.md)** - Variable registration and resolution

## Quick Links

### For Users

1. Start with the [Expression Syntax Guide](expression-syntax.md) to learn the formula language
2. Reference the [Function Reference](function-reference.md) for available functions
3. See practical examples in the [Integration Guide](integration-guide.md)

### For Developers

1. Read the [main README](../README.md) for architecture overview
2. Follow the [Integration Guide](integration-guide.md) for implementation patterns
3. Consult the [API documentation](api/) for technical details

## Common Use Cases

### Shipment Pricing

```javascript
// Basic rate calculation
base_rate * distance * (1 + fuel_surcharge/100)

// Multi-factor pricing
base_rate * distance * 
  (hasHazmat ? 1.25 : 1.0) *
  (requiresTemperatureControl ? 1.15 : 1.0)
```

### Business Rules

```javascript
// Service level determination
if(weight > 10000 || pieces > 50, "ltl",
   if(isExpedited, "expedited", "standard"))

// Delivery time calculation
is_expedited && distance < 100 ? 4 : 
is_expedited && distance < 500 ? 8 : 24
```

### Conditional Charges

```javascript
// Total accessorial charges
(needs_liftgate ? 75 : 0) + 
(is_residential ? 35 : 0) + 
(delivery_hour < 8 || delivery_hour > 17 ? 100 : 0)
```

## Getting Help

- Check the [Expression Syntax Guide](expression-syntax.md) for syntax questions
- See the [Function Reference](function-reference.md) for function usage
- Review the [Integration Guide](integration-guide.md) for implementation help
- Consult the [API documentation](api/) for technical details

## Contributing

When adding new features or functions:

1. Update the relevant documentation files
2. Add examples showing typical usage
3. Document any new error conditions
4. Update the function reference if adding functions
5. Keep the integration guide current with new patterns