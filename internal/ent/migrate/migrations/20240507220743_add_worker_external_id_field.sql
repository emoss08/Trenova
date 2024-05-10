-- Modify "workers" table
ALTER TABLE "workers" ADD COLUMN "external_id" character varying NULL;
-- Set comment to column: "external_id" on table: "workers"
COMMENT ON COLUMN "workers" ."external_id" IS 'External ID usually from HOS integration.';
