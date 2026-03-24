CREATE TYPE "api_key_status_enum" AS ENUM(
    'active',
    'revoked'
);

--bun:split
CREATE TABLE "api_keys"(
    "id" varchar(100) PRIMARY KEY,
    "business_unit_id" varchar(100) NOT NULL REFERENCES business_units(id) ON DELETE CASCADE,
    "organization_id" varchar(100) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    "name" varchar(255) NOT NULL,
    "description" text,
    "key_prefix" varchar(32) NOT NULL UNIQUE,
    "secret_hash" varchar(128) NOT NULL,
    "secret_salt" varchar(64) NOT NULL,
    "status" varchar(20) NOT NULL DEFAULT 'active',
    "expires_at" bigint,
    "last_used_at" bigint,
    "last_used_ip" varchar(45),
    "last_used_user_agent" varchar(255),
    "created_by_id" varchar(100) NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    "revoked_by_id" varchar(100) REFERENCES users(id) ON DELETE SET NULL,
    "revoked_at" bigint,
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp) ::bigint,
    CONSTRAINT "chk_api_keys_status" CHECK (status IN ('active', 'revoked'))
);

CREATE INDEX IF NOT EXISTS "idx_api_keys_org_bu" ON "api_keys"("organization_id", "business_unit_id");

CREATE INDEX IF NOT EXISTS "idx_api_keys_status" ON "api_keys"("status");

CREATE TABLE "api_key_permissions"(
    "id" varchar(100) PRIMARY KEY,
    "api_key_id" varchar(100) NOT NULL REFERENCES api_keys(id) ON DELETE CASCADE,
    "business_unit_id" varchar(100) NOT NULL REFERENCES business_units(id) ON DELETE CASCADE,
    "organization_id" varchar(100) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    "resource" varchar(100) NOT NULL,
    "operations" text[] NOT NULL DEFAULT ARRAY[] ::text[],
    "data_scope" varchar(20) NOT NULL DEFAULT 'organization',
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp) ::bigint
);

CREATE INDEX "idx_api_key_permissions_key" ON "api_key_permissions"("api_key_id");

CREATE INDEX "idx_api_key_permissions_org_bu" ON "api_key_permissions"("organization_id", "business_unit_id");
