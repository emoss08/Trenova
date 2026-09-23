ALTER TABLE "assistant_turns"
    ADD COLUMN IF NOT EXISTS "origin" VARCHAR(30) NOT NULL DEFAULT 'Person';

--bun:split

ALTER TABLE "assistant_turns"
    ADD COLUMN IF NOT EXISTS "input" TEXT;

--bun:split

ALTER TABLE "assistant_turns"
    ADD CONSTRAINT "ck_assistant_turns_origin" CHECK ("origin" IN ('Person', 'DecisionFollowUp'));

--bun:split

COMMENT ON COLUMN "assistant_turns"."origin" IS 'Person for a question somebody asked; DecisionFollowUp for the turn in which the agent reports what came of a decided proposal';

--bun:split

COMMENT ON COLUMN "assistant_turns"."input" IS 'The question the turn answers, so a reader rejoining a reply in progress can show it';
