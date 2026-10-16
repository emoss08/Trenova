DROP TABLE IF EXISTS "extraction_eval_case_pages";

--bun:split
ALTER TABLE "extraction_eval_runs" DROP CONSTRAINT IF EXISTS "ck_extraction_eval_runs_text_source";

--bun:split
ALTER TABLE "extraction_eval_runs" DROP COLUMN IF EXISTS "text_source";

--bun:split
ALTER TABLE "document_controls" DROP CONSTRAINT IF EXISTS "ck_document_controls_extraction_engine";

--bun:split
ALTER TABLE "document_controls" DROP COLUMN IF EXISTS "extraction_engine";

--bun:split
ALTER TABLE "document_contents" DROP CONSTRAINT IF EXISTS "ck_document_contents_converter";

--bun:split
ALTER TABLE "document_contents"
    DROP COLUMN IF EXISTS "raw_object_key",
    DROP COLUMN IF EXISTS "converter_version",
    DROP COLUMN IF EXISTS "converter";

--bun:split
ALTER TABLE "document_content_pages" DROP COLUMN IF EXISTS "markdown";
