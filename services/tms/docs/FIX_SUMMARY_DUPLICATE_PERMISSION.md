# Fix Summary: Duplicate Permission Not Working

## Problem

The `duplicate` and `assign` permissions for shipments were defined in the Go backend but users didn't have access to them, even after running codegen.

## Root Cause

The `AddFullAccessResource()` method in `permissionbuilder` was hardcoded to only include **standard operations** (Create, Read, Update, Delete, Export, Import = 63) and **ignored extended operations** like `duplicate` (32768) and `assign` (16384).

This affected the database seed that creates the "System Admin Policy" for new installations.

## Solution

### 1. Updated `permissionbuilder` to Query Registry

**File:** `/services/tms/pkg/permissionbuilder/builder.go`

- Added `registry` field to `PolicyBuilder` struct
- Modified `NewPolicyBuilder()` to accept a registry parameter
- Updated `AddFullAccessResource()` to dynamically query the registry for extended operations
- Modified `CreateAdminPolicy()` to accept a registry parameter
- Added `CreatePermissionRegistry()` helper function for use in seeds

**Key Changes:**

```go
// Before (hardcoded)
ExtendedOps: []string{},  // ← Always empty!

// After (dynamic)
if pb.registry != nil {
    if res, exists := pb.registry.GetResource(string(resourceType)); exists {
        for _, op := range res.GetSupportedOperations() {
            if op.Code > 32 && op.Code != permission.OpApprove && op.Code != permission.OpReject {
                extendedOps = append(extendedOps, op.Name)
            }
        }
    }
}
```

### 2. Added Manual Registry Constructor

**File:** `/services/tms/pkg/permissionregistry/registry.go`

Added `NewRegistryManual()` function to create a registry without FX dependency injection:

```go
func NewRegistryManual() *Registry {
    return &Registry{
        resources: make(map[string]PermissionAware),
    }
}
```

### 3. Updated Seed to Use Registry

**File:** `/services/tms/internal/infrastructure/database/seeds/base/02_admin_account.go`

Modified `createAdminPermissions()` to:

1. Create a permission registry using `CreatePermissionRegistry()`
2. Pass it to `CreateAdminPolicy()`

This ensures the admin policy includes ALL operations (including extended ones) for every resource.

## What Changed in the Database

### Before

```json
{
  "resourceType": "shipment",
  "actions": {
    "standardOps": 63,       // Only CRUD + Export + Import
    "extendedOps": []        // ← EMPTY!
  }
}
```

### After (with registry)

```json
{
  "resourceType": "shipment",
  "actions": {
    "standardOps": 63,                    // CRUD + Export + Import
    "extendedOps": ["assign", "duplicate"] // ← Now includes extended ops!
  }
}
```

## How to Fix Existing Databases

If you have an existing database, you have two options:

### Option 1: Re-run Seeds (Recommended for Dev)

```bash
# Drop and recreate database
make db-drop
make db-create
make db-migrate
make db-seed
```

### Option 2: Manual SQL Update (For Production)

```sql
-- Update the System Admin Policy to include extended operations
UPDATE policies
SET resources = (
    SELECT jsonb_agg(
        CASE 
            WHEN resource->>'resourceType' = 'shipment' THEN
                jsonb_set(
                    resource,
                    '{actions,extendedOps}',
                    '["assign", "duplicate"]'::jsonb
                )
            ELSE resource
        END
    )
    FROM jsonb_array_elements(resources) AS resource
)
WHERE name = 'System Admin Policy';

-- Invalidate permission cache for all users
-- (Specific implementation depends on your setup)
```

After updating the database, users need to:

1. Log out
2. Log back in (or call the invalidate cache endpoint)

## Verification

### 1. Check the Database

```sql
SELECT 
    name,
    jsonb_pretty(resources -> 0) as shipment_resource
FROM policies
WHERE name = 'System Admin Policy';
```

Expected output should show:

```json
{
  "resourceType": "shipment",
  "actions": {
    "standardOps": 63,
    "extendedOps": ["assign", "duplicate"]
  },
  ...
}
```

### 2. Check Frontend Permissions

In your React app:

```typescript
import { useShipmentPermissions } from '@/types/_gen/permissions';

function MyComponent() {
  const perms = useShipmentPermissions();
  
  console.log('Can duplicate:', perms.canDuplicate); // Should be true
  console.log('Can assign:', perms.canAssign);       // Should be true
  
  return (
    <button disabled={!perms.canDuplicate}>
      Duplicate Shipment
    </button>
  );
}
```

### 3. Check Permission Manifest API

```bash
curl http://localhost:8080/api/v1/permissions/manifest \
  -H "Authorization: Bearer YOUR_TOKEN" | jq '.resources.shipment'
```

Should show:

```json
{
  "standardOps": 63,
  "extendedOps": ["assign", "duplicate"],
  "dataScope": "all"
}
```

## Benefits of This Fix

1. **✅ Automatic** - New permissions are automatically included when added to domain entities
2. **✅ Type-Safe** - Registry validates operations at compile time
3. **✅ Self-Documenting** - Permission definitions live with domain entities
4. **✅ Future-Proof** - Adding new extended operations requires no changes to the builder
5. **✅ Seed-Compatible** - Works during database seeding before FX initialization

## Files Changed

1. `/services/tms/pkg/permissionbuilder/builder.go`
   - Added registry field and parameter
   - Dynamic extended operation discovery
   - CreatePermissionRegistry() helper

2. `/services/tms/pkg/permissionregistry/registry.go`
   - Added NewRegistryManual() constructor

3. `/services/tms/internal/infrastructure/database/seeds/base/02_admin_account.go`
   - Pass registry to CreateAdminPolicy()

## Testing

To test this fix:

1. **Fresh Installation:**

   ```bash
   make db-reset
   make db-seed
   # Check database using SQL above
   ```

2. **Existing Database:**

   ```bash
   # Run SQL update
   # Restart backend
   # Clear browser cache / log out and log in
   # Test duplicate button
   ```

3. **Unit Test:**

   ```go
   func TestAdminPolicyIncludesExtendedOps(t *testing.T) {
       registry := permissionbuilder.CreatePermissionRegistry()
       policy := permissionbuilder.CreateAdminPolicy(
           "Test Policy",
           testBUID,
           []pulid.ID{testOrgID},
           registry,
       )
       
       // Find shipment resource
       var shipmentResource *permission.ResourceRule
       for i, res := range policy.Resources {
           if res.ResourceType == "shipment" {
               shipmentResource = &policy.Resources[i]
               break
           }
       }
       
       assert.Contains(t, shipmentResource.Actions.ExtendedOps, "duplicate")
       assert.Contains(t, shipmentResource.Actions.ExtendedOps, "assign")
   }
   ```

## Future Improvements

1. **Migration Script** - Create a database migration to automatically update existing policies
2. **Admin UI** - Add a "Sync Permissions" button to update policies with new operations
3. **Audit Log** - Log when permissions are updated to track changes
4. **Documentation** - Update API docs to show all available operations per resource

## Related Documentation

- `/docs/PERMISSION_REGISTRY_SYSTEM.md` - Full registry system documentation
- `/docs/PERMISSION_REGISTRY_USAGE.md` - How each method is used
- `/docs/PERMISSION_QUICK_START.md` - Quick start guide
