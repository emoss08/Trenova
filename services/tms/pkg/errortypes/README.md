# errortypes

A structured error management package designed for seamless frontend integration with React Hook Form and comprehensive backend observability.

## Table of Contents

- [Overview](#overview)
- [Quick Start](#quick-start)
- [Error Types](#error-types)
- [Validation Errors](#validation-errors)
- [Error Context & Correlation](#error-context--correlation)
- [Structured Logging](#structured-logging)
- [HTTP Status Mapping](#http-status-mapping)
- [Stack Traces](#stack-traces)
- [Real-World Examples](#real-world-examples)

---

## Overview

The `errortypes` package provides:

- **Field-level validation errors** that map directly to React Hook Form
- **Nested field paths** (`user.address.street`, `items[0].name`)
- **Multiple error collection** in a single response
- **Error codes** for programmatic handling
- **Request correlation** for distributed tracing
- **Structured logging** integration
- **Stack traces** for debugging (dev mode)

---

## Quick Start

```go
import "github.com/emoss08/trenova/pkg/errortypes"

// Create a validation error collector
multiErr := errortypes.NewMultiError()

// Add validation errors
multiErr.Add("email", errortypes.ErrRequired, "Email is required")
multiErr.Add("password", errortypes.ErrInvalidLength, "Password must be at least 8 characters")

// Check and return
if multiErr.HasErrors() {
    return multiErr // Returns 422 with field-level errors
}
```

**Frontend receives:**

```json
{
  "errors": [
    {"field": "email", "code": "REQUIRED", "message": "Email is required"},
    {"field": "password", "code": "INVALID_LENGTH", "message": "Password must be at least 8 characters"}
  ]
}
```

---

## Error Types

### Validation Errors

For field-level validation failures (HTTP 422):

```go
// Single field error
err := errortypes.NewValidationError("email", errortypes.ErrInvalidFormat, "Invalid email format")

// Collect multiple errors
multiErr := errortypes.NewMultiError()
multiErr.Add("name", errortypes.ErrRequired, "Name is required")
multiErr.Add("age", errortypes.ErrInvalid, "Age must be positive")
```

### Business Errors

For business rule violations (HTTP 422):

```go
err := errortypes.NewBusinessError("Cannot delete user with active orders").
    WithParam("userId", "123").
    WithParam("orderCount", "5")

// Check type
if errortypes.IsBusinessError(err) {
    // Handle business logic error
}
```

### Not Found Errors

For missing resources (HTTP 404):

```go
err := errortypes.NewNotFoundError("User not found").
    WithInternal(sql.ErrNoRows)

if errortypes.IsNotFoundError(err) {
    // Return 404
}
```

### Authentication Errors

For auth failures (HTTP 401):

```go
err := errortypes.NewAuthenticationError("Invalid or expired token").
    WithInternal(jwt.ErrTokenExpired)
```

### Authorization Errors

For permission failures (HTTP 403):

```go
err := errortypes.NewAuthorizationError("Insufficient permissions to access this resource")
```

### Database Errors

For database failures (HTTP 500):

```go
err := errortypes.NewDatabaseError("Failed to execute query").
    WithInternal(pgErr)
```

### Rate Limit Errors

For rate limiting (HTTP 429):

```go
err := errortypes.NewRateLimitError("api", "Too many requests. Please try again later.")
```

---

## Validation Errors

### Nested Field Paths

Use `WithPrefix` for nested objects and `WithIndex` for arrays:

```go
multiErr := errortypes.NewMultiError()

// Validate user object
userErrs := multiErr.WithPrefix("user")
userErrs.Add("name", errortypes.ErrRequired, "Name is required")

// Validate nested address
addressErrs := userErrs.WithPrefix("address")
addressErrs.Add("street", errortypes.ErrRequired, "Street is required")
addressErrs.Add("city", errortypes.ErrInvalidLength, "City name too long")

// Validate array items
for i, item := range order.Items {
    itemErrs := multiErr.WithIndex("items", i)
    if item.Quantity <= 0 {
        itemErrs.Add("quantity", errortypes.ErrInvalid, "Quantity must be positive")
    }
}
```

**Result:**

```json
{
  "errors": [
    {"field": "user.name", "code": "REQUIRED", "message": "Name is required"},
    {"field": "user.address.street", "code": "REQUIRED", "message": "Street is required"},
    {"field": "user.address.city", "code": "INVALID_LENGTH", "message": "City name too long"},
    {"field": "items[0].quantity", "code": "INVALID", "message": "Quantity must be positive"}
  ]
}
```

### Ozzo Validation Integration

Seamlessly convert ozzo-validation errors:

```go
func (u *User) Validate(multiErr *errortypes.MultiError) {
    err := validation.ValidateStruct(u,
        validation.Field(&u.Email, validation.Required, is.Email),
        validation.Field(&u.Name, validation.Required, validation.Length(1, 100)),
    )

    // One-liner conversion
    multiErr.AddOzzoError(err)
}
```

### Error Limits

Prevent excessive error collection in batch operations:

```go
// Stop collecting after 10 errors
multiErr := errortypes.NewMultiErrorWithLimit(10)

for _, record := range records {
    if multiErr.IsFull() {
        break // Stop processing
    }
    validateRecord(record, multiErr)
}
```

### Error Priority

Categorize errors by severity:

```go
multiErr.AddWithPriority("criticalField", errortypes.ErrRequired, "This is critical", errortypes.PriorityHigh)
multiErr.AddWithPriority("optionalField", errortypes.ErrInvalid, "Nice to fix", errortypes.PriorityLow)
```

---

## Error Context & Correlation

Attach request context for tracing and debugging:

```go
// Create context with correlation IDs
ctx := errortypes.NewErrorContext().
    WithRequestID("req-abc123").
    WithUserID("user-456").
    WithTraceID("trace-xyz").
    WithSpanID("span-789").
    WithExtra("tenant", "acme-corp").
    WithExtra("endpoint", "/api/v1/users")

// Attach to any error type
err := errortypes.NewBusinessError("Operation failed").
    WithContext(ctx).
    WithInternal(originalErr)

// Or to validation errors
multiErr := errortypes.NewMultiError().WithContext(ctx)
```

**JSON output includes context:**

```json
{
  "code": "BUSINESS_LOGIC",
  "message": "Operation failed",
  "context": {
    "requestId": "req-abc123",
    "userId": "user-456",
    "traceId": "trace-xyz",
    "extra": {"tenant": "acme-corp"}
  }
}
```

---

## Structured Logging

All error types provide `LogFields()` for structured logging:

```go
err := errortypes.NewBusinessError("Payment processing failed").
    WithContext(errortypes.NewErrorContext().WithRequestID("req-123")).
    WithParam("amount", "99.99").
    WithParam("currency", "USD").
    WithInternal(stripeErr)

fields := err.LogFields()
// Returns:
// map[string]any{
//   "error_code":     "BUSINESS_LOGIC",
//   "error_message":  "Payment processing failed",
//   "request_id":     "req-123",
//   "param_amount":   "99.99",
//   "param_currency": "USD",
//   "internal_error": "card declined",
// }

// Use with your logger
logger.Error("payment failed", fields)
// Or with slog
slog.Error("payment failed", slog.Any("error", fields))
```

### MultiError Logging

```go
multiErr := errortypes.NewMultiError().
    WithContext(errortypes.NewErrorContext().WithRequestID("req-456"))
multiErr.Add("email", errortypes.ErrInvalidFormat, "Invalid email")
multiErr.Add("phone", errortypes.ErrRequired, "Phone required")

fields := multiErr.LogFields()
// Returns:
// map[string]any{
//   "error_count":  2,
//   "request_id":   "req-456",
//   "error_fields": []string{"email", "phone"},
// }
```

---

## HTTP Status Mapping

Map errors to appropriate HTTP status codes:

```go
func handleError(err error) int {
    return errortypes.HTTPStatus(err)
}

// Or map by error code
func handleErrorCode(code errortypes.ErrorCode) int {
    return errortypes.HTTPStatusWithCode(code)
}
```

| Error Type | HTTP Status |
|------------|-------------|
| `MultiError` | 422 Unprocessable Entity |
| `BusinessError` | 422 Unprocessable Entity |
| `NotFoundError` | 404 Not Found |
| `AuthenticationError` | 401 Unauthorized |
| `AuthorizationError` | 403 Forbidden |
| `RateLimitError` | 429 Too Many Requests |
| `DatabaseError` | 500 Internal Server Error |

| Error Code | HTTP Status |
|------------|-------------|
| `ErrDuplicate`, `ErrAlreadyExists`, `ErrVersionMismatch` | 409 Conflict |
| `ErrNotFound` | 404 Not Found |
| `ErrUnauthorized` | 401 Unauthorized |
| `ErrForbidden` | 403 Forbidden |
| `ErrTooManyRequests` | 429 Too Many Requests |
| All validation codes | 422 Unprocessable Entity |
| `ErrSystemError` | 500 Internal Server Error |

---

## Stack Traces

Enable stack traces for debugging (typically in development):

```go
// Enable at application startup (dev mode only)
if config.Environment == "development" {
    errortypes.EnableStackTraces()
}

// Stack traces are automatically captured in ErrorContext
ctx := errortypes.NewErrorContext().WithRequestID("req-123")

// ctx.Stack contains:
// []StackFrame{
//   {Function: "main.handleRequest", File: "/app/handler.go", Line: 45},
//   {Function: "main.processUser", File: "/app/user.go", Line: 123},
//   ...
// }
```

**Important:** Stack traces add overhead. Only enable in development/debugging.

```go
// Check if enabled
if errortypes.StackTracesEnabled() {
    // Stack traces are being captured
}

// Disable when done debugging
errortypes.DisableStackTraces()
```

---

## Real-World Examples

### Domain Entity Validation

```go
package user

type User struct {
    ID       string
    Email    string
    Name     string
    Age      int
    Address  Address
    Roles    []string
}

type Address struct {
    Street  string
    City    string
    Country string
}

func (u *User) Validate(multiErr *errortypes.MultiError) {
    // Basic field validation with ozzo
    err := validation.ValidateStruct(u,
        validation.Field(&u.Email, validation.Required, is.Email),
        validation.Field(&u.Name, validation.Required, validation.Length(1, 100)),
        validation.Field(&u.Age, validation.Min(0), validation.Max(150)),
    )
    multiErr.AddOzzoError(err)

    // Nested address validation
    addressErrs := multiErr.WithPrefix("address")
    u.Address.Validate(addressErrs)

    // Custom business rule
    if u.Age < 18 && contains(u.Roles, "admin") {
        multiErr.Add("roles", errortypes.ErrBusinessLogic, "Users under 18 cannot have admin role")
    }
}

func (a *Address) Validate(multiErr *errortypes.MultiError) {
    err := validation.ValidateStruct(a,
        validation.Field(&a.Street, validation.Required),
        validation.Field(&a.City, validation.Required),
        validation.Field(&a.Country, validation.Required, validation.Length(2, 2)),
    )
    multiErr.AddOzzoError(err)
}
```

### Service Layer Error Handling

```go
package service

type UserService struct {
    repo   UserRepository
    logger *slog.Logger
}

func (s *UserService) CreateUser(ctx context.Context, req CreateUserRequest) (*User, error) {
    // Create error context from request
    errCtx := errortypes.NewErrorContext().
        WithRequestID(middleware.GetRequestID(ctx)).
        WithUserID(middleware.GetUserID(ctx))

    // Validate request
    multiErr := errortypes.NewMultiError().WithContext(errCtx)
    req.Validate(multiErr)

    if multiErr.HasErrors() {
        s.logger.Warn("validation failed", multiErr.LogFields())
        return nil, multiErr
    }

    // Check for duplicate email
    existing, err := s.repo.FindByEmail(ctx, req.Email)
    if err != nil && !errortypes.IsNotFoundError(err) {
        return nil, errortypes.NewDatabaseError("Failed to check email").
            WithContext(errCtx).
            WithInternal(err)
    }
    if existing != nil {
        multiErr.Add("email", errortypes.ErrAlreadyExists, "Email already registered")
        return nil, multiErr
    }

    // Create user
    user, err := s.repo.Create(ctx, req.ToUser())
    if err != nil {
        s.logger.Error("failed to create user",
            errortypes.NewDatabaseError("Create failed").
                WithContext(errCtx).
                WithInternal(err).
                LogFields())
        return nil, errortypes.NewDatabaseError("Failed to create user").
            WithContext(errCtx).
            WithInternal(err)
    }

    return user, nil
}
```

### HTTP Handler Integration

```go
package handler

func (h *UserHandler) Create(c *gin.Context) {
    var req CreateUserRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": "Invalid JSON"})
        return
    }

    user, err := h.service.CreateUser(c.Request.Context(), req)
    if err != nil {
        status := errortypes.HTTPStatus(err)

        // Log server errors
        if status >= 500 {
            h.logger.Error("internal error", getLogFields(err))
        }

        c.JSON(status, err)
        return
    }

    c.JSON(201, user)
}

func getLogFields(err error) map[string]any {
    switch e := err.(type) {
    case *errortypes.MultiError:
        return e.LogFields()
    case *errortypes.BusinessError:
        return e.LogFields()
    case *errortypes.DatabaseError:
        return e.BaseError.LogFields()
    default:
        return map[string]any{"error": err.Error()}
    }
}
```

### Batch Processing with Error Limits

```go
func (s *ImportService) ImportUsers(ctx context.Context, records []UserRecord) (*ImportResult, error) {
    // Limit errors to prevent huge responses
    multiErr := errortypes.NewMultiErrorWithLimit(50)

    var imported, skipped int

    for i, record := range records {
        if multiErr.IsFull() {
            skipped = len(records) - i
            break
        }

        recordErr := multiErr.WithIndex("records", i)

        if err := record.Validate(recordErr); err != nil {
            continue
        }

        if err := s.repo.Create(ctx, record.ToUser()); err != nil {
            recordErr.Add("", errortypes.ErrSystemError, "Failed to import")
            continue
        }

        imported++
    }

    result := &ImportResult{
        Imported: imported,
        Skipped:  skipped,
    }

    if multiErr.HasErrors() {
        result.Errors = multiErr
    }

    return result, nil
}
```

---

## Error Codes Reference

| Code | Description | Use Case |
|------|-------------|----------|
| `ErrRequired` | Field is required | Missing required fields |
| `ErrInvalid` | Generic invalid value | Catch-all validation failure |
| `ErrInvalidFormat` | Format validation failed | Email, phone, date formats |
| `ErrInvalidLength` | Length out of range | String length, array size |
| `ErrInvalidReference` | Referenced entity invalid | Foreign key doesn't exist |
| `ErrInvalidOperation` | Operation not allowed | Invalid state transition |
| `ErrDuplicate` | Duplicate value | Unique constraint violation |
| `ErrAlreadyExists` | Resource already exists | Create conflict |
| `ErrNotFound` | Resource not found | 404 scenarios |
| `ErrUnauthorized` | Not authenticated | Missing/invalid token |
| `ErrForbidden` | Not authorized | Insufficient permissions |
| `ErrTooManyRequests` | Rate limited | API throttling |
| `ErrBusinessLogic` | Business rule violation | Domain-specific rules |
| `ErrComplianceViolation` | Compliance rule broken | Regulatory requirements |
| `ErrVersionMismatch` | Optimistic lock failed | Concurrent modification |
| `ErrAlreadyCleared` | Already processed | Double-submit prevention |
| `ErrSystemError` | Internal system error | Database, external service failures |
