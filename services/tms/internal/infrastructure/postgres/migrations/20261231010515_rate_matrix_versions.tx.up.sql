-- Rate matrices are versioned the way formula libraries are.
--
-- The matrix row and its version-zero axes and cells are the draft an editor
-- changes. Approving a submitted draft copies it into an immutable published
-- version (rate_matrix_versions with its axes and cells under that number),
-- and a formula template version pins the published version of every table
-- it reads. Publishing a new version never reprices a contract: adopting it
-- means approving a new template version.
--
-- Every matrix that exists becomes published version 1 of itself, so tariffs
-- keep reading exactly the cells they read before.
ALTER TABLE "rate_matrices"
    ALTER COLUMN "status" DROP DEFAULT;

--bun:split
ALTER TABLE "rate_matrices"
    ALTER COLUMN "status" TYPE formula_template_status_enum
        USING (CASE WHEN "status"::text = 'Active' THEN 'Active' ELSE 'Inactive' END)::formula_template_status_enum,
    ALTER COLUMN "status" SET DEFAULT 'Draft',
    ADD COLUMN IF NOT EXISTS "submitted_by_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "submitted_at" bigint,
    ADD COLUMN IF NOT EXISTS "approved_by_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "approved_at" bigint,
    ADD COLUMN IF NOT EXISTS "review_comment" text,
    ADD COLUMN IF NOT EXISTS "published_version_number" bigint NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS "published_at" bigint,
    ADD COLUMN IF NOT EXISTS "has_draft_changes" boolean NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS "draft_cell_count" bigint NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS "draft_editor_ids" jsonb NOT NULL DEFAULT '[]',
    ADD CONSTRAINT "fk_rate_matrices_submitted_by" FOREIGN KEY ("submitted_by_id")
        REFERENCES "users"("id") ON DELETE SET NULL,
    ADD CONSTRAINT "fk_rate_matrices_approved_by" FOREIGN KEY ("approved_by_id")
        REFERENCES "users"("id") ON DELETE SET NULL,
    ADD CONSTRAINT "ck_rate_matrices_published_version" CHECK ("published_version_number" >= 0),
    ADD CONSTRAINT "ck_rate_matrices_draft_cell_count" CHECK ("draft_cell_count" >= 0);

--bun:split
-- The tables a tariff reads by default: every matrix with a published version
-- that is not archived.
CREATE INDEX IF NOT EXISTS "idx_rate_matrices_served"
    ON "rate_matrices"("organization_id", "business_unit_id")
    INCLUDE ("published_version_number")
    WHERE "published_version_number" > 0 AND "status" <> 'Inactive';

--bun:split
-- The sweep that returns stale submissions to draft.
CREATE INDEX IF NOT EXISTS "idx_rate_matrices_in_review"
    ON "rate_matrices"("submitted_at")
    WHERE "status" = 'InReview';

--bun:split
CREATE TABLE IF NOT EXISTS "rate_matrix_versions"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "rate_matrix_id" varchar(100) NOT NULL,
    "version_number" bigint NOT NULL,
    "code" varchar(64) NOT NULL,
    "name" varchar(100) NOT NULL,
    "description" text,
    "value_type" varchar(32) NOT NULL,
    "cell_count" bigint NOT NULL DEFAULT 0,
    "change_message" text,
    "submitted_by_id" varchar(100),
    "published_by_id" varchar(100),
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_rate_matrix_versions" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "uk_rate_matrix_versions_number"
        UNIQUE ("rate_matrix_id", "organization_id", "business_unit_id", "version_number"),
    CONSTRAINT "fk_rate_matrix_versions_matrix" FOREIGN KEY ("rate_matrix_id", "organization_id", "business_unit_id")
        REFERENCES "rate_matrices"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "fk_rate_matrix_versions_organization" FOREIGN KEY ("organization_id")
        REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_rate_matrix_versions_business_unit" FOREIGN KEY ("business_unit_id")
        REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_rate_matrix_versions_submitted_by" FOREIGN KEY ("submitted_by_id")
        REFERENCES "users"("id") ON DELETE SET NULL,
    CONSTRAINT "fk_rate_matrix_versions_published_by" FOREIGN KEY ("published_by_id")
        REFERENCES "users"("id") ON DELETE SET NULL,
    CONSTRAINT "ck_rate_matrix_versions_number" CHECK ("version_number" >= 1),
    CONSTRAINT "ck_rate_matrix_versions_cell_count" CHECK ("cell_count" >= 0),
    CONSTRAINT "ck_rate_matrix_versions_value_type" CHECK ("value_type" IN ('Money', 'Percent', 'Number', 'Int') OR "value_type" ~ '^Money/(lb|cwt|kg|t|ton|mi|km|in|ft|m|ft3|m3|gal|l|min|h|d|pallet|piece|stop)$')
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_rate_matrix_versions_history"
    ON "rate_matrix_versions"("organization_id", "business_unit_id", "rate_matrix_id", "version_number" DESC);

--bun:split
COMMENT ON TABLE rate_matrix_versions IS 'Published, immutable versions of a rate matrix; formula template versions pin the version of every table they read';

--bun:split
CREATE TABLE IF NOT EXISTS "rate_matrix_reviews"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "rate_matrix_id" varchar(100) NOT NULL,
    "round" smallint NOT NULL,
    "decision" formula_template_review_decision_enum NOT NULL,
    "actor_id" varchar(100),
    "comment" text,
    "base_version_number" bigint NOT NULL DEFAULT 0,
    "requested_reviewer_id" varchar(100),
    "reviewer_note" text,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_rate_matrix_reviews" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_rate_matrix_reviews_matrix" FOREIGN KEY ("rate_matrix_id", "organization_id", "business_unit_id")
        REFERENCES "rate_matrices"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "fk_rate_matrix_reviews_organization" FOREIGN KEY ("organization_id")
        REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_rate_matrix_reviews_business_unit" FOREIGN KEY ("business_unit_id")
        REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_rate_matrix_reviews_actor" FOREIGN KEY ("actor_id")
        REFERENCES "users"("id") ON DELETE SET NULL,
    CONSTRAINT "fk_rate_matrix_reviews_requested_reviewer" FOREIGN KEY ("requested_reviewer_id")
        REFERENCES "users"("id") ON DELETE SET NULL,
    CONSTRAINT "ck_rate_matrix_reviews_round" CHECK ("round" >= 1),
    CONSTRAINT "ck_rate_matrix_reviews_reviewer_note_length" CHECK (
        "reviewer_note" IS NULL OR char_length("reviewer_note") <= 4000
    )
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_rate_matrix_reviews_matrix"
    ON "rate_matrix_reviews"("organization_id", "business_unit_id", "rate_matrix_id", "created_at" DESC);

--bun:split
DROP INDEX IF EXISTS "uq_rate_matrix_dimensions_position";

--bun:split
ALTER TABLE "rate_matrix_dimensions"
    ADD COLUMN IF NOT EXISTS "version_number" bigint NOT NULL DEFAULT 0,
    ADD CONSTRAINT "ck_rate_matrix_dimensions_version" CHECK ("version_number" >= 0);

--bun:split
-- Version zero is the draft; every other number is a published version.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_rate_matrix_dimensions_position"
    ON "rate_matrix_dimensions"("organization_id", "business_unit_id", "rate_matrix_id", "version_number", "position");

--bun:split
DROP INDEX IF EXISTS "uq_rate_matrix_cells_exact";

--bun:split
DROP INDEX IF EXISTS "idx_rate_matrix_cells_lookup";

--bun:split
-- A zip-range axis keeps its range's first postal code in dN_key and its
-- last in dN_key_to; every other axis leaves dN_key_to empty.
ALTER TABLE "rate_matrix_cells"
    ADD COLUMN IF NOT EXISTS "version_number" bigint NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS "d0_key_to" varchar(120) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS "d1_key_to" varchar(120) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS "d2_key_to" varchar(120) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS "d3_key_to" varchar(120) NOT NULL DEFAULT '',
    ADD CONSTRAINT "ck_rate_matrix_cells_version" CHECK ("version_number" >= 0),
    ADD CONSTRAINT "ck_rate_matrix_cells_d0_postal" CHECK ("d0_key_to" = '' OR (char_length("d0_key") = char_length("d0_key_to") AND "d0_key" COLLATE "C" <= "d0_key_to" COLLATE "C")),
    ADD CONSTRAINT "ck_rate_matrix_cells_d1_postal" CHECK ("d1_key_to" = '' OR (char_length("d1_key") = char_length("d1_key_to") AND "d1_key" COLLATE "C" <= "d1_key_to" COLLATE "C")),
    ADD CONSTRAINT "ck_rate_matrix_cells_d2_postal" CHECK ("d2_key_to" = '' OR (char_length("d2_key") = char_length("d2_key_to") AND "d2_key" COLLATE "C" <= "d2_key_to" COLLATE "C")),
    ADD CONSTRAINT "ck_rate_matrix_cells_d3_postal" CHECK ("d3_key_to" = '' OR (char_length("d3_key") = char_length("d3_key_to") AND "d3_key" COLLATE "C" <= "d3_key_to" COLLATE "C"));

--bun:split
-- A wholly keyed version has one cell per key combination, and duplicates
-- would make the rate depend on which row came back first.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_rate_matrix_cells_exact"
    ON "rate_matrix_cells"("organization_id", "business_unit_id", "rate_matrix_id", "version_number", "d0_key", "d1_key", "d2_key", "d3_key")
    WHERE "d0_min" IS NULL AND "d1_min" IS NULL AND "d2_min" IS NULL AND "d3_min" IS NULL;

--bun:split
-- The lookup path, keyed by (matrix, version): exact keys and postal range
-- starts lead so the index stays selective, and banded axes are narrowed by
-- their bounds in the module.
CREATE INDEX IF NOT EXISTS "idx_rate_matrix_cells_lookup"
    ON "rate_matrix_cells"("organization_id", "business_unit_id", "rate_matrix_id", "version_number", "d0_key", "d1_key", "d2_key", "d3_key", "d0_min", "d1_min", "d2_min", "d3_min");

--bun:split
INSERT INTO "rate_matrix_versions"(
    "id", "organization_id", "business_unit_id", "rate_matrix_id", "version_number",
    "code", "name", "description", "value_type", "cell_count", "change_message", "created_at"
)
SELECT
    'rmv_' || upper(substr(md5(rmx."id" || ':v1'), 1, 26)),
    rmx."organization_id",
    rmx."business_unit_id",
    rmx."id",
    1,
    rmx."code",
    rmx."name",
    rmx."description",
    rmx."value_type",
    (SELECT count(*) FROM "rate_matrix_cells" rmc
      WHERE rmc."rate_matrix_id" = rmx."id"
        AND rmc."organization_id" = rmx."organization_id"
        AND rmc."business_unit_id" = rmx."business_unit_id"),
    'Published when rate matrices became versioned',
    rmx."updated_at"
FROM "rate_matrices" rmx
ON CONFLICT DO NOTHING;

--bun:split
INSERT INTO "rate_matrix_dimensions"(
    "id", "business_unit_id", "organization_id", "rate_matrix_id", "version_number",
    "position", "kind", "match_mode", "label", "key_normalization", "range_overflow",
    "created_at", "updated_at"
)
SELECT
    'rmd_' || upper(substr(md5(rmd."id" || ':v1'), 1, 26)),
    rmd."business_unit_id",
    rmd."organization_id",
    rmd."rate_matrix_id",
    1,
    rmd."position",
    rmd."kind",
    rmd."match_mode",
    rmd."label",
    rmd."key_normalization",
    rmd."range_overflow",
    rmd."created_at",
    rmd."updated_at"
FROM "rate_matrix_dimensions" rmd
WHERE rmd."version_number" = 0
ON CONFLICT DO NOTHING;

--bun:split
INSERT INTO "rate_matrix_cells"(
    "id", "business_unit_id", "organization_id", "rate_matrix_id", "version_number",
    "d0_key", "d1_key", "d2_key", "d3_key",
    "d0_key_to", "d1_key_to", "d2_key_to", "d3_key_to",
    "d0_min", "d0_max", "d1_min", "d1_max", "d2_min", "d2_max", "d3_min", "d3_max",
    "value", "min_charge", "deficit_eligible", "created_at", "updated_at"
)
SELECT
    'rmc_' || upper(substr(md5(rmc."id" || ':v1'), 1, 26)),
    rmc."business_unit_id",
    rmc."organization_id",
    rmc."rate_matrix_id",
    1,
    rmc."d0_key", rmc."d1_key", rmc."d2_key", rmc."d3_key",
    rmc."d0_key_to", rmc."d1_key_to", rmc."d2_key_to", rmc."d3_key_to",
    rmc."d0_min", rmc."d0_max", rmc."d1_min", rmc."d1_max",
    rmc."d2_min", rmc."d2_max", rmc."d3_min", rmc."d3_max",
    rmc."value", rmc."min_charge", rmc."deficit_eligible", rmc."created_at", rmc."updated_at"
FROM "rate_matrix_cells" rmc
WHERE rmc."version_number" = 0
ON CONFLICT DO NOTHING;

--bun:split
UPDATE "rate_matrices" rmx
SET
    "published_version_number" = 1,
    "published_at" = rmx."updated_at",
    "has_draft_changes" = FALSE,
    "draft_cell_count" = rmv."cell_count"
FROM "rate_matrix_versions" rmv
WHERE rmv."rate_matrix_id" = rmx."id"
  AND rmv."organization_id" = rmx."organization_id"
  AND rmv."business_unit_id" = rmx."business_unit_id"
  AND rmv."version_number" = 1;

--bun:split
ALTER TABLE "formula_templates"
    ADD COLUMN IF NOT EXISTS "tables" jsonb NOT NULL DEFAULT '[]';

--bun:split
ALTER TABLE "formula_template_versions"
    ADD COLUMN IF NOT EXISTS "tables" jsonb NOT NULL DEFAULT '[]';

--bun:split
CREATE TABLE IF NOT EXISTS "formula_template_version_tables"(
    "template_version_id" varchar(100) NOT NULL,
    "rate_matrix_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "template_id" varchar(100) NOT NULL,
    "template_version" bigint NOT NULL,
    "matrix_version_number" bigint NOT NULL,
    "code" varchar(64) NOT NULL,
    CONSTRAINT "pk_formula_template_version_tables"
        PRIMARY KEY ("organization_id", "business_unit_id", "template_version_id", "rate_matrix_id"),
    CONSTRAINT "fk_formula_template_version_tables_version" FOREIGN KEY ("template_version_id")
        REFERENCES "formula_template_versions"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_formula_template_version_tables_template" FOREIGN KEY ("template_id", "organization_id", "business_unit_id")
        REFERENCES "formula_templates"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "fk_formula_template_version_tables_matrix_version"
        FOREIGN KEY ("rate_matrix_id", "organization_id", "business_unit_id", "matrix_version_number")
        REFERENCES "rate_matrix_versions"("rate_matrix_id", "organization_id", "business_unit_id", "version_number")
        ON DELETE CASCADE,
    CONSTRAINT "fk_formula_template_version_tables_organization" FOREIGN KEY ("organization_id")
        REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_formula_template_version_tables_business_unit" FOREIGN KEY ("business_unit_id")
        REFERENCES "business_units"("id") ON DELETE CASCADE
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_formula_template_version_tables_matrix"
    ON "formula_template_version_tables"("organization_id", "business_unit_id", "rate_matrix_id", "matrix_version_number");

--bun:split
SELECT trenova_rls.reconcile();
