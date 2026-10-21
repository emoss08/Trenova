CREATE TABLE IF NOT EXISTS "extraction_rollout_exclusions" (
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "document_id" varchar(100) NOT NULL,
    "extracted_at" bigint NOT NULL,
    "candidate_provider_id" varchar(100) NOT NULL,
    "reason" varchar(30) NOT NULL,
    "created_at" bigint NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    CONSTRAINT "pk_extraction_rollout_exclusions" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "uq_extraction_rollout_exclusions_extraction" UNIQUE ("organization_id", "business_unit_id", "document_id", "extracted_at"),
    CONSTRAINT "ck_extraction_rollout_exclusions_reason" CHECK ("reason" IN ('docling_input')),
    CONSTRAINT "fk_extraction_rollout_exclusions_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_extraction_rollout_exclusions_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_extraction_rollout_exclusions_created"
    ON "extraction_rollout_exclusions" ("organization_id", "business_unit_id", "candidate_provider_id", "created_at" DESC);

--bun:split
ALTER TABLE "extraction_eval_case_pages" DROP CONSTRAINT IF EXISTS "uq_extraction_eval_case_pages_case";

--bun:split
ALTER TABLE "extraction_eval_case_pages"
    ADD COLUMN IF NOT EXISTS "text_source" varchar(20) NOT NULL DEFAULT 'docling';

--bun:split
ALTER TABLE "extraction_eval_case_pages"
    ADD CONSTRAINT "uq_extraction_eval_case_pages_source" UNIQUE ("organization_id", "business_unit_id", "case_id", "text_source"),
    ADD CONSTRAINT "ck_extraction_eval_case_pages_text_source" CHECK ("text_source" IN ('docling'));

--bun:split
ALTER TABLE "extraction_eval_runs"
    ADD COLUMN IF NOT EXISTS "text_source" varchar(20) NOT NULL DEFAULT 'docling';

--bun:split
ALTER TABLE "extraction_eval_runs"
    ADD CONSTRAINT "ck_extraction_eval_runs_text_source" CHECK ("text_source" IN ('legacy', 'docling'));

--bun:split
ALTER TABLE "document_contents"
    ADD COLUMN IF NOT EXISTS "converter" varchar(20) NOT NULL DEFAULT 'docling';

--bun:split
ALTER TABLE "document_contents"
    ADD CONSTRAINT "ck_document_contents_converter" CHECK ("converter" IN ('legacy', 'docling'));

--bun:split
SELECT trenova_rls.reconcile();
