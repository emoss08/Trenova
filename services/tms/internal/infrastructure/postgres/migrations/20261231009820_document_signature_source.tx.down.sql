ALTER TABLE "documents"
    DROP CONSTRAINT IF EXISTS "ck_documents_signed_at_signed",
    DROP CONSTRAINT IF EXISTS "ck_documents_signature_person",
    DROP CONSTRAINT IF EXISTS "ck_documents_signature_source",
    DROP COLUMN IF EXISTS "signature_updated_at",
    DROP COLUMN IF EXISTS "signature_updated_by_id",
    DROP COLUMN IF EXISTS "signature_source";
