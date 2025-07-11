SET statement_timeout = 0;

--bun:split
-- Drop triggers first
DROP TRIGGER IF EXISTS formula_templates_update_trigger ON formula_templates;

--bun:split
-- Drop the trigger function
DROP FUNCTION IF EXISTS formula_templates_update_timestamps();

--bun:split
-- Drop the table (this will cascade delete all data and foreign key references)
DROP TABLE IF EXISTS "formula_templates";

--bun:split
-- Drop the custom type
DROP TYPE IF EXISTS formula_template_category_enum;

SET statement_timeout = 0;

--bun:split
-- Drop the constraint first
ALTER TABLE "shipments"
    DROP CONSTRAINT IF EXISTS "chk_shipments_formula_template_required";

--bun:split
-- Drop the foreign key constraint
ALTER TABLE "shipments"
    DROP CONSTRAINT IF EXISTS "fk_shipments_formula_template";

--bun:split
-- Drop the column
ALTER TABLE "shipments"
    DROP COLUMN IF EXISTS "formula_template_id";

