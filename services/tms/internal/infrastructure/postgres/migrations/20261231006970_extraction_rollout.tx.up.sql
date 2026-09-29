-- How an organization serves a candidate AI provider to a share of its real
-- document extractions, and the guards that stop it when the candidate does
-- worse than production.
CREATE TABLE IF NOT EXISTS "extraction_rollouts" (
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "enabled" boolean NOT NULL DEFAULT FALSE,
    "provider_id" varchar(100),
    "percent" integer NOT NULL DEFAULT 5,
    "max_accuracy_drop_points" integer NOT NULL DEFAULT 5,
    "max_rejection_increase_points" integer NOT NULL DEFAULT 10,
    "started_at" bigint,
    "halted_at" bigint,
    "halt_reason" varchar(30),
    "halt_candidate_rate" double precision NOT NULL DEFAULT 0,
    "halt_baseline_rate" double precision NOT NULL DEFAULT 0,
    "updated_by_id" varchar(100),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    CONSTRAINT "pk_extraction_rollouts" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "uq_extraction_rollouts_tenant" UNIQUE ("organization_id", "business_unit_id"),
    CONSTRAINT "ck_extraction_rollouts_percent" CHECK ("percent" BETWEEN 1 AND 100),
    CONSTRAINT "ck_extraction_rollouts_accuracy_guard" CHECK ("max_accuracy_drop_points" BETWEEN 1 AND 50),
    CONSTRAINT "ck_extraction_rollouts_rejection_guard" CHECK ("max_rejection_increase_points" BETWEEN 1 AND 100),
    CONSTRAINT "ck_extraction_rollouts_provider" CHECK (NOT "enabled" OR "provider_id" IS NOT NULL),
    CONSTRAINT "ck_extraction_rollouts_started" CHECK (NOT "enabled" OR "started_at" IS NOT NULL),
    CONSTRAINT "ck_extraction_rollouts_halt_reason" CHECK ("halt_reason" IS NULL OR "halt_reason" IN ('AccuracyDrop', 'Rejections')),
    CONSTRAINT "ck_extraction_rollouts_halt" CHECK (("halted_at" IS NULL) = ("halt_reason" IS NULL)),
    CONSTRAINT "fk_extraction_rollouts_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_extraction_rollouts_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
-- Which side of the rollout one production extraction was assigned to, which
-- provider actually served it, and whether its answer was used.
CREATE TABLE IF NOT EXISTS "extraction_rollout_assignments" (
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "document_id" varchar(100) NOT NULL,
    "extracted_at" bigint NOT NULL,
    "arm" varchar(20) NOT NULL,
    "candidate_provider_id" varchar(100) NOT NULL,
    "served_provider_id" varchar(100),
    "served_model" varchar(255),
    "outcome" varchar(20) NOT NULL DEFAULT 'Pending',
    "settled_at" bigint,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    CONSTRAINT "pk_extraction_rollout_assignments" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "uq_extraction_rollout_assignments_extraction" UNIQUE ("organization_id", "business_unit_id", "document_id", "extracted_at"),
    CONSTRAINT "ck_extraction_rollout_assignments_arm" CHECK ("arm" IN ('Candidate', 'Control')),
    CONSTRAINT "ck_extraction_rollout_assignments_outcome" CHECK ("outcome" IN ('Pending', 'Accepted', 'Rejected', 'Failed', 'Superseded')),
    CONSTRAINT "ck_extraction_rollout_assignments_settled" CHECK (("outcome" = 'Pending') = ("settled_at" IS NULL)),
    CONSTRAINT "fk_extraction_rollout_assignments_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_extraction_rollout_assignments_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_extraction_rollout_assignments_created"
    ON "extraction_rollout_assignments" ("organization_id", "business_unit_id", "candidate_provider_id", "created_at" DESC);

COMMENT ON TABLE "extraction_rollouts" IS 'Which candidate AI provider serves a share of an organization''s real document extractions, and the guards that stop it';

COMMENT ON COLUMN "extraction_rollouts"."percent" IS 'The share of documents whose extraction asks the candidate first, chosen by a hash of the document';

COMMENT ON COLUMN "extraction_rollouts"."started_at" IS 'When the current comparison began; the guards and report count only what happened since';

COMMENT ON COLUMN "extraction_rollouts"."halted_at" IS 'When a guard stopped the rollout; cleared when a person starts it again';

COMMENT ON TABLE "extraction_rollout_assignments" IS 'One production extraction during a rollout: the side it was assigned to, the provider that served it, and whether its answer was used';
