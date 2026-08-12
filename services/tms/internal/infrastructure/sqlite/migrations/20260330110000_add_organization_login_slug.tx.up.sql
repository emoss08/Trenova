-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260330110000_add_organization_login_slug.tx.up.sql

ALTER TABLE "organizations" ADD COLUMN "login_slug" TEXT;

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_organizations_login_slug" ON "organizations" ("login_slug")WHERE
    "login_slug" IS NOT NULL
    AND "login_slug" <> '';
