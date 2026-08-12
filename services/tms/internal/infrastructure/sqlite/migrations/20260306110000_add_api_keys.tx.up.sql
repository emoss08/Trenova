-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260306110000_add_api_keys.tx.up.sql

CREATE TABLE IF NOT EXISTS "api_keys"(
    "id" TEXT PRIMARY KEY,
    "business_unit_id" TEXT NOT NULL REFERENCES business_units(id) ON DELETE CASCADE,
    "organization_id" TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "key_prefix" TEXT NOT NULL UNIQUE,
    "secret_hash" TEXT NOT NULL,
    "secret_salt" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'active',
    "expires_at" INTEGER,
    "last_used_at" INTEGER,
    "last_used_ip" TEXT,
    "last_used_user_agent" TEXT,
    "created_by_id" TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    "revoked_by_id" TEXT,
    "revoked_at" INTEGER,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "chk_api_keys_status" CHECK (status IN ('active', 'revoked'))
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_api_keys_org_bu" ON "api_keys" ("organization_id", "business_unit_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_api_keys_status" ON "api_keys" ("status");

--bun:split

CREATE TABLE IF NOT EXISTS "api_key_permissions"(
    "id" TEXT PRIMARY KEY,
    "api_key_id" TEXT NOT NULL REFERENCES api_keys(id) ON DELETE CASCADE,
    "business_unit_id" TEXT NOT NULL REFERENCES business_units(id) ON DELETE CASCADE,
    "organization_id" TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    "resource" TEXT NOT NULL,
    "operations" TEXT NOT NULL,
    "data_scope" TEXT NOT NULL DEFAULT 'organization',
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch())
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_api_key_permissions_key" ON "api_key_permissions" ("api_key_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_api_key_permissions_org_bu" ON "api_key_permissions" ("organization_id", "business_unit_id");
