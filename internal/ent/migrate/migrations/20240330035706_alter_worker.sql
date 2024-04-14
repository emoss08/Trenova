-- Modify "tractors" table
ALTER TABLE "tractors" DROP CONSTRAINT "tractors_workers_primary_worker", DROP CONSTRAINT "tractors_workers_secondary_worker", ADD CONSTRAINT "tractors_workers_primary_tractor" FOREIGN KEY ("primary_worker_id") REFERENCES "workers" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, ADD CONSTRAINT "tractors_workers_secondary_tractor" FOREIGN KEY ("secondary_worker_id") REFERENCES "workers" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- Create index "tractors_primary_worker_id_key" to table: "tractors"
CREATE UNIQUE INDEX "tractors_primary_worker_id_key" ON "tractors" ("primary_worker_id");
-- Create index "tractors_secondary_worker_id_key" to table: "tractors"
CREATE UNIQUE INDEX "tractors_secondary_worker_id_key" ON "tractors" ("secondary_worker_id");
