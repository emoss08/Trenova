--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

CREATE TYPE "formula_template_review_decision_enum" AS ENUM(
    'Submitted',
    'Approved',
    'Rejected',
    'ChangesRequested',
    'Expired'
);

--bun:split
CREATE TABLE IF NOT EXISTS "formula_template_reviews"(
    "id" varchar(100) NOT NULL,
    "template_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "round" smallint NOT NULL,
    "decision" formula_template_review_decision_enum NOT NULL,
    "actor_id" varchar(100),
    "comment" text,
    "base_version_number" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_formula_template_reviews" PRIMARY KEY ("id"),
    CONSTRAINT "chk_formula_template_reviews_round" CHECK ("round" >= 1),
    CONSTRAINT "fk_formula_template_reviews_template" FOREIGN KEY ("template_id", "organization_id", "business_unit_id")
        REFERENCES "formula_templates"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "fk_formula_template_reviews_organization" FOREIGN KEY ("organization_id")
        REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_formula_template_reviews_business_unit" FOREIGN KEY ("business_unit_id")
        REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_formula_template_reviews_actor" FOREIGN KEY ("actor_id")
        REFERENCES "users"("id") ON DELETE SET NULL
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_formula_template_reviews_template"
    ON "formula_template_reviews"("template_id", "organization_id", "business_unit_id", "created_at" DESC);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_formula_templates_in_review_since"
    ON "formula_templates"("submitted_at")
    WHERE "status" = 'InReview';
