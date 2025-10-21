# Permission Middleware Usage Guide

## Overview

The permission middleware enforces authorization at the HTTP handler level, following the principle of fail-fast. This keeps business logic clean and ensures consistent permission checking across all endpoints.

## Architecture

```text
Request → Authentication Middleware → Permission Middleware → Handler → Service → Repository
           (who are you?)              (what can you do?)      (do it)
```

**Benefits:**

- ✅ Fail-fast: Permission denied before expensive operations
- ✅ Consistent: Same permission check logic everywhere
- ✅ Maintainable: Easy to see what permissions each route requires
- ✅ Testable: Mock the permission engine for unit tests
- ✅ Auditable: All permission checks go through one place

## Setup

### 1. Wire Up the Middleware

In your router setup (`internal/api/router/router.go`):

```go
package router

import (
    "github.com/emoss08/trenova/internal/api/middleware"
    "github.com/emoss08/trenova/internal/core/ports"
    "github.com/gin-gonic/gin"
    "go.uber.org/fx"
)

type RouterParams struct {
    fx.In
    PermissionEngine ports.PermissionEngine
    // ... other dependencies
}

func NewRouter(p RouterParams) *gin.Engine {
    router := gin.New()

    // Create permission middleware
    permMiddleware := middleware.NewPermissionMiddleware(p.PermissionEngine)

    // Apply global middlewares
    router.Use(gin.Recovery())
    router.Use(middleware.CORS())
    router.Use(middleware.RequestID())
    router.Use(middleware.Logger())

    // Protected routes
    api := router.Group("/api")
    api.Use(middleware.Authentication()) // Authentication first!

    // Example: Shipment routes
    shipments := api.Group("/shipments")
    {
        shipments.GET("",
            permMiddleware.RequirePermission("shipment", "read"),
            handler.ListShipments,
        )

        shipments.POST("",
            permMiddleware.RequirePermission("shipment", "create"),
            handler.CreateShipment,
        )

        shipments.GET("/:id",
            permMiddleware.RequirePermission("shipment", "read"),
            handler.GetShipment,
        )

        shipments.PUT("/:id",
            permMiddleware.RequirePermission("shipment", "update"),
            handler.UpdateShipment,
        )

        shipments.DELETE("/:id",
            permMiddleware.RequirePermission("shipment", "delete"),
            handler.DeleteShipment,
        )
    }

    return router
}
```

## Usage Patterns

### 1. Single Permission (Most Common)

```go
// Requires exactly one permission
router.POST("/shipments",
    permMiddleware.RequirePermission("shipment", "create"),
    handler.CreateShipment,
)
```

### 2. Any Permission (OR Logic)

```go
// User needs at least one of these permissions
router.GET("/shipments/:id",
    permMiddleware.RequireAnyPermission("shipment", []string{"read", "update"}),
    handler.GetShipment,
)
```

**Use case**: View endpoint that allows both readers and editors

### 3. All Permissions (AND Logic)

```go
// User needs all of these permissions
router.POST("/shipments/:id/approve",
    permMiddleware.RequireAllPermissions("shipment", []string{"read", "approve"}),
    handler.ApproveShipment,
)
```

**Use case**: Complex operations requiring multiple permissions

### 4. Optional Permission

```go
// Check permission but don't block request
router.GET("/shipments",
    permMiddleware.OptionalPermission("shipment", "export"),
    handler.ListShipments,
)

// In handler:
func (h *ShipmentHandler) ListShipments(c *gin.Context) {
    canExport, _ := c.Get("has_shipment_export")

    response := ListResponse{
        Data: shipments,
        CanExport: canExport.(bool), // Include in response
    }

    c.JSON(http.StatusOK, response)
}
```

**Use case**: Show/hide UI elements based on permissions

### 5. Dynamic Permission Check in Handler

```go
// For complex scenarios where middleware isn't enough
func (h *ShipmentHandler) BulkUpdate(c *gin.Context) {
    var req BulkUpdateRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        httputils.RespondError(c, http.StatusBadRequest, err.Error())
        return
    }

    // Check different permissions based on the operation
    for _, operation := range req.Operations {
        action := getActionForOperation(operation.Type)
        if !h.permMiddleware.CheckPermissionInHandler(c, "shipment", action) {
            return // Response already sent
        }
    }

    // Proceed with bulk update
    // ...
}
```

## Common Patterns

### REST Resource Routes

```go
func setupResourceRoutes(
    group *gin.RouterGroup,
    resource string,
    handler ResourceHandler,
    permMiddleware *middleware.PermissionMiddleware,
) {
    group.GET("",
        permMiddleware.RequirePermission(resource, "read"),
        handler.List,
    )

    group.POST("",
        permMiddleware.RequirePermission(resource, "create"),
        handler.Create,
    )

    group.GET("/:id",
        permMiddleware.RequirePermission(resource, "read"),
        handler.Get,
    )

    group.PUT("/:id",
        permMiddleware.RequirePermission(resource, "update"),
        handler.Update,
    )

    group.DELETE("/:id",
        permMiddleware.RequirePermission(resource, "delete"),
        handler.Delete,
    )

    // Custom actions
    group.POST("/:id/approve",
        permMiddleware.RequireAllPermissions(resource, []string{"read", "approve"}),
        handler.Approve,
    )

    group.POST("/export",
        permMiddleware.RequirePermission(resource, "export"),
        handler.Export,
    )
}

// Usage:
setupResourceRoutes(api.Group("/shipments"), "shipment", shipmentHandler, permMiddleware)
setupResourceRoutes(api.Group("/customers"), "customer", customerHandler, permMiddleware)
```

### Admin Routes

```go
// Admin section requires elevated permissions
admin := api.Group("/admin")
{
    // Users management
    users := admin.Group("/users")
    users.Use(permMiddleware.RequirePermission("user", "manage"))
    {
        users.GET("", handler.ListUsers)
        users.POST("", handler.CreateUser)
        users.PUT("/:id", handler.UpdateUser)
        users.DELETE("/:id", handler.DeleteUser)
    }

    // Settings management
    settings := admin.Group("/settings")
    settings.Use(permMiddleware.RequirePermission("setting", "manage"))
    {
        settings.GET("", handler.GetSettings)
        settings.PUT("", handler.UpdateSettings)
    }
}
```

### Public vs Protected Routes

```go
// Public routes (no auth or permissions needed)
public := router.Group("/public")
{
    public.GET("/health", handler.HealthCheck)
    public.POST("/auth/login", handler.Login)
}

// Protected routes (auth + permissions)
api := router.Group("/api")
api.Use(middleware.Authentication())
{
    // All routes here require authentication
    // Add permission middleware as needed per route

    api.GET("/dashboard",
        permMiddleware.RequirePermission("dashboard", "read"),
        handler.GetDashboard,
    )
}
```

## Migration Strategy

### Step 1: Add Middleware to New Routes

```go
// New routes: Use middleware from the start
router.POST("/shipments/v2",
    permMiddleware.RequirePermission("shipment", "create"),
    handler.CreateShipmentV2,
)
```

### Step 2: Migrate Existing Routes

```go
// Before (permission check in service):
func (s *ShipmentService) CreateShipment(ctx context.Context, req CreateRequest) error {
    // Permission check in service (old way)
    if !s.checkPermission(ctx, "shipment", "create") {
        return ErrPermissionDenied
    }
    // ... business logic
}

// After (permission check in middleware):
// Handler:
router.POST("/shipments",
    permMiddleware.RequirePermission("shipment", "create"),
    handler.CreateShipment,
)

// Service (clean):
func (s *ShipmentService) CreateShipment(ctx context.Context, req CreateRequest) error {
    // No permission check needed - already done in middleware
    // ... pure business logic
}
```

### Step 3: Remove Service-Level Checks

Once all routes use middleware, remove permission checks from services:

```go
// Remove these from services:
// - s.permissionEngine.CheckPermission(...)
// - if !hasPermission { return ErrPermissionDenied }
// - permission validation logic

// Services should focus on business logic only
```

## Testing

### Unit Tests for Handlers

```go
func TestCreateShipment_WithPermission(t *testing.T) {
    // Mock permission engine
    mockEngine := &MockPermissionEngine{
        CheckPermissionFunc: func(ctx context.Context, userID, orgID pulid.ID, resource, action string) (bool, error) {
            return true, nil // Grant permission
        },
    }

    // Create middleware with mock
    permMiddleware := middleware.NewPermissionMiddleware(mockEngine)

    // Setup router with middleware
    router := gin.New()
    router.POST("/shipments",
        permMiddleware.RequirePermission("shipment", "create"),
        handler.CreateShipment,
    )

    // Make request
    req := httptest.NewRequest("POST", "/shipments", body)
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)

    // Assert
    assert.Equal(t, http.StatusCreated, w.Code)
}

func TestCreateShipment_WithoutPermission(t *testing.T) {
    mockEngine := &MockPermissionEngine{
        CheckPermissionFunc: func(ctx context.Context, userID, orgID pulid.ID, resource, action string) (bool, error) {
            return false, nil // Deny permission
        },
    }

    permMiddleware := middleware.NewPermissionMiddleware(mockEngine)

    router := gin.New()
    router.POST("/shipments",
        permMiddleware.RequirePermission("shipment", "create"),
        handler.CreateShipment,
    )

    req := httptest.NewRequest("POST", "/shipments", body)
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)

    // Assert
    assert.Equal(t, http.StatusForbidden, w.Code)
}
```

## Best Practices

### ✅ DO

1. **Apply permissions at the handler level**

   ```go
   router.POST("/shipments",
       permMiddleware.RequirePermission("shipment", "create"),
       handler.CreateShipment,
   )
   ```

2. **Use descriptive resource names**

   ```go
   "shipment", "customer", "user", "billing_queue"
   ```

3. **Use standard action names**

   ```go
   "create", "read", "update", "delete", "export", "import", "approve"
   ```

4. **Group routes with common permissions**

   ```go
   adminRoutes := api.Group("/admin")
   adminRoutes.Use(permMiddleware.RequirePermission("admin", "manage"))
   ```

5. **Test permission denial paths**

   ```go
   TestHandler_WithoutPermission(t *testing.T)
   ```

### ❌ DON'T

1. **Don't check permissions in services**

   ```go
   // Bad: Permission check in service layer
   func (s *Service) Create(ctx context.Context) error {
       if !s.hasPermission(...) { // ❌
           return ErrPermissionDenied
       }
   }
   ```

2. **Don't hardcode user/org IDs**

   ```go
   // Bad: Hardcoded IDs
   pm.engine.CheckPermission(ctx, "usr_123", "org_456", ...) // ❌

   // Good: Get from auth context
   userID, _ := authcontext.GetUserID(c)
   orgID, _ := authcontext.GetOrganizationID(c)
   ```

3. **Don't skip permission checks on "read" endpoints**

   ```go
   // Bad: No permission check
   router.GET("/sensitive-data", handler.Get) // ❌

   // Good: Always check permissions
   router.GET("/sensitive-data",
       permMiddleware.RequirePermission("sensitive_data", "read"),
       handler.Get,
   )
   ```

4. **Don't use overly broad permissions**

   ```go
   // Bad: Too broad
   permMiddleware.RequirePermission("*", "admin") // ❌

   // Good: Specific resource
   permMiddleware.RequirePermission("user", "manage")
   ```

## Troubleshooting

### Problem: "Permission check fails but user should have access"

**Debug steps:**

1. Check if user is authenticated
2. Verify user has the role with the policy
3. Check if policy has the correct resource and action
4. Refresh materialized view: `SELECT refresh_user_effective_policies();`
5. Check permission manifest: `GET /api/permissions/manifest`

### Problem: "Middleware blocks all requests"

**Cause**: Authentication middleware not set up correctly

**Fix**: Ensure auth middleware runs before permission middleware:

```go
api.Use(middleware.Authentication()) // Must be before permission checks
api.POST("/shipments",
    permMiddleware.RequirePermission("shipment", "create"),
    handler.CreateShipment,
)
```

### Problem: "Permission check is slow"

**Cause**: Permission engine not using cache

**Fix**: Verify cache is enabled and working:

- Check L1 cache hit rate
- Check Redis connection
- Verify materialized view is indexed

## Related Documentation

- [Permission System V2](PERMISSION_SYSTEM_V2.md)
- [Permission Materialized Views](PERMISSION_MATERIALIZED_VIEWS.md)
- [Permission Engine](../internal/core/services/permissionservice/engine.go)
