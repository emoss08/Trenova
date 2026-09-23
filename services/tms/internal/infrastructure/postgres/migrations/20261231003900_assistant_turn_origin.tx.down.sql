ALTER TABLE "assistant_turns"
    DROP CONSTRAINT IF EXISTS "ck_assistant_turns_origin";

--bun:split

ALTER TABLE "assistant_turns"
    DROP COLUMN IF EXISTS "input";

--bun:split

ALTER TABLE "assistant_turns"
    DROP COLUMN IF EXISTS "origin";
