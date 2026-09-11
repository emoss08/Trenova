-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261212000000_user_organization_locale.tx.up.sql

ALTER TABLE "organizations" ADD COLUMN "locale" TEXT NOT NULL DEFAULT 'en';

--bun:split

ALTER TABLE "users" ADD COLUMN "locale" TEXT NOT NULL DEFAULT 'en';
