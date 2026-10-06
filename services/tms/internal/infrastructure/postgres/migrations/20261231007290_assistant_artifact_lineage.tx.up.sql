-- A table, record or report read again later in the same conversation is a
-- new version of what was read before, not a stranger to it: the Desk lists
-- one artifact with its versions rather than a pile of near-twins.
ALTER TABLE "assistant_artifacts"
    ADD COLUMN IF NOT EXISTS "lineage_key" VARCHAR(64) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS "lineage_id" VARCHAR(100),
    ADD COLUMN IF NOT EXISTS "lineage_seq" INTEGER NOT NULL DEFAULT 1;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_assistant_artifacts_lineage"
    ON "assistant_artifacts" ("organization_id", "business_unit_id", "thread_id", "lineage_key", "created_at" DESC)
    WHERE "lineage_key" <> '';

--bun:split
COMMENT ON COLUMN "assistant_artifacts"."lineage_key" IS 'Digest of the kind, tool and arguments that produced the artifact; a later read of the same source in the same conversation shares it';

--bun:split
COMMENT ON COLUMN "assistant_artifacts"."lineage_id" IS 'The first artifact of the lineage this one is a later version of; null for the first';

--bun:split
COMMENT ON COLUMN "assistant_artifacts"."lineage_seq" IS 'This artifact''s version number within its lineage, from 1';
