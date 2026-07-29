--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md
--
DROP TABLE IF EXISTS "shipment_comment_acknowledgments";

--bun:split
DROP INDEX IF EXISTS "idx_shipment_comments_top_level_list";

DROP INDEX IF EXISTS "idx_shipment_comments_parent";

DROP INDEX IF EXISTS "idx_shipment_comments_pinned";

DROP INDEX IF EXISTS "idx_shipment_comments_unresolved";

DROP INDEX IF EXISTS "idx_shipment_comments_search";

--bun:split
-- Replies cannot survive without threading support
DELETE FROM "shipment_comments"
WHERE "parent_comment_id" IS NOT NULL;

--bun:split
ALTER TABLE "shipment_comments"
    DROP CONSTRAINT IF EXISTS "fk_shipment_comments_parent",
    DROP CONSTRAINT IF EXISTS "fk_shipment_comments_pinned_by",
    DROP CONSTRAINT IF EXISTS "fk_shipment_comments_resolved_by",
    DROP CONSTRAINT IF EXISTS "fk_shipment_comments_deleted_by",
    DROP CONSTRAINT IF EXISTS "chk_shipment_comments_no_self_parent";

--bun:split
ALTER TABLE "shipment_comments"
    DROP COLUMN IF EXISTS "parent_comment_id",
    DROP COLUMN IF EXISTS "body",
    DROP COLUMN IF EXISTS "pinned_at",
    DROP COLUMN IF EXISTS "pinned_by_id",
    DROP COLUMN IF EXISTS "resolved_at",
    DROP COLUMN IF EXISTS "resolved_by_id",
    DROP COLUMN IF EXISTS "requires_acknowledgment",
    DROP COLUMN IF EXISTS "deleted_at",
    DROP COLUMN IF EXISTS "deleted_by_id",
    DROP COLUMN IF EXISTS "search_vector";

--bun:split
-- System comments carry no user; they cannot survive restoring the NOT NULL constraint
DELETE FROM "shipment_comments"
WHERE "user_id" IS NULL;

--bun:split
ALTER TABLE "shipment_comments"
    ALTER COLUMN "user_id" SET NOT NULL;
