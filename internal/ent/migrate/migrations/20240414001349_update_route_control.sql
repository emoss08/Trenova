-- Modify "accessorial_charges" table
ALTER TABLE "accessorial_charges" ALTER COLUMN "amount" DROP DEFAULT;
-- Modify "business_units" table
ALTER TABLE "business_units" ALTER COLUMN "description" TYPE text;
-- Modify "route_controls" table
ALTER TABLE "route_controls" ALTER COLUMN "distance_method" TYPE character varying(8), ALTER COLUMN "distance_method" SET DEFAULT 'Trenova', ALTER COLUMN "mileage_unit" TYPE character varying(9), ALTER COLUMN "mileage_unit" SET DEFAULT 'Metric';
