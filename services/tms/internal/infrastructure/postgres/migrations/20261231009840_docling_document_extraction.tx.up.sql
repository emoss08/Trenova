ALTER TABLE "document_content_pages"
    ADD COLUMN IF NOT EXISTS "markdown" text;

--bun:split
ALTER TABLE "document_contents"
    ADD COLUMN IF NOT EXISTS "converter" varchar(20),
    ADD COLUMN IF NOT EXISTS "converter_version" varchar(40),
    ADD COLUMN IF NOT EXISTS "raw_object_key" text;

--bun:split
UPDATE "document_contents" SET "converter" = 'legacy' WHERE "converter" IS NULL;

--bun:split
ALTER TABLE "document_contents"
    ALTER COLUMN "converter" SET DEFAULT 'legacy',
    ALTER COLUMN "converter" SET NOT NULL,
    ADD CONSTRAINT "ck_document_contents_converter" CHECK ("converter" IN ('legacy', 'docling'));

--bun:split
COMMENT ON COLUMN "document_content_pages"."markdown" IS 'The page as Markdown for the model: pipe tables and the printed header under a page-header marker. Empty for pages read before Docling, which fall back to extracted_text';

--bun:split
COMMENT ON COLUMN "document_contents"."converter" IS 'What read the file: legacy (PDFium, Tesseract, the Office reader) or docling';

--bun:split
COMMENT ON COLUMN "document_contents"."raw_object_key" IS 'Object storage key of the gzipped DoclingDocument, so pages can be rendered again without converting the file again';

--bun:split
ALTER TABLE "document_controls"
    ADD COLUMN IF NOT EXISTS "extraction_engine" varchar(10) NOT NULL DEFAULT 'legacy';

--bun:split
ALTER TABLE "document_controls"
    ADD CONSTRAINT "ck_document_controls_extraction_engine" CHECK ("extraction_engine" IN ('legacy', 'docling'));

--bun:split
CREATE TABLE IF NOT EXISTS "extraction_eval_case_pages" (
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "case_id" varchar(100) NOT NULL,
    "text_source" varchar(20) NOT NULL,
    "pages" jsonb NOT NULL,
    "page_count" integer NOT NULL DEFAULT 0,
    "input_hash" varchar(64) NOT NULL,
    "converter_version" varchar(40),
    "frozen_by_id" varchar(100) NOT NULL,
    "frozen_at" bigint NOT NULL,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_extraction_eval_case_pages" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "uq_extraction_eval_case_pages_source" UNIQUE ("organization_id", "business_unit_id", "case_id", "text_source"),
    CONSTRAINT "ck_extraction_eval_case_pages_text_source" CHECK ("text_source" IN ('docling')),
    CONSTRAINT "ck_extraction_eval_case_pages_input_hash" CHECK (char_length("input_hash") = 64),
    CONSTRAINT "fk_extraction_eval_case_pages_case" FOREIGN KEY ("case_id", "business_unit_id", "organization_id") REFERENCES "extraction_eval_cases"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_extraction_eval_case_pages_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_extraction_eval_case_pages_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
ALTER TABLE "extraction_eval_runs"
    ADD COLUMN IF NOT EXISTS "text_source" varchar(20) NOT NULL DEFAULT 'legacy';

--bun:split
ALTER TABLE "extraction_eval_runs"
    ADD CONSTRAINT "ck_extraction_eval_runs_text_source" CHECK ("text_source" IN ('legacy', 'docling'));

--bun:split
COMMENT ON TABLE "extraction_eval_case_pages" IS 'A case''s page text frozen again from its original file through another reader, so the same cases can be run on each text source. The legacy pages stay on the case row';

--bun:split
SELECT trenova_rls.reconcile();
