-- Set comment to column: "code" on table: "tractors"
COMMENT ON COLUMN "tractors" ."code" IS 'The unique code assigned to each tractor for identification purposes.';
-- Set comment to column: "status" on table: "tractors"
COMMENT ON COLUMN "tractors" ."status" IS 'The operational status of the tractor, indicating availability, maintenance, or other conditions.';
-- Set comment to column: "license_plate_number" on table: "tractors"
COMMENT ON COLUMN "tractors" ."license_plate_number" IS 'The license plate number of the tractor, used for legal identification on roads.';
-- Set comment to column: "vin" on table: "tractors"
COMMENT ON COLUMN "tractors" ."vin" IS 'The Vehicle Identification Number, a unique code used to identify individual motor vehicles.';
-- Set comment to column: "model" on table: "tractors"
COMMENT ON COLUMN "tractors" ."model" IS 'The model of the tractor, which indicates the design and technical specifications.';
-- Set comment to column: "year" on table: "tractors"
COMMENT ON COLUMN "tractors" ."year" IS 'The year the tractor was manufactured, reflecting its age and potentially its technology level.';
-- Set comment to column: "leased" on table: "tractors"
COMMENT ON COLUMN "tractors" ."leased" IS 'Indicates whether the tractor is currently leased or owned outright.';
-- Set comment to column: "leased_date" on table: "tractors"
COMMENT ON COLUMN "tractors" ."leased_date" IS 'The date on which the tractor was leased, if applicable.';
-- Set comment to column: "equipment_type_id" on table: "tractors"
COMMENT ON COLUMN "tractors" ."equipment_type_id" IS 'Identifier for the type of equipment the tractor is classified under.';
-- Set comment to column: "equipment_manufacturer_id" on table: "tractors"
COMMENT ON COLUMN "tractors" ."equipment_manufacturer_id" IS 'The UUID of the manufacturer of the tractor''s equipment, linking to specific company details.';
-- Set comment to column: "state_id" on table: "tractors"
COMMENT ON COLUMN "tractors" ."state_id" IS 'A UUID representing the state in which the tractor is registered, for jurisdiction purposes.';
-- Set comment to column: "primary_worker_id" on table: "tractors"
COMMENT ON COLUMN "tractors" ."primary_worker_id" IS 'The primary worker assigned to operate the tractor, identified by UUID.';
-- Set comment to column: "secondary_worker_id" on table: "tractors"
COMMENT ON COLUMN "tractors" ."secondary_worker_id" IS 'An optional secondary worker who can also operate the tractor, identified by UUID.';
-- Set comment to column: "fleet_code_id" on table: "tractors"
COMMENT ON COLUMN "tractors" ."fleet_code_id" IS 'A UUID linking the tractor to a specific fleet within an organization.';
