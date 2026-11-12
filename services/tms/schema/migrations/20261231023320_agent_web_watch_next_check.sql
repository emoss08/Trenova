ALTER TABLE "agent_web_watches"
    ADD COLUMN IF NOT EXISTS "next_check_at" bigint;

COMMENT ON COLUMN "agent_web_watches"."next_check_at" IS 'When the minute sweep next checks the watch with its vendor; null once a check found it ended';

UPDATE
    "agent_web_watches"
SET
    "next_check_at" = EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint
WHERE
    "status" = 'Active'
    AND "next_check_at" IS NULL;
