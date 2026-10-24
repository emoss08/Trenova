-- A production extraction a rollout left out of its comparison, and why. The
-- fine-tuned candidate was trained on legacy text, so a document read by
-- Docling is kept away from it; counting those keeps the exclusion visible
-- instead of letting the rollout's numbers stall without explanation.
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

CREATE INDEX IF NOT EXISTS "idx_extraction_rollout_exclusions_created"
    ON "extraction_rollout_exclusions" ("organization_id", "business_unit_id", "candidate_provider_id", "created_at" DESC);

COMMENT ON TABLE "extraction_rollout_exclusions" IS 'A production extraction a rollout left out of its comparison, with the reason; never an assignment';

SELECT trenova_rls.reconcile();
