-- Modify "workers" table
ALTER TABLE "workers" DROP COLUMN "primary_worker_id";
-- Modify "tractors" table
ALTER TABLE "tractors" ADD COLUMN "primary_worker_id" uuid NULL, ADD CONSTRAINT "tractors_workers_primary_worker" FOREIGN KEY ("primary_worker_id") REFERENCES "workers" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
