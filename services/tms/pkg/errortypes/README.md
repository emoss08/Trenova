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
    {"path": ["email"], "code": "REQUIRED", "message": "Email is required"},
    {"path": ["password"], "code": "INVALID_LENGTH", "message": "Password must be at least 8 characters"}
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

### Plan Errors

Trenova Cloud plan limits. A quota error carries the meter, the plan's limit and the usage at
the time of the refusal; a plan restriction carries the capability (empty when the whole
organization is read-only or expired), a reason (`plan_restricted`,
`subscription_read_only`, `subscription_expired`, `signups_paused`) and the plan key:

```go
err := errortypes.NewQuotaExceededError("shipments.total", 12, 12, "free_demo") // 402, QUOTA_EXCEEDED
err := errortypes.NewPlanRestrictionError("api_keys", errortypes.PlanRestrictionReasonPlan, "free_demo") // 403, PLAN_RESTRICTED
```

Both expose `Params()`, which the REST problem document and the GraphQL error
extensions carry as `params`. Any error implementing `Parameterized` (`error` plus
`Params() map[string]string`) gets the same treatment.

### Session Context Errors

`SessionContextMiddleware` refuses a request whose `X-Trenova-User-Id`,
`X-Trenova-Organization-Id` or `X-Trenova-Business-Unit-Id` names another session than the
cookie's, because another tab signed in as someone else or switched organization
(`docs/engineering/session-context.md`). Its params name the session the cookie acts for now:

```go
err := errortypes.NewSessionContextStaleError(authCtx.UserID, authCtx.OrganizationID, authCtx.BusinessUnitID) // 409, SESSION_CONTEXT_STALE
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
    {"path": ["user", "name"], "code": "REQUIRED", "message": "Name is required"},
    {"path": ["user", "address", "street"], "code": "REQUIRED", "message": "Street is required"},
    {"path": ["user", "address", "city"], "code": "INVALID_LENGTH", "message": "City name too long"},
    {"path": ["items", 0, "quantity"], "code": "INVALID", "message": "Quantity must be positive"}
  ]
}
```

A path is a list of segments (`pkg/fieldpath`): object keys are strings and list indexes are
numbers, so a key that contains a dot (a rule code such as `authority.revoked`) stays one
segment. `Add` and `WithPrefix` take a path expression (`moves[0].stops[1].status` or
`moves.0.stops.1.status`) and parse it; `AddAt` and `WithPath` take a `fieldpath.Path` when a
key may contain dots or brackets. The client joins the segments with dots, which is the name
react-hook-form registers. `Error.Field()` renders the bracket form for logs and summaries.

### Ozzo Validation Integration

```go
func (u *User) Validate(multiErr *errortypes.MultiError) {
    multiErr.AddOzzoError(validation.ValidateStruct(u,
        validation.Field(&u.Email, validation.Required, is.EmailFormat),
        validation.Field(&u.Name, validation.Required, validation.Length(1, 100)),
    ))
}
```

`AddOzzoError` (through `pkg/ozzoerrors`):

- flattens nested errors into paths: a nested struct becomes `address.city`, `validation.Each`
  and slices of validatable values become `stops[1].name`;
- adds them in sorted order, so the same input always reports the same order;
- maps each error by its ozzo code (`validation_required` is `REQUIRED`, the length rules are
  `INVALID_LENGTH`, `match` and the `is` rules are `INVALID_FORMAT`, everything else `INVALID`);
- replaces ozzo's own English defaults with catalog messages and positional arguments
  ("Must be between {0} and {1} characters"), so they translate; a custom `.Error("…")`
  message is kept, with any `{{.param}}` placeholder turned into `{0}`;
- records an ozzo internal error (a field pointer that is not a field of the struct) as a
  failed run, never as a pass.

### Failed Runs

A validation that could not finish (a rule's query failed, a field pointer was wrong) is not a
refusal. Record it with `AddInternal(err)`. `HasErrors()` is then true, so nothing passes, and
`Err()` returns the failure wrapped as a plain error, not the `MultiError`, so no caller can take
it for a refusal: the presenters answer 500 with a generic message, log the cause and send no
field errors, and a job retries it. A rule in `tenantedvalidation` returns the error instead and
the run stops there.

To ask whether an error is a refusal, use `RefusalOf(err)` (or `IsMultiError`), never
`errors.As(err, &multiErr)`: a `MultiError` that escaped with an internal failure is not one.

```go
if err != nil {
    multiErr.AddInternal(fmt.Errorf("load accounting control: %w", err))
    return multiErr
}
```

Never write a database or system failure as a field error: the client would show the
driver's text on a form and never retry.

### Results

`Err()` is the one way out of a report: nil when nothing was refused (advisories alone never
refuse), the failure when the run failed, the `MultiError` otherwise. `MergeMultiErrors` keeps errors, advisories and internal
failures. A report collects at most `MaxCollectedErrors` (500) errors; the rest are counted
in `Dropped()` and in `LogFields()`.

### Advisories

`Advise(field, code, message, severity, ruleKey)` records a warning that does not refuse the
write. The REST problem document and GraphQL extensions carry them as `advisories`, with paths,
beside `errors`.

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

The presenters in `internal/api/helpers` (REST problem documents) and
`internal/api/graphql/gqlext` (GraphQL extensions) classify an error once
(`helpers.NewDefaultClassifier`) and take its code from `helpers.ErrorCodeOf`. A validation
refusal answers 400 (`validation-error`). A save that lost a race is an `EditConflictError`, from
`NewEditConflictError` (with what changed) or `NewVersionMismatchError` (a message only); it answers
409 (`resource-conflict`) with `code: VERSION_MISMATCH`, and `IsVersionMismatchError` is the only
check for it. A version mismatch is never a field error. A `MultiError` holding an internal
failure is classified by that failure.

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

Handlers return the error to `helpers.ErrorHandler.HandleError`, which classifies, sanitizes,
localizes and logs it; resolvers return it to gqlgen, whose presenter does the same.

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
| `ErrQuotaExceeded` | Plan limit reached | Cloud free tier meter at its limit |
| `ErrPlanRestricted` | Not on this plan | Cloud capability restriction, read-only or expired trial |
| `ErrSessionContextStale` | Session no longer the tab's | A tab another tab's sign-in or organization switch left behind |
| `ErrBusinessLogic` | Business rule violation | Domain-specific rules |
| `ErrComplianceViolation` | Compliance rule broken | Regulatory requirements |
| `ErrVersionMismatch` | Optimistic lock failed | Concurrent modification |
| `ErrAlreadyCleared` | Already processed | Double-submit prevention |
| `ErrSystemError` | Internal system error | Database, external service failures |
