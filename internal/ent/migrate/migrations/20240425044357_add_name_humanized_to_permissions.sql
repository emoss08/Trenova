-- Modify "permissions" table
ALTER TABLE "permissions" ADD COLUMN "name_humanized" character varying NULL;
-- Set comment to column: "name_humanized" on table: "permissions"
COMMENT ON COLUMN "permissions" ."name_humanized" IS 'Name of the permission in human readable format.';
