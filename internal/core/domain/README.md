# `/internal/core/domain` Directory Documentation

## Overview

The `domain` directory contains the core business objects, rules, and logic of the application. This is the heart of your domain-driven design (DDD) implementation, representing the business concepts and their relationships in code.

## Directory Structure

```markdown
/internal/core/domain/
├── shipment/                # Shipment domain objects and logic
│   ├── shipment.go         # Main shipment entity
│   ├── status.go           # Shipment status definitions
│   └── events.go           # Shipment-related domain events
├── vehicle/                # Vehicle domain objects and logic
├── driver/                 # Driver domain objects and logic
├── route/                  # Route planning and optimization
└── tenant/                 # Multi-tenancy domain objects
```

## Guidelines & Rules

### General Rules

- Domain objects should be independent of infrastructure concerns
- Business rules should be enforced within domain objects
- Use value objects for concepts that are defined by their attributes
- Include domain events for significant state changes
- Implement validation within domain objects

### Domain Objects Should Include

1. **Entities**

   ```go
   type Shipment struct {
       ID            string
       TrackingCode  string
       Status        ShipmentStatus
       Origin        Address
       Destination   Address
       Weight        Weight
       CreatedAt     int64
       UpdatedAt     int64
   }
   ```

2. **Value Objects**

   ```go
   type Address struct {
       Street     string
       City       string
       State      string
       PostalCode string
       Country    string
   }
   ```

3. **Domain Events**

   ```go
   type ShipmentStatusChanged struct {
       ShipmentID     string
       PreviousStatus ShipmentStatus
       NewStatus      ShipmentStatus
       ChangedAt      int64
   }
   ```

4. **Business Rules & Validations**

   ```go
   func (s *Shipment) UpdateStatus(newStatus ShipmentStatus) error {
       if !s.canTransitionTo(newStatus) {
           return ErrInvalidStatusTransition
       }
       s.Status = newStatus
       s.UpdatedAt = time.Now()
       return nil
   }
   ```

## Best Practices

1. **Entity Design**
   - Each entity should have a unique identifier
   - Implement domain-specific validation
   - Include business logic methods
   - Use proper encapsulation

2. **Value Objects**
   - Should be immutable
   - Equality based on attributes
   - No identity concept
   - Used for measurements, descriptions, and calculations

3. **Domain Events**
   - Name events in past tense (e.g., ShipmentCreated)
   - Include all relevant event data
   - Add timestamp to events
   - Keep events immutable

4. **Business Rules**
   - Implement as methods on domain objects
   - Use descriptive error types
   - Validate state transitions
   - Enforce invariants

## What Does NOT Belong Here

1. **Infrastructure Concerns**
   - Database operations
   - External service calls
   - Framework-specific code
   - HTTP/gRPC handling

2. **Application Logic**
   - Use case orchestration
   - Transaction management
   - External service coordination

3. **Presentation Logic**
   - DTO transformations
   - JSON/XML handling
   - Request/Response formatting

## Example Entity Implementation

```go
// shipment.go
package shipment

import (
    "time"
    "errors"
)

var (
    ErrInvalidWeight = errors.New("shipment weight must be greater than 0")
    ErrInvalidStatus = errors.New("invalid status transition")
)

type Shipment struct {
    ID           string
    TrackingCode string
    Status       Status
    Weight       Weight
    Route        Route
    createdAt    int64
    updatedAt    int64
}

// NewShipment creates a new shipment with validation
func NewShipment(trackingCode string, weight Weight) (*Shipment, error) {
    if !weight.IsValid() {
        return nil, ErrInvalidWeight
    }

    return &Shipment{
        ID:           generateID(),
        TrackingCode: trackingCode,
        Status:       StatusPending,
        Weight:       weight,
        createdAt:    time.Now(),
        updatedAt:    time.Now(),
    }, nil
}

// Business rule: Status can only transition in a specific order
func (s *Shipment) UpdateStatus(newStatus Status) error {
    if !s.canTransitionTo(newStatus) {
        return ErrInvalidStatus
    }

    s.Status = newStatus
    s.updatedAt = time.Now()
    return nil
}

// Private helper for status transition validation
func (s *Shipment) canTransitionTo(newStatus Status) bool {
    validTransitions := map[Status][]Status{
        StatusPending:   {StatusInTransit},
        StatusInTransit: {StatusDelivered},
        StatusDelivered: {},
    }

    allowedTransitions, exists := validTransitions[s.Status]
    if !exists {
        return false
    }

    for _, status := range allowedTransitions {
        if status == newStatus {
            return true
        }
    }
    return false
}
```

## Testing Guidelines

- Test business rules thoroughly
- Use table-driven tests for validations
- Test domain events
- Ensure invariants are maintained
- Test state transitions
