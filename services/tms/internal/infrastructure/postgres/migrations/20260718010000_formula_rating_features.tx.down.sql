DROP TABLE IF EXISTS "rate_table_entries";

--bun:split
DROP TABLE IF EXISTS "rate_tables";

--bun:split
DROP TYPE IF EXISTS "rate_table_lookup_type_enum";

--bun:split
DROP INDEX IF EXISTS "idx_formula_template_versions_effective";

--bun:split
ALTER TABLE "formula_template_versions"
    DROP COLUMN IF EXISTS "breakdown_definitions",
    DROP COLUMN IF EXISTS "min_charge",
    DROP COLUMN IF EXISTS "max_charge",
    DROP COLUMN IF EXISTS "effective_from";

--bun:split
ALTER TABLE "formula_templates"
    DROP COLUMN IF EXISTS "breakdown_definitions",
    DROP COLUMN IF EXISTS "min_charge",
    DROP COLUMN IF EXISTS "max_charge",
    DROP COLUMN IF EXISTS "submitted_by_id",
    DROP COLUMN IF EXISTS "submitted_at",
    DROP COLUMN IF EXISTS "approved_by_id",
    DROP COLUMN IF EXISTS "approved_at",
    DROP COLUMN IF EXISTS "review_comment";
