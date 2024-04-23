-- Set comment to column: "code" on table: "tractors"
COMMENT ON COLUMN "tractors" ."code" IS 'Unique identifier for the tractor.';
-- Set comment to column: "status" on table: "tractors"
COMMENT ON COLUMN "tractors" ."status" IS 'Current status of the tractor.';
-- Set comment to column: "license_plate_number" on table: "tractors"
COMMENT ON COLUMN "tractors" ."license_plate_number" IS 'License plate number of the tractor.';
-- Set comment to column: "vin" on table: "tractors"
COMMENT ON COLUMN "tractors" ."vin" IS 'Vehicle Identification Number of the tractor.';
-- Set comment to column: "model" on table: "tractors"
COMMENT ON COLUMN "tractors" ."model" IS 'Model of the tractor.';
-- Set comment to column: "year" on table: "tractors"
COMMENT ON COLUMN "tractors" ."year" IS 'Year of the tractor.';
-- Set comment to column: "leased" on table: "tractors"
COMMENT ON COLUMN "tractors" ."leased" IS 'Whether the tractor is leased.';
-- Set comment to column: "leased_date" on table: "tractors"
COMMENT ON COLUMN "tractors" ."leased_date" IS 'Date the tractor was leased.';
-- Set comment to column: "equipment_type_id" on table: "tractors"
COMMENT ON COLUMN "tractors" ."equipment_type_id" IS 'Equipment type ID.';
-- Set comment to column: "equipment_manufacturer_id" on table: "tractors"
COMMENT ON COLUMN "tractors" ."equipment_manufacturer_id" IS 'Equipment manufacturer ID.';
-- Set comment to column: "state_id" on table: "tractors"
COMMENT ON COLUMN "tractors" ."state_id" IS 'State ID.';
-- Set comment to column: "primary_worker_id" on table: "tractors"
COMMENT ON COLUMN "tractors" ."primary_worker_id" IS 'Primary worker ID.';
-- Set comment to column: "secondary_worker_id" on table: "tractors"
COMMENT ON COLUMN "tractors" ."secondary_worker_id" IS 'Secondary worker ID.';
-- Set comment to column: "fleet_code_id" on table: "tractors"
COMMENT ON COLUMN "tractors" ."fleet_code_id" IS 'Fleet code ID.';
