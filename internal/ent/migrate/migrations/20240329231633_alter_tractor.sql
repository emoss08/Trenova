-- Modify "tractors" table
ALTER TABLE "tractors" ADD COLUMN "fleet_code_id" uuid NULL, ADD CONSTRAINT "tractors_fleet_codes_fleet_code" FOREIGN KEY ("fleet_code_id") REFERENCES "fleet_codes" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
