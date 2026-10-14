DROP INDEX IF EXISTS "idx_agent_profile_guidance_suggested";

--bun:split
UPDATE "agent_profile_guidance"
SET "status" = 'Retired', "retired_at" = COALESCE("retired_at", "updated_at")
WHERE "status" IN ('Suggested', 'Dismissed');

--bun:split
ALTER TABLE "agent_profile_guidance"
    DROP CONSTRAINT IF EXISTS "ck_agent_profile_guidance_retired";

--bun:split
ALTER TABLE "agent_profile_guidance"
    ADD CONSTRAINT "ck_agent_profile_guidance_retired" CHECK (("status" = 'Retired') = ("retired_at" IS NOT NULL));

--bun:split
ALTER TABLE "agent_profile_guidance"
    DROP CONSTRAINT IF EXISTS "ck_agent_profile_guidance_status";

--bun:split
ALTER TABLE "agent_profile_guidance"
    ADD CONSTRAINT "ck_agent_profile_guidance_status" CHECK ("status" IN ('Active', 'Retired'));
