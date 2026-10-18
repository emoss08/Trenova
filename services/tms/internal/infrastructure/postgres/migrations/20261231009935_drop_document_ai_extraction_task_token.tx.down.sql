ALTER TABLE "document_ai_extractions"
    ADD COLUMN IF NOT EXISTS "task_token" BYTEA NOT NULL DEFAULT ''::bytea;
