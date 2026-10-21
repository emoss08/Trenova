ALTER TABLE "document_controls" DROP CONSTRAINT IF EXISTS "ck_document_controls_extraction_engine";

--bun:split
ALTER TABLE "document_controls" DROP COLUMN IF EXISTS "extraction_engine";
