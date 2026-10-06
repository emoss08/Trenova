-- The wording a person changed on a pending proposal and has not yet approved
-- with, so the edit outlives the page it was made on. Cleared once the
-- proposal is decided; the decision keeps what was approved.
ALTER TABLE "agent_proposals"
    ADD COLUMN IF NOT EXISTS "pending_modifications" jsonb;
