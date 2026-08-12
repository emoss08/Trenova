-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20250527184726_page_favorite.tx.up.sql

CREATE TABLE IF NOT EXISTS "page_favorites"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "user_id" TEXT NOT NULL,
    "page_url" TEXT NOT NULL,
    "page_title" TEXT NOT NULL,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    PRIMARY KEY ("id"),
    CONSTRAINT "fk_page_favorites_business_unit_id" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id"),
    CONSTRAINT "fk_page_favorites_organization_id" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id"),
    CONSTRAINT "fk_page_favorites_user_id" FOREIGN KEY ("user_id") REFERENCES "users"("id")
);
