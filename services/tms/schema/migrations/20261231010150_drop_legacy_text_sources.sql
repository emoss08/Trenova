-- Docling reads every document, so nothing records which reader did, an
-- evaluation always sends the Docling pages frozen beside its case, and the
-- rollout no longer keeps Docling input away from its candidate.
ALTER TABLE "document_contents" DROP CONSTRAINT IF EXISTS "ck_document_contents_converter";

ALTER TABLE "document_contents" DROP COLUMN IF EXISTS "converter";

ALTER TABLE "extraction_eval_runs" DROP CONSTRAINT IF EXISTS "ck_extraction_eval_runs_text_source";

ALTER TABLE "extraction_eval_runs" DROP COLUMN IF EXISTS "text_source";

ALTER TABLE "extraction_eval_case_pages"
    DROP CONSTRAINT IF EXISTS "uq_extraction_eval_case_pages_source",
    DROP CONSTRAINT IF EXISTS "ck_extraction_eval_case_pages_text_source";

ALTER TABLE "extraction_eval_case_pages" DROP COLUMN IF EXISTS "text_source";

ALTER TABLE "extraction_eval_case_pages"
    ADD CONSTRAINT "uq_extraction_eval_case_pages_case" UNIQUE ("organization_id", "business_unit_id", "case_id");

COMMENT ON TABLE "extraction_eval_case_pages" IS 'A case''s pages as Docling reads them, frozen from its original file and frozen again on request; what every evaluation run sends';

COMMENT ON COLUMN "document_content_pages"."markdown" IS 'The page as Markdown for the model: pipe tables and the printed header under a page-header marker. Empty when Docling produced none for the page, which then falls back to extracted_text';

DROP TABLE IF EXISTS "extraction_rollout_exclusions";
