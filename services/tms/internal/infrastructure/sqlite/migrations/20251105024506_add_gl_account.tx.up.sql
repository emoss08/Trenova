-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20251105024506_add_gl_account.tx.up.sql

CREATE TABLE IF NOT EXISTS "gl_accounts"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Active',
    "account_type_id" TEXT NOT NULL,
    "parent_id" TEXT,
    "account_code" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "is_system" INTEGER NOT NULL DEFAULT 0,
    "allow_manual_je" INTEGER NOT NULL DEFAULT 1,
    "require_project" INTEGER NOT NULL DEFAULT 0,
    "current_balance" INTEGER NOT NULL DEFAULT 0,
    "debit_balance" INTEGER NOT NULL DEFAULT 0,
    "credit_balance" INTEGER NOT NULL DEFAULT 0,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_gl_accounts" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_gl_accounts_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_gl_accounts_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_gl_accounts_account_type" FOREIGN KEY ("account_type_id", "organization_id", "business_unit_id") REFERENCES "account_types"("id", "organization_id", "business_unit_id") ON DELETE RESTRICT,
    CONSTRAINT "fk_gl_accounts_parent" FOREIGN KEY ("parent_id", "organization_id", "business_unit_id") REFERENCES "gl_accounts"("id", "organization_id", "business_unit_id") ON DELETE RESTRICT,
    CONSTRAINT "uq_gl_accounts_code" UNIQUE ("organization_id", "business_unit_id", "account_code"),
    CONSTRAINT "chk_gl_accounts_no_self_parent" CHECK ("id" != "parent_id")
);

--bun:split

CREATE INDEX IF NOT EXISTS idx_gl_accounts_bu_org ON "gl_accounts" ("business_unit_id", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS idx_gl_accounts_created_updated ON "gl_accounts" ("created_at", "updated_at");

--bun:split

CREATE INDEX IF NOT EXISTS idx_gl_accounts_status ON "gl_accounts" ("status");

--bun:split

CREATE INDEX IF NOT EXISTS idx_gl_accounts_account_type ON "gl_accounts" ("account_type_id");

--bun:split

CREATE INDEX IF NOT EXISTS idx_gl_accounts_parent ON "gl_accounts" ("parent_id");

--bun:split

CREATE INDEX IF NOT EXISTS idx_gl_accounts_code ON "gl_accounts" ("account_code");
