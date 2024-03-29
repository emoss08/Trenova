DROP INDEX IF EXISTS revenuecode_code_organization_id CASCADE;

CREATE UNIQUE INDEX revenuecode_code_organization_id ON revenue_codes (LOWER(code), organization_id);