DROP TABLE IF EXISTS "formula_template_version_tables";

--bun:split
ALTER TABLE "formula_template_versions"
    DROP COLUMN IF EXISTS "tables";

--bun:split
ALTER TABLE "formula_templates"
    DROP COLUMN IF EXISTS "tables";

--bun:split
DELETE FROM "rate_matrix_cells" WHERE "version_number" <> 0;

--bun:split
DELETE FROM "rate_matrix_dimensions" WHERE "version_number" <> 0;

--bun:split
DROP INDEX IF EXISTS "uq_rate_matrix_cells_exact";

--bun:split
DROP INDEX IF EXISTS "idx_rate_matrix_cells_lookup";

--bun:split
ALTER TABLE "rate_matrix_cells"
    DROP CONSTRAINT IF EXISTS "ck_rate_matrix_cells_version",
    DROP CONSTRAINT IF EXISTS "ck_rate_matrix_cells_d0_postal",
    DROP CONSTRAINT IF EXISTS "ck_rate_matrix_cells_d1_postal",
    DROP CONSTRAINT IF EXISTS "ck_rate_matrix_cells_d2_postal",
    DROP CONSTRAINT IF EXISTS "ck_rate_matrix_cells_d3_postal",
    DROP COLUMN IF EXISTS "version_number",
    DROP COLUMN IF EXISTS "d0_key_to",
    DROP COLUMN IF EXISTS "d1_key_to",
    DROP COLUMN IF EXISTS "d2_key_to",
    DROP COLUMN IF EXISTS "d3_key_to";

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_rate_matrix_cells_exact" ON "rate_matrix_cells"("organization_id", "business_unit_id", "rate_matrix_id", "d0_key", "d1_key", "d2_key", "d3_key")
WHERE
    "d0_min" IS NULL AND "d1_min" IS NULL AND "d2_min" IS NULL AND "d3_min" IS NULL;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_rate_matrix_cells_lookup" ON "rate_matrix_cells"("organization_id", "business_unit_id", "rate_matrix_id", "d0_key", "d1_key", "d2_key", "d3_key", "d0_min", "d1_min", "d2_min", "d3_min");

--bun:split
DROP INDEX IF EXISTS "uq_rate_matrix_dimensions_position";

--bun:split
ALTER TABLE "rate_matrix_dimensions"
    DROP CONSTRAINT IF EXISTS "ck_rate_matrix_dimensions_version",
    DROP COLUMN IF EXISTS "version_number";

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_rate_matrix_dimensions_position" ON "rate_matrix_dimensions"("organization_id", "business_unit_id", "rate_matrix_id", "position");

--bun:split
DROP TABLE IF EXISTS "rate_matrix_reviews";

--bun:split
DROP TABLE IF EXISTS "rate_matrix_versions";

--bun:split
DROP INDEX IF EXISTS "idx_rate_matrices_served";

--bun:split
DROP INDEX IF EXISTS "idx_rate_matrices_in_review";

--bun:split
ALTER TABLE "rate_matrices"
    DROP CONSTRAINT IF EXISTS "fk_rate_matrices_submitted_by",
    DROP CONSTRAINT IF EXISTS "fk_rate_matrices_approved_by",
    DROP CONSTRAINT IF EXISTS "ck_rate_matrices_published_version",
    DROP CONSTRAINT IF EXISTS "ck_rate_matrices_draft_cell_count",
    DROP COLUMN IF EXISTS "submitted_by_id",
    DROP COLUMN IF EXISTS "submitted_at",
    DROP COLUMN IF EXISTS "approved_by_id",
    DROP COLUMN IF EXISTS "approved_at",
    DROP COLUMN IF EXISTS "review_comment",
    DROP COLUMN IF EXISTS "published_version_number",
    DROP COLUMN IF EXISTS "published_at",
    DROP COLUMN IF EXISTS "has_draft_changes",
    DROP COLUMN IF EXISTS "draft_cell_count",
    DROP COLUMN IF EXISTS "draft_editor_ids",
    ALTER COLUMN "status" DROP DEFAULT;

--bun:split
ALTER TABLE "rate_matrices"
    ALTER COLUMN "status" TYPE status_enum
        USING (CASE WHEN "status"::text = 'Inactive' THEN 'Inactive' ELSE 'Active' END)::status_enum,
    ALTER COLUMN "status" SET DEFAULT 'Active';

--bun:split
SELECT trenova_rls.reconcile();
