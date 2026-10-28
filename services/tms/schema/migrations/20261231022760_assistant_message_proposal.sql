ALTER TABLE "assistant_messages"
    ADD COLUMN IF NOT EXISTS "proposal_id" VARCHAR(100);

COMMENT ON COLUMN "assistant_messages"."proposal_id" IS 'On a Schedule card an agent proposed: the proposal whose approval made it, so the card reads as the approved proposal rather than as words the person typed; null on every other message';
