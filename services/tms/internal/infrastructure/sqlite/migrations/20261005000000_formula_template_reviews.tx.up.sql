-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261005000000_formula_template_reviews.tx.up.sql

CREATE TABLE IF NOT EXISTS "formula_template_reviews"(
    "id" TEXT NOT NULL,
    "template_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "round" INTEGER NOT NULL,
    "decision" TEXT NOT NULL,
    "actor_id" TEXT,
    "comment" TEXT,
    "base_version_number" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
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
