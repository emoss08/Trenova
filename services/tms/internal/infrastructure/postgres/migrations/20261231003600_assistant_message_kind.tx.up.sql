ALTER TABLE "assistant_messages"
    ADD COLUMN IF NOT EXISTS "kind" VARCHAR(50) NOT NULL DEFAULT 'Message';

ALTER TABLE "assistant_messages"
    ADD CONSTRAINT "ck_assistant_messages_kind" CHECK ("kind" IN ('Message', 'DecisionNote'));

COMMENT ON COLUMN "assistant_messages"."kind" IS 'Message for what a person or the model wrote; DecisionNote for the input of the turn that follows a decision on a proposal';
