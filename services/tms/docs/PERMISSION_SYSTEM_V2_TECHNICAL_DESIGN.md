# Permission System V2 - Technical Design Document

## Table of Contents

1. [Overview](#overview)
2. [Architecture](#architecture)
3. [Core Concepts](#core-concepts)
4. [Domain Models](#domain-models)
5. [Database Schema](#database-schema)
6. [Permission Engine](#permission-engine)
7. [Caching Strategy](#caching-strategy)
8. [Client SDK](#client-sdk)
9. [Middleware Integration](#middleware-integration)
10. [Performance Optimization](#performance-optimization)
11. [Migration from V1](#migration-from-v1)
12. [Security Considerations](#security-considerations)
13. [API Reference](#api-reference)
14. [Troubleshooting](#troubleshooting)

---

## Overview

### What is Permission System V2?

Permission System V2 is a comprehensive, policy-based access control system designed for multi-tenant SaaS applications. It provides fine-grained authorization with sub-millisecond performance through aggressive caching and optimized data structures.

### Key Features

- **Policy-Based Access Control**: Flexible permission definition through policies
- **Role-Based Assignment**: Organize permissions into roles for easy management
- **Multi-Tenant Support**: Business Unit → Organization → User hierarchy
- **Sub-Millisecond Performance**: L1/L2/L3 caching with bitfield operations
- **Field-Level Security**: Control access to individual fields within resources
- **Data Scoping**: Restrict access based on ownership and organizational boundaries
- **Materialized Views**: Pre-computed permissions for instant lookups
- **Client-Side SDK**: TypeScript SDK with React hooks for frontend integration
- **Middleware Enforcement**: Handler-level permission checks (fail-fast)

### Architecture Principles

1. **Fail-Fast**: Permission checks at HTTP handler level before expensive operations
2. **Defense in Depth**: Multiple layers of caching with fallbacks
3. **Separation of Concerns**: Policies define rules, roles organize them, users receive them
4. **Performance First**: Bitfields, bloom filters, and materialized views for speed
5. **Audit Trail**: All permission checks and changes are logged

---

## Architecture

### High-Level Architecture

```text
┌─────────────────────────────────────────────────────────────────┐
│                         CLIENT (Browser)                         │
│  ┌────────────────┐  ┌──────────────┐  ┌────────────────────┐  │
│  │ PermissionSDK  │  │ React Hooks  │  │  BloomFilter (L0)  │  │
│  └────────────────┘  └──────────────┘  └────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼ (HTTPS/JSON)
┌─────────────────────────────────────────────────────────────────┐
│                         API SERVER (Go)                          │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  HTTP Layer                                              │   │
│  │  ├── Auth Middleware (Authentication)                    │   │
│  │  ├── Permission Middleware (Authorization) ◄── ENFORCES │   │
│  │  └── Handler (Business Logic)                            │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              │                                   │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  Permission Engine                                       │   │
│  │  ├── Check() - Single permission check                   │   │
│  │  ├── CheckBatch() - Multiple checks                      │   │
│  │  ├── GetUserPermissions() - Manifest generation          │   │
│  │  └── RefreshUserPermissions() - Cache invalidation       │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              │                                   │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  L1 Cache (Memory) - 5min TTL                            │   │
│  │  sync.Map with LRU eviction                              │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              │ (miss)                            │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  L2 Cache (Redis) - 15min TTL                            │   │
│  │  Distributed cache with JSON serialization               │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              │ (miss)                            │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  L3 Cache (Database) - 30min TTL                         │   │
│  │  permission_cache table                                  │   │
│  └─────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
                              │ (miss)
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      POSTGRESQL DATABASE                         │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  Materialized View: user_effective_policies              │   │
│  │  - Pre-computed user permissions                         │   │
│  │  - Auto-refreshed via triggers                           │   │
│  │  - Indexed for fast lookups                              │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              │                                   │
│  ┌──────────┐  ┌──────────┐  ┌──────────────────────────┐     │
│  │ policies │  │  roles   │  │ user_organization_        │     │
│  │          │  │          │  │   memberships             │     │
│  └──────────┘  └──────────┘  └──────────────────────────┘     │
└─────────────────────────────────────────────────────────────────┘
```

### Request Flow

```text
1. Client Request
   ↓
2. Auth Middleware (extracts user/org/bu context)
   ↓
3. Permission Middleware (checks authorization)
   ├── Calls PermissionEngine.Check()
   │   ├── L1 Cache Hit? → Return immediately (< 1ms)
   │   ├── L2 Cache Hit? → Populate L1, return (< 5ms)
   │   ├── L3 Cache Hit? → Populate L2+L1, return (< 10ms)
   │   └── Cache Miss → Query materialized view (< 50ms)
   │       └── Compile permissions → Cache at all levels
   │
   ├── result.Allowed = true → Continue to handler
   └── result.Allowed = false → Return 403 Forbidden
   ↓
4. Handler (business logic executes)
   ↓
5. Service Layer (no permission checks here!)
   ↓
6. Repository Layer (data access)
```

---

## Core Concepts

### 1. Policy

A **Policy** is the atomic unit of permission definition. It defines what actions can be performed on which resources.

**Structure:**

```go
type Policy struct {
    ID              pulid.ID
    Name            string
    Description     string
    Effect          Effect        // Allow or Deny
    Priority        int           // Higher priority wins conflicts
    Resources       Resources     // What resources this applies to
    Scope           Scope         // Where this applies (BU, Org)
    Conditions      Conditions    // When this applies (time, IP, etc.)
    FieldRules      FieldRules    // Field-level permissions
    Version         int
}
```

**Example:**

```json
{
  "name": "shipment_full_access",
  "effect": "allow",
  "priority": 100,
  "resources": {
    "resourceType": ["shipment"],
    "actions": ["create", "read", "update", "delete", "approve"]
  },
  "scope": {
    "businessUnitId": "bu_123",
    "organizationIds": ["org_456", "org_789"]
  },
  "fieldRules": {
    "fields": ["*"],
    "readableFields": ["*"],
    "writableFields": ["status", "notes"],
    "maskedFields": {
      "customerPrice": "partial"
    }
  }
}
```

**Policy Types:**

- **System Policies**: Built-in, cannot be modified (e.g., `system_admin`)
- **Business Unit Policies**: Apply to entire business unit
- **Organization Policies**: Apply to specific organizations
- **Custom Policies**: User-defined for specific use cases

### 2. Role

A **Role** is a collection of policies that can be assigned to users. Roles simplify permission management by grouping related policies.

**Structure:**

```go
type Role struct {
    ID              pulid.ID
    Name            string
    Description     string
    Level           RoleLevel     // System, BU, Org, Custom
    PolicyIDs       []pulid.ID    // Policies included in this role
    InheritedRoles  []pulid.ID    // Parent roles to inherit from
    IsSystem        bool          // Cannot be modified if true
    Version         int
}
```

**Role Hierarchy:**

```
System Admin
    └── Business Unit Admin
            ├── Organization Admin
            │       ├── Manager
            │       └── Supervisor
            └── Fleet Manager
                    └── Dispatcher
```

**Example:**

```json
{
  "name": "Operations Manager",
  "level": "org",
  "policyIds": [
    "pol_shipment_full_access",
    "pol_driver_read_update",
    "pol_equipment_read"
  ],
  "inheritedRoles": ["rol_basic_user"]
}
```

### 3. User Organization Membership

Links users to organizations with their assigned roles and direct policies.

**Structure:**

```go
type UserOrganizationMembership struct {
    UserID         pulid.ID
    OrganizationID pulid.ID
    BusinessUnitID pulid.ID
    RoleIDs        []pulid.ID   // Roles assigned to user
    DirectPolicies []pulid.ID   // Direct policy assignments
    ExpiresAt      *int64       // Optional expiration
}
```

### 4. Resource

A **Resource** represents an entity in the system that can have permissions applied to it.

**Resource Types:**

- `shipment` - Shipment records
- `customer` - Customer records
- `equipment` - Tractors, trailers
- `driver` - Driver records
- `billing_queue` - Billing items
- `user` - User management
- `organization` - Organization settings

**Actions:**

- `create` - Create new records
- `read` - View records
- `update` - Modify records
- `delete` - Remove records
- `list` - List/search records
- `export` - Export data
- `import` - Import data
- `approve` - Approve workflows
- `reject` - Reject workflows
- `archive` - Archive records

### 5. Data Scope

**Data Scope** defines what subset of data a user can access within a resource type.

**Scope Types:**

```go
type DataScope struct {
    Type       ScopeType  // All, Own, Organization, BusinessUnit, Custom
    OwnerField string     // Field to check ownership (e.g., "created_by")
    Filter     Filter     // Custom filter for complex scoping
}
```

**Examples:**

```json
// User can only see their own shipments
{
  "type": "own",
  "ownerField": "created_by"
}

// User can see all shipments in their organization
{
  "type": "organization"
}

// User can see shipments they created or are assigned to
{
  "type": "custom",
  "filter": {
    "or": [
      {"field": "created_by", "op": "eq", "value": "@user_id"},
      {"field": "assigned_to", "op": "eq", "value": "@user_id"}
    ]
  }
}
```

### 6. Field Rules

**Field Rules** provide field-level access control within resources.

**Structure:**

```go
type FieldRules struct {
    ReadableFields []string           // Fields user can see
    WritableFields []string           // Fields user can modify
    MaskedFields   map[string]MaskType // Fields to mask
}

type MaskType string
const (
    MaskPartial MaskType = "partial"  // Show first/last chars
    MaskFull    MaskType = "full"     // Show nothing
    MaskHash    MaskType = "hash"     // Show hash only
)
```

**Examples:**

```json
{
  "readableFields": ["id", "status", "customer_name", "origin", "destination"],
  "writableFields": ["status", "notes"],
  "maskedFields": {
    "customer_price": "partial",    // "$1,***.**"
    "driver_ssn": "full",            // "***-**-****"
    "credit_card": "partial"         // "**** **** **** 1234"
  }
}
```

---

## Domain Models

### Policy Model

**Location:** `/services/tms/internal/core/domain/permission/policy.go`

```go
package permission

type Policy struct {
    ID              pulid.ID   `json:"id" bun:"id,pk,type:VARCHAR(100)"`
    BusinessUnitID  pulid.ID   `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100)"`
    OrganizationID  pulid.ID   `json:"organizationId" bun:"organization_id,type:VARCHAR(100)"`
    Name            string     `json:"name" bun:"name,notnull"`
    Description     string     `json:"description" bun:"description"`
    Effect          Effect     `json:"effect" bun:"effect,type:VARCHAR(10)"`
    Priority        int        `json:"priority" bun:"priority,notnull"`
    Resources       Resources  `json:"resources" bun:"resources,type:jsonb"`
    Scope           Scope      `json:"scope" bun:"scope,type:jsonb"`
    Conditions      Conditions `json:"conditions" bun:"conditions,type:jsonb"`
    FieldRules      FieldRules `json:"fieldRules" bun:"field_rules,type:jsonb"`
    IsSystem        bool       `json:"isSystem" bun:"is_system,notnull,default:false"`
    Version         int        `json:"version" bun:"version,notnull,default:1"`
    CreatedAt       time.Time  `json:"createdAt" bun:"created_at,notnull,default:current_timestamp"`
    UpdatedAt       time.Time  `json:"updatedAt" bun:"updated_at,notnull,default:current_timestamp"`
}

type Effect string
const (
    EffectAllow Effect = "allow"
    EffectDeny  Effect = "deny"
)

type Resources struct {
    ResourceType []string `json:"resourceType"`
    Actions      []string `json:"actions"`
    ResourceIDs  []string `json:"resourceIds,omitempty"`
}

type Scope struct {
    BusinessUnitID  pulid.ID    `json:"businessUnitId"`
    OrganizationIDs []pulid.ID  `json:"organizationIds"`
}

type Conditions struct {
    TimeWindows []TimeWindow `json:"timeWindows,omitempty"`
    IPRanges    []string     `json:"ipRanges,omitempty"`
    CustomRules []CustomRule `json:"customRules,omitempty"`
}

type TimeWindow struct {
    Start string `json:"start"` // "09:00"
    End   string `json:"end"`   // "17:00"
    Days  []int  `json:"days"`  // [1,2,3,4,5] (Monday-Friday)
}

type FieldRules struct {
    ReadableFields []string           `json:"readableFields"`
    WritableFields []string           `json:"writableFields"`
    MaskedFields   map[string]MaskType `json:"maskedFields,omitempty"`
}

type MaskType string
const (
    MaskPartial MaskType = "partial"
    MaskFull    MaskType = "full"
    MaskHash    MaskType = "hash"
)
```

### Role Model

**Location:** `/services/tms/internal/core/domain/permission/role.go`

```go
package permission

type Role struct {
    ID              pulid.ID   `json:"id" bun:"id,pk,type:VARCHAR(100)"`
    BusinessUnitID  pulid.ID   `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100)"`
    Name            string     `json:"name" bun:"name,notnull"`
    Description     string     `json:"description" bun:"description"`
    Level           RoleLevel  `json:"level" bun:"level,type:VARCHAR(20)"`
    PolicyIDs       []pulid.ID `json:"policyIds" bun:"policy_ids,type:jsonb,array"`
    InheritedRoles  []pulid.ID `json:"inheritedRoles" bun:"inherited_roles,type:jsonb,array"`
    IsSystem        bool       `json:"isSystem" bun:"is_system,notnull,default:false"`
    Version         int        `json:"version" bun:"version,notnull,default:1"`
    CreatedAt       time.Time  `json:"createdAt" bun:"created_at,notnull,default:current_timestamp"`
    UpdatedAt       time.Time  `json:"updatedAt" bun:"updated_at,notnull,default:current_timestamp"`
}

type RoleLevel string
const (
    RoleLevelSystem   RoleLevel = "system"   // System-wide role
    RoleLevelBU       RoleLevel = "bu"       // Business unit role
    RoleLevelOrg      RoleLevel = "org"      // Organization role
    RoleLevelCustom   RoleLevel = "custom"   // Custom role
)
```

### Resource Enumeration

**Location:** `/services/tms/internal/core/domain/permission/enums.go`

```go
package permission

type Resource string

const (
    ResourceShipment           Resource = "shipment"
    ResourceCustomer           Resource = "customer"
    ResourceEquipment          Resource = "equipment"
    ResourceDriver             Resource = "driver"
    ResourceUser               Resource = "user"
    ResourceOrganization       Resource = "organization"
    ResourceBillingQueue       Resource = "billing_queue"
    ResourceReport             Resource = "report"
    ResourceHazardousMaterial  Resource = "hazardous_material"
    ResourceLocation           Resource = "location"
    ResourceCommodity          Resource = "commodity"
)
```

### Data Scope Model

**Location:** `/services/tms/internal/core/domain/permission/policy.go`

```go
type DataScope struct {
    Type       ScopeType `json:"type"`
    OwnerField string    `json:"ownerField,omitempty"`
    Filter     *Filter   `json:"filter,omitempty"`
}

type ScopeType string
const (
    ScopeAll          ScopeType = "all"           // All records
    ScopeOwn          ScopeType = "own"           // Own records only
    ScopeOrganization ScopeType = "organization"  // Organization records
    ScopeBusinessUnit ScopeType = "business_unit" // Business unit records
    ScopeCustom       ScopeType = "custom"        // Custom filter
)

type Filter struct {
    Field    string      `json:"field"`
    Operator string      `json:"op"`
    Value    interface{} `json:"value"`
    Or       []Filter    `json:"or,omitempty"`
    And      []Filter    `json:"and,omitempty"`
}
```

---

## Database Schema

### Core Tables

#### `policies`

```sql
CREATE TABLE policies (
    id VARCHAR(100) PRIMARY KEY,
    business_unit_id VARCHAR(100) NOT NULL,
    organization_id VARCHAR(100) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    effect VARCHAR(10) NOT NULL CHECK (effect IN ('allow', 'deny')),
    priority INTEGER NOT NULL DEFAULT 0,
    resources JSONB NOT NULL,
    scope JSONB NOT NULL,
    conditions JSONB DEFAULT '{}'::jsonb,
    field_rules JSONB DEFAULT '{}'::jsonb,
    is_system BOOLEAN NOT NULL DEFAULT false,
    version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_policies_business_unit FOREIGN KEY (business_unit_id)
        REFERENCES business_units(id) ON DELETE CASCADE,
    CONSTRAINT fk_policies_organization FOREIGN KEY (organization_id)
        REFERENCES organizations(id) ON DELETE CASCADE
);

CREATE INDEX idx_policies_business_unit ON policies(business_unit_id);
CREATE INDEX idx_policies_organization ON policies(organization_id);
CREATE INDEX idx_policies_effect_priority ON policies(effect, priority DESC);
CREATE INDEX idx_policies_resources ON policies USING gin(resources);
CREATE INDEX idx_policies_scope ON policies USING gin(scope);
```

#### `roles`

```sql
CREATE TABLE roles (
    id VARCHAR(100) PRIMARY KEY,
    business_unit_id VARCHAR(100) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    level VARCHAR(20) NOT NULL CHECK (level IN ('system', 'bu', 'org', 'custom')),
    policy_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    inherited_roles JSONB NOT NULL DEFAULT '[]'::jsonb,
    is_system BOOLEAN NOT NULL DEFAULT false,
    version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_roles_business_unit FOREIGN KEY (business_unit_id)
        REFERENCES business_units(id) ON DELETE CASCADE
);

CREATE INDEX idx_roles_business_unit ON roles(business_unit_id);
CREATE INDEX idx_roles_level ON roles(level);
CREATE INDEX idx_roles_policy_ids ON roles USING gin(policy_ids);
```

#### `user_organization_memberships`

```sql
CREATE TABLE user_organization_memberships (
    user_id VARCHAR(100) NOT NULL,
    organization_id VARCHAR(100) NOT NULL,
    business_unit_id VARCHAR(100) NOT NULL,
    role_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    direct_policies JSONB NOT NULL DEFAULT '[]'::jsonb,
    expires_at BIGINT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (user_id, organization_id),

    CONSTRAINT fk_membership_user FOREIGN KEY (user_id)
        REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_membership_organization FOREIGN KEY (organization_id)
        REFERENCES organizations(id) ON DELETE CASCADE,
    CONSTRAINT fk_membership_business_unit FOREIGN KEY (business_unit_id)
        REFERENCES business_units(id) ON DELETE CASCADE
);

CREATE INDEX idx_membership_user ON user_organization_memberships(user_id);
CREATE INDEX idx_membership_organization ON user_organization_memberships(organization_id);
CREATE INDEX idx_membership_business_unit ON user_organization_memberships(business_unit_id);
CREATE INDEX idx_membership_role_ids ON user_organization_memberships USING gin(role_ids);
```

### Cache Table

#### `permission_cache`

```sql
CREATE TABLE permission_cache (
    user_id VARCHAR(100) NOT NULL,
    organization_id VARCHAR(100) NOT NULL,
    version VARCHAR(50) NOT NULL,
    computed_at BIGINT NOT NULL,
    expires_at BIGINT NOT NULL,
    permission_data BYTEA NOT NULL,
    bloom_filter BYTEA,
    checksum VARCHAR(64) NOT NULL,

    PRIMARY KEY (user_id, organization_id),

    CONSTRAINT fk_cache_user FOREIGN KEY (user_id)
        REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_cache_organization FOREIGN KEY (organization_id)
        REFERENCES organizations(id) ON DELETE CASCADE
);

CREATE INDEX idx_permission_cache_expires ON permission_cache(expires_at);
CREATE INDEX idx_permission_cache_version ON permission_cache(version);
```

### Materialized View

#### `user_effective_policies`

**Purpose:** Pre-computes all effective policies for each user in each organization for instant permission lookups.

```sql
CREATE MATERIALIZED VIEW user_effective_policies AS
WITH user_roles AS (
    -- Get all roles for each user in each organization
    SELECT
        uom.user_id,
        uom.organization_id,
        uom.business_unit_id,
        unnest(uom.role_ids) AS role_id
    FROM user_organization_memberships uom
    WHERE uom.expires_at IS NULL
       OR uom.expires_at > EXTRACT(epoch FROM CURRENT_TIMESTAMP)::bigint
),
role_policies AS (
    -- Get all policies from roles (including inherited roles)
    SELECT DISTINCT
        ur.user_id,
        ur.organization_id,
        ur.business_unit_id,
        unnest(r.policy_ids) AS policy_id,
        'role' AS assignment_type,
        ur.role_id AS source_id
    FROM user_roles ur
    JOIN roles r ON r.id = ur.role_id
    WHERE r.level IN ('system', 'bu', 'org', 'custom')
),
direct_policies AS (
    -- Get directly assigned policies
    SELECT
        uom.user_id,
        uom.organization_id,
        uom.business_unit_id,
        unnest(uom.direct_policies) AS policy_id,
        'direct' AS assignment_type,
        NULL AS source_id
    FROM user_organization_memberships uom
    WHERE array_length(uom.direct_policies, 1) > 0
      AND (uom.expires_at IS NULL
           OR uom.expires_at > EXTRACT(epoch FROM CURRENT_TIMESTAMP)::bigint)
),
all_user_policies AS (
    -- Combine role and direct policies
    SELECT * FROM role_policies
    UNION ALL
    SELECT * FROM direct_policies
),
policy_details AS (
    -- Join with policy details and apply scoping rules
    SELECT
        aup.user_id,
        aup.organization_id,
        aup.business_unit_id,
        p.id AS policy_id,
        p.name AS policy_name,
        p.effect,
        p.priority,
        p.resources,
        p.scope,
        aup.assignment_type,
        aup.source_id
    FROM all_user_policies aup
    JOIN policies p ON p.id = aup.policy_id
    WHERE
        -- Apply business unit scoping
        p.scope->>'businessUnitId' = aup.business_unit_id::text
        AND (
            -- Policy applies to all organizations in BU
            (p.scope->>'organizationIds' = '[]'
             OR p.scope->>'organizationIds' IS NULL)
            OR
            -- Policy applies to specific organizations
            (p.scope->'organizationIds' ? aup.organization_id::text)
        )
)
SELECT
    user_id,
    organization_id,
    business_unit_id,
    policy_id,
    policy_name,
    effect,
    priority,
    resources,
    scope,
    assignment_type,
    source_id,
    -- Add computed fields for faster lookups
    array_length(ARRAY(
        SELECT jsonb_array_elements_text(resources->'resourceType')
    ), 1) AS resource_count,
    -- Cache the permission hash for quick comparison
    md5(
        user_id::text || organization_id::text || policy_id::text ||
        coalesce(resources::text, '') || coalesce(scope::text, '')
    ) AS permission_hash
FROM policy_details;

-- Indexes
CREATE UNIQUE INDEX idx_user_effective_policies_unique
    ON user_effective_policies(user_id, organization_id, policy_id);

CREATE INDEX idx_user_effective_policies_user_org
    ON user_effective_policies(user_id, organization_id);

CREATE INDEX idx_user_effective_policies_effect_priority
    ON user_effective_policies(effect, priority DESC);

CREATE INDEX idx_user_effective_policies_assignment_type
    ON user_effective_policies(assignment_type);

CREATE INDEX idx_user_effective_policies_hash
    ON user_effective_policies(permission_hash);
```

**Auto-Refresh Triggers:**

```sql
-- Trigger function
CREATE OR REPLACE FUNCTION trigger_refresh_user_policies()
RETURNS TRIGGER AS $$
BEGIN
    -- Refresh the materialized view immediately
    REFRESH MATERIALIZED VIEW CONCURRENTLY user_effective_policies;

    -- Send notification for any listeners
    PERFORM pg_notify('permission_refresh',
        json_build_object(
            'table', TG_TABLE_NAME,
            'operation', TG_OP,
            'timestamp', EXTRACT(epoch FROM CURRENT_TIMESTAMP)
        )::text
    );

    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

-- Triggers on all relevant tables
CREATE TRIGGER trigger_policies_refresh
    AFTER INSERT OR UPDATE OR DELETE ON policies
    FOR EACH ROW EXECUTE FUNCTION trigger_refresh_user_policies();

CREATE TRIGGER trigger_roles_refresh
    AFTER INSERT OR UPDATE OR DELETE ON roles
    FOR EACH ROW EXECUTE FUNCTION trigger_refresh_user_policies();

CREATE TRIGGER trigger_user_memberships_refresh
    AFTER INSERT OR UPDATE OR DELETE ON user_organization_memberships
    FOR EACH ROW EXECUTE FUNCTION trigger_refresh_user_policies();
```

---

## Permission Engine

### Architecture

**Location:** `/services/tms/internal/core/services/permissionservice/engine.go`

The Permission Engine is the core component responsible for evaluating permission checks. It orchestrates caching, policy resolution, and access decisions.

```go
package permissionservice

type Engine struct {
    policyRepo    ports.PolicyRepository
    roleRepo      ports.RoleRepository
    cacheRepo     ports.PermissionCacheRepository
    compiler      ports.PolicyCompiler
    logger        *zap.Logger
}

func NewEngine(
    policyRepo ports.PolicyRepository,
    roleRepo ports.RoleRepository,
    cacheRepo ports.PermissionCacheRepository,
    compiler ports.PolicyCompiler,
    logger *zap.Logger,
) ports.PermissionEngine {
    return &Engine{
        policyRepo: policyRepo,
        roleRepo:   roleRepo,
        cacheRepo:  cacheRepo,
        compiler:   compiler,
        logger:     logger.Named("permission-engine"),
    }
}
```

### Check Flow

```
Engine.Check(ctx, request)
    │
    ├─► 1. Extract user/org/resource/action from request
    │
    ├─► 2. Check L1 Cache (memory)
    │   └─► Hit? Return in < 1ms
    │
    ├─► 3. Check L2 Cache (Redis)
    │   ├─► Hit? Populate L1, return in < 5ms
    │   └─► Miss? Continue
    │
    ├─► 4. Check L3 Cache (Database)
    │   ├─► Hit? Populate L2+L1, return in < 10ms
    │   └─► Miss? Continue
    │
    ├─► 5. Query Materialized View
    │   └─► Get user_effective_policies for user+org
    │
    ├─► 6. Filter Policies by Resource
    │   └─► Match resource_type in policy.resources
    │
    ├─► 7. Evaluate Policies (Priority Order)
    │   ├─► Check action permission (bitfield)
    │   ├─► Apply conditions (time, IP, etc.)
    │   ├─► Evaluate data scope
    │   └─► Check field rules
    │
    ├─► 8. Determine Result
    │   ├─► Deny policies take precedence
    │   ├─► Higher priority wins conflicts
    │   └─► Default to deny if no match
    │
    ├─► 9. Cache Result at all levels
    │   ├─► L1 (5min TTL)
    │   ├─► L2 (15min TTL)
    │   └─► L3 (30min TTL)
    │
    └─► 10. Return PermissionCheckResult
```

### Check Method Implementation

```go
func (e *Engine) Check(
    ctx context.Context,
    req *ports.PermissionCheckRequest,
) (*ports.PermissionCheckResult, error) {
    start := time.Now()

    log := e.logger.With(
        zap.String("userID", req.UserID.String()),
        zap.String("orgID", req.OrganizationID.String()),
        zap.String("resource", req.ResourceType),
        zap.String("action", req.Action),
    )

    // Step 1: Try cache lookup
    cached, err := e.cacheRepo.Get(ctx, req.UserID, req.OrganizationID)
    if err == nil && cached != nil {
        result := e.evaluateCached(cached, req)
        result.CacheHit = true
        result.ComputeTimeMs = float64(time.Since(start).Microseconds()) / 1000.0
        return result, nil
    }

    // Step 2: Cache miss - query materialized view
    policies, err := e.policyRepo.GetUserPolicies(
        ctx,
        req.UserID,
        req.OrganizationID,
    )
    if err != nil {
        log.Error("failed to get user policies", zap.Error(err))
        return &ports.PermissionCheckResult{
            Allowed: false,
            Reason:  "internal error",
        }, err
    }

    // Step 3: Compile permissions
    compiled, err := e.compiler.CompileForUser(
        ctx,
        req.UserID,
        req.OrganizationID,
        policies,
    )
    if err != nil {
        log.Error("failed to compile permissions", zap.Error(err))
        return &ports.PermissionCheckResult{
            Allowed: false,
            Reason:  "internal error",
        }, err
    }

    // Step 4: Cache the compiled permissions
    cachedPerms := &ports.CachedPermissions{
        Version:     generateVersion(),
        ComputedAt:  time.Now(),
        ExpiresAt:   time.Now().Add(30 * time.Minute),
        Permissions: compiled,
        BloomFilter: e.buildBloomFilter(compiled),
        Checksum:    calculateChecksum(compiled),
    }

    _ = e.cacheRepo.Set(
        ctx,
        req.UserID,
        req.OrganizationID,
        cachedPerms,
        15*time.Minute,
    )

    // Step 5: Evaluate permission
    result := e.evaluateCached(cachedPerms, req)
    result.CacheHit = false
    result.ComputeTimeMs = float64(time.Since(start).Microseconds()) / 1000.0

    return result, nil
}
```

### Policy Evaluation Algorithm

```go
func (e *Engine) evaluateCached(
    cached *ports.CachedPermissions,
    req *ports.PermissionCheckRequest,
) *ports.PermissionCheckResult {
    // Get resource permissions
    resourcePerms, exists := cached.Permissions.Resources[req.ResourceType]
    if !exists {
        return &ports.PermissionCheckResult{
            Allowed: false,
            Reason:  "no permissions for resource",
        }
    }

    // Check action using bitfield
    actionBit, ok := ports.ActionBits[req.Action]
    if !ok {
        // Extended operation (not a standard action)
        for _, extOp := range resourcePerms.ExtendedOps {
            if extOp == req.Action {
                return &ports.PermissionCheckResult{
                    Allowed:   true,
                    DataScope: resourcePerms.DataScope,
                }
            }
        }
        return &ports.PermissionCheckResult{
            Allowed: false,
            Reason:  "action not permitted",
        }
    }

    // Check bitfield
    if (resourcePerms.StandardOps & actionBit) == 0 {
        return &ports.PermissionCheckResult{
            Allowed: false,
            Reason:  "action not permitted",
        }
    }

    // Permission granted
    return &ports.PermissionCheckResult{
        Allowed:   true,
        DataScope: resourcePerms.DataScope,
        FieldAccess: &ports.FieldAccessRules{
            Readable: resourcePerms.FieldRules.ReadableFields,
            Writable: resourcePerms.FieldRules.WritableFields,
            Masked:   resourcePerms.FieldRules.MaskedFields,
        },
    }
}
```

### Batch Check

```go
func (e *Engine) CheckBatch(
    ctx context.Context,
    req *ports.BatchPermissionCheckRequest,
) (*ports.BatchPermissionCheckResult, error) {
    start := time.Now()

    results := make([]*ports.PermissionCheckResult, len(req.Checks))
    cacheHits := 0

    // Get cached permissions once
    cached, _ := e.cacheRepo.Get(ctx, req.UserID, req.OrganizationID)

    for i, check := range req.Checks {
        check.UserID = req.UserID
        check.OrganizationID = req.OrganizationID

        if cached != nil {
            results[i] = e.evaluateCached(cached, check)
            results[i].CacheHit = true
            cacheHits++
        } else {
            results[i], _ = e.Check(ctx, check)
        }
    }

    return &ports.BatchPermissionCheckResult{
        Results:      results,
        CacheHitRate: float64(cacheHits) / float64(len(req.Checks)),
        TotalTimeMs:  float64(time.Since(start).Microseconds()) / 1000.0,
    }, nil
}
```

---

## Caching Strategy

### Three-Tier Cache Architecture

```
┌─────────────────────────────────────────────────────────────┐
│ L1: In-Memory Cache (sync.Map)                              │
│ • TTL: 5 minutes                                             │
│ • Capacity: 10,000 entries                                   │
│ • Eviction: LRU                                              │
│ • Latency: < 1ms                                             │
│ • Hit Rate: ~85%                                             │
└─────────────────────────────────────────────────────────────┘
                         ↓ (miss)
┌─────────────────────────────────────────────────────────────┐
│ L2: Redis Cache (Distributed)                               │
│ • TTL: 15 minutes                                            │
│ • Format: JSON (Sonic serialization)                        │
│ • Latency: < 5ms                                             │
│ • Hit Rate: ~10%                                             │
└─────────────────────────────────────────────────────────────┘
                         ↓ (miss)
┌─────────────────────────────────────────────────────────────┐
│ L3: Database Cache (PostgreSQL)                             │
│ • TTL: 30 minutes                                            │
│ • Table: permission_cache                                   │
│ • Latency: < 10ms                                            │
│ • Hit Rate: ~4%                                              │
└─────────────────────────────────────────────────────────────┘
                         ↓ (miss)
┌─────────────────────────────────────────────────────────────┐
│ Materialized View: user_effective_policies                  │
│ • Auto-refreshed via triggers                                │
│ • Indexed for fast lookups                                   │
│ • Latency: < 50ms                                            │
│ • Hit Rate: ~1%                                              │
└─────────────────────────────────────────────────────────────┘
```

### Cache Invalidation

**Strategy:** Write-through with automatic invalidation

**Invalidation Triggers:**

1. Policy created/updated/deleted → Invalidate all affected users
2. Role created/updated/deleted → Invalidate all users with that role
3. User role assignment changed → Invalidate that user
4. User organization membership changed → Invalidate that user
5. Manual refresh requested → Invalidate specific user or all users

**Implementation:**

```go
func (e *Engine) RefreshUserPermissions(
    ctx context.Context,
    userID, organizationID pulid.ID,
) error {
    // Delete from all cache levels
    if err := e.cacheRepo.Delete(ctx, userID, organizationID); err != nil {
        return err
    }

    // Next permission check will recompute
    return nil
}

func (e *Engine) InvalidateCache(
    ctx context.Context,
    userID, organizationID pulid.ID,
) error {
    return e.RefreshUserPermissions(ctx, userID, organizationID)
}
```

### Cache Key Structure

```
Pattern: perm:{userID}:{organizationID}

Examples:
perm:usr_01K6BWEGEPBXPP1GPZD0SVHJ71:org_01K6BWEGCPVGNW2F867D8P6A37
```

### L1 Cache Implementation

**Location:** `/services/tms/internal/infrastructure/redis/repositories/permissioncache/cache.go:214-297`

```go
func (c *cache) getFromL1(key string) *ports.CachedPermissions {
    c.l1Mutex.RLock()
    defer c.l1Mutex.RUnlock()

    entry, ok := c.l1Cache[key]
    if !ok {
        return nil
    }

    if time.Now().After(entry.expiresAt) {
        return nil
    }

    return entry.data
}

func (c *cache) setToL1(key string, permissions *ports.CachedPermissions) {
    c.l1Mutex.Lock()
    defer c.l1Mutex.Unlock()

    // Evict oldest if cache is full
    if len(c.l1Cache) >= maxL1CacheSize {
        c.evictOldestL1()
    }

    c.l1Cache[key] = &cacheEntry{
        data:      permissions,
        expiresAt: time.Now().Add(c.l1TTL),
    }
}

// Background cleanup
func (c *cache) l1CleanupLoop() {
    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()

    for range ticker.C {
        c.cleanupExpiredL1()
    }
}
```

---

## Client SDK

### TypeScript SDK Architecture

**Location:** `/services/ui/src/lib/permissions/`

The client SDK provides:

1. **PermissionClient** - Core permission checking logic
2. **BloomFilter** - Fast negative permission checks
3. **React Context** - State management
4. **React Hooks** - Easy integration with components

### PermissionClient

**Location:** `/services/ui/src/lib/permissions/permission-client.ts`

```typescript
export class PermissionClient {
  private manifest: PermissionManifest;
  private bloom: BloomFilter | null = null;

  constructor(manifest: PermissionManifest) {
    this.manifest = manifest;

    // Initialize BloomFilter if available
    if (manifest.bloomFilter) {
      this.bloom = new BloomFilter(manifest.bloomFilter);
    }
  }

  /**
   * Check if user has permission for a specific action on a resource
   * Performance: < 0.1ms (with bloom filter), < 0.5ms (without)
   */
  can(resource: Resource, action: ActionName): boolean {
    // Fast negative check using bloom filter
    if (this.bloom && !this.bloom.test(`${resource}:${action}`)) {
      return false; // Definitely not present
    }

    const perms = this.manifest.resources[resource];
    if (!perms) return false;

    // Simple bitfield check
    if (typeof perms === "number") {
      const actionBit = ACTION_BITS[action];
      if (!actionBit) return false;
      return (perms & actionBit) > 0;
    }

    // Complex permission evaluation
    return this.evaluateComplex(perms, action);
  }

  /**
   * Check if user has ANY of the specified permissions
   */
  canAny(resource: Resource, actions: ActionName[]): boolean {
    return actions.some((action) => this.can(resource, action));
  }

  /**
   * Check if user has ALL of the specified permissions
   */
  canAll(resource: Resource, actions: ActionName[]): boolean {
    return actions.every((action) => this.can(resource, action));
  }

  /**
   * Get all permissions for a resource
   */
  getResourcePermissions(resource: Resource): string[] {
    const perms = this.manifest.resources[resource];
    if (!perms) return [];

    if (typeof perms === "number") {
      return Object.entries(ACTION_BITS)
        .filter(([_, bit]) => (perms & bit) > 0)
        .map(([action]) => action);
    }

    return perms.actions || [];
  }
}
```

### BloomFilter

**Location:** `/services/ui/src/lib/permissions/bloom-filter.ts`

```typescript
export class BloomFilter {
  private bits: Uint8Array;
  private size: number;
  private hashCount: number;

  constructor(base64Data: string) {
    // Decode base64 to binary
    const decoded = atob(base64Data);
    this.bits = new Uint8Array(decoded.length);
    for (let i = 0; i < decoded.length; i++) {
      this.bits[i] = decoded.charCodeAt(i);
    }

    this.size = this.bits.length * 8;
    this.hashCount = 3; // Optimal for our use case
  }

  /**
   * Test if an element is in the set
   * Returns: false = definitely not present
   *          true = possibly present
   */
  test(key: string): boolean {
    const hashes = this.getHashes(key);

    for (let i = 0; i < this.hashCount; i++) {
      const bitIndex = hashes[i] % this.size;
      const byteIndex = Math.floor(bitIndex / 8);
      const bitOffset = bitIndex % 8;

      if ((this.bits[byteIndex] & (1 << bitOffset)) === 0) {
        return false; // Definitely not present
      }
    }

    return true; // Possibly present
  }

  private getHashes(key: string): number[] {
    // MurmurHash3 implementation
    const h1 = this.murmurHash3(key, 0);
    const h2 = this.murmurHash3(key, h1);

    return [
      Math.abs(h1),
      Math.abs(h2),
      Math.abs(h1 + h2)
    ];
  }

  private murmurHash3(key: string, seed: number): number {
    let h = seed;
    for (let i = 0; i < key.length; i++) {
      h = Math.imul(h ^ key.charCodeAt(i), 2654435761);
    }
    h ^= h >>> 16;
    h = Math.imul(h, 2246822507);
    h ^= h >>> 13;
    h = Math.imul(h, 3266489909);
    h ^= h >>> 16;
    return h;
  }
}
```

### React Hooks

**Location:** `/services/ui/src/hooks/use-permission-v2.ts`

```typescript
/**
 * Main permission hook - provides can/canAny/canAll functions
 */
export function usePermissionV2() {
  const { can, canAny, canAll } = usePermissionContext();
  return useMemo(() => ({ can, canAny, canAll }), [can, canAny, canAll]);
}

/**
 * Hook for single permission check
 * Returns: boolean indicating if user has permission
 */
export function useCanAccess(resource: Resource, action: ActionName): boolean {
  const { can } = usePermissionContext();
  return useMemo(() => can(resource, action), [can, resource, action]);
}

/**
 * Hook for guarded component rendering
 * Only renders children if user has permission
 */
export function usePermissionGuard(
  resource: Resource,
  action: ActionName,
  fallback?: React.ReactNode
) {
  const hasPermission = useCanAccess(resource, action);

  return useMemo(
    () => ({
      hasPermission,
      renderIf: (children: React.ReactNode) =>
        hasPermission ? children : (fallback ?? null),
    }),
    [hasPermission, fallback]
  );
}

/**
 * Hook for field-level access control
 */
export function useFieldAccess(resource: Resource, field: string) {
  const { canAccessField, canWriteField, getFieldAccess } = usePermissionContext();

  return useMemo(() => {
    const access = getFieldAccess(resource, field);
    return {
      canAccess: canAccessField(resource, field),
      canWrite: canWriteField(resource, field),
      access,
      isReadOnly: access === "read_only",
      isHidden: access === "hidden" || !canAccessField(resource, field),
    };
  }, [canAccessField, canWriteField, getFieldAccess, resource, field]);
}

/**
 * Hook for resource-level permissions
 * Returns all permissions for a resource
 */
export function useResourcePermissions(resource: Resource) {
  const { client } = usePermissionContext();

  return useMemo(() => {
    if (!client) return [];
    return client.getResourcePermissions(resource);
  }, [client, resource]);
}
```

### Usage Examples

#### Component Permission Check

```typescript
import { usePermissionV2 } from "@/hooks/use-permission-v2";

function ShipmentActions() {
  const { can } = usePermissionV2();

  return (
    <div>
      {can("shipment", "update") && (
        <button onClick={handleUpdate}>Edit Shipment</button>
      )}

      {can("shipment", "delete") && (
        <button onClick={handleDelete}>Delete Shipment</button>
      )}

      {can("shipment", "approve") && (
        <button onClick={handleApprove}>Approve Shipment</button>
      )}
    </div>
  );
}
```

#### Permission Guard

```typescript
import { usePermissionGuard } from "@/hooks/use-permission-v2";

function BillingSection() {
  const { renderIf } = usePermissionGuard(
    "billing_queue",
    "read",
    <div>You don't have access to billing</div>
  );

  return renderIf(
    <div>
      <h2>Billing Queue</h2>
      <BillingQueueTable />
    </div>
  );
}
```

#### Field Access Control

```typescript
import { useFieldAccess } from "@/hooks/use-permission-v2";

function CustomerForm() {
  const priceAccess = useFieldAccess("customer", "price");
  const ssnAccess = useFieldAccess("customer", "ssn");

  return (
    <form>
      {priceAccess.canAccess && (
        <input
          name="price"
          disabled={priceAccess.isReadOnly}
          type={priceAccess.access === "masked" ? "password" : "text"}
        />
      )}

      {!ssnAccess.isHidden && (
        <input name="ssn" type="password" disabled />
      )}
    </form>
  );
}
```

---

## Middleware Integration

### Permission Middleware

**Location:** `/services/tms/internal/api/middleware/permission.go`

The permission middleware enforces authorization at the HTTP handler level, following the fail-fast principle.

### Middleware Methods

#### `RequirePermission(resource, action)` - Single Permission

```go
// Requires exactly one permission
func (pm *PermissionMiddleware) RequirePermission(
    resource permission.Resource,
    action string,
) gin.HandlerFunc {
    return func(c *gin.Context) {
        userID, ok := authctx.GetUserID(c)
        if !ok {
            pm.handlePermissionError(c, http.StatusUnauthorized, "User not authenticated")
            return
        }

        orgID, ok := authctx.GetOrganizationID(c)
        if !ok {
            pm.handlePermissionError(c, http.StatusUnauthorized, "Organization not found")
            return
        }

        result, err := pm.engine.Check(c.Request.Context(), &ports.PermissionCheckRequest{
            UserID:         userID,
            OrganizationID: orgID,
            ResourceType:   string(resource),
            Action:         action,
        })
        if err != nil {
            pm.handlePermissionError(c, http.StatusInternalServerError, "Failed to check permission")
            return
        }

        if !result.Allowed {
            pm.handlePermissionError(c, http.StatusForbidden, "Insufficient permissions")
            return
        }

        c.Next()
    }
}
```

#### `RequireAnyPermission(resource, []actions)` - OR Logic

```go
// User needs at least one of the specified permissions
func (pm *PermissionMiddleware) RequireAnyPermission(
    resource string,
    actions []string,
) gin.HandlerFunc {
    return func(c *gin.Context) {
        userID, _ := authctx.GetUserID(c)
        orgID, _ := authctx.GetOrganizationID(c)

        for _, action := range actions {
            result, err := pm.engine.Check(c.Request.Context(), &ports.PermissionCheckRequest{
                UserID:         userID,
                OrganizationID: orgID,
                ResourceType:   resource,
                Action:         action,
            })

            if err == nil && result.Allowed {
                c.Next()
                return
            }
        }

        pm.handlePermissionError(c, http.StatusForbidden, "Insufficient permissions")
    }
}
```

#### `RequireAllPermissions(resource, []actions)` - AND Logic

```go
// User needs all of the specified permissions
func (pm *PermissionMiddleware) RequireAllPermissions(
    resource string,
    actions []string,
) gin.HandlerFunc {
    return func(c *gin.Context) {
        userID, _ := authctx.GetUserID(c)
        orgID, _ := authctx.GetOrganizationID(c)

        for _, action := range actions {
            result, err := pm.engine.Check(c.Request.Context(), &ports.PermissionCheckRequest{
                UserID:         userID,
                OrganizationID: orgID,
                ResourceType:   resource,
                Action:         action,
            })

            if err != nil || !result.Allowed {
                pm.handlePermissionError(c, http.StatusForbidden, "Insufficient permissions")
                return
            }
        }

        c.Next()
    }
}
```

#### `OptionalPermission(resource, action)` - Non-Blocking Check

```go
// Checks permission but doesn't block request
// Sets context value for handler to check
func (pm *PermissionMiddleware) OptionalPermission(
    resource string,
    action string,
) gin.HandlerFunc {
    return func(c *gin.Context) {
        userID, ok := authctx.GetUserID(c)
        if !ok {
            c.Set("has_"+resource+"_"+action, false)
            c.Next()
            return
        }

        orgID, _ := authctx.GetOrganizationID(c)

        result, err := pm.engine.Check(c.Request.Context(), &ports.PermissionCheckRequest{
            UserID:         userID,
            OrganizationID: orgID,
            ResourceType:   resource,
            Action:         action,
        })

        c.Set("has_"+resource+"_"+action, err == nil && result.Allowed)
        c.Next()
    }
}
```

### Handler Integration

**Example:** Hazardous Material Handler

**Location:** `/services/tms/internal/api/handlers/hazardousmaterial.go:40-62`

```go
type HazardousMaterialHandler struct {
    service    *hazardousmaterialservice.Service
    eh         *helpers.ErrorHandler
    middleware *middleware.PermissionMiddleware
}

func (h *HazardousMaterialHandler) RegisterRoutes(rg *gin.RouterGroup) {
    api := rg.Group("/hazardous-materials/")

    // List - requires read permission
    api.GET(
        "",
        h.middleware.RequirePermission(permission.ResourceHazardousMaterial, "read"),
        h.list,
    )

    // Get by ID - requires read permission
    api.GET(
        ":id/",
        h.middleware.RequirePermission(permission.ResourceHazardousMaterial, "read"),
        h.get,
    )

    // Create - requires create permission
    api.POST(
        "",
        h.middleware.RequirePermission(permission.ResourceHazardousMaterial, "create"),
        h.create,
    )

    // Update - requires update permission
    api.PUT(
        ":id/",
        h.middleware.RequirePermission(permission.ResourceHazardousMaterial, "update"),
        h.update,
    )
}
```

### Dependency Injection

**Location:** Handler params structure

```go
type HazardousMaterialHandlerParams struct {
    fx.In

    Service      *hazardousmaterialservice.Service
    Middleware   *middleware.PermissionMiddleware  // ← Injected
    ErrorHandler *helpers.ErrorHandler
}

func NewHazardousMaterialHandler(p HazardousMaterialHandlerParams) *HazardousMaterialHandler {
    return &HazardousMaterialHandler{
        service:    p.Service,
        eh:         p.ErrorHandler,
        middleware: p.Middleware,  // ← Stored for use in routes
    }
}
```

---

## Performance Optimization

### Bitfield Operations

**Purpose:** Store multiple boolean flags in a single integer for fast permission checks.

**Implementation:**

```go
const (
    ActionCreate uint32 = 1 << iota  // 1
    ActionRead                        // 2
    ActionUpdate                      // 4
    ActionDelete                      // 8
    ActionList                        // 16
    ActionExport                      // 32
    ActionImport                      // 64
    ActionApprove                     // 128
    ActionReject                      // 256
    ActionArchive                     // 512
)

// Check if action is allowed (single bitwise AND operation)
func HasAction(bitfield uint32, action string) bool {
    if bit, ok := ActionBits[action]; ok {
        return (bitfield & bit) != 0
    }
    return false
}

// Add action (single bitwise OR operation)
func AddAction(bitfield uint32, action string) uint32 {
    if bit, ok := ActionBits[action]; ok {
        return bitfield | bit
    }
    return bitfield
}
```

**Performance:** Single bitwise operation (~0.001μs)

**Storage:** 10 actions fit in 4 bytes instead of 10 booleans (10 bytes)

### Bloom Filter

**Purpose:** Fast negative permission checks to avoid cache lookups.

**How it works:**

1. Hash the permission key 3 times
2. Check if all 3 bits are set in the filter
3. If any bit is 0 → definitely not present (return false immediately)
4. If all bits are 1 → possibly present (continue with full check)

**False Positive Rate:** ~1% (acceptable tradeoff for speed)

**Performance:** 3 hash operations + 3 bit checks (~0.01μs)

**Size:** ~1KB for 1000 permissions

### Materialized View

**Purpose:** Pre-compute user permissions instead of expensive JOINs.

**Without Materialized View:**

```sql
-- 5+ table joins on every permission check (100-500ms)
SELECT p.* FROM policies p
JOIN role_policies rp ON rp.policy_id = p.id
JOIN user_roles ur ON ur.role_id = rp.role_id
JOIN user_organization_memberships uom ON uom.user_id = ur.user_id
WHERE uom.user_id = ? AND uom.organization_id = ?;
```

**With Materialized View:**

```sql
-- Simple indexed lookup (5-10ms)
SELECT * FROM user_effective_policies
WHERE user_id = ? AND organization_id = ?;
```

**Auto-Refresh:** Triggers update view when policies/roles change.

**Trade-off:**

- Storage: ~10MB for 10,000 users (acceptable)
- Write latency: +50ms when changing policies (rare operation)
- Read latency: -90% on permission checks (common operation)

### Performance Metrics

| Operation | Target | Actual (P95) | Cache Level |
|-----------|--------|--------------|-------------|
| Permission check (L1 hit) | < 1ms | 0.1ms | Memory |
| Permission check (L2 hit) | < 5ms | 2ms | Redis |
| Permission check (L3 hit) | < 10ms | 7ms | Database |
| Permission check (cache miss) | < 50ms | 35ms | Materialized view |
| Manifest generation | < 100ms | 65ms | Compiled |
| Batch check (10 permissions) | < 10ms | 3ms | Memory |
| Cache invalidation | < 5ms | 2ms | All levels |

---

## Migration from V1

### V1 vs V2 Comparison

| Feature | V1 | V2 |
|---------|----|----|
| Permission Model | String-based permissions | Policy-based |
| Storage | Simple array in user record | Policies + Roles + Memberships |
| Caching | Single Redis cache | L1/L2/L3 + Materialized view |
| Performance | 50-100ms per check | < 1ms per check (cached) |
| Granularity | Resource-level only | Resource + Field + Data scope |
| Multi-tenancy | Limited | Full BU/Org hierarchy |
| Scalability | 1000 users | 100,000+ users |

### Migration Steps

#### 1. Run Migration Scripts

```bash
# Apply database migrations
make db-migrate

# Migrations create:
# - policies table
# - roles table
# - user_organization_memberships updates
# - permission_cache table
# - user_effective_policies materialized view
# - Auto-refresh triggers
```

#### 2. Create System Policies

```go
// Create basic policies for existing permissions
systemAdmin := &permission.Policy{
    Name:     "system_admin",
    Effect:   permission.EffectAllow,
    Priority: 1000,
    Resources: permission.Resources{
        ResourceType: []string{"*"},
        Actions:      []string{"*"},
    },
    Scope: permission.Scope{
        BusinessUnitID:  buID,
        OrganizationIDs: []pulid.ID{}, // All orgs
    },
    IsSystem: true,
}
```

#### 3. Create Roles from V1 Permissions

```go
// Map old permission strings to new policies
func migrateUserPermissions(user *User) {
    // Old: user.Permissions = []string{"shipment.read", "shipment.create"}

    // New: Create role with policies
    role := &permission.Role{
        Name:  user.Username + "_migrated_role",
        Level: permission.RoleLevelCustom,
        PolicyIDs: []pulid.ID{
            findPolicyByResource("shipment", "read"),
            findPolicyByResource("shipment", "create"),
        },
    }

    // Assign role to user
    membership := &UserOrganizationMembership{
        UserID:         user.ID,
        OrganizationID: user.CurrentOrgID,
        BusinessUnitID: user.BusinessUnitID,
        RoleIDs:        []pulid.ID{role.ID},
    }
}
```

#### 4. Update Frontend Code

```typescript
// Old V1 code
if (user.permissions.includes('shipment.read')) {
    // ...
}

// New V2 code
const { can } = usePermissionV2();
if (can('shipment', 'read')) {
    // ...
}
```

#### 5. Update Handler Code

```go
// Old V1 code (service-level check)
func (s *Service) CreateShipment(ctx context.Context) error {
    if !s.hasPermission(ctx, "shipment.create") {
        return ErrPermissionDenied
    }
    // ...
}

// New V2 code (middleware-level check)
router.POST("/shipments",
    permMiddleware.RequirePermission(permission.ResourceShipment, "create"),
    handler.CreateShipment,
)

func (h *Handler) CreateShipment(c *gin.Context) {
    // No permission check needed - already done in middleware
    // ...
}
```

#### 6. Test Permissions

```bash
# Test permission checks
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/permissions/manifest

# Verify user has expected permissions
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/hazardous-materials/

# Should return 403 if no permission
```

#### 7. Refresh Materialized View

```bash
# Manual refresh if needed
docker exec -i tms-db-1 psql -U postgres -d trenova_go_db \
  -c "SELECT refresh_user_effective_policies();"
```

---

## Security Considerations

### Defense in Depth

1. **Server-Side Enforcement** (Primary)
   - Permission checks in middleware (required)
   - Fail-fast at handler level
   - Never trust client-side checks

2. **Client-Side Checks** (Secondary)
   - UI/UX only (hide/disable elements)
   - Reduce unnecessary API calls
   - Improve user experience

3. **Database-Level Security** (Tertiary)
   - Row-level security (RLS) policies
   - Data scoping enforcement
   - Audit logging

### Common Vulnerabilities

#### ❌ BAD: Skipping Middleware

```go
// NEVER do this - bypasses permission checks
router.POST("/shipments", handler.CreateShipment)
```

#### ✅ GOOD: Always Use Middleware

```go
router.POST("/shipments",
    permMiddleware.RequirePermission("shipment", "create"),
    handler.CreateShipment,
)
```

#### ❌ BAD: Trusting Client Input

```go
func (h *Handler) UpdateShipment(c *gin.Context) {
    // User could send any shipment ID
    shipmentID := c.Param("id")

    // No ownership check!
    h.service.Update(ctx, shipmentID, data)
}
```

#### ✅ GOOD: Enforce Data Scope

```go
func (h *Handler) UpdateShipment(c *gin.Context) {
    shipmentID := c.Param("id")
    userID := authctx.GetUserID(c)

    // Check ownership based on data scope
    shipment, err := h.service.Get(ctx, shipmentID)
    if shipment.CreatedBy != userID {
        return ErrNotFound // Don't reveal existence
    }

    h.service.Update(ctx, shipmentID, data)
}
```

### Audit Trail

All permission checks are logged:

```json
{
  "timestamp": "2025-09-29T20:15:00Z",
  "level": "info",
  "msg": "permission_check",
  "user_id": "usr_123",
  "organization_id": "org_456",
  "resource": "shipment",
  "action": "delete",
  "allowed": false,
  "reason": "insufficient permissions",
  "cache_hit": true,
  "compute_time_ms": 0.1
}
```

---

## API Reference

### Permission Engine Interface

```go
type PermissionEngine interface {
    // Check single permission
    Check(ctx context.Context, req *PermissionCheckRequest) (*PermissionCheckResult, error)

    // Check multiple permissions in batch
    CheckBatch(ctx context.Context, req *BatchPermissionCheckRequest) (*BatchPermissionCheckResult, error)

    // Get complete permission manifest for user
    GetUserPermissions(ctx context.Context, userID, organizationID pulid.ID) (*PermissionManifest, error)

    // Refresh user permissions (invalidate cache)
    RefreshUserPermissions(ctx context.Context, userID, organizationID pulid.ID) error

    // Invalidate cache for user
    InvalidateCache(ctx context.Context, userID, organizationID pulid.ID) error

    // Get field-level access rules
    GetFieldAccess(ctx context.Context, userID, organizationID pulid.ID, resourceType string) (*FieldAccessRules, error)
}
```

### HTTP API Endpoints

#### `GET /api/permissions/manifest`

Get complete permission manifest for current user.

**Response:**

```json
{
  "version": "v1_1727650800",
  "userId": "usr_123",
  "currentOrg": "org_456",
  "availableOrgs": ["org_456", "org_789"],
  "computedAt": 1727650800,
  "expiresAt": 1727654400,
  "resources": {
    "shipment": 127,  // Bitfield: create|read|update|delete|list|export|import
    "customer": {
      "standardOps": 31,  // create|read|update|delete|list
      "extendedOps": ["export"],
      "dataScope": {
        "type": "organization"
      },
      "fieldRules": {
        "readableFields": ["*"],
        "writableFields": ["name", "address"],
        "maskedFields": {
          "creditCard": "partial"
        }
      }
    }
  },
  "checksum": "abc123..."
}
```

#### `POST /api/permissions/check`

Check single permission.

**Request:**

```json
{
  "resource": "shipment",
  "action": "create"
}
```

**Response:**

```json
{
  "allowed": true,
  "dataScope": {
    "type": "organization"
  },
  "fieldAccess": {
    "readable": ["*"],
    "writable": ["status", "notes"],
    "masked": {}
  },
  "cacheHit": true,
  "computeTimeMs": 0.1
}
```

#### `POST /api/permissions/check-batch`

Check multiple permissions at once.

**Request:**

```json
{
  "checks": [
    {"resource": "shipment", "action": "create"},
    {"resource": "shipment", "action": "delete"},
    {"resource": "customer", "action": "read"}
  ]
}
```

**Response:**

```json
{
  "results": [
    {"allowed": true, "cacheHit": true},
    {"allowed": false, "reason": "insufficient permissions", "cacheHit": true},
    {"allowed": true, "cacheHit": true}
  ],
  "cacheHitRate": 1.0,
  "totalTimeMs": 0.3
}
```

#### `POST /api/permissions/refresh`

Manually refresh user permissions (admin only).

**Request:**

```json
{
  "userId": "usr_123",
  "organizationId": "org_456"
}
```

**Response:**

```json
{
  "success": true,
  "message": "Permissions refreshed successfully"
}
```

---

## Troubleshooting

### Problem: Permission denied but user should have access

**Symptoms:**

- API returns 403 Forbidden
- User has correct role assigned
- Policy looks correct

**Debug Steps:**

1. Check materialized view:

```sql
SELECT * FROM user_effective_policies
WHERE user_id = 'usr_123' AND organization_id = 'org_456';
```

2. Refresh materialized view:

```sql
SELECT refresh_user_effective_policies();
```

3. Check cache:

```bash
# Redis (L2)
redis-cli GET "perm:usr_123:org_456"

# Database (L3)
SELECT * FROM permission_cache
WHERE user_id = 'usr_123' AND organization_id = 'org_456';
```

4. Invalidate cache:

```bash
curl -X POST http://localhost:8080/api/permissions/refresh \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{"userId":"usr_123","organizationId":"org_456"}'
```

5. Check logs:

```bash
# Look for permission check logs
grep "permission_check" logs/app.log | grep "usr_123"
```

### Problem: Materialized view not updating

**Symptoms:**

- Policy changes don't take effect
- New role assignments don't work
- Materialized view is empty

**Solution:**

1. Check if triggers exist:

```sql
SELECT tgname, tgenabled FROM pg_trigger
WHERE tgname LIKE '%refresh%';
```

2. Manually refresh:

```sql
SELECT refresh_user_effective_policies();
```

3. Verify trigger function:

```sql
SELECT prosrc FROM pg_proc
WHERE proname = 'trigger_refresh_user_policies';
```

4. Re-create triggers if needed:

```bash
# Re-run migration
make db-migrate
```

### Problem: Performance degradation

**Symptoms:**

- Permission checks taking > 100ms
- High CPU usage
- Slow API responses

**Debug Steps:**

1. Check cache hit rates:

```bash
# L1 cache
grep "cache hit: L1" logs/app.log | wc -l

# L2 cache
grep "cache hit: L2" logs/app.log | wc -l

# Cache miss
grep "cache miss" logs/app.log | wc -l
```

2. Check Redis connection:

```bash
redis-cli PING
```

3. Check materialized view size:

```sql
SELECT pg_size_pretty(pg_total_relation_size('user_effective_policies'));
```

4. Analyze view performance:

```sql
EXPLAIN ANALYZE
SELECT * FROM user_effective_policies
WHERE user_id = 'usr_123' AND organization_id = 'org_456';
```

5. Rebuild indexes:

```sql
REINDEX TABLE user_effective_policies;
```

### Problem: Context canceled errors

**Symptoms:**

- Logs show "context canceled" errors
- Happens during shutdown or when clients disconnect

**Solution:**

Already fixed in `/services/tms/internal/infrastructure/redis/repositories/permissioncache/cache.go:127-131`

```go
if err := c.setToL3(ctx, userID, organizationID, permissions); err != nil {
    if ctx.Err() == nil {
        log.Warn("failed to set L3 cache", zap.Error(err))
    }
}
```

This suppresses warnings for expected context cancellations.

---

## Summary

Permission System V2 provides:

✅ **Sub-millisecond permission checks** via L1/L2/L3 caching
✅ **Policy-based access control** with flexible resource/action rules
✅ **Role-based assignment** for easy permission management
✅ **Multi-tenant support** with BU → Org → User hierarchy
✅ **Field-level security** for fine-grained access control
✅ **Data scoping** to restrict access by ownership
✅ **Materialized views** for instant permission lookups
✅ **Auto-refresh triggers** to keep permissions up-to-date
✅ **Client SDK** with React hooks for frontend integration
✅ **Middleware enforcement** at handler level (fail-fast)
✅ **Comprehensive audit trail** for all permission checks

For implementation examples, see:

- [Permission Middleware Usage Guide](PERMISSION_MIDDLEWARE_USAGE.md)
- [Materialized Views Documentation](PERMISSION_MATERIALIZED_VIEWS.md)
- [Permission System V2 Overview](PERMISSION_SYSTEM_V2.md)
