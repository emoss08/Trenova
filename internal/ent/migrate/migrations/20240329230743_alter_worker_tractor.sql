-- Modify "tractors" table
ALTER TABLE "tractors" ADD COLUMN "secondary_worker_id" uuid NULL, ADD CONSTRAINT "tractors_workers_secondary_worker" FOREIGN KEY ("secondary_worker_id") REFERENCES "workers" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
