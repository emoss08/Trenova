-- Who last said whether a document is signed. A person's answer is final:
-- extraction reads the signature line only while no person has answered.
ALTER TABLE "documents"
    ADD COLUMN IF NOT EXISTS "signature_source" varchar(20),
    ADD COLUMN IF NOT EXISTS "signature_updated_by_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "signature_updated_at" bigint;

ALTER TABLE "documents"
    DROP CONSTRAINT IF EXISTS "ck_documents_signature_source",
    DROP CONSTRAINT IF EXISTS "ck_documents_signature_person",
    DROP CONSTRAINT IF EXISTS "ck_documents_signed_at_signed";

ALTER TABLE "documents"
    ADD CONSTRAINT "ck_documents_signature_source" CHECK ("signature_source" IS NULL OR "signature_source" IN ('Extraction', 'Person'));

-- A person's answer names the person.
ALTER TABLE "documents"
    ADD CONSTRAINT "ck_documents_signature_person" CHECK ("signature_source" IS DISTINCT FROM 'Person' OR "signature_updated_by_id" IS NOT NULL);

-- Only a signed document has a signing date.
ALTER TABLE "documents"
    ADD CONSTRAINT "ck_documents_signed_at_signed" CHECK ("signed_at" IS NULL OR "signature_status" = 'Signed');
