-- Modify "commodities" table
ALTER TABLE "commodities" ALTER COLUMN "min_temp" TYPE bigint, ALTER COLUMN "max_temp" TYPE bigint, DROP COLUMN "set_point_temp";
-- Modify "hazardous_materials" table
ALTER TABLE "hazardous_materials" ADD COLUMN "status" character varying NOT NULL DEFAULT 'A';
