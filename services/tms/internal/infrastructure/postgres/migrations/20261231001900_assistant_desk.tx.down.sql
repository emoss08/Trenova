DROP TABLE IF EXISTS "assistant_artifacts";

--bun:split
DROP INDEX IF EXISTS "idx_assistant_threads_user_pinned";

--bun:split
ALTER TABLE "assistant_threads"
    DROP CONSTRAINT IF EXISTS "ck_assistant_threads_origin";

--bun:split
ALTER TABLE "assistant_threads"
    DROP COLUMN IF EXISTS "subject_id";

--bun:split
ALTER TABLE "assistant_threads"
    DROP COLUMN IF EXISTS "subject_type";

--bun:split
ALTER TABLE "assistant_threads"
    DROP COLUMN IF EXISTS "pinned";

--bun:split
ALTER TABLE "assistant_threads"
    DROP COLUMN IF EXISTS "origin";
