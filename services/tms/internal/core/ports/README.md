<!--
Copyright 2023-2025 Eric Moss
Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md-->
# `/internal/core/ports` Directory Documentation

## Overview

The `ports` directory contains interface definitions that establish boundaries between different layers of the application following clean architecture principles. These interfaces act as contracts between the domain layer and external adapters.

## Directory Structure

```markdown
/internal/core/ports/
├── repositories/           # Database operation interfaces
├── services/              # Business service interfaces
└── types.go              # Shared types and interfaces
```

## Purpose & Guidelines

### General Rules

- Contains ONLY interface definitions, no implementations
- Each interface should have a single responsibility
- Use clear, descriptive names that indicate the interface's purpose
- Document each interface method with clear input/output expectations
- Include any custom types needed by the interfaces

### `/repositories`

- Define interfaces for data persistence operations
- Each domain entity should have its own repository interface
- Methods should be CRUD-focused and domain-specific
- Example interface structure:

```go
type ShipmentRepository interface {
    Create(ctx context.Context, shipment *domain.Shipment) error
    GetByID(ctx context.Context, id string) (*domain.Shipment, error)
    Update(ctx context.Context, shipment *domain.Shipment) error
    Delete(ctx context.Context, id string) error
    List(ctx context.Context, filter ShipmentFilter) ([]domain.Shipment, error)
}
```

### `/services`

- Define interfaces for business operations and external services
- Each business capability should have its own service interface
- Methods should represent business use cases
- Example interface structure:

```go
type RoutingService interface {
    OptimizeRoute(ctx context.Context, shipments []domain.Shipment) (*domain.Route, error)
    CalculateETA(ctx context.Context, route domain.Route) (time.Time, error)
    UpdateTrafficConditions(ctx context.Context, routeID string) error
}
```

## Best Practices

1. **Interface Segregation**
   - Keep interfaces small and focused
   - Split large interfaces into smaller, more specific ones
   - Clients should not depend on methods they don't use

2. **Naming Conventions**
   - Repository interfaces should end with `Repository`
   - Service interfaces should end with `Service`
   - Method names should be clear and action-oriented

3. **Error Handling**
   - Define domain-specific errors when needed
   - Use meaningful error types that help with error handling
   - Include error documentation in interface comments

4. **Context Usage**
   - All methods should accept context.Context as first parameter
   - Use for cancellation, timeouts, and request-scoped values

5. **Documentation**
   - Document interface purpose and usage
   - Include examples for non-obvious use cases
   - Document any special error conditions

## What Does NOT Belong Here

1. **Implementations**
   - No concrete implementations of interfaces
   - No business logic
   - No direct database code

2. **Infrastructure Concerns**
   - No SQL queries
   - No external service clients
   - No configuration code

3. **Domain Models**
   - No domain entity definitions
   - No value objects
   - No domain logic

## Example Interface Documentation

```go
// PaymentService handles payment processing and verification
// for shipment transactions.
type PaymentService interface {
    // ProcessPayment handles the payment for a shipment
    // Returns ErrInsufficientFunds if payment fails due to funds
    // Returns ErrPaymentDeclined if payment is declined for any other reason
    ProcessPayment(ctx context.Context, payment domain.Payment) error

    // RefundPayment processes a refund for a previously successful payment
    // Returns ErrPaymentNotFound if original payment cannot be found
    // Returns ErrRefundFailed if refund cannot be processed
    RefundPayment(ctx context.Context, paymentID string, amount decimal.Decimal) error
}
```
