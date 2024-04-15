-- Modify "customers" table
ALTER TABLE "customers" ALTER COLUMN "state_id" SET NOT NULL;
-- Modify "worker_profiles" table
ALTER TABLE "worker_profiles" ALTER COLUMN "license_state_id" SET NOT NULL, ADD CONSTRAINT "worker_profiles_us_states_state" FOREIGN KEY ("license_state_id") REFERENCES "us_states" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
