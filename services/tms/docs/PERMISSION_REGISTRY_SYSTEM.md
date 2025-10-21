# Permission Registry System

## Overview

The Permission Registry System is a decentralized, self-documenting approach to defining permissions in the Trenova TMS. Instead of hardcoding resource names and operations in a central location, **each domain entity defines its own permission requirements** using interfaces.

This eliminates:

- ✅ Import cycles
- ✅ Manual enum maintenance
- ✅ Disconnection between entities and their permissions
- ✅ Need to remember what operations each resource supports

And provides:

- ✅ Type-safe permission definitions
- ✅ Self-documenting API (UI can query what operations are available)
- ✅ Composite operations (e.g., "manage" = CRUD automatically)
- ✅ Field-level metadata for dynamic forms
- ✅ Workflow support for approval processes

---

## Architecture

### Structure

```
pkg/permissionregistry/          ← Core interfaces & registry (no domain imports)
    types.go                      ← Interface definitions
    registry.go                   ← Global registry implementation
    helpers.go                    ← Operation constants & helpers

pkg/domainregistry/              ← Re-exports for convenience
    permission.go                 ← Type aliases to permissionregistry

internal/core/domain/{entity}/   ← Domain implementations
    {entity}.go                   ← Entity definition
    permission.go                 ← Implements PermissionAware interface
```

### Flow

```
1. Entity defines permissions in permission.go
   ↓
2. init() registers with global registry
   ↓
3. Permission engine queries registry at runtime
   ↓
4. Admin UI queries registry to show available operations
   ↓
5. Policies reference resource names & operations dynamically
```

---

## Core Interfaces

### PermissionAware (Base Interface)

Every entity that requires permissions MUST implement this interface.

```go
type PermissionAware interface {
    // GetResourceName returns the unique resource identifier
    GetResourceName() string

    // GetResourceDescription returns human-readable description
    GetResourceDescription() string

    // GetSupportedOperations returns available operations
    GetSupportedOperations() []OperationDefinition

    // GetCompositeOperations returns named operation combinations
    GetCompositeOperations() map[string]uint32

    // GetDefaultOperation returns default operation (usually Read)
    GetDefaultOperation() uint32

    // GetOperationsRequiringApproval returns operations requiring approval
    GetOperationsRequiringApproval() []uint32
}
```

### FieldPermissionAware (Extended Interface)

For entities with field-level permissions:

```go
type FieldPermissionAware interface {
    PermissionAware

    // GetFieldDefinitions returns metadata for all fields
    GetFieldDefinitions() []FieldDefinition

    // GetSensitiveFields returns fields that should be masked
    GetSensitiveFields() []string

    // GetReadOnlyFields returns fields that cannot be modified
    GetReadOnlyFields() []string
}
```

### DataScopeAware (Extended Interface)

For entities that support data scoping (own, org, all):

```go
type DataScopeAware interface {
    PermissionAware

    // GetSupportedDataScopes returns available data scopes
    GetSupportedDataScopes() []string

    // GetDefaultDataScope returns default scope
    GetDefaultDataScope() string

    // GetOwnerField returns field name for ownership checks
    GetOwnerField() string
}
```

### WorkflowAware (Extended Interface)

For entities with approval workflows:

```go
type WorkflowAware interface {
    PermissionAware

    // GetWorkflowStates returns possible workflow states
    GetWorkflowStates() []WorkflowState

    // GetWorkflowTransitions returns allowed state transitions
    GetWorkflowTransitions() []WorkflowTransition
}
```

---

## Implementation Example

### Step 1: Create `permission.go` in your domain package

**File:** `/services/tms/internal/core/domain/hazardousmaterial/permission.go`

```go
package hazardousmaterial

import (
    "github.com/emoss08/trenova/pkg/permissionregistry"
)

// Ensure HazardousMaterial implements interfaces (compile-time check)
var (
    _ permissionregistry.PermissionAware      = (*HazardousMaterialPermission)(nil)
    _ permissionregistry.FieldPermissionAware = (*HazardousMaterialPermission)(nil)
    _ permissionregistry.DataScopeAware       = (*HazardousMaterialPermission)(nil)
)

// HazardousMaterialPermission implements permission metadata
type HazardousMaterialPermission struct{}

// GetResourceName returns the unique resource identifier
func (h *HazardousMaterialPermission) GetResourceName() string {
    return "hazardous_material"
}

// GetResourceDescription returns human-readable description
func (h *HazardousMaterialPermission) GetResourceDescription() string {
    return "Hazardous materials database for managing dangerous goods, UN numbers, and shipping requirements"
}

// GetSupportedOperations defines what operations this resource supports
func (h *HazardousMaterialPermission) GetSupportedOperations() []permissionregistry.OperationDefinition {
    return []permissionregistry.OperationDefinition{
        permissionregistry.BuildOperationDefinition(
            permissionregistry.OpCreate,
            "create",
            "Create",
            "Add new hazardous materials to the database",
            "plus",
        ),
        permissionregistry.BuildOperationDefinition(
            permissionregistry.OpRead,
            "read",
            "Read",
            "View hazardous material information",
            "eye",
        ),
        permissionregistry.BuildOperationDefinition(
            permissionregistry.OpUpdate,
            "update",
            "Update",
            "Modify hazardous material details",
            "edit",
        ),
        permissionregistry.BuildOperationDefinition(
            permissionregistry.OpDelete,
            "delete",
            "Delete",
            "Remove hazardous materials (requires approval)",
            "trash",
        ),
        permissionregistry.BuildOperationDefinition(
            permissionregistry.OpExport,
            "export",
            "Export",
            "Export hazmat data for compliance reporting",
            "download",
        ),
        permissionregistry.BuildOperationDefinition(
            permissionregistry.OpImport,
            "import",
            "Import",
            "Import hazmats from UN database",
            "upload",
        ),
    }
}

// GetCompositeOperations defines shortcut operation combinations
func (h *HazardousMaterialPermission) GetCompositeOperations() map[string]uint32 {
    return map[string]uint32{
        // "manage" gives all operations
        "manage": permissionregistry.OpCreate | permissionregistry.OpRead |
                  permissionregistry.OpUpdate | permissionregistry.OpDelete |
                  permissionregistry.OpExport | permissionregistry.OpImport,

        // Safety officer gets CRUD + export, no delete
        "safety_officer": permissionregistry.OpCreate | permissionregistry.OpRead |
                          permissionregistry.OpUpdate | permissionregistry.OpExport,

        // Compliance role gets read + export only
        "compliance": permissionregistry.OpRead | permissionregistry.OpExport,

        // Read-only access
        "read_only": permissionregistry.OpRead,
    }
}

// GetDefaultOperation returns the default operation for basic access
func (h *HazardousMaterialPermission) GetDefaultOperation() uint32 {
    return permissionregistry.OpRead
}

// GetOperationsRequiringApproval lists operations requiring approval
func (h *HazardousMaterialPermission) GetOperationsRequiringApproval() []uint32 {
    return []uint32{
        permissionregistry.OpCreate, // Creating hazmat requires approval due to safety
        permissionregistry.OpUpdate, // Updating requires approval for regulatory compliance
        permissionregistry.OpDelete, // Deleting requires approval for safety records
    }
}

// GetFieldDefinitions provides metadata for each field
func (h *HazardousMaterialPermission) GetFieldDefinitions() []permissionregistry.FieldDefinition {
    return []permissionregistry.FieldDefinition{
        {
            Name:        "code",
            DisplayName: "Code",
            Description: "Short code for quick reference",
            Type:        "string",
            IsRequired:  true,
            Group:       "basic",
            Tags:        []string{"required", "unique"},
        },
        {
            Name:            "emergencyContactPhoneNumber",
            DisplayName:     "Emergency Phone",
            Description:     "24-hour emergency response phone number",
            Type:            "string",
            IsSensitive:     true,
            DefaultMaskType: "partial",
            IsRequired:      false,
            Group:           "emergency",
            Tags:            []string{"sensitive", "pii"},
        },
        // ... more fields
    }
}

// GetSensitiveFields returns fields that should be masked
func (h *HazardousMaterialPermission) GetSensitiveFields() []string {
    return []string{
        "emergencyContact",
        "emergencyContactPhoneNumber",
    }
}

// GetReadOnlyFields returns fields that cannot be modified
func (h *HazardousMaterialPermission) GetReadOnlyFields() []string {
    return []string{
        "id",
        "version",
        "createdAt",
        "updatedAt",
    }
}

// GetSupportedDataScopes defines what scoping this resource supports
func (h *HazardousMaterialPermission) GetSupportedDataScopes() []string {
    return []string{
        "all",           // System admins see all
        "business_unit", // BU admins see across orgs
        "organization",  // Most users see only their org
    }
}

// GetDefaultDataScope returns the default data scope
func (h *HazardousMaterialPermission) GetDefaultDataScope() string {
    return "organization"
}

// GetOwnerField returns the field name used for ownership checks
func (h *HazardousMaterialPermission) GetOwnerField() string {
    // Hazmat doesn't have an owner field - scoped by organization
    return ""
}

// Register this resource with the global registry on init
func init() {
    permissionregistry.Register(&HazardousMaterialPermission{})
}
```

---

## Using the Registry

### Query Available Resources

```go
import "github.com/emoss08/trenova/pkg/permissionregistry"

// Get all registered resources
resources := permissionregistry.GetAllResources()

// Get specific resource
if res, exists := permissionregistry.GetResource("hazardous_material"); exists {
    fmt.Println(res.GetResourceDescription())

    // Get operations
    operations := res.GetSupportedOperations()
    for _, op := range operations {
        fmt.Printf("%s: %s\n", op.Name, op.Description)
    }
}
```

### Query Composite Operations

```go
registry := permissionregistry.GetGlobalRegistry()

// Get composite operations for hazardous_material
if composites, exists := registry.GetCompositeOperationsForResource("hazardous_material"); exists {
    // Get "manage" operation bitfield
    manageBitfield := composites["manage"]

    // Expand to see what it includes
    operations := permissionregistry.GetOperationsFromBitfield(manageBitfield)
    // Returns: ["create", "read", "update", "delete", "export", "import"]
}
```

### Build Policies Dynamically

```go
func CreatePolicyForRole(resourceName string, compositeOp string) (*Policy, error) {
    registry := permissionregistry.GetGlobalRegistry()

    // Expand the composite operation
    bitfield, exists := registry.ExpandCompositeOperation(resourceName, compositeOp)
    if !exists {
        return nil, fmt.Errorf("composite operation %s not found for %s", compositeOp, resourceName)
    }

    // Get individual operations
    operations := permissionregistry.GetOperationsFromBitfield(bitfield)

    // Create policy
    policy := &Policy{
        Name: fmt.Sprintf("%s_%s", resourceName, compositeOp),
        Resources: Resources{
            ResourceType: []string{resourceName},
            Actions:      operations,
        },
        Effect:   EffectAllow,
        Priority: 100,
    }

    return policy, nil
}
```

---

## API Endpoints for Admin UI

### GET `/api/permissions/registry`

Returns all registered resources with their operations.

```json
{
  "resources": {
    "hazardous_material": {
      "name": "hazardous_material",
      "description": "Hazardous materials database for managing dangerous goods...",
      "operations": [
        {
          "code": 1,
          "name": "create",
          "displayName": "Create",
          "description": "Add new hazardous materials to the database",
          "icon": "plus"
        },
        {
          "code": 2,
          "name": "read",
          "displayName": "Read",
          "description": "View hazardous material information",
          "icon": "eye"
        }
      ],
      "compositeOperations": {
        "manage": 63,
        "safety_officer": 31,
        "compliance": 18,
        "read_only": 2
      },
      "defaultOperation": 2,
      "operationsRequiringApproval": [1, 4, 8]
    }
  }
}
```

### GET `/api/permissions/registry/:resource`

Returns details for a specific resource.

```json
{
  "name": "hazardous_material",
  "description": "Hazardous materials database...",
  "operations": [...],
  "compositeOperations": {...},
  "fields": [
    {
      "name": "code",
      "displayName": "Code",
      "description": "Short code for quick reference",
      "type": "string",
      "isRequired": true,
      "isSensitive": false,
      "group": "basic",
      "tags": ["required", "unique"]
    }
  ],
  "sensitiveFields": ["emergencyContact", "emergencyContactPhoneNumber"],
  "readOnlyFields": ["id", "version", "createdAt", "updatedAt"],
  "supportedDataScopes": ["all", "business_unit", "organization"],
  "defaultDataScope": "organization"
}
```

---

## Benefits of This Approach

### 1. No Manual Enum Maintenance

**Before (Manual):**

```go
const (
    ResourceHazardousMaterial = "hazardous_material"
    ResourceCustomer = "customer"
    ResourceShipment = "shipment"
    // Add new resource? Update enum, operation maps, docs...
)
```

**After (Automatic):**

```go
// Just create permission.go in your domain
// Everything else is discovered automatically
```

### 2. Self-Documenting API

Admin UI can dynamically build permission management forms:

```typescript
// Fetch available resources
const resources = await api.get('/api/permissions/registry');

// Build form dynamically
resources.forEach(resource => {
  addResourceSection(resource.name);

  // Show composite operations as shortcuts
  resource.compositeOperations.forEach(comp => {
    addCheckbox(comp.name, comp.displayName);
  });

  // Show individual operations for fine-grained control
  resource.operations.forEach(op => {
    addCheckbox(op.name, op.displayName, op.description);
  });
});
```

### 3. Composite Operations (The "Manage" Problem Solved)

Instead of clicking create, read, update, delete, export, import one by one:

```go
// Define once in the entity
func (h *HazardousMaterialPermission) GetCompositeOperations() map[string]uint32 {
    return map[string]uint32{
        "manage": permissionregistry.OpCreate | permissionregistry.OpRead |
                  permissionregistry.OpUpdate | permissionregistry.OpDelete |
                  permissionregistry.OpExport | permissionregistry.OpImport,
    }
}
```

Admin UI shows:

```
☐ Manage (all operations)
or
☐ Create
☐ Read
☐ Update
☐ Delete
☐ Export
☐ Import
```

Checking "Manage" automatically selects all operations.

### 4. Type Safety

```go
// Compile-time enforcement
var _ permissionregistry.PermissionAware = (*HazardousMaterialPermission)(nil)

// If you forget to implement a required method, it won't compile
```

### 5. No Import Cycles

```
domain/hazardousmaterial → pkg/permissionregistry ✅
pkg/permissionregistry → NO domain imports ✅
permission engine → pkg/permissionregistry ✅
```

---

## Code Generation Opportunities

Since all resources are registered, we can auto-generate:

### 1. Resource Enum

```go
// Generated file: permission/enums_generated.go
package permission

const (
    ResourceHazardousMaterial = Resource("hazardous_material")
    ResourceCustomer = Resource("customer")
    ResourceShipment = Resource("shipment")
    // ... auto-generated from registry
)
```

### 2. TypeScript Types

```typescript
// Generated file: types/permission.ts
export type Resource =
  | "hazardous_material"
  | "customer"
  | "shipment";

export type HazardousMaterialOperation =
  | "create"
  | "read"
  | "update"
  | "delete"
  | "export"
  | "import";
```

### 3. OpenAPI Documentation

Auto-generate permission-related API specs from registry.

---

## Migration Path

### Step 1: Create Permission Files

For each existing entity, create `{entity}/permission.go`:

```bash
# Template
cat > internal/core/domain/customer/permission.go << 'EOF'
package customer

import "github.com/emoss08/trenova/pkg/permissionregistry"

type CustomerPermission struct{}

func (c *CustomerPermission) GetResourceName() string {
    return "customer"
}

// ... implement remaining methods

func init() {
    permissionregistry.Register(&CustomerPermission{})
}
EOF
```

### Step 2: Update Permission Engine

Query registry instead of hardcoded maps:

```go
func (e *Engine) GetResourceOperations(resourceName string) ([]string, error) {
    res, exists := permissionregistry.GetResource(resourceName)
    if !exists {
        return nil, fmt.Errorf("resource %s not found", resourceName)
    }

    operations := res.GetSupportedOperations()
    names := make([]string, len(operations))
    for i, op := range operations {
        names[i] = op.Name
    }

    return names, nil
}
```

### Step 3: Build Admin UI

Create dynamic permission management interface that queries registry:

```typescript
function PermissionBuilder({ resource }: { resource: string }) {
  const { data: metadata } = useQuery({
    queryKey: ['permission-metadata', resource],
    queryFn: () => api.get(`/api/permissions/registry/${resource}`),
  });

  return (
    <div>
      <h3>{metadata.description}</h3>

      {/* Composite operations */}
      <h4>Quick Access</h4>
      {Object.entries(metadata.compositeOperations).map(([name, bitfield]) => (
        <label key={name}>
          <input type="checkbox" value={name} />
          {name} (grants: {getOperationsFromBitfield(bitfield).join(', ')})
        </label>
      ))}

      {/* Individual operations */}
      <h4>Individual Permissions</h4>
      {metadata.operations.map(op => (
        <label key={op.name} title={op.description}>
          <input type="checkbox" value={op.name} />
          <Icon name={op.icon} />
          {op.displayName}
        </label>
      ))}
    </div>
  );
}
```

---

## Testing

### Unit Test Example

```go
func TestHazardousMaterialPermissions(t *testing.T) {
    perm := &HazardousMaterialPermission{}

    // Test resource name
    assert.Equal(t, "hazardous_material", perm.GetResourceName())

    // Test operations
    ops := perm.GetSupportedOperations()
    assert.NotEmpty(t, ops)

    // Test composite operations include expected bitfields
    composites := perm.GetCompositeOperations()
    manageBitfield := composites["manage"]

    assert.True(t, permissionregistry.HasOperation(manageBitfield, permissionregistry.OpCreate))
    assert.True(t, permissionregistry.HasOperation(manageBitfield, permissionregistry.OpRead))
    assert.True(t, permissionregistry.HasOperation(manageBitfield, permissionregistry.OpUpdate))
    assert.True(t, permissionregistry.HasOperation(manageBitfield, permissionregistry.OpDelete))
}
```

### Integration Test Example

```go
func TestPermissionRegistry(t *testing.T) {
    // Ensure hazmat is registered
    res, exists := permissionregistry.GetResource("hazardous_material")
    assert.True(t, exists)
    assert.NotNil(t, res)

    // Test operation expansion
    registry := permissionregistry.GetGlobalRegistry()
    bitfield, found := registry.ExpandCompositeOperation("hazardous_material", "manage")
    assert.True(t, found)
    assert.NotZero(t, bitfield)

    // Test bitfield conversion
    operations := permissionregistry.GetOperationsFromBitfield(bitfield)
    assert.Contains(t, operations, "create")
    assert.Contains(t, operations, "read")
}
```

---

## Summary

The Permission Registry System solves the original problem:

**Question:** "How does the admin know that 'manage' operation exists for a resource? How do they know what operations are available?"

**Answer:** The registry! Each entity self-documents its operations, and the admin UI queries the registry to show available options.

**Benefits:**

- ✅ No manual enum maintenance
- ✅ Self-documenting (UI queries registry)
- ✅ Composite operations solve "click fatigue"
- ✅ Type-safe at compile time
- ✅ No import cycles
- ✅ Scalable (add entities without touching permission code)
- ✅ Field-level metadata included
- ✅ Workflow support built-in
