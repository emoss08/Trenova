-- An AI extraction no longer waits in an activity a global poller completes by
-- task token: its workflow polls the model on its own durable timer, so the
-- token is never stored.
ALTER TABLE "document_ai_extractions"
    DROP COLUMN IF EXISTS "task_token";
