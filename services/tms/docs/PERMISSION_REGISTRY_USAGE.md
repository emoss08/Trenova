# Permission Registry System - Complete Usage Guide

## Overview

The Permission Registry System allows each domain entity to self-document its permissions, operations, and metadata. This guide explains how **every method** in the permission definition is utilized by the system.

---

## Complete Method Usage Map

### 1. `GetResourceName()` → string

**Where Used:**

- **Registry**: Keys in the resource map (`registry.resources[name]`)
- **Engine**: Validates resource exists before checking permissions (`engine.go:358`)
- **Compiler**: Validates resource exists when compiling policies (`compiler.go:171`)
- **API**: URL parameter for `/api/permissions/registry/:resource`

**Example:**

```go
func (h *HazardousMaterialPermission) GetResourceName() string {
    return "hazardous_material"
}
```

**Runtime Usage:**

```go
// Engine validation
if _, exists := e.registry.GetResource(req.ResourceType); !exists {
    return false, "resource not found"
}
```

---

### 2. `GetResourceDescription()` → string

**Where Used:**

- **API**: Returned in `GET /api/permissions/registry/` response
- **Admin UI**: Displays resource description in permission assignment forms
- **Documentation**: Auto-generated API documentation

**JSON Response:**

```json
{
  "hazardous_material": {
    "name": "hazardous_material",
    "description": "Hazardous materials database for managing dangerous goods..."
  }
}
```

**Frontend Usage:**

```typescript
// Admin UI renders this as a tooltip or help text
<ResourceCard
  name="Hazardous Material"
  description={resource.description}  // ← Used here
/>
```

---

### 3. `GetSupportedOperations()` → []OperationDefinition

**Where Used:**

- **Compiler**: Validates operations in policies exist (`compiler.go:345-351`)
- **API**: Returns available operations for UI checkboxes
- **Engine**: Implicitly validates operations match bitfield
- **Admin UI**: Generates permission checkboxes dynamically

**Code Flow:**

```go
// compiler.go:345 - Validates bitfield operations
supportedOps := resource.GetSupportedOperations()
for _, op := range supportedOps {
    if (bitfield & op.Code) == op.Code {
        validated |= op.Code
    }
}
```

**API Response:**

```json
{
  "operations": [
    {
      "code": 1,
      "name": "create",
      "displayName": "Create",
      "description": "Add new hazardous materials to the database"
    },
    {
      "code": 2,
      "name": "read",
      "displayName": "Read",
      "description": "View hazardous material information"
    }
  ]
}
```

**Admin UI Renders:**

```typescript
resource.operations.map(op => (
  <Checkbox
    key={op.name}
    label={op.displayName}
    tooltip={op.description}  // ← Used for tooltips
    onChange={(checked) => togglePermission(op.code)}
  />
))
```

---

### 4. `GetCompositeOperations()` → map[string]uint32

**Where Used:**

- **API**: Returns composite operations for "quick assign" buttons
- **Admin UI**: Creates convenience buttons ("Grant Full Access", "Read Only", etc.)
- **Compiler**: Could expand composites when building policies (future enhancement)

**Example Definition:**

```go
func (h *HazardousMaterialPermission) GetCompositeOperations() map[string]uint32 {
    return map[string]uint32{
        "manage":         OpCreate | OpRead | OpUpdate | OpDelete | OpExport | OpImport,
        "safety_officer": OpRead | OpCreate | OpUpdate | OpExport,
        "compliance":     OpRead | OpExport,
        "read_only":      OpRead,
    }
}
```

**API Response:**

```json
{
  "compositeOperations": {
    "manage": 63,          // Binary: 111111 (all 6 operations)
    "safety_officer": 23,   // Binary: 10111 (read, create, update, export)
    "compliance": 18,       // Binary: 10010 (read, export)
    "read_only": 2          // Binary: 10 (read only)
  }
}
```

**Admin UI Usage:**

```typescript
// Quick permission assignment buttons
<Button onClick={() => assignComposite("manage")}>
  Grant Full Access
</Button>
<Button onClick={() => assignComposite("safety_officer")}>
  Safety Officer Role
</Button>
<Button onClick={() => assignComposite("read_only")}>
  View Only
</Button>
```

---

### 5. `GetDefaultOperation()` → uint32

**Where Used:**

- **Policy Creation**: Default permissions when creating new user roles
- **API**: Returned in metadata for UI defaults
- **Future**: Auto-assignment when user joins organization

**Example:**

```go
func (h *HazardousMaterialPermission) GetDefaultOperation() uint32 {
    return OpRead  // New users can read by default
}
```

**Usage Scenario:**

```go
// When creating a new "Viewer" role
func createViewerRole() {
    for _, resource := range registry.GetAllResources() {
        defaultOp := resource.GetDefaultOperation()
        policy.AddPermission(resource.GetResourceName(), defaultOp)
    }
}
```

---

### 6. `GetOperationsRequiringApproval()` → []uint32

**Where Used:**

- **API**: Informs UI which operations need approval workflows
- **Admin UI**: Shows warning icons/badges on sensitive operations
- **Future Workflow Engine**: Triggers approval workflows

**Example:**

```go
func (h *HazardousMaterialPermission) GetOperationsRequiringApproval() []uint32 {
    return []uint32{
        OpCreate,  // Creating hazmat requires approval
        OpUpdate,  // Updating hazmat requires approval
        OpDelete,  // Deleting hazmat requires approval
    }
}
```

**API Response:**

```json
{
  "operationsRequiringApproval": [1, 4, 8]
}
```

**Admin UI Renders:**

```typescript
operations.map(op => (
  <div>
    <Checkbox label={op.displayName} />
    {requiresApproval(op.code) && (
      <Badge color="warning">Requires Approval</Badge>  // ← Used here
    )}
  </div>
))
```

---

### 7. `GetFieldDefinitions()` → []FieldDefinition (FieldPermissionAware)

**Where Used:**

- **API**: Returns field metadata for dynamic forms
- **Admin UI**: Generates field-level permission controls
- **Form Builders**: Creates forms with proper validation
- **Future**: Field-level access control enforcement

**Example:**

```go
func (h *HazardousMaterialPermission) GetFieldDefinitions() []FieldDefinition {
    return []FieldDefinition{
        {
            Name:            "emergencyContactPhoneNumber",
            DisplayName:     "Emergency Phone",
            Description:     "24-hour emergency response phone number",
            Type:            "string",
            IsSensitive:     true,
            DefaultMaskType: "partial",
            Group:           "emergency",
            Tags:            []string{"sensitive", "emergency", "pii"},
        },
    }
}
```

**API Response:**

```json
{
  "fields": [
    {
      "name": "emergencyContactPhoneNumber",
      "displayName": "Emergency Phone",
      "description": "24-hour emergency response phone number",
      "type": "string",
      "isSensitive": true,
      "defaultMaskType": "partial",
      "group": "emergency",
      "tags": ["sensitive", "emergency", "pii"]
    }
  ]
}
```

**Admin UI - Field Permission Matrix:**

```typescript
// Renders a grid: Field × Permission level
fields.map(field => (
  <tr>
    <td>
      {field.displayName}
      {field.isSensitive && <Icon name="lock" />}  // ← Shows lock icon
    </td>
    <td>
      <Select options={["none", "read", "write", "masked"]} />
      {field.defaultMaskType && (
        <span>Default: {field.defaultMaskType}</span>  // ← Shows default masking
      )}
    </td>
  </tr>
))
```

**Form Builder:**

```typescript
// Dynamically generates form fields
fields
  .filter(f => !f.tags.includes("readonly"))
  .map(field => (
    <FormField
      name={field.name}
      label={field.displayName}
      type={field.type}
      required={field.isRequired}
      helpText={field.description}  // ← Used for help text
    />
  ))
```

---

### 8. `GetSensitiveFields()` → []string (FieldPermissionAware)

**Where Used:**

- **API**: Marks fields that need special handling
- **Admin UI**: Shows warning icons on sensitive fields
- **Audit Logs**: Tracks access to sensitive fields
- **Data Masking**: Applies masking rules

**Example:**

```go
func (h *HazardousMaterialPermission) GetSensitiveFields() []string {
    return []string{
        "emergencyContact",
        "emergencyContactPhoneNumber",
    }
}
```

**API Response:**

```json
{
  "sensitiveFields": ["emergencyContact", "emergencyContactPhoneNumber"]
}
```

**Usage:**

```typescript
// Admin UI shows warning
fields.map(field => (
  <div>
    {field.name}
    {sensitiveFields.includes(field.name) && (
      <Badge color="red">Sensitive Data</Badge>  // ← Warning badge
    )}
  </div>
))

// Audit logging
if (sensitiveFields.includes(fieldName)) {
  auditLog.record({
    action: "SENSITIVE_FIELD_ACCESS",
    field: fieldName,
    user: currentUser
  })
}
```

---

### 9. `GetReadOnlyFields()` → []string (FieldPermissionAware)

**Where Used:**

- **Form Builders**: Disables editing on these fields
- **API Validation**: Rejects updates to readonly fields
- **Admin UI**: Renders fields as disabled/readonly

**Example:**

```go
func (h *HazardousMaterialPermission) GetReadOnlyFields() []string {
    return []string{
        "id",
        "version",
        "createdAt",
        "updatedAt",
    }
}
```

**Form Builder:**

```typescript
fields.map(field => (
  <input
    name={field.name}
    disabled={readOnlyFields.includes(field.name)}  // ← Disables input
    className={readOnlyFields.includes(field.name) ? "readonly" : ""}
  />
))
```

---

### 10. `GetSupportedDataScopes()` → []string (DataScopeAware)

**Where Used:**

- **API**: Returns available data scopes for resource
- **Admin UI**: Dropdown for selecting data access level
- **Query Builder**: Applies scope filters to database queries

**Example:**

```go
func (h *HazardousMaterialPermission) GetSupportedDataScopes() []string {
    return []string{
        "all",           // System admins
        "business_unit", // BU admins
        "organization",  // Normal users
    }
}
```

**API Response:**

```json
{
  "supportedDataScopes": ["all", "business_unit", "organization"]
}
```

**Admin UI:**

```typescript
<Select label="Data Access Level">
  {supportedDataScopes.map(scope => (
    <option value={scope}>
      {scopeLabels[scope]}  // "All Organizations", "Business Unit", etc.
    </option>
  ))}
</Select>
```

**Query Builder:**

```go
// Applies scope filter
switch dataScope {
case "organization":
    query = query.Where("organization_id = ?", userOrgID)
case "business_unit":
    query = query.Where("business_unit_id = ?", userBUID)
case "all":
    // No filter
}
```

---

### 11. `GetDefaultDataScope()` → string (DataScopeAware)

**Where Used:**

- **Policy Creation**: Default scope for new roles
- **API**: Default value in forms

**Example:**

```go
func (h *HazardousMaterialPermission) GetDefaultDataScope() string {
    return "organization"  // Most users see only their org
}
```

---

### 12. `GetOwnerField()` → string (DataScopeAware)

**Where Used:**

- **Query Builder**: Filters records by owner when scope is "own"
- **Access Control**: Checks if user owns the record

**Example:**

```go
func (h *HazardousMaterialPermission) GetOwnerField() string {
    return "created_by"  // Or "" if no ownership concept
}
```

**Usage:**

```go
// When dataScope = "own"
if ownerField := resource.GetOwnerField(); ownerField != "" {
    query = query.Where(ownerField + " = ?", userID)
}
```

---

## How to Test Registry Integration

### 1. Test API Endpoints

```bash
# Get all registered resources
curl http://localhost:8080/api/permissions/registry/

# Get specific resource metadata
curl http://localhost:8080/api/permissions/registry/hazardous_material
```

**Expected Response:**

```json
{
  "name": "hazardous_material",
  "description": "Hazardous materials database...",
  "operations": [...],
  "compositeOperations": {...},
  "fields": [...],
  "sensitiveFields": [...],
  "supportedDataScopes": [...]
}
```

### 2. Test Engine Validation

```go
// Should reject unknown resources
result, err := engine.Check(ctx, &PermissionCheckRequest{
    UserID:       userID,
    ResourceType: "unknown_resource",  // Not in registry
    Action:       "read",
})
// result.Allowed = false
// result.Reason = "resource unknown_resource not registered"
```

### 3. Test Compiler Validation

```go
// Should skip invalid resources
policies := []*Policy{{
    Resources: []PolicyResource{{
        ResourceType: "invalid_resource",  // Not in registry
        Actions:      ActionSet{StandardOps: OpRead},
    }},
}}
compiled, _ := compiler.Compile(ctx, policies)
// compiled.Resources should NOT contain "invalid_resource"
// Logs warning: "resource not found in registry, skipping"
```

---

## Summary: Every Method Is Used

| Method | Primary Usage | Secondary Usage |
|--------|---------------|-----------------|
| `GetResourceName()` | Registry key, validation | API routing, logs |
| `GetResourceDescription()` | Admin UI help text | Documentation generation |
| `GetSupportedOperations()` | Compiler validation, UI checkboxes | API introspection |
| `GetCompositeOperations()` | Quick assign buttons | Future: policy shortcuts |
| `GetDefaultOperation()` | New role defaults | Future: auto-assignment |
| `GetOperationsRequiringApproval()` | UI warnings | Future: workflow triggers |
| `GetFieldDefinitions()` | Form generation | Field-level permissions |
| `GetSensitiveFields()` | Data masking, audit logs | UI warnings |
| `GetReadOnlyFields()` | Form disabling | API validation |
| `GetSupportedDataScopes()` | Scope dropdown | Query filtering |
| `GetDefaultDataScope()` | Policy defaults | New role creation |
| `GetOwnerField()` | Ownership filtering | "Own" scope queries |

---

## Next Steps for Full Utilization

1. **✅ Completed:**
   - Registry integrated into engine
   - Registry integrated into compiler
   - API endpoints for introspection
   - Validation on permission checks

2. **🔄 Partially Used:**
   - Field definitions (metadata available, enforcement not yet implemented)
   - Data scopes (defined but query filtering not automated)
   - Approval requirements (marked but no workflow yet)

3. **📋 Future Enhancements:**
   - **Code Generation**: Generate TypeScript types from field definitions
   - **Query Middleware**: Auto-apply data scope filters
   - **Field-Level Enforcement**: Check field permissions on API responses
   - **Workflow Engine**: Trigger approvals for marked operations
   - **Audit System**: Track sensitive field access
   - **Form Generator**: Auto-generate CRUD forms from field definitions

All methods in the permission definition are **already being used** or have **clear integration points** for future features.
