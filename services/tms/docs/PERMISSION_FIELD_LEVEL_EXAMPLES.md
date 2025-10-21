# Field-Level Permission Examples

## Scenario: User Can Update, But Not All Fields

This document shows how to handle the case where a user has permission to perform an action (e.g., "update"), but shouldn't be able to modify specific fields.

---

## Approach 1: Check Permission in Handler (Recommended)

Use the middleware for the main action, then check field permissions inside the handler.

### Handler Example

```go
package handlers

import (
    "net/http"

    authctx "github.com/emoss08/trenova/internal/api/context"
    "github.com/emoss08/trenova/internal/api/helpers"
    "github.com/emoss08/trenova/internal/api/middleware"
    "github.com/emoss08/trenova/internal/core/domain/shipment"
    "github.com/emoss08/trenova/internal/core/domain/permission"
    "github.com/emoss08/trenova/internal/core/ports"
    "github.com/emoss08/trenova/pkg/errortypes"
    "github.com/gin-gonic/gin"
)

type ShipmentHandler struct {
    service    ShipmentService
    eh         *helpers.ErrorHandler
    pm         *middleware.PermissionMiddleware
    permEngine ports.PermissionEngine
}

func (h *ShipmentHandler) RegisterRoutes(rg *gin.RouterGroup) {
    api := rg.Group("/shipments/")

    // Middleware checks if user can "update" shipments
    api.PUT(
        ":id/",
        h.pm.RequirePermission(permission.ResourceShipment, "update"),
        h.update,
    )
}

type UpdateShipmentRequest struct {
    Status          *string  `json:"status"`
    Notes           *string  `json:"notes"`
    BillingAmount   *float64 `json:"billingAmount"`
    CustomerPrice   *float64 `json:"customerPrice"`
    AssignedDriver  *string  `json:"assignedDriver"`
}

func (h *ShipmentHandler) update(c *gin.Context) {
    authCtx := authctx.GetAuthContext(c)

    // Parse request
    var req UpdateShipmentRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        h.eh.HandleError(c, errortypes.NewValidationError("request", errortypes.ErrInvalidFormat, err.Error()))
        return
    }

    // Get field access rules for shipment resource
    fieldAccess, err := h.permEngine.GetFieldAccess(
        c.Request.Context(),
        authCtx.UserID,
        authCtx.OrganizationID,
        "shipment",
    )
    if err != nil {
        h.eh.HandleError(c, err)
        return
    }

    // Validate field permissions
    if err := h.validateFieldPermissions(&req, fieldAccess); err != nil {
        h.eh.HandleError(c, err)
        return
    }

    // All field permissions validated - proceed with update
    updated, err := h.service.Update(c.Request.Context(), req)
    if err != nil {
        h.eh.HandleError(c, err)
        return
    }

    c.JSON(http.StatusOK, updated)
}

func (h *ShipmentHandler) validateFieldPermissions(
    req *UpdateShipmentRequest,
    fieldAccess *ports.FieldAccessRules,
) error {
    // Check each field that was provided in the request

    if req.Status != nil {
        if !h.canWriteField("status", fieldAccess) {
            return errortypes.NewAuthorizationError("You cannot modify the 'status' field")
        }
    }

    if req.Notes != nil {
        if !h.canWriteField("notes", fieldAccess) {
            return errortypes.NewAuthorizationError("You cannot modify the 'notes' field")
        }
    }

    if req.BillingAmount != nil {
        if !h.canWriteField("billingAmount", fieldAccess) {
            return errortypes.NewAuthorizationError("You cannot modify the 'billingAmount' field")
        }
    }

    if req.CustomerPrice != nil {
        if !h.canWriteField("customerPrice", fieldAccess) {
            return errortypes.NewAuthorizationError("You cannot modify the 'customerPrice' field")
        }
    }

    if req.AssignedDriver != nil {
        if !h.canWriteField("assignedDriver", fieldAccess) {
            return errortypes.NewAuthorizationError("You cannot modify the 'assignedDriver' field")
        }
    }

    return nil
}

func (h *ShipmentHandler) canWriteField(field string, fieldAccess *ports.FieldAccessRules) bool {
    // Check if field is in writable fields
    for _, writable := range fieldAccess.Writable {
        if writable == "*" || writable == field {
            return true
        }
    }
    return false
}
```

### Request/Response Flow

```text
1. Client sends PUT /api/shipments/123
   Body: {
     "status": "delivered",
     "customerPrice": 5000.00
   }

2. Permission Middleware checks: Can user "update" shipment?
   ✅ Yes → Continue

3. Handler gets field access rules:
   {
     "readable": ["*"],
     "writable": ["status", "notes"],
     "masked": {"customerPrice": "partial"}
   }

4. Handler validates each field in request:
   - status: ✅ In writable list
   - customerPrice: ❌ NOT in writable list

5. Return 403 Forbidden:
   {
     "error": "You cannot modify the 'customerPrice' field"
   }
```

---

## Approach 2: Use CheckPermissionInHandler for Complex Logic

For more complex scenarios, use the middleware's `CheckPermissionInHandler` method.

### Handler Example

```go
func (h *ShipmentHandler) bulkUpdate(c *gin.Context) {
    authCtx := authctx.GetAuthContext(c)

    var req BulkUpdateRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        h.eh.HandleError(c, err)
        return
    }

    // Check different permissions based on operation type
    for _, operation := range req.Operations {
        var requiredAction string

        switch operation.Type {
        case "status_change":
            requiredAction = "update"
        case "assign_driver":
            requiredAction = "assign"
        case "approve":
            requiredAction = "approve"
        case "delete":
            requiredAction = "delete"
        }

        // Check permission for each operation type
        if !h.pm.CheckPermissionInHandler(c, permission.ResourceShipment, requiredAction) {
            return // Response already sent by CheckPermissionInHandler
        }
    }

    // All operations validated - proceed
    results, err := h.service.BulkUpdate(c.Request.Context(), req)
    if err != nil {
        h.eh.HandleError(c, err)
        return
    }

    c.JSON(http.StatusOK, results)
}
```

---

## Approach 3: Batch Permission Check (Most Efficient)

For checking multiple permissions at once, use the `CheckBatch` method.

### Handler Example

```go
func (h *ShipmentHandler) complexOperation(c *gin.Context) {
    authCtx := authctx.GetAuthContext(c)

    // Check multiple permissions at once
    batchReq := &ports.BatchPermissionCheckRequest{
        UserID:         authCtx.UserID,
        OrganizationID: authCtx.OrganizationID,
        Checks: []*ports.PermissionCheckRequest{
            {
                UserID:         authCtx.UserID,
                OrganizationID: authCtx.OrganizationID,
                ResourceType:   "shipment",
                Action:         "update",
            },
            {
                UserID:         authCtx.UserID,
                OrganizationID: authCtx.OrganizationID,
                ResourceType:   "billing_queue",
                Action:         "create",
            },
            {
                UserID:         authCtx.UserID,
                OrganizationID: authCtx.OrganizationID,
                ResourceType:   "customer",
                Action:         "read",
            },
        },
    }

    results, err := h.permEngine.CheckBatch(c.Request.Context(), batchReq)
    if err != nil {
        h.eh.HandleError(c, err)
        return
    }

    // Check results
    canUpdateShipment := results.Results[0].Allowed
    canCreateBilling := results.Results[1].Allowed
    canReadCustomer := results.Results[2].Allowed

    if !canUpdateShipment {
        h.eh.HandleError(c, errortypes.NewAuthorizationError("Cannot update shipment"))
        return
    }

    if !canCreateBilling {
        h.eh.HandleError(c, errortypes.NewAuthorizationError("Cannot create billing"))
        return
    }

    if !canReadCustomer {
        h.eh.HandleError(c, errortypes.NewAuthorizationError("Cannot read customer"))
        return
    }

    // All permissions granted - proceed
    // ...
}
```

**Performance:** Batch check is ~3x faster than individual checks (one cache lookup instead of three).

---

## Approach 4: Return Field Access in Response (Client-Side Validation)

For better UX, include field access rules in the response so the client can disable/hide fields.

### Handler Example

```go
type ShipmentDetailResponse struct {
    Shipment    *shipment.Shipment `json:"shipment"`
    FieldAccess *FieldAccessInfo   `json:"fieldAccess"`
}

type FieldAccessInfo struct {
    CanRead  []string          `json:"canRead"`
    CanWrite []string          `json:"canWrite"`
    Masked   map[string]string `json:"masked"`
}

func (h *ShipmentHandler) get(c *gin.Context) {
    authCtx := authctx.GetAuthContext(c)
    shipmentID := c.Param("id")

    // Get shipment
    s, err := h.service.Get(c.Request.Context(), shipmentID)
    if err != nil {
        h.eh.HandleError(c, err)
        return
    }

    // Get field access rules
    fieldAccess, err := h.permEngine.GetFieldAccess(
        c.Request.Context(),
        authCtx.UserID,
        authCtx.OrganizationID,
        "shipment",
    )
    if err != nil {
        h.eh.HandleError(c, err)
        return
    }

    // Build response with field access info
    response := ShipmentDetailResponse{
        Shipment: s,
        FieldAccess: &FieldAccessInfo{
            CanRead:  fieldAccess.Readable,
            CanWrite: fieldAccess.Writable,
            Masked:   convertMasked(fieldAccess.Masked),
        },
    }

    c.JSON(http.StatusOK, response)
}
```

### Client-Side Usage

```typescript
import { useQuery } from '@tanstack/react-query';

function ShipmentForm({ shipmentId }: { shipmentId: string }) {
  const { data } = useQuery({
    queryKey: ['shipment', shipmentId],
    queryFn: () => api.get(`/shipments/${shipmentId}`),
  });

  const shipment = data?.shipment;
  const fieldAccess = data?.fieldAccess;

  return (
    <form>
      {/* Status field - check if writable */}
      <input
        name="status"
        value={shipment?.status}
        disabled={!fieldAccess?.canWrite.includes('status')}
      />

      {/* Customer price - check if readable and if masked */}
      {fieldAccess?.canRead.includes('customerPrice') && (
        <input
          name="customerPrice"
          value={shipment?.customerPrice}
          type={fieldAccess?.masked?.customerPrice ? 'password' : 'text'}
          disabled={!fieldAccess?.canWrite.includes('customerPrice')}
        />
      )}

      {/* Billing amount - hide if not readable */}
      {fieldAccess?.canRead.includes('billingAmount') && (
        <input
          name="billingAmount"
          value={shipment?.billingAmount}
          disabled={!fieldAccess?.canWrite.includes('billingAmount')}
        />
      )}
    </form>
  );
}
```

---

## Approach 5: Helper Function for Reusable Field Validation

Create a reusable helper for field validation across all handlers.

### Helper Implementation

```go
package helpers

import (
    "reflect"
    "github.com/emoss08/trenova/internal/core/ports"
    "github.com/emoss08/trenova/pkg/errortypes"
)

type FieldValidator struct {
    fieldAccess *ports.FieldAccessRules
}

func NewFieldValidator(fieldAccess *ports.FieldAccessRules) *FieldValidator {
    return &FieldValidator{fieldAccess: fieldAccess}
}

// ValidateWritableFields checks if user can write to any non-nil fields in the struct
func (v *FieldValidator) ValidateWritableFields(data interface{}) error {
    val := reflect.ValueOf(data)
    if val.Kind() == reflect.Ptr {
        val = val.Elem()
    }

    typ := val.Type()

    for i := 0; i < val.NumField(); i++ {
        field := val.Field(i)
        fieldType := typ.Field(i)

        // Skip if field is nil (not provided in request)
        if field.Kind() == reflect.Ptr && field.IsNil() {
            continue
        }

        // Get JSON field name
        jsonTag := fieldType.Tag.Get("json")
        if jsonTag == "" || jsonTag == "-" {
            continue
        }

        // Check if user can write this field
        if !v.canWriteField(jsonTag) {
            return errortypes.NewAuthorizationError(
                "You cannot modify the '" + jsonTag + "' field",
            )
        }
    }

    return nil
}

func (v *FieldValidator) canWriteField(field string) bool {
    for _, writable := range v.fieldAccess.Writable {
        if writable == "*" || writable == field {
            return true
        }
    }
    return false
}

// ValidateReadableFields removes fields user cannot read from response
func (v *FieldValidator) ValidateReadableFields(data interface{}) interface{} {
    val := reflect.ValueOf(data)
    if val.Kind() == reflect.Ptr {
        val = val.Elem()
    }

    typ := val.Type()
    result := reflect.New(typ).Elem()

    for i := 0; i < val.NumField(); i++ {
        field := val.Field(i)
        fieldType := typ.Field(i)

        // Get JSON field name
        jsonTag := fieldType.Tag.Get("json")
        if jsonTag == "" || jsonTag == "-" {
            result.Field(i).Set(field)
            continue
        }

        // Check if user can read this field
        if v.canReadField(jsonTag) {
            result.Field(i).Set(field)
        }
    }

    return result.Interface()
}

func (v *FieldValidator) canReadField(field string) bool {
    for _, readable := range v.fieldAccess.Readable {
        if readable == "*" || readable == field {
            return true
        }
    }
    return false
}
```

### Usage in Handler

```go
func (h *ShipmentHandler) update(c *gin.Context) {
    authCtx := authctx.GetAuthContext(c)

    var req UpdateShipmentRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        h.eh.HandleError(c, err)
        return
    }

    // Get field access
    fieldAccess, err := h.permEngine.GetFieldAccess(
        c.Request.Context(),
        authCtx.UserID,
        authCtx.OrganizationID,
        "shipment",
    )
    if err != nil {
        h.eh.HandleError(c, err)
        return
    }

    // Validate all fields at once using helper
    validator := helpers.NewFieldValidator(fieldAccess)
    if err := validator.ValidateWritableFields(&req); err != nil {
        h.eh.HandleError(c, err)
        return
    }

    // All fields validated - proceed
    updated, err := h.service.Update(c.Request.Context(), req)
    if err != nil {
        h.eh.HandleError(c, err)
        return
    }

    c.JSON(http.StatusOK, updated)
}
```

---

## Summary: When to Use Each Approach

| Approach | Use Case | Performance | Complexity |
|----------|----------|-------------|------------|
| **Approach 1: Check in Handler** | Single resource, few fields | Fast | Low |
| **Approach 2: CheckPermissionInHandler** | Dynamic actions, complex logic | Fast | Medium |
| **Approach 3: Batch Check** | Multiple resources/actions at once | Fastest | Medium |
| **Approach 4: Include in Response** | Client-side validation, better UX | Fast | Low |
| **Approach 5: Helper Function** | Reusable across many handlers | Fast | Low |

**Recommended Pattern:**

- Use **middleware** for the main action (update, delete, etc.)
- Use **Approach 1 or 5** for field-level validation inside handlers
- Use **Approach 4** to improve client-side UX

---

## Complete Example: Shipment Handler with All Checks

```go
package handlers

import (
    "net/http"

    authctx "github.com/emoss08/trenova/internal/api/context"
    "github.com/emoss08/trenova/internal/api/helpers"
    "github.com/emoss08/trenova/internal/api/middleware"
    "github.com/emoss08/trenova/internal/core/domain/permission"
    "github.com/emoss08/trenova/internal/core/ports"
    "github.com/emoss08/trenova/pkg/errortypes"
    "github.com/gin-gonic/gin"
)

type ShipmentHandler struct {
    service    ShipmentService
    eh         *helpers.ErrorHandler
    pm         *middleware.PermissionMiddleware
    permEngine ports.PermissionEngine
}

func (h *ShipmentHandler) RegisterRoutes(rg *gin.RouterGroup) {
    api := rg.Group("/shipments/")

    // Standard CRUD with permission middleware
    api.GET("", h.pm.RequirePermission(permission.ResourceShipment, "read"), h.list)
    api.GET(":id/", h.pm.RequirePermission(permission.ResourceShipment, "read"), h.get)
    api.POST("", h.pm.RequirePermission(permission.ResourceShipment, "create"), h.create)
    api.PUT(":id/", h.pm.RequirePermission(permission.ResourceShipment, "update"), h.update)
    api.DELETE(":id/", h.pm.RequirePermission(permission.ResourceShipment, "delete"), h.delete)

    // Complex operations requiring multiple permissions
    api.POST(":id/approve", h.pm.RequireAllPermissions(permission.ResourceShipment, []string{"read", "approve"}), h.approve)
    api.POST("/bulk", h.bulkOperation)
}

// Simple GET with field filtering
func (h *ShipmentHandler) get(c *gin.Context) {
    authCtx := authctx.GetAuthContext(c)
    shipmentID := c.Param("id")

    shipment, err := h.service.Get(c.Request.Context(), shipmentID)
    if err != nil {
        h.eh.HandleError(c, err)
        return
    }

    // Get field access for response
    fieldAccess, err := h.permEngine.GetFieldAccess(
        c.Request.Context(),
        authCtx.UserID,
        authCtx.OrganizationID,
        "shipment",
    )
    if err != nil {
        h.eh.HandleError(c, err)
        return
    }

    // Filter response based on readable fields
    validator := helpers.NewFieldValidator(fieldAccess)
    filtered := validator.ValidateReadableFields(shipment)

    c.JSON(http.StatusOK, gin.H{
        "shipment":    filtered,
        "fieldAccess": fieldAccess,
    })
}

// UPDATE with field-level validation
func (h *ShipmentHandler) update(c *gin.Context) {
    authCtx := authctx.GetAuthContext(c)

    var req UpdateShipmentRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        h.eh.HandleError(c, err)
        return
    }

    // Get field access rules
    fieldAccess, err := h.permEngine.GetFieldAccess(
        c.Request.Context(),
        authCtx.UserID,
        authCtx.OrganizationID,
        "shipment",
    )
    if err != nil {
        h.eh.HandleError(c, err)
        return
    }

    // Validate writable fields
    validator := helpers.NewFieldValidator(fieldAccess)
    if err := validator.ValidateWritableFields(&req); err != nil {
        h.eh.HandleError(c, err)
        return
    }

    // All validations passed - update
    updated, err := h.service.Update(c.Request.Context(), req)
    if err != nil {
        h.eh.HandleError(c, err)
        return
    }

    c.JSON(http.StatusOK, updated)
}

// BULK operation with dynamic permission checks
func (h *ShipmentHandler) bulkOperation(c *gin.Context) {
    authCtx := authctx.GetAuthContext(c)

    var req BulkOperationRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        h.eh.HandleError(c, err)
        return
    }

    // Build batch permission check
    checks := make([]*ports.PermissionCheckRequest, 0)
    for _, op := range req.Operations {
        checks = append(checks, &ports.PermissionCheckRequest{
            UserID:         authCtx.UserID,
            OrganizationID: authCtx.OrganizationID,
            ResourceType:   "shipment",
            Action:         op.Action,
        })
    }

    // Check all permissions at once
    batchReq := &ports.BatchPermissionCheckRequest{
        UserID:         authCtx.UserID,
        OrganizationID: authCtx.OrganizationID,
        Checks:         checks,
    }

    results, err := h.permEngine.CheckBatch(c.Request.Context(), batchReq)
    if err != nil {
        h.eh.HandleError(c, err)
        return
    }

    // Validate all operations are allowed
    for i, result := range results.Results {
        if !result.Allowed {
            h.eh.HandleError(
                c,
                errortypes.NewAuthorizationError(
                    "Operation '"+req.Operations[i].Action+"' not permitted",
                ),
            )
            return
        }
    }

    // All permissions granted - execute
    operationResults, err := h.service.BulkExecute(c.Request.Context(), req)
    if err != nil {
        h.eh.HandleError(c, err)
        return
    }

    c.JSON(http.StatusOK, operationResults)
}
```

---

## Policy Configuration Example

To restrict field access, configure the policy:

```json
{
  "name": "shipment_limited_update",
  "effect": "allow",
  "priority": 100,
  "resources": {
    "resourceType": ["shipment"],
    "actions": ["read", "update"]
  },
  "fieldRules": {
    "readableFields": ["*"],
    "writableFields": ["status", "notes", "assignedDriver"],
    "maskedFields": {
      "customerPrice": "partial",
      "billingAmount": "full"
    }
  },
  "scope": {
    "businessUnitId": "bu_123",
    "organizationIds": ["org_456"]
  }
}
```

**Result:**

- User can **update** shipments (middleware allows)
- User can only **write** to: status, notes, assignedDriver
- User can **read** customerPrice but it's partially masked
- User **cannot read** billingAmount at all
