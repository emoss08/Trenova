ALTER TABLE "assistant_messages"
    ADD COLUMN IF NOT EXISTS "delegate_report" JSONB;

COMMENT ON COLUMN "assistant_messages"."delegate_report" IS 'The bounded account of the task a delegate_task call handed to another agent (status, answer, writes made and awaiting, documents published), set only on that call''s result';
