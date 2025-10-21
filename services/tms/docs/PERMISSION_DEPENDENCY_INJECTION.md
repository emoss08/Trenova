# Permission Registry - Dependency Injection Guide

## Overview

The Permission Registry System uses **uber-go/fx** for dependency injection. No globals, no init functions - everything is properly wired through the dependency injection container.

---

## Architecture

```text
1. Each entity provides a constructor function
   ↓
2. Bootstrap module registers entities with fx group
   ↓
3. Registry receives all entities via fx.In group
   ↓
4. Services/handlers receive registry via dependency injection
```

---

## Step 1: Define Permission for Your Entity

**File:** `/services/tms/internal/core/domain/{entity}/permission.go`

```go
package myentity

import "github.com/emoss08/trenova/pkg/permissionregistry"

// Ensure interface compliance at compile time
var (
    _ permissionregistry.PermissionAware = (*MyEntityPermission)(nil)
)

type MyEntityPermission struct{}

func (m *MyEntityPermission) GetResourceName() string {
    return "my_entity"
}

func (m *MyEntityPermission) GetResourceDescription() string {
    return "My entity description"
}

func (m *MyEntityPermission) GetSupportedOperations() []permissionregistry.OperationDefinition {
    return permissionregistry.StandardCRUDOperations()
}

func (m *MyEntityPermission) GetCompositeOperations() map[string]uint32 {
    return map[string]uint32{
        "manage":    permissionregistry.CompositeBasicCRUD,
        "read_only": permissionregistry.OpRead,
    }
}

func (m *MyEntityPermission) GetDefaultOperation() uint32 {
    return permissionregistry.OpRead
}

func (m *MyEntityPermission) GetOperationsRequiringApproval() []uint32 {
    return []uint32{permissionregistry.OpDelete}
}

// Constructor for dependency injection (NO init function!)
func NewMyEntityPermission() permissionregistry.PermissionAware {
    return &MyEntityPermission{}
}
```

---

## Step 2: Register Entity in Bootstrap Module

**File:** `/services/tms/internal/bootstrap/modules/permissionregistry/module.go`

```go
package permissionregistry

import (
    "github.com/emoss08/trenova/internal/core/domain/hazardousmaterial"
    "github.com/emoss08/trenova/internal/core/domain/customer"
    "github.com/emoss08/trenova/internal/core/domain/shipment"
    "github.com/emoss08/trenova/internal/core/domain/myentity"
    "github.com/emoss08/trenova/pkg/permissionregistry"
    "go.uber.org/fx"
)

var Module = fx.Module("permission-registry",
    // Provide the registry
    fx.Provide(permissionregistry.NewRegistry),

    // Register all permission entities
    fx.Provide(
        fx.Annotate(
            hazardousmaterial.NewHazardousMaterialPermission,
            fx.ResultTags(`group:"permission_entities"`),
        ),
        fx.Annotate(
            customer.NewCustomerPermission,
            fx.ResultTags(`group:"permission_entities"`),
        ),
        fx.Annotate(
            shipment.NewShipmentPermission,
            fx.ResultTags(`group:"permission_entities"`),
        ),
        fx.Annotate(
            myentity.NewMyEntityPermission,
            fx.ResultTags(`group:"permission_entities"`),
        ),
    ),
)
```

**Key Points:**

- Each entity's constructor is registered with the `permission_entities` group tag
- The registry will receive ALL entities through its `fx.In` parameter
- No globals, no manual registration needed

---

## Step 3: Use Registry in Services

### In Permission Engine

```go
package permissionservice

import (
    "github.com/emoss08/trenova/pkg/permissionregistry"
    "go.uber.org/fx"
)

type EngineParams struct {
    fx.In

    Registry      *permissionregistry.Registry
    PolicyRepo    ports.PolicyRepository
    CacheRepo     ports.PermissionCacheRepository
    // ... other dependencies
}

type Engine struct {
    registry   *permissionregistry.Registry
    policyRepo ports.PolicyRepository
    cacheRepo  ports.PermissionCacheRepository
}

func NewEngine(p EngineParams) ports.PermissionEngine {
    return &Engine{
        registry:   p.Registry,
        policyRepo: p.PolicyRepo,
        cacheRepo:  p.CacheRepo,
    }
}

func (e *Engine) GetResourceOperations(resourceName string) ([]string, error) {
    // Query the registry
    ops, exists := e.registry.GetOperationsForResource(resourceName)
    if !exists {
        return nil, fmt.Errorf("resource %s not found", resourceName)
    }

    names := make([]string, len(ops))
    for i, op := range ops {
        names[i] = op.Name
    }
    return names, nil
}

func (e *Engine) ExpandCompositeOperation(resource, operation string) (uint32, error) {
    bitfield, found := e.registry.ExpandCompositeOperation(resource, operation)
    if !found {
        return 0, fmt.Errorf("composite operation %s not found for %s", operation, resource)
    }
    return bitfield, nil
}
```

### In API Handler

```go
package handlers

import (
    "github.com/emoss08/trenova/pkg/permissionregistry"
    "go.uber.org/fx"
)

type PermissionHandlerParams struct {
    fx.In

    Registry     *permissionregistry.Registry
    Engine       ports.PermissionEngine
    ErrorHandler *helpers.ErrorHandler
}

type PermissionHandler struct {
    registry *permissionregistry.Registry
    engine   ports.PermissionEngine
    eh       *helpers.ErrorHandler
}

func NewPermissionHandler(p PermissionHandlerParams) *PermissionHandler {
    return &PermissionHandler{
        registry: p.Registry,
        engine:   p.Engine,
        eh:       p.ErrorHandler,
    }
}

func (h *PermissionHandler) RegisterRoutes(rg *gin.RouterGroup) {
    api := rg.Group("/permissions/")
    api.GET("registry/", h.getRegistry)
    api.GET("registry/:resource/", h.getResourceMetadata)
}

// GET /api/permissions/registry
func (h *PermissionHandler) getRegistry(c *gin.Context) {
    resources := h.registry.GetAllResources()

    response := make(map[string]any)
    for name, res := range resources {
        response[name] = map[string]any{
            "name":                name,
            "description":         res.GetResourceDescription(),
            "operations":          res.GetSupportedOperations(),
            "compositeOperations": res.GetCompositeOperations(),
            "defaultOperation":    res.GetDefaultOperation(),
        }
    }

    c.JSON(http.StatusOK, response)
}

// GET /api/permissions/registry/:resource
func (h *PermissionHandler) getResourceMetadata(c *gin.Context) {
    resourceName := c.Param("resource")

    res, exists := h.registry.GetResource(resourceName)
    if !exists {
        h.eh.HandleError(c, errortypes.NewNotFoundError("Resource not found"))
        return
    }

    response := map[string]any{
        "name":                     res.GetResourceName(),
        "description":              res.GetResourceDescription(),
        "operations":               res.GetSupportedOperations(),
        "compositeOperations":      res.GetCompositeOperations(),
        "defaultOperation":         res.GetDefaultOperation(),
        "operationsRequiringApproval": res.GetOperationsRequiringApproval(),
    }

    // Add field definitions if available
    if fields, ok := h.registry.GetFieldDefinitionsForResource(resourceName); ok {
        response["fields"] = fields
    }

    c.JSON(http.StatusOK, response)
}
```

---

## Step 4: Wire Everything in Main Application

**File:** `/services/tms/cmd/api/main.go` or bootstrap setup

```go
package main

import (
    "github.com/emoss08/trenova/internal/bootstrap/modules/permissionregistry"
    "github.com/emoss08/trenova/internal/bootstrap/modules/api"
    // ... other imports
    "go.uber.org/fx"
)

func main() {
    app := fx.New(
        // Infrastructure
        postgres.Module,
        redis.Module,

        // Permission system
        permissionregistry.Module,  // ← Registers registry + all entities

        // Core services
        permissionservice.Module,   // ← Uses registry

        // API
        api.Module,                 // ← Handlers use registry

        // Start server
        fx.Invoke(func(*gin.Engine) {}),
    )

    app.Run()
}
```

---

## Benefits of This Approach

### ✅ No Globals

**Before (with globals):**

```go
var globalRegistry = NewRegistry()

func init() {
    globalRegistry.Register(&MyEntityPermission{})
}
```

**After (with DI):**

```go
func NewMyEntityPermission() permissionregistry.PermissionAware {
    return &MyEntityPermission{}
}

// Registered via fx.Provide in module
```

### ✅ Testable

```go
func TestPermissionEngine(t *testing.T) {
    // Create test registry
    registry := permissionregistry.NewRegistry(permissionregistry.RegistryParams{
        Entities: []permissionregistry.PermissionAware{
            &MyEntityPermission{},
            &AnotherEntityPermission{},
        },
    })

    // Create engine with test registry
    engine := NewEngine(EngineParams{
        Registry: registry,
        // ... mock other dependencies
    })

    // Test
    ops, err := engine.GetResourceOperations("my_entity")
    assert.NoError(t, err)
    assert.NotEmpty(t, ops)
}
```

### ✅ Explicit Dependencies

Every component declares what it needs:

```go
type EngineParams struct {
    fx.In

    Registry   *permissionregistry.Registry  // ← Explicitly required
    PolicyRepo ports.PolicyRepository         // ← Explicitly required
    Logger     *zap.Logger                    // ← Explicitly required
}
```

### ✅ Lifecycle Management

fx handles initialization order automatically:

```
1. Create all permission entities (NewMyEntityPermission, etc.)
   ↓
2. Create registry with all entities
   ↓
3. Create engine with registry
   ↓
4. Create handlers with engine
   ↓
5. Start server
```

---

## Adding a New Entity

### Step 1: Create permission file

```go
// internal/core/domain/newentity/permission.go
package newentity

func NewNewEntityPermission() permissionregistry.PermissionAware {
    return &NewEntityPermission{}
}
```

### Step 2: Register in module

```go
// internal/bootstrap/modules/permissionregistry/module.go
fx.Provide(
    // ... existing entities
    fx.Annotate(
        newentity.NewNewEntityPermission,
        fx.ResultTags(`group:"permission_entities"`),
    ),
)
```

### Step 3: Done

The entity is now:

- ✅ Registered in the registry
- ✅ Available to all services via DI
- ✅ Queryable from API endpoints
- ✅ No globals touched

---

## Testing

### Unit Test (Single Entity)

```go
func TestMyEntityPermission(t *testing.T) {
    perm := NewMyEntityPermission()

    assert.Equal(t, "my_entity", perm.GetResourceName())

    ops := perm.GetSupportedOperations()
    assert.Len(t, ops, 4) // CRUD

    composites := perm.GetCompositeOperations()
    assert.Contains(t, composites, "manage")
}
```

### Integration Test (With Registry)

```go
func TestPermissionRegistry(t *testing.T) {
    // Create registry with test entities
    registry := permissionregistry.NewRegistry(permissionregistry.RegistryParams{
        Entities: []permissionregistry.PermissionAware{
            NewMyEntityPermission(),
            hazardousmaterial.NewHazardousMaterialPermission(),
        },
    })

    // Test resource lookup
    res, exists := registry.GetResource("my_entity")
    assert.True(t, exists)
    assert.Equal(t, "my_entity", res.GetResourceName())

    // Test operation expansion
    bitfield, found := registry.ExpandCompositeOperation("my_entity", "manage")
    assert.True(t, found)
    assert.NotZero(t, bitfield)
}
```

### Full Integration Test (With fx)

```go
func TestPermissionSystemIntegration(t *testing.T) {
    var registry *permissionregistry.Registry

    app := fxtest.New(t,
        permissionregistry.Module,
        fx.Populate(&registry),
    )

    app.RequireStart()
    defer app.RequireStop()

    // Test registry
    resources := registry.GetAllResources()
    assert.NotEmpty(t, resources)
}
```

---

## Summary

**No globals, no init functions, just clean dependency injection:**

1. ✅ Each entity exports a constructor: `NewMyEntityPermission()`
2. ✅ Bootstrap module registers with fx groups
3. ✅ Registry receives entities via `fx.In` group injection
4. ✅ Services receive registry via `fx.In`
5. ✅ Everything is testable and explicit

**Result:** A clean, maintainable, testable permission system following Go best practices!
