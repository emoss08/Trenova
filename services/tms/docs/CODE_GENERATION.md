# Code Generation - Permission Registry

## Overview

The permission registry system includes automatic code generation to keep Go enums and TypeScript metadata in sync with permission definitions.

## Generated Files

### TypeScript (Frontend)

**Location:** `services/ui/src/types/_gen/permissions.ts`

**What's Generated:**

- ✅ Permission operation constants (`PermissionOperations`)
- ✅ Resource metadata (operations, composite operations, sensitive fields)
- ✅ React permission hooks (`useHazardousMaterialPermissions()`)
- ✅ Type guards (`isValidResource`, `isValidOperation`)

**What's NOT Generated:**

- ❌ Entity TypeScript types (use Zod schemas instead, e.g., `hazardous-material-schema.ts`)

### Go (Backend)

**Location:** `internal/core/domain/permission/resource_gen.go`

**What's Generated:**

- ✅ Resource enum (`Resource` type with constants)
- ✅ Helper methods (`IsValid()`, `AllResources()`, `ResourceDescriptions()`)

**Note:** This conflicts with existing `enums.go`. You'll need to decide whether to keep the manual enum or switch to generated.

---

## Usage

### Generate Everything

```bash
# From services/tms directory
make codegen

# Or manually
./build/trenova codegen generate
```

### Generate TypeScript Only

```bash
./build/trenova codegen types
```

### Generate Go Enum Only

```bash
./build/trenova codegen enum
```

---

## Adding a New Resource

### 1. Create Permission Definition

```go
// internal/core/domain/customer/permission.go
package customer

import "github.com/emoss08/trenova/pkg/permissionregistry"

type CustomerPermission struct{}

func (c *CustomerPermission) GetResourceName() string {
    return "customer"
}

func (c *CustomerPermission) GetResourceDescription() string {
    return "Customer management and billing"
}

func (c *CustomerPermission) GetSupportedOperations() []permissionregistry.OperationDefinition {
    return permissionregistry.StandardCRUDOperations()
}

func (c *CustomerPermission) GetCompositeOperations() map[string]uint32 {
    return map[string]uint32{
        "manage":    permissionregistry.CompositeBasicCRUD,
        "read_only": permissionregistry.OpRead,
    }
}

func (c *CustomerPermission) GetDefaultOperation() uint32 {
    return permissionregistry.OpRead
}

func (c *CustomerPermission) GetOperationsRequiringApproval() []uint32 {
    return []uint32{permissionregistry.OpDelete}
}

func NewCustomerPermission() permissionregistry.PermissionAware {
    return &CustomerPermission{}
}
```

### 2. Register in Bootstrap Module

```go
// internal/bootstrap/modules/permissionregistry/module.go

fx.Provide(
    fx.Annotate(
        customer.NewCustomerPermission,
        fx.ResultTags(`group:"permission_entities"`),
    ),
)
```

### 3. Update Code Generator

```go
// cmd/cli/codegen/generate_enum.go

func createRegistry() *permregistry.Registry {
    entities := []permregistry.PermissionAware{
        hazardousmaterial.NewHazardousMaterialPermission(),
        customer.NewCustomerPermission(), // ← Add here
    }

    registry := permregistry.NewRegistry(permregistry.RegistryParams{
        Entities: entities,
    })

    return registry
}
```

### 4. Regenerate Code

```bash
./build/trenova codegen generate
```

### 5. Use Generated Code

**Frontend:**

```typescript
import {
  useCustomerPermissions,
  CustomerMetadata
} from '@/types/_gen/permissions';

function CustomerPage() {
  const perms = useCustomerPermissions();

  return (
    <div>
      {perms.canCreate && <Button>Create Customer</Button>}
      {perms.canUpdate && <Button>Edit</Button>}
      {perms.hasManage && <AdminPanel />}
    </div>
  );
}

// Access metadata
console.log(CustomerMetadata.operations);
console.log(CustomerMetadata.compositeOperations);
console.log(CustomerMetadata.sensitiveFields);
```

**Backend:**

```go
import "github.com/emoss08/trenova/internal/core/domain/permission"

// Use generated enum
resource := permission.ResourceCustomer
fmt.Println(resource.String()) // "customer"

// Validate
if !resource.IsValid() {
    return errors.New("invalid resource")
}

// Get all resources
allResources := permission.AllResources()
```

---

## Generated File Structure

### TypeScript Output

```typescript
// services/ui/src/types/_gen/permissions.ts

export const PermissionOperations = {
  CREATE: 1,
  READ: 2,
  UPDATE: 4,
  DELETE: 8,
  // ...
} as const;

export const CustomerMetadata = {
  resourceName: 'customer',
  description: 'Customer management and billing',
  operations: [
    { code: 1, name: 'create', displayName: 'Create', description: '...' },
    // ...
  ],
  compositeOperations: {
    manage: 15,    // CRUD bits combined
    read_only: 2,  // Just read
  },
  sensitiveFields: ['creditCard', 'ssn'],
  readOnlyFields: ['id', 'createdAt'],
  requiredFields: ['name', 'email'],
  supportedDataScopes: ['all', 'organization'],
  defaultDataScope: 'organization',
} as const;

export const useCustomerPermissions = () => {
  const { hasPermission } = usePermissions();

  return {
    canCreate: hasPermission('customer', 'create'),
    canRead: hasPermission('customer', 'read'),
    canUpdate: hasPermission('customer', 'update'),
    canDelete: hasPermission('customer', 'delete'),
    hasManage: hasPermission('customer', 'manage'),
    hasReadOnly: hasPermission('customer', 'read_only'),
  };
};
```

### Go Output

```go
// internal/core/domain/permission/resource_gen.go

package permission

type Resource string

const (
    ResourceCustomer          Resource = "customer"
    ResourceHazardousMaterial Resource = "hazardous_material"
    // ...
)

func (r Resource) IsValid() bool {
    switch r {
    case ResourceCustomer, ResourceHazardousMaterial:
        return true
    default:
        return false
    }
}

func AllResources() []Resource {
    return []Resource{
        ResourceCustomer,
        ResourceHazardousMaterial,
        // ...
    }
}

func ResourceDescriptions() map[Resource]string {
    return map[Resource]string{
        ResourceCustomer: "Customer management and billing",
        ResourceHazardousMaterial: "Hazardous materials database...",
    }
}
```

---

## Makefile Integration

Add to `Makefile`:

```makefile
# Code generation
.PHONY: codegen
codegen: build-cli
 @echo "🔧 Generating code from permission registry..."
 ./build/trenova codegen generate

.PHONY: codegen-check
codegen-check: codegen
 @echo "🔍 Checking for uncommitted generated code..."
 @git diff --exit-code services/ui/src/types/_gen/ || \
  (echo "❌ Generated code is out of sync! Run 'make codegen' and commit." && exit 1)

.PHONY: build-cli
build-cli:
 @mkdir -p build
 go build -o build/trenova ./cmd/cli/main.go
```

Usage:

```bash
make codegen         # Generate code
make codegen-check   # Verify generated code is up to date (CI)
```

---

## CI/CD Integration

### Pre-commit Hook

```bash
#!/bin/bash
# .git/hooks/pre-commit

make codegen-check
if [ $? -ne 0 ]; then
    echo "Generated code is out of sync. Please run 'make codegen' and stage the changes."
    exit 1
fi
```

### GitHub Actions

```yaml
# .github/workflows/codegen-check.yml
name: Check Code Generation

on: [pull_request]

jobs:
  codegen-check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.23'

      - name: Check generated code
        working-directory: services/tms
        run: make codegen-check
```

---

## Best Practices

### ✅ DO

- **Run codegen after adding new resources**
- **Commit generated files** (they're part of the codebase)
- **Add CI check** to ensure generated code is up to date
- **Use Zod schemas** for entity types (not generated types)
- **Use generated hooks** in React components

### ❌ DON'T

- **Don't edit generated files manually** (they'll be overwritten)
- **Don't generate entity TypeScript types** (you have Zod schemas)
- **Don't commit with out-of-sync generated code**

---

## Troubleshooting

### Generated file conflicts with existing code

**Problem:** `resource_gen.go` conflicts with `enums.go`

**Solution:** Choose one approach:

1. **Option A:** Use generated enum - Delete/rename old `enums.go` Resource type
2. **Option B:** Keep manual enum - Don't generate enum (only generate TypeScript)

### Import cycle errors

**Problem:** Domain package can't import generated types

**Solution:** Generated types are in `permission` package, domains import `permissionregistry`

### TypeScript hook not working

**Problem:** `useCustomerPermissions` returns all false

**Solution:** Make sure:

1. Resource is registered in bootstrap module
2. Code generation includes the resource
3. Permission policies are assigned to user
4. `usePermissions()` hook is implemented

---

## Future Enhancements

- 🔜 **Zod schema hints** - Generate validation hints from field definitions
- 🔜 **GraphQL schema** - Generate GraphQL types
- 🔜 **OpenAPI spec** - Generate API documentation
- 🔜 **Permission middleware** - Auto-generate route protection
- 🔜 **Database migrations** - Generate migration hints from field definitions

---

## See Also

- [PERMISSION_QUICK_START.md](PERMISSION_QUICK_START.md) - Quick reference
- [PERMISSION_REGISTRY_USAGE.md](PERMISSION_REGISTRY_USAGE.md) - Complete usage guide
- [PERMISSION_DEPENDENCY_INJECTION.md](PERMISSION_DEPENDENCY_INJECTION.md) - DI patterns
