-- Enum for permission scopes with descriptions
CREATE TYPE "scope_enum" AS ENUM (
    'global', -- System-wide scope
    'business_unit', -- Limited to a specific business unit
    'organization', -- Limited to a specific organization
    'department', -- Limited to a specific department
    'team', -- Limited to a specific team
    'personal', -- Limited to an individual user
    'region', -- Limited to a geographic region
    'fleet', -- Limited to a specific fleet
    'customer_group' -- Limited to a specific customer group
);

-- Enum for permission actions with descriptions
CREATE TYPE "action_enum" AS ENUM (
    'create', -- Permission to create new resources
    'read', -- Permission to view resources
    'update', -- Permission to modify existing resources
    'delete', -- Permission to remove resources
    'modify_field', -- Permission to modify specific fields
    'view_field', -- Permission to view specific fields
    'approve', -- Permission to approve requests/changes
    'reject', -- Permission to reject requests/changes
    'ready_to_bill', -- Permission to mark a shipment as ready to bill
    'release_to_billing', -- Permission to release a shipment to billing
    'bulk_transfer', -- Permission to bulk transfer shipments to the billing queue
    'review_invoice', -- Permission to review an invoice
    'post_invoice', -- Permission to post an invoice
    'split', -- Permission to split resources
    'submit', -- Permission to submit for approval
    'cancel', -- Permission to cancel operations
    'duplicate', -- Permission to duplicate resources
    'export', -- Permission to export data
    'import', -- Permission to import data
    'archive', -- Permission to archive resources
    'restore', -- Permission to restore archived resources
    'manage', -- Full management permissions
    'share', -- Permission to share resources
    'audit', -- Permission to view audit logs
    'delegate', -- Permission to delegate authority
    'configure', -- Permission to configure settings
    'manage_defaults', -- Permission to manage table configurations
    'assign', -- Permission to assign a resource to a user or group
    'reassign', -- Permission to assign a resource to a different user or group
    'complete' -- Permission to mark a resource or action as completed
);

--bun:split
CREATE TABLE IF NOT EXISTS "permissions" (
    "id" varchar(100) NOT NULL,
    "resource" varchar(50) NOT NULL,
    "action" action_enum NOT NULL,
    "scope" scope_enum NOT NULL,
    "description" text,
    "is_system_level" boolean NOT NULL DEFAULT FALSE,
    "field_permissions" jsonb DEFAULT '[]' ::jsonb,
    "conditions" jsonb DEFAULT '[]' ::jsonb,
    "dependencies" jsonb DEFAULT '[]' ::jsonb,
    "custom_settings" jsonb DEFAULT '{}' ::jsonb,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    PRIMARY KEY ("id"),
    -- Ensure JSONB fields contain valid objects
    CONSTRAINT "check_field_permissions_format" CHECK (jsonb_typeof(field_permissions) = 'array'),
    CONSTRAINT "check_conditions_format" CHECK (jsonb_typeof(conditions) = 'array'),
    CONSTRAINT "check_dependencies_format" CHECK (jsonb_typeof(dependencies) = 'array'),
    CONSTRAINT "check_custom_settings_format" CHECK (jsonb_typeof(custom_settings) = 'object')
);

--bun:split
-- Indexes for common query patterns
CREATE UNIQUE INDEX "idx_permissions_resource_action_scope" ON "permissions" ("resource", "action", "scope");

CREATE INDEX "idx_permissions_resource" ON "permissions" ("resource");

CREATE INDEX "idx_permissions_action" ON "permissions" ("action");

CREATE INDEX "idx_permissions_scope" ON "permissions" ("scope");

CREATE INDEX "idx_permissions_system_level" ON "permissions" ("is_system_level");

CREATE INDEX "idx_permissions_created_updated" ON "permissions" ("created_at", "updated_at");

-- JSONB indexes for efficient querying
CREATE INDEX "idx_permissions_field_permissions" ON "permissions" USING gin ("field_permissions");

CREATE INDEX "idx_permissions_conditions" ON "permissions" USING gin ("conditions");

CREATE INDEX "idx_permissions_dependencies" ON "permissions" USING gin ("dependencies");

-- Table and column comments
COMMENT ON TABLE permissions IS 'Stores permission definitions for system access control';

--bun:split
CREATE TYPE "role_type_enum" AS ENUM (
    'System', -- Built-in system roles
    'Organization', -- Organization-specific roles
    'Custom', -- User-defined custom roles
    'Temporary' -- Time-limited temporary roles
);

--bun:split
CREATE TABLE IF NOT EXISTS "roles" (
    "id" varchar(100) NOT NULL,
    "name" varchar(100) NOT NULL,
    "description" text,
    "role_type" role_type_enum NOT NULL,
    "is_system" boolean NOT NULL DEFAULT FALSE,
    "priority" int NOT NULL DEFAULT 0,
    "status" status_enum NOT NULL DEFAULT 'Active',
    "expires_at" bigint,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "parent_role_id" varchar(100),
    "metadata" jsonb DEFAULT '{}' ::jsonb,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_roles_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_roles_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_roles_parent" FOREIGN KEY ("parent_role_id", "business_unit_id", "organization_id") REFERENCES "roles" ("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "check_parent_not_self" CHECK ("id" != "parent_role_id"),
    CONSTRAINT "check_priority_range" CHECK ("priority" >= 0),
    CONSTRAINT "check_metadata_format" CHECK (jsonb_typeof(metadata) = 'object'),
    -- Ensure system roles have appropriate settings
    CONSTRAINT "check_system_role_type" CHECK ((is_system = TRUE AND role_type = 'System') OR (is_system = FALSE)),
    -- Ensure temporary roles have expiration
    CONSTRAINT "check_temporary_role_expires" CHECK ((role_type = 'Temporary' AND expires_at IS NOT NULL) OR (role_type != 'Temporary'))
);

--bun:split
-- Indexes for common query patterns
CREATE UNIQUE INDEX "idx_roles_name_org" ON "roles" (lower("name"), "organization_id");

CREATE INDEX "idx_roles_business_unit" ON "roles" ("business_unit_id");

CREATE INDEX "idx_roles_organization" ON "roles" ("organization_id");

CREATE INDEX "idx_roles_parent" ON "roles" ("parent_role_id");

CREATE INDEX "idx_roles_type" ON "roles" ("role_type");

CREATE INDEX "idx_roles_status" ON "roles" ("status");

CREATE INDEX "idx_roles_expires_at" ON "roles" ("expires_at");

CREATE INDEX "idx_roles_system" ON "roles" ("is_system");

CREATE INDEX "idx_roles_created_updated" ON "roles" ("created_at", "updated_at");

CREATE INDEX "idx_roles_metadata" ON "roles" USING gin ("metadata");

ALTER TABLE roles
    ALTER COLUMN status SET STATISTICS 1000;

ALTER TABLE roles
    ALTER COLUMN business_unit_id SET STATISTICS 1000;

ALTER TABLE roles
    ALTER COLUMN organization_id SET STATISTICS 1000;

-- Table and column comments
COMMENT ON TABLE roles IS 'Stores role definitions for access control management';

--bun:split
-- Enum for permission grant statuses with descriptions
CREATE TYPE "permission_status_enum" AS ENUM (
    'Active', -- Permission is currently active and valid
    'Inactive', -- Permission is temporarily inactive
    'Suspended', -- Permission is suspended due to policy/violation
    'Archived' -- Permission is archived for historical reference
);

--bun:split
CREATE TABLE IF NOT EXISTS "permission_grants" (
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "user_id" varchar(100) NOT NULL,
    "permission_id" varchar(100) NOT NULL,
    "resource_id" varchar(100),
    "granted_by" varchar(100) NOT NULL,
    "revoked_by" varchar(100),
    "status" permission_status_enum NOT NULL DEFAULT 'Active',
    "expires_at" bigint,
    "revoked_at" bigint,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "reason" text,
    "field_overrides" jsonb DEFAULT '{}' ::jsonb,
    "conditions" jsonb DEFAULT '{}' ::jsonb,
    "audit_trail" jsonb DEFAULT '[]' ::jsonb,
    PRIMARY KEY ("id"),
    CONSTRAINT "fk_permission_grants_user" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_permission_grants_permission" FOREIGN KEY ("permission_id") REFERENCES "permissions" ("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_permission_grants_granted_by" FOREIGN KEY ("granted_by") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_permission_grants_revoked_by" FOREIGN KEY ("revoked_by") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_permission_grants_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_permission_grants_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "check_revocation_status" CHECK ((status = 'Archived' AND revoked_at IS NOT NULL AND revoked_by IS NOT NULL) OR (status != 'Archived' AND (revoked_at IS NULL AND revoked_by IS NULL))),
    CONSTRAINT "check_expiry_date" CHECK (expires_at IS NULL OR expires_at > created_at),
    CONSTRAINT "check_revoked_date" CHECK (revoked_at IS NULL OR revoked_at >= created_at),
    CONSTRAINT "check_field_overrides_format" CHECK (jsonb_typeof(field_overrides) = 'object'),
    CONSTRAINT "check_conditions_format" CHECK (jsonb_typeof(conditions) = 'object'),
    CONSTRAINT "check_audit_trail_format" CHECK (jsonb_typeof(audit_trail) = 'array')
);

--bun:split
-- Indexes for common query patterns
CREATE UNIQUE INDEX "idx_permission_grants_user_permission_resource" ON "permission_grants" ("user_id", "permission_id", COALESCE(resource_id, ''))
WHERE
    status = 'Active';

CREATE INDEX "idx_permission_grants_user" ON "permission_grants" ("user_id");

CREATE INDEX "idx_permission_grants_permission" ON "permission_grants" ("permission_id");

CREATE INDEX "idx_permission_grants_organization" ON "permission_grants" ("organization_id");

CREATE INDEX "idx_permission_grants_business_unit" ON "permission_grants" ("business_unit_id");

CREATE INDEX "idx_permission_grants_granted_by" ON "permission_grants" ("granted_by");

CREATE INDEX "idx_permission_grants_status" ON "permission_grants" ("status");

CREATE INDEX "idx_permission_grants_expires_at" ON "permission_grants" ("expires_at")
WHERE
    expires_at IS NOT NULL;

CREATE INDEX "idx_permission_grants_created" ON "permission_grants" ("created_at");

CREATE INDEX "idx_permission_grants_revoked" ON "permission_grants" ("revoked_at")
WHERE
    revoked_at IS NOT NULL;

CREATE INDEX "idx_permission_grants_field_overrides" ON "permission_grants" USING gin ("field_overrides");

CREATE INDEX "idx_permission_grants_conditions" ON "permission_grants" USING gin ("conditions");

CREATE INDEX "idx_permission_grants_audit_trail" ON "permission_grants" USING gin ("audit_trail");

ALTER TABLE permission_grants
    ALTER COLUMN status SET STATISTICS 1000;

ALTER TABLE permission_grants
    ALTER COLUMN business_unit_id SET STATISTICS 1000;

ALTER TABLE permission_grants
    ALTER COLUMN organization_id SET STATISTICS 1000;

ALTER TABLE permission_grants
    ALTER COLUMN user_id SET STATISTICS 1000;

ALTER TABLE permission_grants
    ALTER COLUMN permission_id SET STATISTICS 1000;

-- Table and column comments
COMMENT ON TABLE permission_grants IS 'Stores granted permissions to users with associated metadata and audit information';

--bun:split
-- Permission Templates table
CREATE TABLE IF NOT EXISTS "permission_templates" (
    "id" varchar(100) NOT NULL,
    "name" varchar(100) NOT NULL,
    "description" text,
    "permissions" jsonb DEFAULT '[]' ::jsonb,
    "field_settings" jsonb DEFAULT '{}' ::jsonb,
    "is_system" boolean NOT NULL DEFAULT FALSE,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    PRIMARY KEY ("id"),
    CONSTRAINT "check_permissions_format" CHECK (jsonb_typeof(permissions) = 'array'),
    CONSTRAINT "check_field_settings_format" CHECK (jsonb_typeof(field_settings) = 'object')
);

-- Indexes for permission_templates
CREATE UNIQUE INDEX "idx_permission_templates_name" ON "permission_templates" (lower("name"));

CREATE INDEX "idx_permission_templates_system" ON "permission_templates" ("is_system");

CREATE INDEX "idx_permission_templates_created_updated" ON "permission_templates" ("created_at", "updated_at");

CREATE INDEX "idx_permission_templates_permissions" ON "permission_templates" USING gin ("permissions");

CREATE INDEX "idx_permission_templates_field_settings" ON "permission_templates" USING gin ("field_settings");

COMMENT ON TABLE permission_templates IS 'Stores predefined permission templates for quick role setup';

--bun:split
-- Role Permissions junction table
CREATE TABLE IF NOT EXISTS "role_permissions" (
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "role_id" varchar(100) NOT NULL,
    "permission_id" varchar(100) NOT NULL,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    PRIMARY KEY ("role_id", "permission_id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_role_permissions_role" FOREIGN KEY ("role_id", "organization_id", "business_unit_id") REFERENCES "roles" ("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_role_permissions_permission" FOREIGN KEY ("permission_id") REFERENCES "permissions" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

CREATE INDEX "idx_role_permissions_role" ON "role_permissions" ("role_id");

CREATE INDEX "idx_role_permissions_permission" ON "role_permissions" ("permission_id");

CREATE INDEX "idx_role_permissions_created" ON "role_permissions" ("created_at");

COMMENT ON TABLE role_permissions IS 'Junction table linking roles to their assigned permissions';

--bun:split
-- User Roles junction table
CREATE TABLE IF NOT EXISTS "user_roles" (
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "user_id" varchar(100) NOT NULL,
    "role_id" varchar(100) NOT NULL,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "granted_by" varchar(100),
    "expires_at" bigint,
    PRIMARY KEY ("user_id", "role_id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_user_roles_user" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_user_roles_role" FOREIGN KEY ("role_id", "organization_id", "business_unit_id") REFERENCES "roles" ("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_user_roles_granted_by" FOREIGN KEY ("granted_by") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "check_expiry_date" CHECK (expires_at IS NULL OR expires_at > created_at)
);

CREATE INDEX "idx_user_roles_user" ON "user_roles" ("user_id");

CREATE INDEX "idx_user_roles_role" ON "user_roles" ("role_id");

CREATE INDEX "idx_user_roles_granted_by" ON "user_roles" ("granted_by");

CREATE INDEX "idx_user_roles_expires_at" ON "user_roles" ("expires_at")
WHERE
    expires_at IS NOT NULL;

CREATE INDEX "idx_user_roles_created" ON "user_roles" ("created_at");

--bun:split
ALTER TABLE user_roles
    ALTER COLUMN business_unit_id SET STATISTICS 1000;

--bun:split
ALTER TABLE user_roles
    ALTER COLUMN organization_id SET STATISTICS 1000;

COMMENT ON TABLE user_roles IS 'Junction table linking users to their assigned roles';

