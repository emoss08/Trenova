-- Document Intelligence is metered in pages, not calls: a managed instance reports the pages of
-- the documents it extracted each hour. An extraction knows its document's page count before it
-- calls the model, keeps it on its record so the deferred answer is attributed the same way, and
-- writes it on the usage record of the call that answered.
ALTER TABLE "ai_usage_records"
    ADD COLUMN IF NOT EXISTS "pages" integer NOT NULL DEFAULT 0;

ALTER TABLE "ai_usage_records"
    DROP CONSTRAINT IF EXISTS "ck_ai_usage_records_pages";

ALTER TABLE "ai_usage_records"
    ADD CONSTRAINT "ck_ai_usage_records_pages" CHECK ("pages" >= 0);

COMMENT ON COLUMN "ai_usage_records"."pages" IS 'The pages of the document a Document Intelligence extraction read; zero for every other call';

ALTER TABLE "document_ai_extractions"
    ADD COLUMN IF NOT EXISTS "page_count" integer NOT NULL DEFAULT 0;

ALTER TABLE "document_ai_extractions"
    DROP CONSTRAINT IF EXISTS "ck_document_ai_extractions_page_count";

ALTER TABLE "document_ai_extractions"
    ADD CONSTRAINT "ck_document_ai_extractions_page_count" CHECK ("page_count" >= 0);

COMMENT ON COLUMN "document_ai_extractions"."page_count" IS 'The pages of the document when the extraction was submitted';
