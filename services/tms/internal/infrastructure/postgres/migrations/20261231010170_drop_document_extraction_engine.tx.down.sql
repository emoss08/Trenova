ALTER TABLE "document_controls"
    ADD COLUMN IF NOT EXISTS "extraction_engine" varchar(10) NOT NULL DEFAULT 'legacy';

--bun:split
ALTER TABLE "document_controls"
    ADD CONSTRAINT "ck_document_controls_extraction_engine" CHECK ("extraction_engine" IN ('legacy', 'docling'));
