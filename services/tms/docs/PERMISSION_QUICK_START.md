# Permission System - Quick Start Guide

## TL;DR

Each domain entity defines its own permissions. No manual enum updates needed. Just implement `PermissionAware` interface.

---

## Adding Permissions to a New Entity

### 1. Create `permission.go` in your domain package

```go
package myentity

import "github.com/emoss08/trenova/pkg/permissionregistry"

type MyEntityPermission struct{}

func (m *MyEntityPermission) GetResourceName() string {
    return "my_entity"
}

func (m *MyEntityPermission) GetResourceDescription() string {
    return "My entity does XYZ"
}

func (m *MyEntityPermission) GetSupportedOperations() []permissionregistry.OperationDefinition {
    return permissionregistry.StandardCRUDOperations() // create, read, update, delete
}

func (m *MyEntityPermission) GetCompositeOperations() map[string]uint32 {
    return map[string]uint32{
        "manage":    permissionregistry.CompositeBasicCRUD, // All CRUD
        "read_only": permissionregistry.OpRead,              // Just read
    }
}

func (m *MyEntityPermission) GetDefaultOperation() uint32 {
    return permissionregistry.OpRead
}

func (m *MyEntityPermission) GetOperationsRequiringApproval() []uint32 {
    return []uint32{permissionregistry.OpDelete}
}

// Constructor for dependency injection
func NewMyEntityPermission() permissionregistry.PermissionAware {
    return &MyEntityPermission{}
}
```

### 2. Register in Bootstrap Module

**File:** `/services/tms/internal/bootstrap/modules/permissionregistry/module.go`

Add your entity to the module:

```go
fx.Provide(
    // ... existing entities
    fx.Annotate(
        myentity.NewMyEntityPermission,
        fx.ResultTags(`group:"permission_entities"`),
    ),
)
```

### 3. Use in Handler

```go
func (h *MyEntityHandler) RegisterRoutes(rg *gin.RouterGroup) {
    api := rg.Group("/my-entities/")
    api.GET("", h.middleware.RequirePermission("my_entity", "read"), h.list)
    api.POST("", h.middleware.RequirePermission("my_entity", "create"), h.create)
    api.PUT(":id/", h.middleware.RequirePermission("my_entity", "update"), h.update)
    api.DELETE(":id/", h.middleware.RequirePermission("my_entity", "delete"), h.delete)
}
```

### 4. Done

The permission system now knows:

- ✅ What operations `my_entity` supports
- ✅ That "manage" = all CRUD operations
- ✅ That delete requires approval
- ✅ Admin UI can query this and show appropriate checkboxes
- ✅ Everything is properly dependency injected (no globals!)

---

## Standard Operation Helpers

### Basic CRUD

```go
permissionregistry.StandardCRUDOperations()
// Returns: create, read, update, delete
```

### Export/Import

```go
ops := permissionregistry.StandardCRUDOperations()
ops = append(ops, permissionregistry.StandardExportImportOperations()...)
// Returns: create, read, update, delete, export, import
```

### Workflow

```go
ops := permissionregistry.StandardCRUDOperations()
ops = append(ops, permissionregistry.StandardWorkflowOperations()...)
// Returns: create, read, update, delete, approve, reject, submit
```

### All Operations

```go
ops := []permissionregistry.OperationDefinition{}
ops = append(ops, permissionregistry.StandardCRUDOperations()...)
ops = append(ops, permissionregistry.StandardExportImportOperations()...)
ops = append(ops, permissionregistry.StandardArchiveOperations()...)
ops = append(ops, permissionregistry.StandardWorkflowOperations()...)
ops = append(ops, permissionregistry.StandardAssignmentOperations()...)
// Returns: create, read, update, delete, export, import, archive, restore, approve, reject, submit, assign, share
```

---

## Standard Composite Operations

```go
func (m *MyEntityPermission) GetCompositeOperations() map[string]uint32 {
    return map[string]uint32{
        "manage":     permissionregistry.CompositeManageFull, // All operations
        "basic_crud": permissionregistry.CompositeBasicCRUD,  // CRUD only
        "read_only":  permissionregistry.CompositeReadOnly,   // Read + Export
        "workflow":   permissionregistry.CompositeWorkflow,   // Approve + Reject + Submit
    }
}
```

---

## Field-Level Permissions

Implement `FieldPermissionAware`:

```go
func (m *MyEntityPermission) GetFieldDefinitions() []permissionregistry.FieldDefinition {
    return []permissionregistry.FieldDefinition{
        {
            Name:        "price",
            DisplayName: "Price",
            Description: "Customer pricing",
            Type:        "number",
            IsSensitive: true,
            DefaultMaskType: "partial",
            Group:       "financial",
            Tags:        []string{"sensitive", "pii"},
        },
        {
            Name:        "status",
            DisplayName: "Status",
            Description: "Current status",
            Type:        "enum",
            IsRequired:  true,
            Group:       "basic",
            Tags:        []string{"required"},
        },
    }
}

func (m *MyEntityPermission) GetSensitiveFields() []string {
    return []string{"price", "ssn", "creditCard"}
}

func (m *MyEntityPermission) GetReadOnlyFields() []string {
    return []string{"id", "createdAt", "updatedAt"}
}
```

---

## Data Scoping

Implement `DataScopeAware`:

```go
func (m *MyEntityPermission) GetSupportedDataScopes() []string {
    return []string{
        "all",           // System admin sees all
        "business_unit", // BU admin sees across orgs
        "organization",  // Normal users see only their org
        "own",           // Users see only their own records
    }
}

func (m *MyEntityPermission) GetDefaultDataScope() string {
    return "organization"
}

func (m *MyEntityPermission) GetOwnerField() string {
    return "created_by" // Field name for ownership check
}
```

---

## Workflow Support

Implement `WorkflowAware`:

```go
func (m *MyEntityPermission) GetWorkflowStates() []permissionregistry.WorkflowState {
    return []permissionregistry.WorkflowState{
        {Name: "draft", DisplayName: "Draft", Description: "Initial state"},
        {Name: "pending", DisplayName: "Pending Approval", Description: "Waiting for approval"},
        {Name: "approved", DisplayName: "Approved", Description: "Approved and active", IsFinal: true},
        {Name: "rejected", DisplayName: "Rejected", Description: "Rejected", IsFinal: true},
    }
}

func (m *MyEntityPermission) GetWorkflowTransitions() []permissionregistry.WorkflowTransition {
    return []permissionregistry.WorkflowTransition{
        {
            From: "draft",
            To: "pending",
            RequiredOperation: permissionregistry.OpSubmit,
            DisplayName: "Submit for Approval",
        },
        {
            From: "pending",
            To: "approved",
            RequiredOperation: permissionregistry.OpApprove,
            DisplayName: "Approve",
        },
        {
            From: "pending",
            To: "rejected",
            RequiredOperation: permissionregistry.OpReject,
            DisplayName: "Reject",
        },
    }
}
```

---

## Querying the Registry

### Get all resources

```go
resources := permissionregistry.GetAllResources()
for name, resource := range resources {
    fmt.Println(name, resource.GetResourceDescription())
}
```

### Get specific resource

```go
if res, exists := permissionregistry.GetResource("my_entity"); exists {
    operations := res.GetSupportedOperations()
    for _, op := range operations {
        fmt.Printf("%s: %s\n", op.DisplayName, op.Description)
    }
}
```

### Expand composite operation

```go
registry := permissionregistry.GetGlobalRegistry()
bitfield, _ := registry.ExpandCompositeOperation("my_entity", "manage")
operations := permissionregistry.GetOperationsFromBitfield(bitfield)
// operations = ["create", "read", "update", "delete", ...]
```

---

## Complete Example

See `/services/tms/internal/core/domain/hazardousmaterial/permission.go` for a full implementation with:

- Standard operations
- Composite operations
- Field definitions
- Sensitive field handling
- Data scoping
- Approval requirements

---

## Testing

```go
func TestMyEntityPermissions(t *testing.T) {
    perm := &MyEntityPermission{}

    // Test resource name
    assert.Equal(t, "my_entity", perm.GetResourceName())

    // Test operations
    ops := perm.GetSupportedOperations()
    assert.Len(t, ops, 4) // CRUD = 4 operations

    // Test composite operations
    composites := perm.GetCompositeOperations()
    manageBitfield := composites["manage"]

    assert.True(t, permissionregistry.HasOperation(manageBitfield, permissionregistry.OpCreate))
    assert.True(t, permissionregistry.HasOperation(manageBitfield, permissionregistry.OpRead))
}
```

---

## FAQ

**Q: Do I need to update the Resource enum when adding a new entity?**
A: No! Just implement `PermissionAware` and register it. The resource name comes from `GetResourceName()`.

**Q: How does the admin UI know what operations are available?**
A: It queries `/api/permissions/registry/{resource}` which returns metadata from the registry.

**Q: What if I want custom operations beyond CRUD?**
A: Define them using `BuildOperationDefinition()` with any bitfield value you want.

**Q: Can I have resource-specific composite operations?**
A: Yes! Define them in `GetCompositeOperations()`. For example, hazmat has "safety_officer" and "compliance" roles.

**Q: Do I have to implement all interfaces?**
A: No. Only implement what you need:

- `PermissionAware` - Required (base interface)
- `FieldPermissionAware` - Optional (field-level permissions)
- `DataScopeAware` - Optional (data scoping)
- `WorkflowAware` - Optional (approval workflows)

---

## See Also

- [PERMISSION_REGISTRY_SYSTEM.md](PERMISSION_REGISTRY_SYSTEM.md) - Complete technical documentation
- [PERMISSION_SYSTEM_V2_TECHNICAL_DESIGN.md](PERMISSION_SYSTEM_V2_TECHNICAL_DESIGN.md) - Overall permission system design
- [PERMISSION_FIELD_LEVEL_EXAMPLES.md](PERMISSION_FIELD_LEVEL_EXAMPLES.md) - Field-level permission patterns
