-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20250724214710_add_shipment_comments.tx.up.sql

CREATE TABLE IF NOT EXISTS "shipment_comments"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "shipment_id" TEXT NOT NULL,
    "user_id" TEXT NOT NULL,
    "comment" TEXT NOT NULL,
    "type" TEXT NOT NULL DEFAULT 'Internal',
    "visibility" TEXT NOT NULL DEFAULT 'Internal',
    "priority" TEXT NOT NULL DEFAULT 'Normal',
    "source" TEXT NOT NULL DEFAULT 'User',
    "edited_at" INTEGER,
    "metadata" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_shipment_comments" PRIMARY KEY ("id", "business_unit_id", "organization_id", "shipment_id"),
    CONSTRAINT "fk_shipment_comments_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_shipment_comments_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shipment_comments_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shipment_comments_shipment" FOREIGN KEY ("shipment_id", "business_unit_id", "organization_id") REFERENCES "shipments"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_shipment_comments_shipment" ON "shipment_comments" ("shipment_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_shipment_comments_business_unit" ON "shipment_comments" ("business_unit_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_shipment_comments_organization" ON "shipment_comments" ("organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_shipment_comments_org_bu_user" ON "shipment_comments" ("organization_id", "business_unit_id", "user_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_shipment_comments_created_updated" ON "shipment_comments" ("created_at", "updated_at");

--bun:split

CREATE TABLE IF NOT EXISTS "shipment_comment_mentions"(
    "id" TEXT NOT NULL,
    "comment_id" TEXT NOT NULL,
    "mentioned_user_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "shipment_id" TEXT NOT NULL,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_shipment_comment_mentions" PRIMARY KEY ("id"),
    CONSTRAINT "fk_shipment_comment_mentions_comment" FOREIGN KEY ("comment_id", "business_unit_id", "organization_id", "shipment_id") REFERENCES "shipment_comments"("id", "business_unit_id", "organization_id", "shipment_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shipment_comment_mentions_user" FOREIGN KEY ("mentioned_user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shipment_comment_mentions_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "uk_shipment_comment_mentions_comment_user" UNIQUE ("comment_id", "mentioned_user_id")
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_shipment_comment_mentions_comment" ON "shipment_comment_mentions" ("comment_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_shipment_comment_mentions_user" ON "shipment_comment_mentions" ("mentioned_user_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_shipment_comment_mentions_organization" ON "shipment_comment_mentions" ("organization_id");
