ALTER TABLE "assistant_messages"
    ADD COLUMN IF NOT EXISTS "hand_off_agents" JSONB;

COMMENT ON COLUMN "assistant_messages"."hand_off_agents" IS 'On a find_tools result: the ids of the person''s other agents it named as holding what the conversation''s agent could not call, so the hand-off menu offers them first; null on every other message';
