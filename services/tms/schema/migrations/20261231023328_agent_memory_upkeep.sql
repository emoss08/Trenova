-- A memory outlives the thing it describes: a dock changes its hours, a customer its rule. A
-- person who reads a memory and says it still holds is recorded here, so AI Control can ask
-- about the facts nobody has stood behind for a long time and the memories no turn has used,
-- and stop asking about the ones a person just confirmed.
ALTER TABLE "agent_memories"
    ADD COLUMN IF NOT EXISTS "confirmed_by_user_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "confirmed_at" bigint;

COMMENT ON COLUMN "agent_memories"."confirmed_by_user_id" IS 'The person who last read the memory and said it still holds';

COMMENT ON COLUMN "agent_memories"."confirmed_at" IS 'When a person last said the memory still holds, as Unix seconds';

ALTER TABLE "agent_memories"
    DROP CONSTRAINT IF EXISTS "chk_agent_memories_confirmed";

ALTER TABLE "agent_memories"
    ADD CONSTRAINT "chk_agent_memories_confirmed" CHECK (
        ("confirmed_at" IS NULL) = ("confirmed_by_user_id" IS NULL)
    );

-- A look back used to start only when a person corrected an agent or used a phrase such as
-- "from now on", so what people say in passing about a customer or a dock was never kept. It
-- now also starts when a person states something while asking for something else; this turns
-- that off for an organization that does not want the extra model call.
ALTER TABLE "agent_controls"
    ADD COLUMN IF NOT EXISTS "passing_learning_off" boolean NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN "agent_controls"."passing_learning_off" IS 'Agents no longer look back over a conversation only because a person stated something in passing';
