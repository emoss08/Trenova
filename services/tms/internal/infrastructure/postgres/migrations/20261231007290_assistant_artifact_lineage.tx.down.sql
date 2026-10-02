DROP INDEX IF EXISTS "idx_assistant_artifacts_lineage";

--bun:split
ALTER TABLE "assistant_artifacts"
    DROP COLUMN IF EXISTS "lineage_seq",
    DROP COLUMN IF EXISTS "lineage_id",
    DROP COLUMN IF EXISTS "lineage_key";
