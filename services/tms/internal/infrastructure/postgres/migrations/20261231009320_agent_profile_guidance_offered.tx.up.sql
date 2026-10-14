-- Guidance about the organization's profile used to be kept the moment it was
-- given. It shapes what every agent is told, so it is now an administrator's to
-- keep: a line from anyone else, or written down by an agent, is offered
-- (Suggested) until one approves it, or dismissed. A dismissed line keeps who
-- closed it and when, as a retired one does.
ALTER TABLE "agent_profile_guidance"
    DROP CONSTRAINT IF EXISTS "ck_agent_profile_guidance_status";

--bun:split
ALTER TABLE "agent_profile_guidance"
    ADD CONSTRAINT "ck_agent_profile_guidance_status" CHECK ("status" IN ('Active', 'Suggested', 'Retired', 'Dismissed'));

--bun:split
ALTER TABLE "agent_profile_guidance"
    DROP CONSTRAINT IF EXISTS "ck_agent_profile_guidance_retired";

--bun:split
ALTER TABLE "agent_profile_guidance"
    ADD CONSTRAINT "ck_agent_profile_guidance_retired" CHECK (("status" IN ('Retired', 'Dismissed')) = ("retired_at" IS NOT NULL));

--bun:split
CREATE INDEX IF NOT EXISTS "idx_agent_profile_guidance_suggested" ON "agent_profile_guidance"("organization_id", "business_unit_id", "subject", "created_at")
WHERE
    "status" = 'Suggested';
