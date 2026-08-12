-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260728100000_shipment_comments_overhaul.tx.up.sql

ALTER TABLE "shipment_comments" ADD COLUMN "parent_comment_id" TEXT;

--bun:split

ALTER TABLE "shipment_comments" ADD COLUMN "body" TEXT;

--bun:split

ALTER TABLE "shipment_comments" ADD COLUMN "pinned_at" INTEGER;

--bun:split

ALTER TABLE "shipment_comments" ADD COLUMN "pinned_by_id" TEXT;

--bun:split

ALTER TABLE "shipment_comments" ADD COLUMN "resolved_at" INTEGER;

--bun:split

ALTER TABLE "shipment_comments" ADD COLUMN "resolved_by_id" TEXT;

--bun:split

ALTER TABLE "shipment_comments" ADD COLUMN "requires_acknowledgment" INTEGER NOT NULL DEFAULT 0;

--bun:split

ALTER TABLE "shipment_comments" ADD COLUMN "deleted_at" INTEGER;

--bun:split

ALTER TABLE "shipment_comments" ADD COLUMN "deleted_by_id" TEXT;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_shipment_comments_top_level_list" ON "shipment_comments" ("organization_id", "business_unit_id", "shipment_id", "created_at" DESC, "id" DESC)WHERE "parent_comment_id" IS NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_shipment_comments_parent" ON "shipment_comments" ("parent_comment_id")WHERE "parent_comment_id" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_shipment_comments_pinned" ON "shipment_comments" ("shipment_id", "pinned_at" DESC)WHERE "pinned_at" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_shipment_comments_unresolved" ON "shipment_comments" ("shipment_id")WHERE "resolved_at" IS NULL AND "deleted_at" IS NULL;

--bun:split

CREATE TABLE IF NOT EXISTS "shipment_comment_acknowledgments"(
    "id" TEXT NOT NULL,
    "comment_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "shipment_id" TEXT NOT NULL,
    "user_id" TEXT NOT NULL,
    "acknowledged_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_shipment_comment_acknowledgments" PRIMARY KEY ("id"),
    CONSTRAINT "fk_shipment_comment_acks_comment" FOREIGN KEY ("comment_id", "business_unit_id", "organization_id", "shipment_id") REFERENCES "shipment_comments"("id", "business_unit_id", "organization_id", "shipment_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shipment_comment_acks_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shipment_comment_acks_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "uk_shipment_comment_acks_comment_user" UNIQUE ("comment_id", "user_id")
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_shipment_comment_acks_comment" ON "shipment_comment_acknowledgments" ("comment_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_shipment_comment_acks_user" ON "shipment_comment_acknowledgments" ("user_id");
