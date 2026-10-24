ALTER TABLE "document_types"
    ADD COLUMN IF NOT EXISTS "is_system" boolean NOT NULL DEFAULT FALSE;
