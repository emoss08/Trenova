DROP TABLE IF EXISTS "equipment_types" CASCADE;

-- bun:split

DROP TYPE IF EXISTS equipment_class_enum CASCADE;

-- bun:split

DROP INDEX IF EXISTS "equipment_types_code_organization_id_unq" CASCADE;