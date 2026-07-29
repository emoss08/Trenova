--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md
--
ALTER TABLE "shipment_comments"
    ALTER COLUMN "user_id" DROP NOT NULL;

--bun:split
ALTER TABLE "shipment_comments"
    ADD COLUMN IF NOT EXISTS "parent_comment_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "body" jsonb,
    ADD COLUMN IF NOT EXISTS "pinned_at" bigint,
    ADD COLUMN IF NOT EXISTS "pinned_by_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "resolved_at" bigint,
    ADD COLUMN IF NOT EXISTS "resolved_by_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "requires_acknowledgment" boolean NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS "deleted_at" bigint,
    ADD COLUMN IF NOT EXISTS "deleted_by_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "search_vector" tsvector GENERATED ALWAYS AS (to_tsvector('english', coalesce("comment", ''))) STORED;

--bun:split
ALTER TABLE "shipment_comments"
    DROP CONSTRAINT IF EXISTS "fk_shipment_comments_parent";

--bun:split
ALTER TABLE "shipment_comments"
    ADD CONSTRAINT "fk_shipment_comments_parent" FOREIGN KEY ("parent_comment_id", "business_unit_id", "organization_id", "shipment_id") REFERENCES "shipment_comments"("id", "business_unit_id", "organization_id", "shipment_id") ON UPDATE NO ACTION ON DELETE RESTRICT;

--bun:split
ALTER TABLE "shipment_comments"
    DROP CONSTRAINT IF EXISTS "fk_shipment_comments_pinned_by";

--bun:split
ALTER TABLE "shipment_comments"
    ADD CONSTRAINT "fk_shipment_comments_pinned_by" FOREIGN KEY ("pinned_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL;

--bun:split
ALTER TABLE "shipment_comments"
    DROP CONSTRAINT IF EXISTS "fk_shipment_comments_resolved_by";

--bun:split
ALTER TABLE "shipment_comments"
    ADD CONSTRAINT "fk_shipment_comments_resolved_by" FOREIGN KEY ("resolved_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL;

--bun:split
ALTER TABLE "shipment_comments"
    DROP CONSTRAINT IF EXISTS "fk_shipment_comments_deleted_by";

--bun:split
ALTER TABLE "shipment_comments"
    ADD CONSTRAINT "fk_shipment_comments_deleted_by" FOREIGN KEY ("deleted_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL;

--bun:split
ALTER TABLE "shipment_comments"
    DROP CONSTRAINT IF EXISTS "chk_shipment_comments_no_self_parent";

--bun:split
ALTER TABLE "shipment_comments"
    ADD CONSTRAINT "chk_shipment_comments_no_self_parent" CHECK ("parent_comment_id" IS NULL OR "parent_comment_id" <> "id");

--bun:split
-- Indexes for threading, pinning, resolution, and search
CREATE INDEX IF NOT EXISTS "idx_shipment_comments_top_level_list" ON "shipment_comments"("organization_id", "business_unit_id", "shipment_id", "created_at" DESC, "id" DESC)
WHERE "parent_comment_id" IS NULL;

CREATE INDEX IF NOT EXISTS "idx_shipment_comments_parent" ON "shipment_comments"("parent_comment_id")
WHERE "parent_comment_id" IS NOT NULL;

CREATE INDEX IF NOT EXISTS "idx_shipment_comments_pinned" ON "shipment_comments"("shipment_id", "pinned_at" DESC)
WHERE "pinned_at" IS NOT NULL;

CREATE INDEX IF NOT EXISTS "idx_shipment_comments_unresolved" ON "shipment_comments"("shipment_id")
WHERE "resolved_at" IS NULL AND "deleted_at" IS NULL;

CREATE INDEX IF NOT EXISTS "idx_shipment_comments_search" ON "shipment_comments" USING gin("search_vector");

--bun:split
-- Per-user acknowledgment tracking for comments that request acknowledgment
CREATE TABLE IF NOT EXISTS "shipment_comment_acknowledgments"(
    "id" varchar(100) NOT NULL,
    "comment_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "shipment_id" varchar(100) NOT NULL,
    "user_id" varchar(100) NOT NULL,
    "acknowledged_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    CONSTRAINT "pk_shipment_comment_acknowledgments" PRIMARY KEY ("id"),
    CONSTRAINT "fk_shipment_comment_acks_comment" FOREIGN KEY ("comment_id", "business_unit_id", "organization_id", "shipment_id") REFERENCES "shipment_comments"("id", "business_unit_id", "organization_id", "shipment_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shipment_comment_acks_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shipment_comment_acks_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "uk_shipment_comment_acks_comment_user" UNIQUE ("comment_id", "user_id")
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_shipment_comment_acks_comment" ON "shipment_comment_acknowledgments"("comment_id");

CREATE INDEX IF NOT EXISTS "idx_shipment_comment_acks_user" ON "shipment_comment_acknowledgments"("user_id");

COMMENT ON TABLE "shipment_comment_acknowledgments" IS 'Tracks per-user acknowledgments of shipment comments that request acknowledgment';
