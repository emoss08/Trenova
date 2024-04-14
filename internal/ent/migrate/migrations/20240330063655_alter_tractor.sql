-- Modify "tractors" table
ALTER TABLE "tractors" ALTER COLUMN "license_plate_number" TYPE character varying(50), ALTER COLUMN "vin" TYPE character varying(17), ALTER COLUMN "model" TYPE character varying(50), ALTER COLUMN "year" TYPE smallint, ALTER COLUMN "leased_date" TYPE date;
